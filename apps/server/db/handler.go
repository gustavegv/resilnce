package db

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	sessions "github.com/gustavegv/resilnce/apps/server/redis"
	scookie "github.com/gustavegv/resilnce/apps/server/scookies"

	"github.com/jackc/pgx/v5/pgxpool"
)

type SupabaseCFG struct {
	DB         *pgxpool.Pool
	RedisStore *sessions.Store
}

func (supa *SupabaseCFG) getValidatedMail(w http.ResponseWriter, r *http.Request) string {

	userMail := r.URL.Query().Get("mail")
	SID, userMail, _, err := scookie.ValidateSignedCookie(r)
	if err != nil {
		println("Verification failed.", err.Error())
		http.Error(w, "verification failed", http.StatusBadRequest)
		return ""
	}

	if userMail == "" {
		println("No mail")
		http.Error(w, "missing mail", http.StatusBadRequest)
		return ""
	}

	data, err := supa.RedisStore.GetSession(r.Context(), SID)
	if err != nil {
		http.Error(w, "Authentication error", http.StatusBadRequest)
		return ""
	}
	if data == nil {
		http.Error(w, "Session expired, please log in again", http.StatusUnauthorized)
		return ""
	}

	return userMail
}

func getSesID(w http.ResponseWriter, r *http.Request) int {
	idStr := r.URL.Query().Get("sesID")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return -1
	}
	return id
}

func getExID(w http.ResponseWriter, r *http.Request) int {
	idStr := r.URL.Query().Get("exID")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "Invalid ExID", http.StatusBadRequest)
		return -1
	}
	return id
}

func (supa *SupabaseCFG) GetUserSessions(w http.ResponseWriter, r *http.Request) {
	userMail := supa.getValidatedMail(w, r)
	if userMail == "" {
		http.Error(w, "Invalid credentials", http.StatusUnauthorized)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
	defer cancel()

	sesMetaData, err := supa.UserSessions(userMail, ctx)
	if err != nil {
		println("Error (GetUserSessions):", err.Error())
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(sesMetaData)
}

func (supa *SupabaseCFG) GetSessionExercises(w http.ResponseWriter, r *http.Request) {
	sesID := getSesID(w, r)
	if sesID == -1 {
		return
	}
	userMail := supa.getValidatedMail(w, r)
	if userMail == "" || sesID == -1 {
		http.Error(w, "Invalid credentials", http.StatusUnauthorized)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
	defer cancel()

	exInfo, err := supa.SessionExercises(userMail, sesID, ctx)
	if err != nil {
		println("Error (GetSessionExercises):", err.Error())
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(exInfo)
}

func (supa *SupabaseCFG) GetActiveSession(w http.ResponseWriter, r *http.Request) {
	userMail := supa.getValidatedMail(w, r)
	if userMail == "" {
		http.Error(w, "Invalid credentials", http.StatusUnauthorized)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
	defer cancel()

	activeSesName, activeSesID, err := supa.CheckIfActive(userMail, ctx)
	if err != nil {
		println(err.Error())
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	jsonPayload := struct {
		ActiveSession     string
		ActiveSessionName string
	}{
		ActiveSession:     activeSesID,
		ActiveSessionName: activeSesName,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(jsonPayload)
}

func (supa *SupabaseCFG) GetFinishedExercises(w http.ResponseWriter, r *http.Request) {
	sesID := getSesID(w, r)
	if sesID == -1 {
		return
	}
	userMail := supa.getValidatedMail(w, r)
	if userMail == "" || sesID == -1 {
		http.Error(w, "Invalid credentials", http.StatusUnauthorized)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
	defer cancel()

	finishedExIDs, err := supa.CheckFinishedExercises(userMail, sesID, ctx)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	jsonPayload := struct {
		Finished []int
	}{
		Finished: finishedExIDs,
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(jsonPayload)
}

type CompactExercise struct {
	Reps    []int     `json:"reps"`
	Weights []float64 `json:"weights"`
	ExID    string    `json:"id"`
}

type ExerciseEdit struct {
	ExID         string   `json:"id"`
	Name         *string  `json:"name,omitempty"`
	Sets         *int     `json:"sets,omitempty"`
	Weight       *float64 `json:"weight,omitempty"`
	RepThreshold *int     `json:"repThreshold,omitempty"`
	AutoIncrease *float64 `json:"autoIncrease,omitempty"`
}

func (supa *SupabaseCFG) CallUpdateExercise(w http.ResponseWriter, r *http.Request) {
	var exercise CompactExercise

	sesID := getSesID(w, r)
	if sesID == -1 {
		return
	}
	userMail := supa.getValidatedMail(w, r)
	if userMail == "" || sesID == -1 {
		http.Error(w, "Invalid credentials", http.StatusUnauthorized)
		return
	}

	if err := json.NewDecoder(r.Body).Decode(&exercise); err != nil {
		http.Error(w, "Invalid JSON: "+err.Error(), http.StatusBadRequest)
		return
	}

	defer r.Body.Close()

	err := supa.UpdateExercise(userMail, sesID, exercise, r.Context())
	if err != nil {
		http.Error(w, "Update failed: "+err.Error(), http.StatusExpectationFailed)
		return
	}

	err = supa.SaveHistory(userMail, sesID, exercise, r.Context())
	if err != nil {
		http.Error(w, "Save history failed: "+err.Error(), http.StatusExpectationFailed)
		return
	}

	err = supa.UpdateLastRan(userMail, sesID, r.Context())
	if err != nil {
		http.Error(w, "Update last ran failed: "+err.Error(), http.StatusExpectationFailed)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Update received successfully"))
}

func validateExerciseEdit(edit *ExerciseEdit) error {
	const maxSets = 20
	const maxWeight = 9999
	const maxNameLen = 100
	const maxRepThreshold = 99
	const maxAutoIncrease = 99

	if edit.ExID == "" {
		return errors.New("missing exercise id")
	}

	if edit.Name != nil {
		trimmedName := strings.TrimSpace(*edit.Name)
		if trimmedName == "" {
			return errors.New("exercise name cannot be empty")
		}
		if len(trimmedName) > maxNameLen {
			return errors.New("exercise name cannot be longer than 100 characters")
		}
		edit.Name = &trimmedName
	}

	if edit.Sets != nil {
		if *edit.Sets < 1 || *edit.Sets > maxSets {
			return errors.New("sets must be between 1 and 20")
		}
	}

	if edit.Weight != nil {
		if *edit.Weight < 0 || *edit.Weight > maxWeight {
			return errors.New("weight must be between 0 and 9999")
		}
	}

	if edit.RepThreshold != nil {
		if *edit.RepThreshold < 1 || *edit.RepThreshold > maxRepThreshold {
			return errors.New("rep threshold must be between 1 and 99")
		}
	}

	if edit.AutoIncrease != nil {
		if *edit.AutoIncrease < 0.25 || *edit.AutoIncrease > maxAutoIncrease {
			return errors.New("auto increase must be between 0.25 and 99")
		}
	}

	if edit.Name == nil &&
		edit.Sets == nil &&
		edit.Weight == nil &&
		edit.RepThreshold == nil &&
		edit.AutoIncrease == nil {
		return errors.New("no edit fields were provided")
	}

	return nil
}

func (supa *SupabaseCFG) CallEditExercise(w http.ResponseWriter, r *http.Request) {
	var edit ExerciseEdit

	sesID := getSesID(w, r)
	if sesID == -1 {
		return
	}
	userMail := supa.getValidatedMail(w, r)
	if userMail == "" || sesID == -1 {
		http.Error(w, "Invalid credentials", http.StatusUnauthorized)
		return
	}

	if err := json.NewDecoder(r.Body).Decode(&edit); err != nil {
		http.Error(w, "Invalid JSON: "+err.Error(), http.StatusBadRequest)
		return
	}

	defer r.Body.Close()

	if err := validateExerciseEdit(&edit); err != nil {
		http.Error(w, "Edit validation failed: "+err.Error(), http.StatusBadRequest)
		return
	}

	if err := supa.EditExercise(userMail, sesID, edit, r.Context()); err != nil {
		http.Error(w, "Edit failed: "+err.Error(), http.StatusExpectationFailed)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Edit received successfully"))
}

func (supa *SupabaseCFG) CallCompleteExercise(w http.ResponseWriter, r *http.Request) {
	sesID := getSesID(w, r)
	if sesID == -1 {
		return
	}
	userMail := supa.getValidatedMail(w, r)
	if userMail == "" || sesID == -1 {
		http.Error(w, "Invalid credentials", http.StatusUnauthorized)
		return
	}

	supa.CompleteSession(userMail, sesID, r.Context())
}

func getURLParam(r *http.Request, param string) string {
	varContent := r.URL.Query().Get(param)

	return varContent
}

func (supa *SupabaseCFG) CallSetActiveSession(w http.ResponseWriter, r *http.Request) {
	sesID := getSesID(w, r)
	if sesID == -1 {
		return
	}
	userMail := supa.getValidatedMail(w, r)
	if userMail == "" {
		http.Error(w, "Invalid credentials", http.StatusUnauthorized)
		return
	}

	supa.SetActiveSession(userMail, sesID, r.Context())
}

func validateNewSessionDataHelper(id string, exI []ExInfo) error {
	if exI == nil || id == "" {
		return errors.New("Error (validateNewSessionDataHelper - 1): New data empty when trying to create new session")
	}
	const maxExPerSes = 100
	const maxWeight = 9999
	const maxSets = 20
	const maxNameLen = 100

	if len(exI) > maxExPerSes {
		return errors.New("Error (validateNewSessionDataHelper - 2): Too many exercises pushed.")
	}
	for _, exercise := range exI {
		w := exercise.Weights[0]
		s, err := strconv.Atoi(exercise.SetCount)
		nl := len(exercise.ExName)

		if err != nil {
			return errors.New("Error (validateNewSessionDataHelper - 3): Non numeric set number")
		}
		if w > maxWeight || s > maxSets || nl > maxNameLen {
			return errors.New("Error (validateNewSessionDataHelper - 4): Exercise data exceeding max limits")
		}
	}

	return nil
}

func (supa *SupabaseCFG) MakeNewSession(w http.ResponseWriter, r *http.Request) {
	var data struct {
		SesID string   `json:"sesID"`
		ExI   []ExInfo `json:"exI"`
	}

	userMail := supa.getValidatedMail(w, r)
	if userMail == "" {
		http.Error(w, "Invalid credentials", http.StatusUnauthorized)
		return
	}

	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(&data); err != nil {
		http.Error(w, "Invalid JSON: "+err.Error(), http.StatusBadRequest)
		return
	}

	err := validateNewSessionDataHelper(data.SesID, data.ExI)
	if err != nil {
		log.Printf("invalid session data: %v", err.Error())
		http.Error(w, "Session data invalid", http.StatusFailedDependency)
		return
	}

	fmt.Println("\n\nHandler data-struct:")
	fmt.Printf("data = %#v\n", data)

	err = supa.NewSession(userMail, data.SesID, data.ExI, r.Context())
	if err != nil {
		http.Error(w, "NewSession add error: "+err.Error(), http.StatusBadRequest)
		return
	}

	defer r.Body.Close()
}

func (supa *SupabaseCFG) DeleteSessionH(w http.ResponseWriter, r *http.Request) {
	sesID := getSesID(w, r)
	if sesID == -1 {
		return
	}
	userMail := supa.getValidatedMail(w, r)
	if userMail == "" {
		http.Error(w, "Invalid credentials", http.StatusUnauthorized)
		return
	}

	err := supa.DeleteSession(userMail, sesID, r.Context())
	if err != nil {
		http.Error(w, "Delete failed", http.StatusBadRequest)
		return
	}
}

func (supa *SupabaseCFG) EditSessionH(w http.ResponseWriter, r *http.Request) {
	sesID := getSesID(w, r)
	if sesID == -1 {
		return
	}

	userMail := supa.getValidatedMail(w, r)
	if userMail == "" {
		http.Error(w, "Invalid credentials", http.StatusUnauthorized)
		return
	}

	var data struct {
		Name string `json:"name"`
	}

	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(&data); err != nil {
		http.Error(w, "Invalid JSON: "+err.Error(), http.StatusBadRequest)
		return
	}

	data.Name = strings.TrimSpace(data.Name)
	if data.Name == "" || len(data.Name) > 100 {
		http.Error(w, "Invalid session name", http.StatusBadRequest)
		return
	}

	err := supa.EditSessionName(userMail, sesID, data.Name, r.Context())
	if err != nil {
		http.Error(w, "Edit session failed", http.StatusBadRequest)
		return
	}

	defer r.Body.Close()
}
