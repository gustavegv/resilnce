package api

import (
	"context"
	"net/http"
	"strings"

	"github.com/gustavegv/resilnce/apps/server/ai"
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
	pool := db.NewPool()
	supaCFG := &db.SupabaseCFG{DB: pool, RedisStore: service.RedisStore}
	service.SBDB = *supaCFG

	mux.HandleFunc("/api/me", service.GetUserInfo)
	mux.HandleFunc("/login/google", service.LoginGoogle)
	mux.HandleFunc("/oauth/google/callback", service.CallbackGoogle)

	// DB (supabase)
	mux.HandleFunc("/db/mySessions", service.SBDB.GetUserSessions)
	mux.HandleFunc("/db/getExercises", service.SBDB.GetSessionExercises)
	mux.HandleFunc("/db/active", service.SBDB.GetActiveSession)
	mux.HandleFunc("/db/finished", service.SBDB.GetFinishedExercises)
	mux.HandleFunc("/db/update", service.SBDB.CallUpdateExercise)
	mux.HandleFunc("/db/complete", service.SBDB.CallCompleteExercise)
	mux.HandleFunc("/db/setActive", service.SBDB.CallSetActiveSession)
	mux.HandleFunc("/db/newSession", service.SBDB.MakeNewSession)

	// Other
	mux.HandleFunc("/ai/quick", ai.AutoCreation(service.RedisStore))

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
