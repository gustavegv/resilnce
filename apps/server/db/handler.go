package db

import (
	"encoding/json"
	"net/http"

	"github.com/gustavegv/resilnce/apps/server/auth"
	"github.com/jackc/pgx/v5/pgxpool"
)

type SupabaseCFG struct {
	DB *pgxpool.Pool
}

func (supa *SupabaseCFG) GetUserSessions(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userMail := r.URL.Query().Get("mail")
	_, userMail, _, success := auth.ValidateSignedCookie(r)
	if !success {
		println("Verification failed")
		http.Error(w, "verification failed", http.StatusBadRequest)
	}

	if userMail == "" {
		println("No mail")
		http.Error(w, "missing mail", http.StatusBadRequest)
		return
	}

	names, err := supa.UserSessions(userMail, ctx)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(names)
}
