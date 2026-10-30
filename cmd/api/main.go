package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	repo "github.com/nikola-golijanin/greenlight/internal/data/sqlc"
	"github.com/nikola-golijanin/greenlight/internal/movies"
	"github.com/nikola-golijanin/greenlight/internal/users"
)

const version = "1.0.0"

type config struct {
	port int
	env  string
	db   struct {
		dsn          string
		maxOpenConns int
		maxIdleTime  time.Duration
	}
	limiter struct {
		requests int
		window   time.Duration
		enabled  bool
	}
}

type application struct {
	config config
	logger *slog.Logger
	movies movies.MovieService
	users  users.UserService
}

func main() {
	var cfg config

	flag.IntVar(&cfg.port, "port", 4000, "API server port")
	flag.StringVar(&cfg.env, "env", "development", "Environment (development|production)")

	flag.StringVar(&cfg.db.dsn, "db-dsn", "postgres://postgres:a-secure-password@localhost/greenlight?sslmode=disable", "PostgreSQL DSN")
	flag.IntVar(&cfg.db.maxOpenConns, "db-max-open-conns", 25, "PostgreSQL max open connections")
	flag.DurationVar(&cfg.db.maxIdleTime, "db-max-idle-time", 15*time.Minute, "PostgreSQL max connection idle time")

	flag.IntVar(&cfg.limiter.requests, "limiter-requests", 10, "Rate limiter max requests per window")
	flag.DurationVar(&cfg.limiter.window, "limiter-window", 5*time.Second, "Rate limiter window length")
	flag.BoolVar(&cfg.limiter.enabled, "limiter-enabled", true, "Enable rate limiter")

	flag.Parse()

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	db, err := openDB(cfg)
	if err != nil {
		logger.Error(err.Error())
		os.Exit(1)
	}

	defer db.Close()
	logger.Info("database connection pool established")

	app := &application{
		config: cfg,
		logger: logger,
		movies: movies.NewMovieService(repo.New(db)),
		users:  users.NewUserService(repo.New(db)),
	}

	err = app.serve()
	if err != nil {
		logger.Error(err.Error())
		os.Exit(1)
	}
}

func openDB(cfg config) (*pgxpool.Pool, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	dbCfg, err := pgxpool.ParseConfig(cfg.db.dsn)
	dbCfg.MaxConns = int32(cfg.db.maxOpenConns)
	dbCfg.MaxConnIdleTime = cfg.db.maxIdleTime

	dbpool, err := pgxpool.NewWithConfig(ctx, dbCfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Unable to create connection pool: %v\n", err)
		return nil, err
	}

	if err := dbpool.Ping(ctx); err != nil {
		dbpool.Close()
		return nil, fmt.Errorf("ping db: %w", err)
	}

	return dbpool, nil
}
