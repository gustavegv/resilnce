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
	"time"

	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/gustavegv/resilnce/apps/server/db"
	sessions "github.com/gustavegv/resilnce/apps/server/redis"
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

func (s *Service) UpsertUser(ctx context.Context, provider, providerSID, email, name string) (userID string, err error) {

	var data = sessions.SessionData{
		Email: email,
		Name:  name,
	}

	err = s.store().SaveSession(ctx, providerSID, data, 120*24*time.Hour)
	if err != nil {
		log.Println("Redis SaveSession failed (auth.go)", err)
		return "", err
	}

	err = s.SBDB.AddUser(email, name, ctx)
	if err != nil {
		log.Println("SBDB SaveSession failed (auth.go)", err)
		return "", err
	}

	return email, nil
}

func (s *Service) IssueSignedCookie(w http.ResponseWriter, r *http.Request, userID, sub, name string) error {
	pl := scookie.BuildPayload(userID, sub, name)

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
		SameSite: http.SameSiteNoneMode,
		Secure:   true,
		MaxAge:   ttl,
	})

	return nil
}

type Service struct {
	cfg            Config
	googleOAuth    *oauth2.Config
	googleVerifier *oidc.IDTokenVerifier
	appleIssuerURL string
	appleTokenURL  string
	appleAuthURL   string
	appleVerifier  *oidc.IDTokenVerifier
	appleOAuth     *oauth2.Config

	RedisStore *sessions.Store

	SBDB db.SupabaseCFG

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
func New(ctx context.Context, cfg Config) (*Service, error) {
	redisStore, err := sessions.NewStore()
	if err != nil {
		return nil, err
	}

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

	s := &Service{
		cfg:              cfg,
		googleOAuth:      googleOAuth,
		googleVerifier:   googleVerifier,
		appleIssuerURL:   appleIssuer,
		appleAuthURL:     appleAuthURL,
		appleTokenURL:    appleTokenURL,
		appleVerifier:    appleVerifier,
		appleOAuth:       appleOAuth,
		stateCookieAttrs: cookieAttrs{Path: "/", HTTPOnly: true, SameSite: http.SameSiteNoneMode, Secure: cfg.SecureCookies, MaxAge: cfg.StateTTLSeconds},
		pkceCookieAttrs:  cookieAttrs{Path: "/", HTTPOnly: true, SameSite: http.SameSiteNoneMode, Secure: cfg.SecureCookies, MaxAge: cfg.PKCETTLSeconds},
		RedisStore:       redisStore,
	}
	return s, nil
}

func (s *Service) setCookie(w http.ResponseWriter, name, value string, a cookieAttrs) {
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

func (s *Service) clearCookie(w http.ResponseWriter, name string) {
	http.SetCookie(w, &http.Cookie{
		Name:     name,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		SameSite: http.SameSiteNoneMode,
		Secure:   s.cfg.SecureCookies,
	})
}

func (s *Service) getCookie(r *http.Request, name string) (string, error) {
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
	var cfg Config = Config{
		BaseURL:         baseURL,
		StateCookieName: "oauth_state",
		PKCECookieName:  "oauth_pkce",
		SecureCookies:   true,
	}

	getOAuthEnvByVendor("google", &cfg)

	return cfg
}

func getOAuthEnvByVendor(vendor string, c *Config) {
	v := strings.ToLower(vendor)

	switch v {
	case "google":
		gCliID := os.Getenv("GOOGLE_CLIENT_ID")
		gCliSec := os.Getenv("GOOGLE_CLIENT_SECRET")
		if gCliID == "" || gCliSec == "" {
			log.Fatal("Failed to get Google ENV variables (auth.go)")
		}
		c.GoogleClientID = gCliID
		c.GoogleClientSecret = gCliSec

	case "apple":
		aCliID := os.Getenv("APPLE_CLIENT_ID")
		aTeamID := os.Getenv("APPLE_TEAM_ID")
		aKeyID := os.Getenv("APPLE_KEY_ID")
		aKeyPEM := os.Getenv("APPLE_PRIVATE_KEY_PEM")

		if aCliID == "" || aTeamID == "" || aKeyID == "" || aKeyPEM == "" {
			log.Fatal("Failed to get Apple ENV variables (auth.go)")
		}
		c.AppleClientID = aCliID
		c.AppleTeamID = aTeamID
		c.AppleKeyID = aKeyID
		c.AppleKeyPEM = aKeyPEM
	}
}

func GetBaseURL() string {
	base := os.Getenv("FRONTEND_BASE_URL")
	if base == "" {
		log.Fatal("No base URL found in .env (auth.go)")
	}
	return base
}

func (s *Service) store() *sessions.Store {
	if s == nil || s.RedisStore == nil {
		panic("auth.Service.redisStore is nil; construct Service via auth.New")
	}
	return s.RedisStore
}
