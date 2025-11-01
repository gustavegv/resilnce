package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/coreos/go-oidc/v3/oidc"
	scookie "github.com/gustavegv/resilnce/apps/server/scookies"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

type Config struct {
	BaseURL string

	GoogleClientID     string
	GoogleClientSecret string

	AppleClientID string // service ID
	AppleTeamID   string
	AppleKeyID    string
	AppleKeyPEM   string // contents of the .p8 private key

	StateCookieName string // "oauth_state"
	PKCECookieName  string // "oauth_pkce"
	SecureCookies   bool

	StateTTLSeconds int // default 600
	PKCETTLSeconds  int // default 600
}

func UpsertUser(provider, providerID, email, name string) (userID string, err error) {
	// TODO: save data in redis
	return email, nil
}

func (s *service) IssueSignedCookie(w http.ResponseWriter, r *http.Request, userID string) error {
	pl := scookie.BuildPayload(userID)

	sec := scookie.GetSecret()

	cookie, err := scookie.Create(pl, sec)
	if err != nil {
		return errors.New("make signed cookie failed")
	}

	const ttl = 86400 * 120
	http.SetCookie(w, &http.Cookie{
		Name:     "SignedCookie",
		Value:    cookie,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Secure:   true,
		MaxAge:   ttl,
	})

	return nil
}

type service struct {
	cfg            Config
	googleOAuth    *oauth2.Config
	googleVerifier *oidc.IDTokenVerifier
	appleIssuerURL string
	appleTokenURL  string
	appleAuthURL   string
	appleVerifier  *oidc.IDTokenVerifier
	appleOAuth     *oauth2.Config

	stateCookieAttrs cookieAttrs
	pkceCookieAttrs  cookieAttrs
}

type cookieAttrs struct {
	Path     string
	HTTPOnly bool
	SameSite http.SameSite
	Secure   bool
	MaxAge   int
}

// New constructs the service with provider configs and OIDC verifiers.
func New(ctx context.Context, cfg Config) (*service, error) {
	if cfg.StateCookieName == "" {
		cfg.StateCookieName = "oauth_state"
	}
	if cfg.PKCECookieName == "" {
		cfg.PKCECookieName = "oauth_pkce"
	}
	if cfg.StateTTLSeconds == 0 {
		cfg.StateTTLSeconds = 600
	}
	if cfg.PKCETTLSeconds == 0 {
		cfg.PKCETTLSeconds = 600
	}

	// Google
	googleProvider, err := oidc.NewProvider(ctx, "https://accounts.google.com")
	if err != nil {
		return nil, fmt.Errorf("google provider: %w", err)
	}
	googleVerifier := googleProvider.Verifier(&oidc.Config{ClientID: cfg.GoogleClientID})
	googleOAuth := &oauth2.Config{
		ClientID:     cfg.GoogleClientID,
		ClientSecret: cfg.GoogleClientSecret,
		Endpoint:     google.Endpoint,
		RedirectURL:  strings.TrimRight(cfg.BaseURL, "/") + "/oauth/google/callback",
		Scopes:       []string{"openid", "email", "profile"},
	}

	// Apple
	appleIssuer := "https://appleid.apple.com"
	appleAuthURL := "https://appleid.apple.com/auth/authorize"
	appleTokenURL := "https://appleid.apple.com/auth/token"

	appleProvider, err := oidc.NewProvider(ctx, appleIssuer)
	if err != nil {
		return nil, fmt.Errorf("apple provider: %w", err)
	}
	appleVerifier := appleProvider.Verifier(&oidc.Config{ClientID: cfg.AppleClientID})
	appleOAuth := &oauth2.Config{
		ClientID: cfg.AppleClientID,
		Endpoint: oauth2.Endpoint{
			AuthURL:  appleAuthURL,
			TokenURL: appleTokenURL,
		},
		RedirectURL: strings.TrimRight(cfg.BaseURL, "/") + "/oauth/apple/callback",
		Scopes:      []string{"openid", "email", "name"},
	}

	s := &service{
		cfg:              cfg,
		googleOAuth:      googleOAuth,
		googleVerifier:   googleVerifier,
		appleIssuerURL:   appleIssuer,
		appleAuthURL:     appleAuthURL,
		appleTokenURL:    appleTokenURL,
		appleVerifier:    appleVerifier,
		appleOAuth:       appleOAuth,
		stateCookieAttrs: cookieAttrs{Path: "/", HTTPOnly: true, SameSite: http.SameSiteLaxMode, Secure: cfg.SecureCookies, MaxAge: cfg.StateTTLSeconds},
		pkceCookieAttrs:  cookieAttrs{Path: "/", HTTPOnly: true, SameSite: http.SameSiteLaxMode, Secure: cfg.SecureCookies, MaxAge: cfg.PKCETTLSeconds},
	}
	return s, nil
}

func (s *service) setCookie(w http.ResponseWriter, name, value string, a cookieAttrs) {
	http.SetCookie(w, &http.Cookie{
		Name:     name,
		Value:    value,
		Path:     a.Path,
		HttpOnly: a.HTTPOnly,
		SameSite: a.SameSite,
		Secure:   a.Secure,
		MaxAge:   a.MaxAge,
	})
}

func (s *service) clearCookie(w http.ResponseWriter, name string) {
	http.SetCookie(w, &http.Cookie{
		Name:     name,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Secure:   s.cfg.SecureCookies,
	})
}

func (s *service) getCookie(r *http.Request, name string) (string, error) {
	c, err := r.Cookie(name)
	if err != nil {
		return "", err
	}
	return c.Value, nil
}

func randB64(n int) string {
	b := make([]byte, n)
	_, _ = rand.Read(b)
	return base64.RawURLEncoding.EncodeToString(b)
}

func pkceS256(verifier string) string {
	h := sha256.Sum256([]byte(verifier))
	return base64.RawURLEncoding.EncodeToString(h[:])
}

func ConfigFromEnv(baseURL string) Config {
	return Config{
		BaseURL:            baseURL,
		GoogleClientID:     os.Getenv("GOOGLE_CLIENT_ID"),
		GoogleClientSecret: os.Getenv("GOOGLE_CLIENT_SECRET"),

		AppleClientID: os.Getenv("APPLE_CLIENT_ID"),
		AppleTeamID:   os.Getenv("APPLE_TEAM_ID"),
		AppleKeyID:    os.Getenv("APPLE_KEY_ID"),
		AppleKeyPEM:   os.Getenv("APPLE_PRIVATE_KEY_PEM"),

		StateCookieName: "oauth_state",
		PKCECookieName:  "oauth_pkce",
		SecureCookies:   true,
	}
}

func GetBaseURL() string {
	base := os.Getenv("FRONTEND_BASE_URL")
	if base == "" {
		log.Fatal("No base URL found in .env")
	}
	return base
}
