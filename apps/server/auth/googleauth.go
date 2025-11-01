package auth

import (
	"fmt"
	"net/http"

	"golang.org/x/oauth2"
)

func (s *service) LoginGoogle(w http.ResponseWriter, r *http.Request) {
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

func (s *service) CallbackGoogle(w http.ResponseWriter, r *http.Request) {
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
	userID, err := UpsertUser("google", claims.Sub, claims.Email, claims.Name)
	if err != nil {
		http.Error(w, "user upsert failed", http.StatusInternalServerError)
		return
	}

	err = s.IssueSignedCookie(w, r, userID)

	if err != nil {
		http.Error(w, "signed cookie issue failed", http.StatusInternalServerError)
		return
	}

	fmt.Println("Succesful sign in. User:", userID, "\n Named:", claims.Name)

	var redirectURL string = GetBaseURL() + "/oauth"
	// TODO: Switch back: "/?sonner=loggedin"

	http.Redirect(w, r, redirectURL, http.StatusSeeOther)
}
