package ai

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"

	sessions "github.com/gustavegv/resilnce/apps/server/redis"
	scookie "github.com/gustavegv/resilnce/apps/server/scookies"
)

func getValidatedMail(w http.ResponseWriter, r *http.Request) (string, string) {

	// todo, add redis validation, to check if mail expired, if so log out user
	userMail := r.URL.Query().Get("mail")
	SID, userMail, _, success := scookie.ValidateSignedCookie(r)
	if !success {
		println("Verification failed")
		http.Error(w, "verification failed", http.StatusBadRequest)
		return "", ""
	}

	if userMail == "" {
		println("No mail")
		http.Error(w, "missing mail", http.StatusBadRequest)
		return "", ""
	}
	return SID, userMail
}

func GetQuickCreationData(promptSelections []bool, userData string, sesID string, store *sessions.Store, ctx context.Context) (string, error) {

	accountAICallCount, err := store.GetOrInitAICalls(ctx, sesID)
	println("AI calls and account number (", accountAICallCount, ",", sesID, ")")
	const AI_CALL_LIMIT = 25
	if accountAICallCount > AI_CALL_LIMIT {
		println("AI calls exceeded")
		return "", errors.New("AI call count exceeded")
	}

	_, err = store.IncrAICalls(ctx, sesID, 1)
	if err != nil {
		return "", err
	}

	callPrompt, err := getPrompt(promptSelections, userData)
	if err != nil {
		return "", err
	}

	jsonString, err := callChatAPI(callPrompt)
	if err != nil {
		return "", err
	}

	return jsonString, nil
}

func AutoCreation(store *sessions.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		SID, userMail := getValidatedMail(w, r)
		if userMail == "" {
			http.Error(w, "Invalid credentials", http.StatusUnauthorized)
			return
		}

		var data struct {
			UserInput        string `json:"userInput"`
			PromptSelections []bool `json:"promptSelections"`
		}

		dec := json.NewDecoder(r.Body)
		dec.DisallowUnknownFields()
		if err := dec.Decode(&data); err != nil {
			http.Error(w, "Invalid JSON: "+err.Error(), http.StatusBadRequest)
			return
		}

		output, err := GetQuickCreationData(data.PromptSelections, data.UserInput, SID, store, r.Context())

		if err != nil {
			if err.Error() == "AI call count exceeded" {
				http.Error(w, "AI call count exceeded", http.StatusUnauthorized)
				return
			}
			http.Error(w, "Agent failiure", http.StatusBadRequest)
			return
		}

		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		_, _ = w.Write([]byte(output))
	}
}
