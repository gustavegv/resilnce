package db

import (
	"context"
	"errors"
	"fmt"
	"math"
	"strconv"
)

func (supa *SupabaseCFG) UpdateExercise(userMail string, sesID int, ex CompactExercise, ctx context.Context) error {
	const query = `
		update "SesExercise" sx
		set
			finished = true,
			rep_per_set = $1,
			weight_per_set = $2
		from "Session" se
			where se.ses_id = sx.ses_id
				and sx.ex_id = $3
				and se.mail = $4
				and sx.ses_id = $5
	`

	_, err := supa.DB.Exec(ctx, query, ex.Reps, ex.Weights, ex.ExID, userMail, sesID)

	if err != nil {
		println("DB query fail (Write: Update)")
		println(err.Error())
		return err
	}

	return nil
}

type exerciseState struct {
	Reps    []int
	Weights []float64
}

func (supa *SupabaseCFG) EditExercise(userMail string, sesID int, edit ExerciseEdit, ctx context.Context) error {
	if edit.Name != nil {
		if err := supa.updateExerciseName(userMail, edit.ExID, *edit.Name, ctx); err != nil {
			return err
		}
	}

	if edit.AutoIncrease != nil {
		if err := supa.updateExerciseAutoIncrease(userMail, edit.ExID, *edit.AutoIncrease, ctx); err != nil {
			return err
		}
	}

	if edit.Sets != nil {
		if err := supa.updateSessionSetCount(userMail, sesID, edit.ExID, *edit.Sets, ctx); err != nil {
			return err
		}
	}

	if edit.Weight != nil {
		if err := supa.updateSessionWeight(userMail, sesID, edit.ExID, *edit.Weight, ctx); err != nil {
			return err
		}
	}

	if edit.RepThreshold != nil {
		if err := supa.updateSessionRepThreshold(userMail, sesID, edit.ExID, *edit.RepThreshold, ctx); err != nil {
			return err
		}
	}

	return nil
}

func (supa *SupabaseCFG) updateExerciseName(userMail string, exID string, name string, ctx context.Context) error {
	const query = `
		update "Exercise"
		set
			ex_name = $1
		where mail = $2
			and ex_id = $3
	`

	_, err := supa.DB.Exec(ctx, query, name, userMail, exID)
	if err != nil {
		println("DB query fail (Write: Edit exercise name)")
		println(err.Error())
		return err
	}

	return nil
}

func (supa *SupabaseCFG) loadExerciseState(userMail string, sesID int, exID string, ctx context.Context) (exerciseState, error) {
	const query = `
		select sx.rep_per_set, sx.weight_per_set
		from "SesExercise" sx
		join "Session" se on se.ses_id = sx.ses_id
		where se.mail = $1
			and sx.ses_id = $2
			and sx.ex_id = $3
	`

	var state exerciseState
	err := supa.DB.QueryRow(ctx, query, userMail, sesID, exID).Scan(&state.Reps, &state.Weights)
	if err != nil {
		println("DB query fail (Write: Load edit state)")
		println(err.Error())
		return exerciseState{}, err
	}

	return state, nil
}

func (supa *SupabaseCFG) updateExerciseAutoIncrease(userMail string, exID string, autoIncrease float64, ctx context.Context) error {
	const query = `
		update "Exercise"
		set
			auto_inc = $1
		where mail = $2
			and ex_id = $3
	`

	_, err := supa.DB.Exec(ctx, query, autoIncrease, userMail, exID)
	if err != nil {
		println("DB query fail (Write: Edit exercise auto increase)")
		println(err.Error())
		return err
	}

	return nil
}

func resizeIntSlice(values []int, nextLength int) []int {
	resized := append([]int{}, values...)

	if len(resized) > nextLength {
		return resized[:nextLength]
	}

	for len(resized) < nextLength {
		resized = append(resized, 0)
	}

	return resized
}

func buildWeightArray(weight float64, setCount int) []float64 {
	weights := make([]float64, setCount)
	for i := range weights {
		weights[i] = weight
	}
	return weights
}

func baseWeightFromState(state exerciseState) float64 {
	if len(state.Weights) == 0 {
		return 0
	}
	return state.Weights[0]
}

func (supa *SupabaseCFG) updateSessionSetCount(
	userMail string,
	sesID int,
	exID string,
	setCount int,
	ctx context.Context,
) error {
	state, err := supa.loadExerciseState(userMail, sesID, exID, ctx)
	if err != nil {
		return err
	}

	reps := resizeIntSlice(state.Reps, setCount)
	weights := buildWeightArray(baseWeightFromState(state), setCount)

	const query = `
		update "SesExercise" sx
		set
			set_count = $1,
			rep_per_set = $2,
			weight_per_set = $3
		from "Session" se
		where se.ses_id = sx.ses_id
			and se.mail = $4
			and sx.ses_id = $5
			and sx.ex_id = $6
	`

	_, err = supa.DB.Exec(ctx, query, setCount, reps, weights, userMail, sesID, exID)
	if err != nil {
		println("DB query fail (Write: Edit session set count)")
		println(err.Error())
		return err
	}

	return nil
}

func (supa *SupabaseCFG) updateSessionWeight(
	userMail string,
	sesID int,
	exID string,
	weight float64,
	ctx context.Context,
) error {
	state, err := supa.loadExerciseState(userMail, sesID, exID, ctx)
	if err != nil {
		return err
	}

	setCount := len(state.Reps)
	weights := buildWeightArray(weight, setCount)

	const query = `
		update "SesExercise" sx
		set
			weight_per_set = $1
		from "Session" se
		where se.ses_id = sx.ses_id
			and se.mail = $2
			and sx.ses_id = $3
			and sx.ex_id = $4
	`

	_, err = supa.DB.Exec(ctx, query, weights, userMail, sesID, exID)
	if err != nil {
		println("DB query fail (Write: Edit session weight)")
		println(err.Error())
		return err
	}

	return nil
}

func (supa *SupabaseCFG) updateSessionRepThreshold(
	userMail string,
	sesID int,
	exID string,
	repThreshold int,
	ctx context.Context,
) error {
	const query = `
		update "SesExercise" sx
		set
			rep_threshold = $1
		from "Session" se
		where se.ses_id = sx.ses_id
			and se.mail = $2
			and sx.ses_id = $3
			and sx.ex_id = $4
	`

	_, err := supa.DB.Exec(ctx, query, repThreshold, userMail, sesID, exID)
	if err != nil {
		println("DB query fail (Write: Edit session rep threshold)")
		println(err.Error())
		return err
	}

	return nil
}

func avg(arr []int) float64 {
	var sum int = 0
	for i := 0; i < len(arr); i++ {
		sum += arr[i]
	}
	var avg float64 = float64(sum) / float64(len(arr))
	rounded := math.Round(avg*10) / 10
	return rounded
}

func avgF(arr []float64) float64 {
	var sum float64 = 0
	for i := 0; i < len(arr); i++ {
		sum += arr[i]
	}
	var avg float64 = float64(sum) / float64(len(arr))
	rounded := math.Round(avg*10) / 10
	return rounded
}

func (supa *SupabaseCFG) SaveHistory(userMail string, sesID int, ex CompactExercise, ctx context.Context) error {

	var avgR float64 = avg(ex.Reps)
	var avgW float64 = avgF(ex.Weights)
	var sets int = len(ex.Reps)

	const query = `
		INSERT INTO "History" (save_date, ex_id, avg_rep, avg_weight, no_of_sets)
		SELECT now(), ex.ex_id, $1, $2, $3
		FROM "Exercise" AS ex
			WHERE ex.mail = $4
			AND ex.ex_id = $5;
	`

	_, err := supa.DB.Exec(ctx, query, avgR, avgW, sets, userMail, ex.ExID)

	if err != nil {
		println("DB query fail (Write: Save history)")
		println(err.Error())
		return err
	}

	return nil
}
func (supa *SupabaseCFG) UpdateLastRan(userMail string, sesID int, ctx context.Context) error {

	const query = `
		update "Session"
		set
			date_last_ran = now()
			where mail = $1
				and ses_id = $2
	`

	_, err := supa.DB.Exec(ctx, query, userMail, sesID)

	if err != nil {
		println("DB query fail (Write: Update date)")
		println(err.Error())
		return err
	}

	return nil
}

func (supa *SupabaseCFG) CompleteSession(userMail string, sesID int, ctx context.Context) error {
	const query = `
		update "SesExercise" sx
		set
			finished = false
		from "Session" se
			where se.ses_id = sx.ses_id
				and se.mail = $1
				and sx.ses_id = $2
	`

	_, err := supa.DB.Exec(ctx, query, userMail, sesID)
	if err != nil {
		println("DB query fail (Write: Complete session)")
		println(err.Error())
		return err
	}

	err = supa.SetActiveSession(userMail, -1, ctx)
	if err != nil {
		println(err.Error())
		return err
	}

	return nil
}

func (supa *SupabaseCFG) SetActiveSession(userMail string, sesID int, ctx context.Context) error {
	const query = `
		update "User"
		set
			active_session = $1
			where mail = $2
	`
	var err error
	if sesID == -1 {
		_, err = supa.DB.Exec(ctx, query, nil, userMail)
	} else {
		_, err = supa.DB.Exec(ctx, query, strconv.Itoa(sesID), userMail)

	}

	if err != nil {
		println("DB query fail (Write: Set session active)")
		println(err.Error())
		return err
	}

	return nil
}

func generateSQLArgsArray(info []ExInfo) ([]any, error) {
	loopTimes := len(info)
	if loopTimes < 1 {
		err := errors.New("Empty array (Batch add exercises)")
		return nil, err
	}
	parametersPerRow := 2
	mailParamArgCount := 1
	args := make([]any, mailParamArgCount+loopTimes*parametersPerRow)
	return args, nil
}

/*
documentation of function for my future self:
We want in our SB call:
STEP 1:
query values:

	(mail, auto incrementing ID, ExName1, AutoInc1)
	(mail, auto incrementing ID, ExName2, AutoInc2)
	etc.

the inputs need to be put in parameters:

	($1, DEFAULT, $2, $3)
	($1, DEFAULT, $4, $5)

for any given length of input array.

STEP 2:
we also need to pass the inputs in the correct as arguments

args =	(ID, ExName1, AutoInc1, ExName2, AutoInc2)

	$1.   $2.       $3.      $4.       $5.

which corresponds to the parameter under.
*/
func fillExerciseMultiAddSQLQuery(args []any, query string, info []ExInfo, loopTimes int, usedVars int, userMail string) (string, []any) {
	args[0] = userMail

	for i := 0; i < loopTimes; i++ {
		offset := usedVars * i
		pos_name := offset + 2
		pos_auto := offset + 3
		next := fmt.Sprintf(`($1, DEFAULT, $%d, $%d)`, pos_name, pos_auto)

		if i == loopTimes-1 {
			query = query + "\n" + next + ""
		} else {
			query = query + "\n" + next + ","
		}

		current := info[i]
		args[pos_name-1] = current.ExName
		args[pos_auto-1] = current.AutoInc
	}
	query = query + "\n returning ex_id"
	return query, args

}

func generateMultiAddSQLQuery(info []ExInfo, userMail string) (string, []any, error) {
	loopTimes := len(info)
	if loopTimes < 1 {
		err := errors.New("Empty array (Batch add exercises)")
		return "", nil, err
	}
	parametersPerRow := 2
	mailParamArgCount := 1
	args := make([]any, mailParamArgCount+loopTimes*parametersPerRow)

	var base_query = `insert into "Exercise" (mail, ex_id, ex_name, auto_inc) values`

	query, args := fillExerciseMultiAddSQLQuery(args, base_query, info, loopTimes, parametersPerRow, userMail)
	return query, args, nil
}

func (supa *SupabaseCFG) AddMultipleExercises(userMail string, info []ExInfo, ctx context.Context) ([]ExInfo, error) {
	query, args, err := generateMultiAddSQLQuery(info, userMail)
	if err != nil {
		return []ExInfo{}, err
	}

	rows, err := supa.DB.Query(ctx, query, args...)
	if err != nil {
		println("DB query fail (Write: Batch Add Exercises)")
		println(err.Error())
		return []ExInfo{}, err
	}
	defer rows.Close()

	i := 0
	for rows.Next() {

		err := rows.Scan(&info[i].ExID)
		if err != nil {
			return nil, fmt.Errorf("failed to scan row: %w", err)
		}
		i++
	}

	return info, nil
}

func (supa *SupabaseCFG) AddSession(userMail string, sesName string, ctx context.Context) (int, error) {

	const query = `
		insert into "Session" (ses_id, mail, ses_name, date_last_ran)
		values (DEFAULT, $1, $2, NOW())
		returning ses_id
	`
	var newID int
	err := supa.DB.QueryRow(ctx, query, userMail, sesName).Scan(&newID)
	if err != nil {
		println("DB query fail (Write: Add session)")
		println(err.Error())
		return -1, err
	}

	return newID, nil
}

func makeSesExQuery(newInfo []ExInfo, sesID int) (string, []any, error) {
	var query = `
	insert into "SesExercise" (ses_id, ex_id, rep_threshold, set_count, rep_per_set, weight_per_set, rest_seconds, "order", finished)
	values 

	`

	loopTimes := len(newInfo)
	if loopTimes < 1 {
		err := errors.New("Empty array (SesEx add exercises)")
		return "", nil, err
	}
	used_vars := 6
	args := make([]any, 1+loopTimes*used_vars)
	args[0] = sesID

	for i := 0; i < loopTimes; i++ {
		offset := used_vars * i

		p_exID := offset + 2
		p_repT := offset + 3
		p_setC := offset + 4
		p_rperS := offset + 5
		p_wperS := offset + 6
		p_ordr := offset + 7

		next := fmt.Sprintf(`($1, $%d, $%d, $%d, $%d, $%d, 0, $%d, false)`, p_exID, p_repT, p_setC, p_rperS, p_wperS, p_ordr)

		if i == loopTimes-1 {
			query = query + "\n" + next + ";"
		} else {
			query = query + "\n" + next + ","
		}

		current := newInfo[i]
		args[p_exID-1] = current.ExID
		args[p_repT-1] = current.RepThres
		args[p_setC-1] = current.SetCount
		args[p_rperS-1] = current.Reps
		args[p_wperS-1] = current.Weights
		args[p_ordr-1] = current.Order
	}
	return query, args, nil
}

func (supa *SupabaseCFG) AddSesExercise(sesID int, newInfo []ExInfo, ctx context.Context) error {

	query, args, err := makeSesExQuery(newInfo, sesID)
	if err != nil {
		return err
	}
	fmt.Println("AddSesExercise query:")

	fmt.Println(query)

	_, err = supa.DB.Exec(ctx, query, args...)
	if err != nil {
		println("DB query fail (Write: Add SesExercises)")
		println(err.Error())
		return err
	}

	return nil
}

func (supa *SupabaseCFG) NewSession(userMail string, sesName string, info []ExInfo, ctx context.Context) error {
	fmt.Println("\nBeginning NewSession struct:")
	fmt.Printf("data = %#v\n", info)

	sesID, err := supa.AddSession(userMail, sesName, ctx)
	if err != nil {
		return err
	}

	// här kan man lägga en check om exercises redan finns. (aka exid finns i exinfo)
	// sen kan man lägga dom som inte finns i en egen array och skicka till add exercises
	// sen mergea dom som fanns med dom som blivit tillagda i addmultipleexs

	newInfo, err := supa.AddMultipleExercises(userMail, info, ctx)
	if err != nil {
		return err
	}

	err = supa.AddSesExercise(sesID, newInfo, ctx)
	if err != nil {
		return err
	}

	return nil
}

func (supa *SupabaseCFG) AddUser(userMail string, name string, ctx context.Context) error {
	const query = `
		insert into "User" (mail, name, signup_date)
		values ($1, $2, NOW())
		ON CONFLICT (mail) DO NOTHING;
	`

	_, err := supa.DB.Exec(ctx, query, userMail, name)

	if err != nil {
		println("DB query fail (Write: Add user)")
		println(err.Error())
		return err
	}

	return nil
}

func (supa *SupabaseCFG) DeleteSession(userMail string, sessionID int, ctx context.Context) error {
	const query = `
		delete from "Session"
		where mail = $1 and ses_id = $2;
	`

	_, err := supa.DB.Exec(ctx, query, userMail, sessionID)

	if err != nil {
		println("DB query fail (Write: Delete Session)")
		println(err.Error())
		return err
	}

	return nil
}
