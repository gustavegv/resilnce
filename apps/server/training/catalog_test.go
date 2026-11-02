package training

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"slices"
	"testing"

	"github.com/gustavegv/resilnce/apps/server/application"
	"github.com/gustavegv/resilnce/apps/server/predefined"
)

func TestCatalogServesExerciseNameSuggestionsWithoutDatabaseAccess(t *testing.T) {
	mux := http.NewServeMux()
	// No Service or database is supplied: these are reference names, not seeded exercises.
	HTTP{Boundary: application.Boundary{
		Config: application.Config{Model: application.ExerciseCentric},
		Resolve: func(*http.Request) (application.Identity, error) {
			return application.Identity{Email: "owner@example.com"}, nil
		},
	}}.Register(mux)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/training/catalog", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("catalog response: %d %s", w.Code, w.Body.String())
	}
	var catalog struct {
		ExerciseNames []string                 `json:"exerciseNames"`
		MuscleGroups  []predefined.MuscleGroup `json:"muscleGroups"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &catalog); err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(catalog.ExerciseNames, predefined.ExerciseNames()) || len(catalog.MuscleGroups) != 26 {
		t.Fatalf("catalog must include exercise names alongside existing muscle groups: %s", w.Body.String())
	}
	if w.Header().Get("Cache-Control") != "no-store" {
		t.Fatal("catalog must preserve the authenticated response's no-store policy")
	}
}
