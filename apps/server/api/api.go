package api

import (
	"encoding/json"
	"fmt"
	"net/http"

	scookie "github.com/gustavegv/resilnce/apps/server/scookies"
)

func BaseReq(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	fmt.Fprintf(w, "Base request reached: %s %s\n", r.Method, r.URL.Path)
}

func GetUserInfo(w http.ResponseWriter, r *http.Request) {
	uID, success := validateSignedCookie(r)
	if !success {
		// todo: redirect to login screen because they are probably not logged in
		print("Failed to validate.\nLog:", uID)
		http.Error(w, "fail to logout", http.StatusInternalServerError)
		return
	}

	// todo (replace): Look up the sessionID (sub) in Redis, and get username
	const firstName string = "GuenDB"

	resp := map[string]any{
		"name": firstName,
		"id":   uID,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
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
		SameSite: http.SameSiteLaxMode,
		Secure:   true,
	})

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("success"))
	print("User logged out.\n")
}

func validateSignedCookie(r *http.Request) (string, bool) {
	cookie, err := r.Cookie("SignedCookie")
	if err != nil || cookie.Value == "" {
		return "No cookie value", false
	}
	secret := scookie.GetSecret()
	payload, success := scookie.Verify(cookie.Value, secret)
	if !success {
		return "SCookie Verification failed", false
	}
	return payload.UID, true
}
