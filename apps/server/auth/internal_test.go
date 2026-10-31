package auth

import (
	"context"
	"encoding/base64"
	"net/http"
	"net/http/httptest"
	"testing"

	sessions "github.com/gustavegv/resilnce/apps/server/redis"
	scookie "github.com/gustavegv/resilnce/apps/server/scookies"
)

type identityStore struct{ data *sessions.SessionData }

func (s identityStore) GetSession(context.Context, string) (*sessions.SessionData, error) {
	return s.data, nil
}

func TestVerifiedIdentityBindsCookieToLiveVerifiedSession(t *testing.T) {
	secret := make([]byte, 32)
	t.Setenv("SIGNED_COOKIE_SECRET", base64.RawURLEncoding.EncodeToString(secret))
	cookie, err := scookie.Create(scookie.BuildPayload("gustavengvall@yahoo.se", "subject", "Gustav"), secret)
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		name string
		data *sessions.SessionData
		ok   bool
	}{
		{"expired", nil, false},
		{"old session", &sessions.SessionData{Email: "gustavengvall@yahoo.se"}, false},
		{"different identity", &sessions.SessionData{Email: "other@example.com", EmailVerified: true}, false},
		{"verified", &sessions.SessionData{Email: "gustavengvall@yahoo.se", EmailVerified: true}, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			r := httptest.NewRequest("GET", "/", nil)
			r.AddCookie(&http.Cookie{Name: "__Host-SignedCookie", Value: cookie})
			mail, err := VerifiedIdentity(r, identityStore{tc.data})
			if (err == nil) != tc.ok {
				t.Fatalf("mail %q, err %v", mail, err)
			}
		})
	}
}
