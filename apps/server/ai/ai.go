package ai

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"

	"github.com/gustavegv/resilnce/apps/server/application"
)

type Quota func(context.Context, string, int64) (bool, error)
type Generator func(context.Context, []bool, string) (string, error)

func GenerateDraft(_ context.Context, selections []bool, input string) (string, error) {
	prompt, err := getPrompt(selections, input)
	if err != nil {
		return "", err
	}
	return callChatAPI(prompt)
}

// HTTP generates untrusted review data; it never writes training records.
func HTTP(quota Quota, generate Generator, origin string) http.HandlerFunc {
	if generate == nil {
		generate = GenerateDraft
	}
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Origin") != strings.TrimRight(origin, "/") || origin == "" {
			application.WriteError(w, 403, "origin-forbidden", "Use the configured application origin", nil)
			return
		}
		identity, ok := application.FromContext(r.Context())
		if !ok {
			application.WriteError(w, 401, "unauthenticated", "Sign in to continue", nil)
			return
		}
		var data struct {
			UserInput        string `json:"userInput"`
			PromptSelections []bool `json:"promptSelections"`
		}
		dec := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20))
		dec.DisallowUnknownFields()
		if err := dec.Decode(&data); err != nil {
			application.WriteError(w, 400, "malformed-request", "Provide a valid AI draft request", nil)
			return
		}
		if err := dec.Decode(new(any)); err != io.EOF {
			application.WriteError(w, 400, "malformed-request", "Provide one JSON object", nil)
			return
		}
		if quota == nil {
			application.WriteError(w, 503, "ai-unavailable", "AI quota service is unavailable", nil)
			return
		}
		allowed, err := quota(r.Context(), identity.QuotaKey(), 25)
		if err != nil {
			application.WriteError(w, 503, "ai-unavailable", "AI quota service is unavailable", nil)
			return
		}
		if !allowed {
			application.WriteError(w, 403, "ai-quota-exceeded", "AI call count exceeded", nil)
			return
		}
		output, err := generate(r.Context(), data.PromptSelections, data.UserInput)
		if err != nil {
			application.WriteError(w, 422, "ai-generation-failed", "Could not generate a draft; edit the input and retry", nil)
			return
		}
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.Write([]byte(output))
	}
}
