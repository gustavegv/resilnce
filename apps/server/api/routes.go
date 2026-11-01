package api

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/gustavegv/resilnce/apps/server/ai"
	"github.com/gustavegv/resilnce/apps/server/application"
	"github.com/gustavegv/resilnce/apps/server/auth"
	"github.com/gustavegv/resilnce/apps/server/db"
	"github.com/gustavegv/resilnce/apps/server/porting"
	sessions "github.com/gustavegv/resilnce/apps/server/redis"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/cors"
)

// Options keeps provider/network bootstrap outside the application HTTP seam.
type Options struct {
	Pool          *pgxpool.Pool
	Sessions      application.SessionLookup
	RedisStore    *sessions.Store
	Application   application.Config
	Resolve       application.Resolver
	Origin        string
	Auth          *auth.Service
	AIQuota       ai.Quota
	GenerateDraft ai.Generator
	// RegisterTraining mounts new operations with the same identity and policy.
	RegisterTraining func(*http.ServeMux, application.Boundary)
}

func RoutingCreation() http.Handler {
	cfg, err := application.ConfigFromEnv()
	if err != nil {
		log.Fatal(err)
	}
	options := Options{Pool: db.NewPool(), Application: cfg, Origin: auth.GetBaseURL()}
	if cfg.DevelopmentMode {
		// No OAuth configuration, provider discovery, or synthetic login in dev mode.
		if os.Getenv("REDIS_ADDR") != "" {
			options.RedisStore, err = sessions.NewStore()
			if err != nil {
				log.Fatal(err)
			}
			options.Sessions = options.RedisStore
		}
	} else {
		options.Auth, err = auth.New(context.Background(), auth.ConfigFromEnv(options.Origin))
		if err != nil {
			log.Fatal(err)
		}
		options.RedisStore = options.Auth.RedisStore
		options.Sessions = options.RedisStore
	}
	h, err := NewHandler(context.Background(), options)
	if err != nil {
		log.Fatal(err)
	}
	return h
}

func NewHandler(ctx context.Context, options Options) (http.Handler, error) {
	cfg, err := options.Application.Validate()
	if err != nil {
		return nil, err
	}
	if options.Pool == nil {
		return nil, fmt.Errorf("application database is required")
	}
	resolve := options.Resolve
	if cfg.DevelopmentMode {
		resolve, err = application.ProvisionDevelopmentAccount(ctx, options.Pool, cfg)
		if err != nil {
			return nil, err
		}
	} else if resolve == nil {
		resolve = application.NormalIdentity(options.Sessions)
	}
	boundary := application.Boundary{Resolve: resolve, Config: cfg, Origin: strings.TrimRight(options.Origin, "/")}
	mux := http.NewServeMux()
	mux.HandleFunc("/", BaseReq)
	mux.HandleFunc("/api/logout", boundary.Handler("POST", "", false, LogOutUser))
	mux.HandleFunc("/api/me", boundary.Handler("GET", "", false, boundary.Account))
	supaCFG := &db.SupabaseCFG{DB: options.Pool, RedisStore: options.RedisStore}
	if options.Auth != nil && !cfg.DevelopmentMode {
		options.Auth.SBDB = *supaCFG
		mux.HandleFunc("/login/google", options.Auth.LoginGoogle)
		mux.HandleFunc("/oauth/google/callback", options.Auth.CallbackGoogle)
	} else {
		unavailable := func(w http.ResponseWriter, r *http.Request) {
			application.WriteError(w, 503, "login-unavailable", "Normal login is unavailable on this server", nil)
		}
		mux.HandleFunc("/login/google", unavailable)
		mux.HandleFunc("/oauth/google/callback", unavailable)
	}
	porting.HTTP{
		Service:      &porting.Service{Pool: options.Pool, AllowApply: porting.ApplyEnabled()},
		Authenticate: func(r *http.Request) (string, error) { return auth.VerifiedIdentity(r, options.Sessions) },
		Origin:       options.Origin,
	}.Register(mux)
	reads := map[string]http.HandlerFunc{
		"/db/mySessions": supaCFG.GetUserSessions, "/db/getExercises": supaCFG.GetSessionExercises,
		"/db/history": supaCFG.GetExerciseHistory, "/db/active": supaCFG.GetActiveSession,
		"/db/finished": supaCFG.GetFinishedExercises,
	}
	for path, h := range reads {
		mux.HandleFunc(path, boundary.Handler("GET", "", false, h))
	}
	writes := map[string]http.HandlerFunc{
		"/db/update": supaCFG.CallUpdateExercise, "/db/edit": supaCFG.CallEditExercise,
		"/db/complete": supaCFG.CallCompleteExercise, "/db/setActive": supaCFG.CallSetActiveSession,
		"/db/newSession": supaCFG.MakeNewSession, "/db/editSession": supaCFG.EditSessionH,
		"/db/addSessionExercises":    supaCFG.AddSessionExercisesH,
		"/db/removeSessionExercises": supaCFG.RemoveSessionExercisesH, "/db/deleteSession": supaCFG.DeleteSessionH,
	}
	for path, h := range writes {
		mux.HandleFunc(path, boundary.Handler("POST", application.Legacy, true, h))
	}
	quota := options.AIQuota
	if quota == nil && options.RedisStore != nil {
		quota = options.RedisStore.TakeAICall
	}
	mux.HandleFunc("/ai/quick", boundary.Handler("POST", "", false, ai.HTTP(quota, options.GenerateDraft, boundary.Origin)))
	if options.RegisterTraining != nil {
		options.RegisterTraining(mux, boundary)
	}
	return cors.New(cors.Options{
		AllowedOrigins: []string{boundary.Origin, "http://localhost:5173"}, AllowCredentials: true,
		AllowedMethods: []string{"GET", "POST", "OPTIONS", "PUT", "PATCH", "DELETE"},
		AllowedHeaders: []string{"Authorization", "Content-Type"},
	}).Handler(mux), nil
}
