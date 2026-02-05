package auth

import (
	"encoding/json"
	"fmt"
	"net/http"

	scookie "github.com/gustavegv/resilnce/apps/server/scookies"
	"golang.org/x/oauth2"
)

func (s *Service) LoginGoogle(w http.ResponseWriter, r *http.Request) {
	state := randB64(32)
	verifier := randB64(64) // PKCE code_verifier
	challenge := pkceS256(verifier)

	s.setCookie(w, s.cfg.StateCookieName, state, s.stateCookieAttrs)
	s.setCookie(w, s.cfg.PKCECookieName, verifier, s.pkceCookieAttrs)

	authURL := s.googleOAuth.AuthCodeURL(
		state,
		oauth2.SetAuthURLParam("access_type", "offline"),
		oauth2.SetAuthURLParam("prompt", "consent"),
		oauth2.SetAuthURLParam("code_challenge", challenge),
		oauth2.SetAuthURLParam("code_challenge_method", "S256"),
	)

	http.Redirect(w, r, authURL, http.StatusFound)
}

func (s *Service) CallbackGoogle(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// validate state
	state := r.URL.Query().Get("state")
	storedState, err := s.getCookie(r, s.cfg.StateCookieName)
	if err != nil {
		fmt.Println("Cookie error:", err.Error())
		http.Error(w, "Cookie error", http.StatusBadRequest)
		return
	}
	if state == "" || storedState == "" || state != storedState {
		fmt.Println("Invalid state. \nCurrent state", state, " \nStored state:", storedState)
		http.Error(w, "Invalid state", http.StatusBadRequest)
		return
	}
	s.clearCookie(w, s.cfg.StateCookieName)

	// PKCE
	codeVerifier, err := s.getCookie(r, s.cfg.PKCECookieName)
	if err != nil {
		fmt.Println("Code verifier cookie error:", err.Error())
		http.Error(w, "Cookie error", http.StatusBadRequest)
		return
	}
	s.clearCookie(w, s.cfg.PKCECookieName)

	// exchang code for token
	code := r.URL.Query().Get("code")
	if code == "" {
		http.Error(w, "missing code", http.StatusBadRequest)
		return
	}
	tok, err := s.googleOAuth.Exchange(ctx, code,
		oauth2.SetAuthURLParam("code_verifier", codeVerifier),
	)
	if err != nil {
		http.Error(w, "code exchange failed", http.StatusBadRequest)
		return
	}

	// verify token
	rawID, _ := tok.Extra("id_token").(string)
	if rawID == "" {
		http.Error(w, "missing id_token", http.StatusBadRequest)
		return
	}
	idTok, err := s.googleVerifier.Verify(ctx, rawID)
	if err != nil {
		http.Error(w, "id_token verify failed", http.StatusUnauthorized)
		return
	}

	var claims struct {
		Sub   string `json:"sub"`
		Email string `json:"email"`
		Name  string `json:"name"`
	}
	_ = idTok.Claims(&claims)

	if claims.Sub == "" {
		http.Error(w, "missing sub", http.StatusUnauthorized)
		return
	}

	// save / update session in db
	email, err := s.UpsertUser(ctx, "google", claims.Sub, claims.Email, claims.Name)
	if err != nil {
		println("Error (UpsertUser - 1): User upsert failed.", err.Error())
		http.Error(w, "user upsert failed", http.StatusInternalServerError)
		return
	}

	err = s.IssueSignedCookie(w, r, email, claims.Sub, claims.Name)

	if err != nil {
		println("Error (UpsertUser - 2): Signed cookie failed.", err.Error())
		http.Error(w, "signed cookie issue failed", http.StatusInternalServerError)
		return
	}

	fmt.Println("Succesful sign in. User:", email, "\n Named:", claims.Name)

	redirectTo("home", w, r)
}

func redirectTo(loc string, w http.ResponseWriter, r *http.Request) {
	var directory string
	switch loc {
	case "home":
		directory = "/?sonner=loggedin"

	case "login":
		directory = "/account"

	default:
		println("Error (redirectTo): Unknown redirect.")
		http.Error(w, "unknown redirect location (googleauth.go)", http.StatusInternalServerError)
		return
	}
	base := GetBaseURL()
	directory = base + directory
	http.Redirect(w, r, directory, http.StatusSeeOther)
}

func (s *Service) GetUserInfo(w http.ResponseWriter, r *http.Request) {
	_, mail, name, err := scookie.ValidateSignedCookie(r)
	if err != nil {
		println("Error (GetUserInfo):", err.Error())
		http.Error(w, "Try signing in.", http.StatusUnauthorized)
		return
	}

	resp := map[string]any{
		"name": name,
		"id":   mail,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}
