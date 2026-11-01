package auth

import (
	"context"
	"errors"
	"net/http"

	sessions "github.com/gustavegv/resilnce/apps/server/redis"
	scookie "github.com/gustavegv/resilnce/apps/server/scookies"
)

type SessionLookup interface {
	GetSession(context.Context, string) (*sessions.SessionData, error)
}

// VerifiedIdentity binds a valid signed cookie to a live server-side session.
// Old sessions have no verification claim and require a renewed Google login.
func VerifiedIdentity(r *http.Request, store SessionLookup) (string, error) {
	sid, mail, _, err := scookie.ValidateSignedCookie(r)
	if err != nil {
		return "", err
	}
	if store == nil {
		return "", errors.New("renew sign-in with a verified email")
	}
	data, err := store.GetSession(r.Context(), sid)
	if err != nil || data == nil || mail == "" || data.Email != mail || !data.EmailVerified {
		return "", errors.New("renew sign-in with a verified email")
	}
	return mail, nil
}
