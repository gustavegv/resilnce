package db

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

type SessionMetaData struct {
	ID   string     `json:"id"`             // mail
	Name string     `json:"name"`           // ses_name
	Date *time.Time `json:"date,omitempty"` // date_last_ran
}

type ExInfo struct {
	ExID     string    `json:"id"`
	ExName   string    `json:"ex_name"`
	RepThres string    `json:"rep_threshold,omitempty"`
	AutoInc  string    `json:"auto_increase,omitempty"`
	SetCount string    `json:"set_count"`
	Weights  []float64 `json:"weight_per_set"`
	Reps     []int     `json:"rep_per_set"`
	Order    string    `json:"order"`
	Finished bool      `json:"finished"`
}

func (supa *SupabaseCFG) UserSessions(userMail string, ctx context.Context) ([]SessionMetaData, error) {
	const query = `
		select ses_id, ses_name, date_last_ran from "Session"
		where mail = $1
		order by date_last_ran DESC
    `
	rows, err := supa.DB.Query(ctx, query, userMail)

	sessions := []SessionMetaData{}

	for rows.Next() {
		var s SessionMetaData

		err := rows.Scan(&s.ID, &s.Name, &s.Date)
		if err != nil {
			return nil, fmt.Errorf("failed to scan row: %w", err)
		}

		sessions = append(sessions, s)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating rows: %w", err)
	}

	defer rows.Close()
	return sessions, nil
}

func (supa *SupabaseCFG) SessionExercises(userMail string, sesID int, ctx context.Context) ([]ExInfo, error) {

	const query = `
		select  sx.ex_id, ex_name, rep_threshold, auto_inc, set_count,weight_per_set,rep_per_set,"order",finished from "SesExercise" sx
		join "Exercise" ex on sx.ex_id = ex.ex_id
		WHERE sx.ses_id = $1 AND ex.mail = $2
		order by "order"
	`

	exs := []ExInfo{}

	rows, err := supa.DB.Query(ctx, query, sesID, userMail)

	if err != nil {
		println("DB Query fail (Read session exercises)")
		println(err.Error())
		return exs, err

	}
	for rows.Next() {
		var ex ExInfo

		err := rows.Scan(
			&ex.ExID,
			&ex.ExName,
			&ex.RepThres,
			&ex.AutoInc,
			&ex.SetCount,
			&ex.Weights,
			&ex.Reps,
			&ex.Order,
			&ex.Finished,
		)

		if err != nil {
			println("Err row scan (SessionExercises)")
			println(err.Error())
			return exs, err
		}
		exs = append(exs, ex)
	}
	defer rows.Close()
	return exs, nil
}

func (supa *SupabaseCFG) CheckIfActive(userMail string, ctx context.Context) (string, error) {
	const query = `
	select active_session from "User"
	where mail = $1
`
	rows, err := supa.DB.Query(ctx, query, userMail)
	if err != nil {
		println("DB Query fail (Read active session)")
		println(err.Error())
		return "", err

	}
	var activeSession sql.NullString
	rows.Next()
	err = rows.Scan(&activeSession)
	if err != nil {
		println("Err row scan (CheckActive)")
		println(err.Error())

		return "", err
	}
	if activeSession.Valid {
		return activeSession.String, nil
	} else {
		return "", nil
	}
}

func (supa *SupabaseCFG) CheckFinishedExercises(userMail string, sesID int, ctx context.Context) ([]int, error) {
	const query = `
	select finished_ex_ids from "Session"
	where mail = $1 and ses_id = $2
`
	var arr []int

	rows, err := supa.DB.Query(ctx, query, userMail, sesID)
	if err != nil {
		println("DB Query fail (Read finished exercises)")
		println(err.Error())
		return arr, err
	}

	rows.Next()
	err = rows.Scan(&arr)
	if err != nil {
		println("Err row scan (CheckFinished)")
		println(err.Error())

		return arr, err
	}

	return arr, nil
}
