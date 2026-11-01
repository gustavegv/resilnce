package ai_test

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gustavegv/resilnce/apps/server/ai"
	"github.com/gustavegv/resilnce/apps/server/application"
)

// The production generator makes real HTTP requests. Redirect only its network
// transport to a local external-service stub; no internal generator is mocked.
func TestQuickDraftReportsExternalErrorsAndUsesSelectedAccount(t *testing.T) {
	t.Setenv("OPENAI_API_KEY", "isolated-test-key")
	t.Setenv("OPENAI_MODEL", "test-model")
	response := `{"error":{"message":"Unavailable","type":"server_error"}}`
	received := make(chan string, 2)
	external := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" || r.URL.Path != "/v1/responses" || r.Header.Get("Authorization") != "Bearer isolated-test-key" {
			t.Errorf("external request: %s %s", r.Method, r.URL.Path)
		}
		var body struct{ Model, Input string }
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Error(err)
		}
		if body.Model != "test-model" {
			t.Errorf("model: %q", body.Model)
		}
		received <- body.Input
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(response))
	}))
	defer external.Close()
	original := http.DefaultTransport
	transport := original.(*http.Transport).Clone()
	transport.Proxy = nil
	transport.TLSClientConfig = &tls.Config{InsecureSkipVerify: true} // owned local TLS stub only
	transport.DialContext = func(ctx context.Context, network, _ string) (net.Conn, error) {
		return (&net.Dialer{}).DialContext(ctx, network, external.Listener.Addr().String())
	}
	http.DefaultTransport = transport
	t.Cleanup(func() { http.DefaultTransport = original; transport.CloseIdleConnections() })
	identity := application.Identity{Email: "development@resilnce.invalid", Name: "Development"}
	boundary := application.Boundary{Resolve: func(*http.Request) (application.Identity, error) { return identity, nil }, Config: application.Config{Model: application.ExerciseCentric}, Origin: "https://app.example"}
	quotaKey := ""
	h := boundary.Handler("POST", "", false, ai.HTTP(func(_ context.Context, key string, limit int64) (bool, error) {
		quotaKey = key
		if limit != 25 {
			t.Error("AI limit changed")
		}
		return true, nil
	}, nil, "https://app.example"))
	call := func() *httptest.ResponseRecorder {
		r := httptest.NewRequest("POST", "/ai/quick?mail=other@example.com", strings.NewReader(`{"userInput":"Bench press 3 sets at 40kg","promptSelections":[false,false,false,false]}`))
		r.Header.Set("Origin", "https://app.example")
		w := httptest.NewRecorder()
		h.ServeHTTP(w, r)
		return w
	}
	w := call()
	if w.Code != 422 || !strings.Contains(w.Body.String(), "ai-generation-failed") {
		t.Fatalf("external error must not be a valid draft: %d %s", w.Code, w.Body.String())
	}
	if quotaKey != identity.QuotaKey() {
		t.Fatal("generation quota did not use selected identity")
	}
	if !strings.Contains(<-received, "Bench press 3 sets at 40kg") {
		t.Fatal("notes missing from external request")
	}
	response = `{"output":[{"type":"message","role":"assistant","content":[{"type":"output_text","text":"[{\"name\":\"Bench press\",\"sets\":3,\"weight\":40}]"}]}]}`
	w = call()
	if w.Code != 200 || w.Body.String() != `[{"name":"Bench press","sets":3,"weight":40}]` {
		t.Fatalf("review draft: %d %s", w.Code, w.Body.String())
	}
}
