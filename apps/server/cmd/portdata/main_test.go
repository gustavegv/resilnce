package main

import (
	"os"
	"testing"
)

func TestDatabaseConfigurationLoadsDotenvAndRespectsOverrides(t *testing.T) {
	for _, tc := range []struct {
		name                    string
		file                    *string
		database, porting, want string
		wantError               bool
	}{
		{name: "same dotenv as server", file: contents("DATABASE_URL=postgres://dotenv/example\nPORTING_APPLY_ENABLED=true\n"), want: "postgres://dotenv/example"},
		{name: "exported database wins over dotenv", file: contents("DATABASE_URL=postgres://dotenv/example\n"), database: "postgres://exported/example", want: "postgres://exported/example"},
		{name: "explicit porting override", file: contents("DATABASE_URL=postgres://dotenv/example\nPORTING_DATABASE_URL=postgres://dotenv-porting/example\n"), porting: "postgres://rehearsal/example", want: "postgres://rehearsal/example"},
		{name: "porting override in dotenv", file: contents("DATABASE_URL=postgres://dotenv/example\nPORTING_DATABASE_URL=postgres://rehearsal/example\n"), want: "postgres://rehearsal/example"},
		{name: "environment without dotenv", database: "postgres://deployed/example", want: "postgres://deployed/example"},
		{name: "missing configuration", wantError: true},
		{name: "invalid dotenv", file: contents("DATABASE_URL='unclosed\n"), wantError: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Chdir(t.TempDir())
			t.Setenv("DATABASE_URL", tc.database)
			t.Setenv("PORTING_DATABASE_URL", tc.porting)
			t.Setenv("PORTING_APPLY_ENABLED", "")
			// godotenv correctly preserves even empty exported variables. Unset
			// empty test values so the file can supply them, restoring them on exit.
			for _, key := range []string{"DATABASE_URL", "PORTING_DATABASE_URL", "PORTING_APPLY_ENABLED"} {
				if os.Getenv(key) == "" {
					if err := os.Unsetenv(key); err != nil {
						t.Fatal(err)
					}
				}
			}
			if tc.file != nil {
				if err := os.WriteFile(".env", []byte(*tc.file), 0600); err != nil {
					t.Fatal(err)
				}
			}
			got, err := loadDatabaseURL()
			if (err != nil) != tc.wantError {
				t.Fatalf("unexpected configuration error: %v", err)
			}
			if got != tc.want {
				t.Fatal("wrong database configuration selected")
			}
			if tc.name == "same dotenv as server" && os.Getenv("PORTING_APPLY_ENABLED") != "true" {
				t.Fatal("worker options were not loaded from .env")
			}
		})
	}
}

func contents(value string) *string { return &value }
