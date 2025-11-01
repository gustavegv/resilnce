package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"time"

	"github.com/gustavegv/resilnce/apps/server/api"
	"github.com/joho/godotenv"
)

func main() {
	godotenv.Load()

	httpHandler := api.RoutingCreation()
	addr := ":" + getenv("PORT", "8080")
	srv := startServer(httpHandler, addr)

	shutdownServer(srv)
}

func getenv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func startServer(hh http.Handler, addr string) *http.Server {
	// seconds
	const readTO = 10
	const writeTO = 60
	const idleTO = 60

	srv := &http.Server{
		Addr:         addr,
		Handler:      hh,
		ReadTimeout:  readTO * time.Second,
		WriteTimeout: writeTO * time.Second,
		IdleTimeout:  idleTO * time.Second,
	}

	go func() {
		log.Printf("listening on %s", addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("server error: %v", err)
		}
	}()
	return srv
}

func shutdownServer(srv *http.Server) {
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt)
	<-stop
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_ = srv.Shutdown(ctx)
	log.Println("shutdown complete")
}
