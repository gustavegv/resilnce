package api

import (
	"context"
	"net/http"
	"strings"

	"github.com/gustavegv/resilnce/apps/server/auth"
	"github.com/gustavegv/resilnce/apps/server/db"
	"github.com/rs/cors"
)

func RoutingCreation() http.Handler {
	var siteDirectory string = auth.GetBaseURL()

	mux := http.NewServeMux()

	mux.HandleFunc("/", BaseReq)
	mux.HandleFunc("/api/logout", LogOutUser)

	// Auth
	cfg := auth.ConfigFromEnv(siteDirectory)
	ctx := context.Background()
	service, _ := auth.New(ctx, cfg)

	mux.HandleFunc("/api/me", service.GetUserInfo)
	mux.HandleFunc("/login/google", service.LoginGoogle)
	mux.HandleFunc("/oauth/google/callback", service.CallbackGoogle)

	// DB (supabase)
	// TODO: flytta till pool.go?
	pool := db.NewPool()

	supaCFG := &db.SupabaseCFG{DB: pool}
	mux.HandleFunc("/db/mySessions", supaCFG.GetUserSessions)

	// -

	originURL := trimURLSlash(siteDirectory)

	corsHandler := cors.New(cors.Options{
		AllowedOrigins: []string{
			originURL,
			"http://localhost:5173",
		},
		AllowCredentials: true,
		AllowedMethods:   []string{"GET", "POST", "OPTIONS", "PUT"},
		AllowedHeaders:   []string{"Authorization", "Content-Type"},
	}).Handler(mux)

	return corsHandler
}

func trimURLSlash(u string) string {
	if i := strings.LastIndex(u, "/"); i > len("http://") {
		return u[:i]
	}
	return u
}
