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
	storedState, _ := s.getCookie(r, s.cfg.StateCookieName)
	if state == "" || storedState == "" || state != storedState {
		fmt.Println("Invalid state. \nCurrent state", state, " \nStored state:", storedState)
		http.Error(w, "Invalid state", http.StatusBadRequest)

		return
	}
	s.clearCookie(w, s.cfg.StateCookieName)

	// PKCE
	codeVerifier, _ := s.getCookie(r, s.cfg.PKCECookieName)
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
		http.Error(w, "user upsert failed", http.StatusInternalServerError)
		return
	}

	err = s.IssueSignedCookie(w, r, email, claims.Sub, claims.Name)

	if err != nil {
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
		// TODO: Switch back: "/?sonner=loggedin"

	case "login":
		directory = "/account"

	default:
		http.Error(w, "unknown redirect location (googleauth.go)", http.StatusInternalServerError)
		return
	}
	base := GetBaseURL()
	directory = base + directory
	http.Redirect(w, r, directory, http.StatusSeeOther)
}

func (s *Service) GetUserInfoOld(w http.ResponseWriter, r *http.Request) {
	SID, _, _, success := scookie.ValidateSignedCookie(r)
	if !success {
		redirectTo("login", w, r)
		return
	}

	ctx := r.Context()
	data, err := s.store().GetSession(ctx, SID)
	if err != nil {
		print("getSession error")
		http.Error(w, "fail to get session from redis store", http.StatusInternalServerError)
		return
	}

	if data == nil {
		print("redis fetched data empty (googleauth.go)")
		http.Error(w, "redis data empty", http.StatusNotFound)
		return
	}
	println("redis data sent (googleauth.go):", data)
	firstName := data.Name
	mail := data.Email

	resp := map[string]any{
		"name": firstName,
		"id":   mail,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func (s *Service) GetUserInfo(w http.ResponseWriter, r *http.Request) {
	_, mail, name, success := scookie.ValidateSignedCookie(r)
	if !success {
		http.Error(w, "unknown redirect location (googleauth.go)", http.StatusInternalServerError)

		return
	}

	resp := map[string]any{
		"name": name,
		"id":   mail,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// TODO: Move function. Make it return only payload. Make it return err not bool
