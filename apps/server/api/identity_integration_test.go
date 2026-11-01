//go:build integration

package api

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gustavegv/resilnce/apps/server/application"
	"github.com/gustavegv/resilnce/apps/server/internal/testdb"
	"github.com/gustavegv/resilnce/apps/server/migrations"
	sessions "github.com/gustavegv/resilnce/apps/server/redis"
	scookie "github.com/gustavegv/resilnce/apps/server/scookies"
)

func readAccount(t *testing.T, h http.Handler) application.Account {
	t.Helper()
	w := httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest("GET", "/api/me?mail=other@example.com", nil))
	if w.Code != 200 {
		t.Fatalf("account: %d %s", w.Code, w.Body.String())
	}
	var account application.Account
	if err := json.Unmarshal(w.Body.Bytes(), &account); err != nil {
		t.Fatal(err)
	}
	return account
}

func TestAIUsesPersistentSelectedAccountQuotaWithoutLogin(t *testing.T) {
	pool, ctx := testdb.New(t, "../porting/testdata/legacy.sql")
	counts := map[string]int{}
	options := Options{Pool: pool, Application: application.Config{DevelopmentMode: true}, Origin: "https://app.example",
		AIQuota: func(_ context.Context, key string, limit int64) (bool, error) {
			counts[key]++
			return counts[key] <= int(limit), nil
		},
		GenerateDraft: func(context.Context, []bool, string) (string, error) { return `{"sessionName":"Draft"}`, nil },
	}
	for restart := 0; restart < 2; restart++ {
		h, err := NewHandler(ctx, options)
		if err != nil {
			t.Fatal(err)
		}
		for call := 0; call < 13; call++ {
			r := httptest.NewRequest("POST", "/ai/quick?mail=someone@example.com", strings.NewReader(`{"userInput":"legs","promptSelections":[true]}`))
			r.Header.Set("Origin", "https://app.example")
			w := httptest.NewRecorder()
			h.ServeHTTP(w, r)
			want := 200
			if restart == 1 && call == 12 {
				want = 403
			}
			if w.Code != want {
				t.Fatalf("restart %d call %d: %d %s", restart, call, w.Code, w.Body.String())
			}
		}
	}
	if len(counts) != 1 {
		t.Fatalf("quota identity changed across restart: %v", counts)
	}
}

func TestDevelopmentAccountIsProvisionedOnceWithoutOAuthAndSurvivesRestart(t *testing.T) {
	pool, ctx := testdb.New(t, "../porting/testdata/legacy.sql")
	if err := migrations.Apply(ctx, pool); err != nil {
		t.Fatal(err)
	}
	options := Options{Pool: pool, Application: application.Config{DevelopmentMode: true}, Origin: "https://app.example"}
	h, err := NewHandler(ctx, options)
	if err != nil {
		t.Fatal(err)
	}
	first := readAccount(t, h)
	if first.ID != "development@resilnce.invalid" || !first.DevelopmentMode || first.Capabilities.ApplicationModel != application.ExerciseCentric {
		t.Fatalf("wrong authoritative account: %+v", first)
	}
	var library, workouts int
	if err := pool.QueryRow(ctx, `SELECT (SELECT count(*) FROM public.exercise),(SELECT count(*) FROM public.workout)`).Scan(&library, &workouts); err != nil {
		t.Fatal(err)
	}
	if library != 0 || workouts != 0 {
		t.Fatal("provisioning invented training data")
	}
	// Existing account data is fixture setup; startup must never overwrite it.
	if _, err := pool.Exec(ctx, `UPDATE public."User" SET name='Retained test account' WHERE mail=$1;`, first.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `
INSERT INTO public.exercise(id,owner_mail,name) VALUES ('10000000-0000-4000-8000-000000000001','development@resilnce.invalid','Retained lift');
INSERT INTO public.training_session(id,owner_mail,name) VALUES ('10000000-0000-4000-8000-000000000002','development@resilnce.invalid','Retained plan');
INSERT INTO public.workout(id,owner_mail,source_session_id,origin,status) VALUES ('10000000-0000-4000-8000-000000000003','development@resilnce.invalid','10000000-0000-4000-8000-000000000002','session','active');`); err != nil {
		t.Fatal(err)
	}
	h, err = NewHandler(ctx, options)
	if err != nil {
		t.Fatal(err)
	}
	second := readAccount(t, h)
	if second.ID != first.ID || second.Name != "Retained test account" {
		t.Fatalf("restart reset account: %+v", second)
	}
	w := httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest("GET", "/db/mySessions?mail=other@example.com", nil))
	if w.Code != 200 || w.Body.String() != "[]\n" {
		t.Fatalf("new account must be empty: %d %s", w.Code, w.Body.String())
	}
	options.Application.DevelopmentMode = false
	h, err = NewHandler(ctx, options)
	if err != nil {
		t.Fatal(err)
	}
	w = httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest("GET", "/api/me", nil))
	if w.Code != 401 {
		t.Fatalf("disabling development mode manufactured login: %d %s", w.Code, w.Body.String())
	}
	options.Application.DevelopmentMode = true
	h, err = NewHandler(ctx, options)
	if err != nil {
		t.Fatal(err)
	}
	if got := readAccount(t, h); got.Name != "Retained test account" {
		t.Fatal("disabling mode deleted account data")
	}
	// Persistence invariants cover the domain tables before their read operations
	// are introduced by the following vertical slices.
	if err := pool.QueryRow(ctx, `SELECT (SELECT count(*) FROM public.exercise WHERE name='Retained lift'),(SELECT count(*) FROM public.workout WHERE status='active')`).Scan(&library, &workouts); err != nil {
		t.Fatal(err)
	}
	if library != 1 || workouts != 1 {
		t.Fatal("restarting or disabling development identity reset training data")
	}
}

type loginStore struct{ data *sessions.SessionData }

func (s *loginStore) GetSession(context.Context, string) (*sessions.SessionData, error) {
	return s.data, nil
}

func signedLogin(t *testing.T, email string, expired bool) *http.Cookie {
	t.Helper()
	secret := make([]byte, 32)
	t.Setenv("SIGNED_COOKIE_SECRET", base64.RawURLEncoding.EncodeToString(secret))
	payload := scookie.BuildPayload(email, "provider-subject", "Untrusted cookie name")
	if expired {
		payload.Exp = 1
	}
	value, err := scookie.Create(payload, secret)
	if err != nil {
		t.Fatal(err)
	}
	return &http.Cookie{Name: "__Host-SignedCookie", Value: value}
}

func request(t *testing.T, h http.Handler, method, path, body, origin string, cookie *http.Cookie) *httptest.ResponseRecorder {
	t.Helper()
	r := httptest.NewRequest(method, path, strings.NewReader(body))
	if cookie != nil {
		r.AddCookie(cookie)
	}
	if origin != "" {
		r.Header.Set("Origin", origin)
	}
	r.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	return w
}

func TestNormalAccountBindsCookieAndLiveSessionAcrossAccountTrainingAndAI(t *testing.T) {
	pool, ctx := testdb.New(t, "../porting/testdata/legacy.sql")
	if _, err := pool.Exec(ctx, `INSERT INTO public."User" (mail,name) VALUES ('owner@example.com','Owner'),('other@example.com','Other'); INSERT INTO public."Session" VALUES (1,'Owned',NULL,'owner@example.com'),(2,'Other plan',NULL,'other@example.com');`); err != nil {
		t.Fatal(err)
	}
	store := &loginStore{}
	var quotaKey string
	h, err := NewHandler(ctx, Options{Pool: pool, Sessions: store, Origin: "https://app.example",
		AIQuota:       func(_ context.Context, key string, _ int64) (bool, error) { quotaKey = key; return true, nil },
		GenerateDraft: func(context.Context, []bool, string) (string, error) { return `{}`, nil },
	})
	if err != nil {
		t.Fatal(err)
	}
	cookie := signedLogin(t, "owner@example.com", false)
	paths := []struct{ method, path, body string }{{"GET", "/api/me?mail=other@example.com", ""}, {"GET", "/db/mySessions?mail=other@example.com", ""}, {"POST", "/ai/quick?mail=other@example.com", `{"userInput":"legs","promptSelections":[]}`}}
	for _, data := range []*sessions.SessionData{nil, {Email: "other@example.com", Name: "Other"}} {
		store.data = data
		for _, endpoint := range paths {
			w := request(t, h, endpoint.method, endpoint.path, endpoint.body, "https://app.example", cookie)
			if w.Code != 401 {
				t.Fatalf("invalid live session admitted %s: %d %s", endpoint.path, w.Code, w.Body.String())
			}
		}
	}
	store.data = &sessions.SessionData{Email: "owner@example.com", Name: "Authoritative owner"}
	for _, bad := range []*http.Cookie{signedLogin(t, "owner@example.com", true), {Name: "__Host-SignedCookie", Value: cookie.Value + "tampered"}, nil} {
		for _, endpoint := range paths {
			w := request(t, h, endpoint.method, endpoint.path, endpoint.body, "https://app.example", bad)
			if w.Code != 401 {
				t.Fatalf("invalid cookie admitted %s: %d", endpoint.path, w.Code)
			}
		}
	}
	w := request(t, h, "GET", paths[0].path, "", "", cookie)
	var account application.Account
	if err := json.Unmarshal(w.Body.Bytes(), &account); err != nil {
		t.Fatal(err)
	}
	if account.Email != "owner@example.com" || account.Name != "Authoritative owner" || account.DevelopmentMode || account.Capabilities.ApplicationModel != application.Legacy {
		t.Fatalf("wrong normal account: %+v", account)
	}
	w = request(t, h, "GET", paths[1].path, "", "", cookie)
	if w.Code != 200 || !strings.Contains(w.Body.String(), "Owned") || strings.Contains(w.Body.String(), "Other plan") {
		t.Fatalf("browser owner changed training identity: %d %s", w.Code, w.Body.String())
	}
	w = request(t, h, "POST", paths[2].path, paths[2].body, "https://app.example", cookie)
	if w.Code != 200 || quotaKey != (application.Identity{Email: account.Email}).QuotaKey() {
		t.Fatalf("AI identity differs: %d %s %q", w.Code, w.Body.String(), quotaKey)
	}
	// Legacy mode remains usable, with ownership from the signed/live identity.
	w = request(t, h, "POST", "/db/editSession?sesID=1&mail=other@example.com", `{"name":"Changed"}`, "https://app.example", cookie)
	if w.Code != 200 {
		t.Fatalf("legacy mutation failed: %d %s", w.Code, w.Body.String())
	}
	w = request(t, h, "GET", paths[1].path, "", "", cookie)
	if !strings.Contains(w.Body.String(), "Changed") {
		t.Fatal("legacy update unavailable")
	}
	w = request(t, h, "POST", "/db/editSession?sesID=1", `{"name":"Untrusted"}`, "https://evil.example", cookie)
	if w.Code != 403 {
		t.Fatalf("untrusted origin allowed: %d", w.Code)
	}
}

var legacyMutationPaths = []string{"update", "edit", "complete", "setActive", "newSession", "editSession", "addSessionExercises", "removeSessionExercises", "deleteSession"}

func TestModelSwitchRejectsEveryLegacyMutationBeforeDecodingAndMaintenancePreservesReads(t *testing.T) {
	pool, ctx := testdb.New(t, "../porting/testdata/legacy.sql")
	cookie := signedLogin(t, "owner@example.com", false)
	store := &loginStore{data: &sessions.SessionData{Email: "owner@example.com", Name: "Owner"}}
	for _, model := range []application.Model{application.Legacy, application.ExerciseCentric} {
		for _, frozen := range []bool{false, true} {
			options := Options{Pool: pool, Sessions: store, Origin: "https://app.example", Application: application.Config{Model: model, WriteMaintenance: frozen}}
			reached := false
			options.RegisterTraining = func(mux *http.ServeMux, b application.Boundary) {
				mux.HandleFunc("/api/training/test", b.Handler("POST", application.ExerciseCentric, true, func(w http.ResponseWriter, r *http.Request) { reached = true; w.WriteHeader(200) }))
			}
			h, err := NewHandler(ctx, options)
			if err != nil {
				t.Fatal(err)
			}
			for _, path := range legacyMutationPaths {
				if model == application.Legacy && !frozen {
					continue
				}
				w := request(t, h, "POST", "/db/"+path, "{", "https://app.example", cookie)
				want, code := 426, "upgrade-required"
				if frozen {
					want, code = 503, "write-maintenance"
				}
				if w.Code != want || !strings.Contains(w.Body.String(), code) {
					t.Fatalf("%s model=%s frozen=%v: %d %s", path, model, frozen, w.Code, w.Body.String())
				}
			}
			w := request(t, h, "GET", "/api/me", "", "", cookie)
			if w.Code != 200 {
				t.Fatalf("maintenance blocked read: %d", w.Code)
			}
			w = request(t, h, "POST", "/api/training/test", "{}", "https://app.example", cookie)
			want := 200
			if model == application.Legacy {
				want = 409
			}
			if frozen {
				want = 503
			}
			if w.Code != want || reached != (want == 200) {
				t.Fatalf("new model policy failed: %d %s", w.Code, w.Body.String())
			}
			w = request(t, h, "GET", "/db/deleteSession", "", "", cookie)
			if w.Code != 405 {
				t.Fatalf("mutation GET allowed: %d", w.Code)
			}
		}
	}
}

func TestDevelopmentIdentityNeverGrantsImporterAuthorization(t *testing.T) {
	pool, ctx := testdb.New(t, "../porting/testdata/legacy.sql")
	store := &loginStore{}
	options := Options{Pool: pool, Sessions: store, Application: application.Config{DevelopmentMode: true, WriteMaintenance: true}, Origin: "https://app.example"}
	h, err := NewHandler(ctx, options)
	if err != nil {
		t.Fatal(err)
	}
	w := request(t, h, "GET", "/internal/porting/schema", "", "", nil)
	if w.Code != 401 {
		t.Fatalf("development identity reached importer: %d %s", w.Code, w.Body.String())
	}
	cookie := signedLogin(t, "gustavengvall@yahoo.se", false)
	for _, data := range []*sessions.SessionData{nil, {Email: "gustavengvall@yahoo.se"}, {Email: "other@example.com", EmailVerified: true}} {
		store.data = data
		w = request(t, h, "GET", "/internal/porting/schema", "", "", cookie)
		if w.Code != 401 {
			t.Fatalf("unverified or mismatched importer session admitted: %d", w.Code)
		}
	}
	store.data = &sessions.SessionData{Email: "owner@example.com", EmailVerified: true}
	w = request(t, h, "GET", "/internal/porting/schema", "", "", signedLogin(t, "owner@example.com", false))
	if w.Code != 403 {
		t.Fatalf("nonallowlisted login admitted: %d", w.Code)
	}
	store.data = &sessions.SessionData{Email: "gustavengvall@yahoo.se", EmailVerified: true}
	w = request(t, h, "GET", "/internal/porting/schema", "", "", cookie)
	if w.Code != 200 {
		t.Fatalf("separate verified administration unavailable during development/maintenance: %d %s", w.Code, w.Body.String())
	}
	w = request(t, h, "POST", "/internal/porting/schema/apply", "{}", "https://evil.example", cookie)
	if w.Code != 403 {
		t.Fatalf("import origin policy bypassed: %d", w.Code)
	}
}

func TestApplicationRejectsInvalidBackendSelectionBeforeProvisioning(t *testing.T) {
	pool, ctx := testdb.New(t, "../porting/testdata/legacy.sql")
	for _, cfg := range []application.Config{{DevelopmentMode: true, Model: "typo"}, {DevelopmentMode: true, DevelopmentEmail: "real@example.com"}, {DevelopmentMode: true, DevelopmentEmail: "gustavengvall@yahoo.se"}} {
		if _, err := NewHandler(ctx, Options{Pool: pool, Application: cfg}); err == nil {
			t.Fatalf("unsafe/invalid config accepted: %+v", cfg)
		}
	}
	var count int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM public."User"`).Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 0 {
		t.Fatal("invalid configuration provisioned an account")
	}
}
