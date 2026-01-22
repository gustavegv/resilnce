package db

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strconv"
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
	SID, userMail, _, success := scookie.ValidateSignedCookie(r)
	if !success {
		println("Verification failed")
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
		return errors.New("Error: New data empty when trying to create new session")
	}
	const maxExPerSes = 100
	const maxWeight = 9999
	const maxSets = 20
	const maxNameLen = 100

	if len(exI) > maxExPerSes {
		return errors.New("Error: Too many exercises pushed.")
	}
	for _, exercise := range exI {
		w := exercise.Weights[0]
		s, err := strconv.Atoi(exercise.SetCount)
		nl := len(exercise.ExName)

		if err != nil {
			return errors.New("Error: Non numeric set number")
		}
		if w > maxWeight || s > maxSets || nl > maxNameLen {
			return errors.New("Error: Exercise data exceeding max limits")
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
		log.Printf("invalid session data: %v", err)
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
