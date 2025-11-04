package api

import (
	"fmt"
	"net/http"
)

func BaseReq(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	fmt.Fprintf(w, "Base request reached: %s %s\n", r.Method, r.URL.Path)
}

func LogOutUser(w http.ResponseWriter, r *http.Request) {

	// todo: delete SessionID from Redis
	const redisDeleteSuccess = true
	if !redisDeleteSuccess {
		http.Error(w, "fail to logout", http.StatusInternalServerError)
	}

	http.SetCookie(w, &http.Cookie{
		Name:     "SignedCookie",
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		SameSite: http.SameSiteNoneMode,
		Secure:   true,
	})

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("success"))
	print("User logged out.\n")
}
