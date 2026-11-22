package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"

	repo "github.com/nikola-golijanin/greenlight/internal/data/sqlc"
	"github.com/nikola-golijanin/greenlight/internal/mailer"
	"github.com/nikola-golijanin/greenlight/internal/movies"
	"github.com/nikola-golijanin/greenlight/internal/permissions"
	"github.com/nikola-golijanin/greenlight/internal/users"
)

// Package-level state shared across all integration tests.
var (
	testApp    *application
	testRouter http.Handler // built once to avoid duplicate expvar registration
	testDB     *pgxpool.Pool
	testMailer *capturingMailer
)

// capturingMailer satisfies mailer.MailSender and records every sent email
// in memory instead of contacting an SMTP server.
type capturingMailer struct {
	mu   sync.Mutex
	sent []sentEmail
}

type sentEmail struct {
	recipient    string
	templateFile string
	data         map[string]any
}

func (m *capturingMailer) Send(recipient, templateFile string, data any) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	d, _ := data.(map[string]any)
	m.sent = append(m.sent, sentEmail{recipient, templateFile, d})
	return nil
}

// pop returns and removes the oldest captured email, failing the test if none
// is available within 2 seconds (to account for background goroutines).
func (m *capturingMailer) pop(t *testing.T) sentEmail {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		m.mu.Lock()
		if len(m.sent) > 0 {
			e := m.sent[0]
			m.sent = m.sent[1:]
			m.mu.Unlock()
			return e
		}
		m.mu.Unlock()
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatal("timeout waiting for captured email")
	return sentEmail{}
}

func (m *capturingMailer) drain() {
	m.mu.Lock()
	m.sent = nil
	m.mu.Unlock()
}

// Compile-time check that capturingMailer satisfies the interface.
var _ mailer.MailSender = (*capturingMailer)(nil)

func TestMain(m *testing.M) {
	ctx := context.Background()

	pgContainer, err := postgres.Run(ctx,
		"postgres:17-alpine",
		postgres.WithDatabase("greenlight_test"),
		postgres.WithUsername("greenlight"),
		postgres.WithPassword("pa55word"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).
				WithStartupTimeout(30*time.Second),
		),
	)
	if err != nil {
		fmt.Fprintf(os.Stderr, "start postgres container: %v\n", err)
		os.Exit(1)
	}

	defer func() {
		if err := testcontainers.TerminateContainer(pgContainer); err != nil {
			fmt.Fprintf(os.Stderr, "terminate container: %v\n", err)
		}
	}()

	dsn, err := pgContainer.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		fmt.Fprintf(os.Stderr, "get connection string: %v\n", err)
		os.Exit(1)
	}

	cfg := pgxpool.Config{MaxConns: 5}
	parsedCfg, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		fmt.Fprintf(os.Stderr, "parse dsn: %v\n", err)
		os.Exit(1)
	}
	parsedCfg.MaxConns = cfg.MaxConns

	testDB, err = pgxpool.NewWithConfig(ctx, parsedCfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "open db pool: %v\n", err)
		os.Exit(1)
	}
	defer testDB.Close()

	if err := testDB.Ping(ctx); err != nil {
		fmt.Fprintf(os.Stderr, "ping db: %v\n", err)
		os.Exit(1)
	}

	if err := runMigrations(ctx, testDB, "../../internal/data/migrations"); err != nil {
		fmt.Fprintf(os.Stderr, "run migrations: %v\n", err)
		os.Exit(1)
	}

	testMailer = &capturingMailer{}
	repository := repo.New(testDB)

	testApp = &application{
		config: config{
			auth: struct{ enabled bool }{enabled: true},
			limiter: struct {
				requests int
				window   time.Duration
				enabled  bool
			}{enabled: false},
		},
		logger:      slog.New(slog.NewTextHandler(io.Discard, nil)),
		movies:      movies.NewMovieService(repository),
		users:       users.NewUserService(repository, testDB),
		tokens:      users.NewTokenService(repository),
		permissions: permissions.NewPermissionsService(repository, testDB),
		mailer:      testMailer,
	}

	// Build the router exactly once so that expvar names are only registered once.
	testRouter = testApp.routes()

	os.Exit(m.Run())
}

// cleanDB registers a t.Cleanup that truncates mutable tables and drains the
// mailer after the test completes.
func cleanDB(t *testing.T) {
	t.Helper()
	t.Cleanup(func() {
		testApp.wg.Wait() // let any background goroutines finish
		ctx := context.Background()
		if _, err := testDB.Exec(ctx, "TRUNCATE movies, users CASCADE"); err != nil {
			t.Logf("cleanDB: truncate error: %v", err)
		}
		testMailer.drain()
	})
}

// doRequest sends method+path to the test application's router and returns the recorder.
func doRequest(t *testing.T, method, path string, body io.Reader, headers map[string]string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(method, path, body)
	req.Header.Set("Content-Type", "application/json")
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	rr := httptest.NewRecorder()
	testRouter.ServeHTTP(rr, req)
	return rr
}

// jsonBody encodes v as JSON and returns a reader.
func jsonBody(t *testing.T, v any) io.Reader {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatalf("jsonBody: %v", err)
	}
	return bytes.NewReader(b)
}

// bearerHeader returns a header map carrying the given bearer token.
func bearerHeader(token string) map[string]string {
	return map[string]string{"Authorization": "Bearer " + token}
}

// registerUser registers a new user and returns the activation token captured
// from the outgoing welcome email.
func registerUser(t *testing.T, name, email, password string) string {
	t.Helper()
	rr := doRequest(t, http.MethodPost, "/v1/users", jsonBody(t, map[string]any{
		"name":     name,
		"email":    email,
		"password": password,
	}), nil)
	if rr.Code != http.StatusAccepted {
		t.Fatalf("registerUser: want 202, got %d: %s", rr.Code, rr.Body.String())
	}
	testApp.wg.Wait()
	mail := testMailer.pop(t)
	return mail.data["activationToken"].(string)
}

// activateUser activates the user with the given activation token.
func activateUser(t *testing.T, token string) {
	t.Helper()
	rr := doRequest(t, http.MethodPut, "/v1/users/activated", jsonBody(t, map[string]any{
		"token": token,
	}), nil)
	if rr.Code != http.StatusOK {
		t.Fatalf("activateUser: want 200, got %d: %s", rr.Code, rr.Body.String())
	}
}

// loginUser authenticates and returns a bearer token string.
func loginUser(t *testing.T, email, password string) string {
	t.Helper()
	rr := doRequest(t, http.MethodPost, "/v1/tokens/authentication", jsonBody(t, map[string]any{
		"email":    email,
		"password": password,
	}), nil)
	if rr.Code != http.StatusCreated {
		t.Fatalf("loginUser: want 201, got %d: %s", rr.Code, rr.Body.String())
	}
	var body struct {
		Token string `json:"token"`
	}
	if err := json.NewDecoder(rr.Body).Decode(&body); err != nil {
		t.Fatalf("loginUser: decode response: %v", err)
	}
	return body.Token
}

// grantPermission inserts a movies:write permission for the given user directly
// via the DB, so integration tests can test write endpoints without going through
// the full user-creation flow every time.
func grantWritePermission(t *testing.T, userID int) {
	t.Helper()
	_, err := testDB.Exec(context.Background(),
		`INSERT INTO users_permissions (user_id, permission_id)
         SELECT $1, id FROM permissions WHERE code = 'movies:write'
         ON CONFLICT DO NOTHING`, userID)
	if err != nil {
		t.Fatalf("grantWritePermission: %v", err)
	}
}

// runMigrations reads all *.sql goose migration files from dir in filename order
// and executes the "Up" section of each against db.
func runMigrations(ctx context.Context, db *pgxpool.Pool, dir string) error {
	// Enable extensions required by the schema before running migrations.
	if _, err := db.Exec(ctx, "CREATE EXTENSION IF NOT EXISTS citext"); err != nil {
		return fmt.Errorf("create citext extension: %w", err)
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		return fmt.Errorf("read dir %s: %w", dir, err)
	}

	names := make([]string, 0, len(entries))
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".sql") {
			names = append(names, e.Name())
		}
	}
	sort.Strings(names)

	for _, name := range names {
		content, err := os.ReadFile(filepath.Join(dir, name))
		if err != nil {
			return fmt.Errorf("read %s: %w", name, err)
		}
		up := extractUpSection(string(content))
		if up == "" {
			continue
		}
		if _, err := db.Exec(ctx, up); err != nil {
			return fmt.Errorf("exec %s: %w", name, err)
		}
	}
	return nil
}

// extractUpSection returns the SQL between "-- +goose Up" and "-- +goose Down"
// (or the end of the file if there is no Down section).
func extractUpSection(content string) string {
	const upMarker = "-- +goose Up"
	const downMarker = "-- +goose Down"
	start := strings.Index(content, upMarker)
	if start < 0 {
		return ""
	}
	content = content[start+len(upMarker):]
	if end := strings.Index(content, downMarker); end >= 0 {
		content = content[:end]
	}
	return strings.TrimSpace(content)
}
