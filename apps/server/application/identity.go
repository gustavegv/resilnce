// Package application defines the shared account identity and HTTP boundary.
package application

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/mail"
	"os"
	"strconv"
	"strings"

	sessions "github.com/gustavegv/resilnce/apps/server/redis"
	scookie "github.com/gustavegv/resilnce/apps/server/scookies"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Model string

const (
	Legacy           Model = "legacy"
	ExerciseCentric  Model = "exercise-centric"
	DevelopmentEmail       = "development@resilnce.invalid"
)

type Config struct {
	DevelopmentMode  bool
	DevelopmentEmail string
	Model            Model
	WriteMaintenance bool
}

func (c Config) Validate() (Config, error) {
	if c.Model == "" {
		c.Model = Legacy
	}
	if c.Model != Legacy && c.Model != ExerciseCentric {
		return c, errors.New("APPLICATION_MODEL must be legacy or exercise-centric")
	}
	if c.DevelopmentEmail == "" {
		c.DevelopmentEmail = DevelopmentEmail
	}
	address, err := mail.ParseAddress(c.DevelopmentEmail)
	if err != nil || address.Address != c.DevelopmentEmail || !strings.HasSuffix(c.DevelopmentEmail, ".invalid") {
		return c, errors.New("development account must use a dedicated .invalid email address")
	}
	return c, nil
}

func ConfigFromEnv() (Config, error) {
	c := Config{Model: Model(os.Getenv("APPLICATION_MODEL")), DevelopmentEmail: os.Getenv("DEVELOPMENT_ACCOUNT_EMAIL")}
	for _, flag := range []struct {
		name  string
		value *bool
	}{{"DEVELOPMENT_ACCOUNT_ENABLED", &c.DevelopmentMode}, {"TRAINING_WRITE_MAINTENANCE", &c.WriteMaintenance}} {
		if raw := os.Getenv(flag.name); raw != "" {
			v, err := strconv.ParseBool(raw)
			if err != nil {
				return c, fmt.Errorf("%s must be a boolean", flag.name)
			}
			*flag.value = v
		}
	}
	return c.Validate()
}

type Identity struct {
	Email           string
	Name            string
	DevelopmentMode bool
}

func (i Identity) QuotaKey() string {
	return fmt.Sprintf("ai:account:%x", sha256.Sum256([]byte(i.Email)))
}

type Resolver func(*http.Request) (Identity, error)
type SessionLookup interface {
	GetSession(context.Context, string) (*sessions.SessionData, error)
}

func NormalIdentity(store SessionLookup) Resolver {
	return func(r *http.Request) (Identity, error) {
		sid, email, _, err := scookie.ValidateSignedCookie(r)
		if err != nil || email == "" || store == nil {
			return Identity{}, errors.New("sign in to continue")
		}
		data, err := store.GetSession(r.Context(), sid)
		if err != nil || data == nil || data.Email != email {
			return Identity{}, errors.New("session expired or identity changed; sign in again")
		}
		return Identity{Email: email, Name: data.Name}, nil
	}
}

// ProvisionDevelopmentAccount never creates a login session or changes existing data.
func ProvisionDevelopmentAccount(ctx context.Context, pool *pgxpool.Pool, cfg Config) (Resolver, error) {
	cfg, err := cfg.Validate()
	if err != nil {
		return nil, err
	}
	if !cfg.DevelopmentMode {
		return nil, errors.New("development identity is disabled")
	}
	if _, err := pool.Exec(ctx, `INSERT INTO public."User" (mail,name,signup_date) VALUES ($1,'Development account',now()) ON CONFLICT (mail) DO NOTHING`, cfg.DevelopmentEmail); err != nil {
		return nil, err
	}
	return func(r *http.Request) (Identity, error) {
		i := Identity{Email: cfg.DevelopmentEmail, DevelopmentMode: true}
		err := pool.QueryRow(r.Context(), `SELECT COALESCE(name,'Development account') FROM public."User" WHERE mail=$1`, i.Email).Scan(&i.Name)
		return i, err
	}, nil
}

type Capabilities struct {
	ApplicationModel Model `json:"applicationModel"`
	TrainingWrites   bool  `json:"trainingWrites"`
	WriteMaintenance bool  `json:"writeMaintenance"`
}
type Account struct {
	ID              string       `json:"id"`
	Email           string       `json:"email"`
	Name            string       `json:"name"`
	DevelopmentMode bool         `json:"developmentMode"`
	Capabilities    Capabilities `json:"capabilities"`
}

type identityKey struct{}

func FromContext(ctx context.Context) (Identity, bool) {
	i, ok := ctx.Value(identityKey{}).(Identity)
	return i, ok
}

type Boundary struct {
	Resolve Resolver
	Config  Config
	Origin  string
}

func (b Boundary) ModelFor(i Identity) Model {
	if i.DevelopmentMode {
		return ExerciseCentric
	}
	if b.Config.Model == "" {
		return Legacy
	}
	return b.Config.Model
}

// Handler authenticates once, applies method/origin/model/write policy, and binds
// the authoritative identity for downstream application operations.
func (b Boundary) Handler(method string, model Model, write bool, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != method {
			w.Header().Set("Allow", method)
			WriteError(w, 405, "method-not-allowed", "Use "+method, nil)
			return
		}
		i, err := b.Resolve(r)
		if err != nil || i.Email == "" {
			WriteError(w, 401, "unauthenticated", "Sign in to continue", nil)
			return
		}
		if write {
			if b.Config.WriteMaintenance {
				WriteError(w, 503, "write-maintenance", "Training writes are paused; try again after maintenance", nil)
				return
			}
			if model == Legacy && b.ModelFor(i) != Legacy {
				WriteError(w, 426, "upgrade-required", "Refresh the application before saving", nil)
				return
			}
			if r.Header.Get("Origin") != strings.TrimRight(b.Origin, "/") || b.Origin == "" {
				WriteError(w, 403, "origin-forbidden", "Use the configured application origin", nil)
				return
			}
		}
		if model == ExerciseCentric && b.ModelFor(i) != ExerciseCentric {
			WriteError(w, 409, "model-unavailable", "Exercise-centric training is not enabled for this account", nil)
			return
		}
		next(w, r.WithContext(context.WithValue(r.Context(), identityKey{}, i)))
	}
}

func (b Boundary) Account(w http.ResponseWriter, r *http.Request) {
	i, _ := FromContext(r.Context())
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	json.NewEncoder(w).Encode(Account{ID: i.Email, Email: i.Email, Name: i.Name, DevelopmentMode: i.DevelopmentMode, Capabilities: Capabilities{ApplicationModel: b.ModelFor(i), TrainingWrites: !b.Config.WriteMaintenance, WriteMaintenance: b.Config.WriteMaintenance}})
}

type ErrorResponse struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Details any    `json:"details,omitempty"`
}

func WriteError(w http.ResponseWriter, status int, code, message string, details any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(ErrorResponse{Code: code, Message: message, Details: details})
}
