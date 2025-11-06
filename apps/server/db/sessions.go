package db

import (
	"context"
)

func (supa *SupabaseCFG) UserSessions(userMail string, ctx context.Context) ([]string, error) {
	var names []string

	const query = `
        SELECT "ses_name" FROM "Session"
        WHERE "mail" = $1
        ORDER BY "ses_name";
    `
	rows, err := supa.DB.Query(ctx, query, userMail)

	if err != nil {
		println("DB Query fail")
		println(err.Error())
		return names, err

	}
	defer rows.Close()

	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			println("Err row scan")

			return names, err
		}
		names = append(names, name)
	}
	return names, nil
}
