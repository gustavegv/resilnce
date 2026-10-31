// portdata loads the same .env and database configuration as the server.
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"
	_ "time/tzdata"

	"github.com/gustavegv/resilnce/apps/server/migrations"
	"github.com/gustavegv/resilnce/apps/server/porting"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
)

func main() { os.Exit(run()) }

func run() int {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "usage: portdata migrate|audit|run|resume|status|worker [--owner email] [--source-timezone zone] [--delete-unused-exercises] [--delete-unused-history] [--actor email] [--id uuid]")
		return 1
	}
	command := os.Args[1]
	flags := flag.NewFlagSet(command, flag.ContinueOnError)
	owner := flags.String("owner", "", "single-account scope; empty means all accounts")
	zone := flags.String("source-timezone", porting.DefaultSourceTimezone, "IANA timezone for naive legacy timestamps")
	deleteUnused := flags.Bool("delete-unused-exercises", false, "delete unmapped exercises with no session placements or recorded history during import")
	deleteHistory := flags.Bool("delete-unused-history", false, "also delete legacy history for unused exercises (requires --delete-unused-exercises)")
	actor := flags.String("actor", "", "identity initiating an operational apply run")
	id := flags.String("id", "", "durable run ID for status/resume")
	if err := flags.Parse(os.Args[2:]); err != nil || flags.NArg() != 0 {
		return 1
	}
	switch command {
	case "migrate", "audit", "run", "resume", "status", "worker":
	default:
		fmt.Fprintln(os.Stderr, "unknown command")
		return 1
	}
	if command == "run" && *actor == "" {
		fmt.Fprintln(os.Stderr, "--actor is required")
		return 1
	}
	if (command == "resume" || command == "status") && *id == "" {
		fmt.Fprintln(os.Stderr, "--id is required")
		return 1
	}
	opts := porting.Options{SourceTimezone: *zone, DeleteUnusedExercises: *deleteUnused, DeleteUnusedHistory: *deleteHistory}
	if err := opts.Validate(); err != nil {
		fmt.Fprintln(os.Stderr, "invalid import options:", err)
		return 1
	}
	dsn, err := loadDatabaseURL()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	config, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		fmt.Fprintln(os.Stderr, "invalid database connection URL (details omitted)")
		return 1
	}
	config.MaxConns = 6
	config.ConnConfig.ConnectTimeout = 10 * time.Second
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()
	pool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		fmt.Fprintln(os.Stderr, "database connection failed (details omitted)")
		return 1
	}
	defer pool.Close()
	if err = pool.Ping(ctx); err != nil {
		fmt.Fprintln(os.Stderr, "database connection failed (details omitted)")
		return 1
	}
	s := porting.Service{Pool: pool, AllowApply: porting.ApplyEnabled()}
	var result any
	switch command {
	case "migrate":
		err = migrations.Apply(ctx, pool)
		result = map[string]string{"status": "migrations applied"}
	case "audit":
		result, err = s.Audit(ctx, *owner, opts)
	case "status":
		result, err = s.GetRun(ctx, *id)
	case "run", "resume":
		if command == "run" {
			*id, err = s.StartRun(ctx, *actor, "apply", *owner, opts)
		} else {
			err = s.ResumeRun(ctx, *id)
		}
		if err == nil {
			log.Printf("run %s queued", *id)
			_, err = s.WorkOne(ctx, *id)
			if err == nil {
				result, err = s.GetRun(ctx, *id)
			}
		}
	case "worker":
		for ctx.Err() == nil {
			worked, e := s.WorkOne(ctx, "")
			if e != nil && ctx.Err() == nil {
				log.Printf("porting worker: %v", e)
			}
			if !worked || e != nil {
				select {
				case <-ctx.Done():
				case <-time.After(2 * time.Second):
				}
			}
		}
		return 0
	}
	if err != nil {
		if !errors.Is(err, context.Canceled) {
			fmt.Fprintln(os.Stderr, err)
		}
		return 1
	}
	encoded, err := porting.JSONReport(result)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	fmt.Println(string(encoded))
	if r, ok := result.(porting.Run); ok && r.Status != "completed" {
		return 2
	}
	if reports, ok := result.([]porting.OwnerReport); ok {
		for _, r := range reports {
			if !r.Ready {
				return 2
			}
		}
	}
	return 0
}

func loadDatabaseURL() (string, error) {
	// Load preserves exported values. A missing file is normal in deployments
	// that supply their configuration entirely through the process environment.
	if err := godotenv.Load(); err != nil && !errors.Is(err, os.ErrNotExist) {
		return "", errors.New("could not load .env (details omitted)")
	}
	if dsn := os.Getenv("PORTING_DATABASE_URL"); dsn != "" {
		return dsn, nil
	}
	if dsn := os.Getenv("DATABASE_URL"); dsn != "" {
		return dsn, nil
	}
	return "", errors.New("set DATABASE_URL in .env or the environment, or override it with PORTING_DATABASE_URL")
}
