package porting

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestInternalAPIDeniesBeforeReadingPrivateData(t *testing.T) {
	paths := []struct{ method, path, body string }{
		{"GET", "schema", ""}, {"POST", "schema/apply", `{}`},
		{"GET", "overview", ""}, {"POST", "audits", `{}`}, {"POST", "runs", `{}`},
		{"GET", "runs/aaaaaaaa-aaaa-4aaa-aaaa-aaaaaaaaaaaa", ""},
		{"POST", "runs/aaaaaaaa-aaaa-4aaa-aaaa-aaaaaaaaaaaa/resume", `{}`},
	}
	for _, path := range paths {
		for _, tc := range []struct {
			email string
			err   error
			want  int
		}{
			{"", errors.New("no session"), 401}, {"someone@example.com", nil, 403},
		} {
			t.Run(path.method+path.path+tc.email, func(t *testing.T) {
				h := HTTP{Authenticate: func(*http.Request) (string, error) { return tc.email, tc.err }, Origin: "https://app.example"}
				r := httptest.NewRequest(path.method, "/internal/porting/"+path.path, strings.NewReader(path.body))
				r.Header.Set("Origin", "https://app.example")
				r.Header.Set("Content-Type", "application/json")
				w := httptest.NewRecorder()
				h.ServeHTTP(w, r)
				if w.Code != tc.want {
					t.Fatalf("got %d", w.Code)
				}
			})
		}
	}
}

func TestInternalMutationsRequireConfiguredOrigin(t *testing.T) {
	for _, path := range []string{"runs", "schema/apply"} {
		for _, origin := range []string{"", "https://evil.example", "null"} {
			h := HTTP{Authenticate: func(*http.Request) (string, error) { return "gustavengvall@yahoo.se", nil }, Origin: "https://app.example/"}
			r := httptest.NewRequest("POST", "/internal/porting/"+path, strings.NewReader(`{}`))
			r.Header.Set("Origin", origin)
			r.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()
			h.ServeHTTP(w, r)
			if w.Code != 403 {
				t.Fatalf("origin %q: %d", origin, w.Code)
			}
		}
	}
}
