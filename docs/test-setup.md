# Test Setup

## Overview

Tests are split into two layers with no build tags — all files compile normally, so the IDE resolves every symbol and can run any test directly.

| Layer | Location | Needs Docker |
|---|---|---|
| Unit | `internal/...` | No |
| Integration | `cmd/api/...` | Yes |

```bash
make test/unit         # go test -race ./internal/...
make test/integration  # go test -race -count=1 ./cmd/api/...
make test/all          # go test -race -count=1 ./...
```

---

## Why `TestMain`

Go allows a package to define a single `TestMain(m *testing.M)` function. When present, the test binary calls it instead of running tests directly. `TestMain` is responsible for setup, calling `m.Run()` to execute the tests, and cleanup.

`cmd/api/testmain_test.go` uses this to build shared infrastructure once for the entire `cmd/api` test run:

```
TestMain
  ├── start Postgres container
  ├── run migrations
  ├── build *application
  ├── build http.Handler (router) — once only
  └── m.Run()  ← all tests execute here
      └── defer: terminate container
```

---

## Testcontainers

[testcontainers-go](https://golang.testcontainers.org) starts a real Docker container from within the test process. No external setup or running database is needed.

```go
pgContainer, err := postgres.Run(ctx,
    "postgres:17-alpine",
    postgres.WithDatabase("greenlight_test"),
    postgres.WithUsername("greenlight"),
    postgres.WithPassword("pa55word"),
    testcontainers.WithWaitStrategy(
        wait.ForLog("database system is ready to accept connections").
            WithOccurrence(2),
    ),
)
```

- The image is pulled from Docker Hub on first run, cached locally after that.
- `WithWaitStrategy` blocks until Postgres logs the ready message twice (once for template1, once for the actual database), so the first connection is never refused.
- The container is terminated in a deferred function inside `TestMain`, guaranteeing cleanup even if tests panic.
- Testcontainers also runs a Ryuk sidecar container that acts as a reaper — if the test process is killed hard, Ryuk cleans up the containers automatically.

---

## Migrations

The project uses [Goose](https://pressboard.github.io/goose/) for migrations in development, but Goose is a CLI tool, not imported as a library. To avoid adding a dependency just for tests, `runMigrations` does the same work inline:

1. Reads all `*.sql` files from `internal/data/migrations/` sorted by filename (the timestamp prefix guarantees order).
2. Strips the `-- +goose Down` section from each file, keeping only the `-- +goose Up` SQL.
3. Executes each UP block against the container database.

Before running any migration, `CREATE EXTENSION IF NOT EXISTS citext` is run because the users table uses the `citext` column type, which lives in a Postgres extension not enabled by default.

---

## Shared package-level state

```go
var (
    testApp    *application   // the fully wired application
    testRouter http.Handler   // the Chi router, built once
    testDB     *pgxpool.Pool  // direct DB access for setup/teardown
    testMailer *capturingMailer
)
```

**Why build the router once?**
The `RecordRequestMetrics` middleware registers `expvar` counters (e.g. `total_requests_received`) globally by name when it is first wired into the router. Calling `testApp.routes()` a second time would try to register the same names again and panic. Building the router once in `TestMain` avoids this.

---

## capturingMailer

The real mailer sends email over SMTP. In tests, registering a user triggers a background goroutine that sends a welcome email containing the activation token. That token is not returned by the HTTP response — it only exists in the email.

`capturingMailer` implements the `mailer.MailSender` interface and stores every sent email in memory instead of contacting an SMTP server:

```go
type capturingMailer struct {
    mu   sync.Mutex
    sent []sentEmail
}

func (m *capturingMailer) Send(recipient, templateFile string, data any) error {
    m.mu.Lock()
    defer m.mu.Unlock()
    m.sent = append(m.sent, sentEmail{...})
    return nil
}
```

The mutex is required because `Send` is called from a background goroutine while the test goroutine may be reading `sent`.

**Retrieving the activation token** in tests works like this:

```go
activationToken := registerUser(t, "Alice", "alice@example.com", "pa55word!")
```

Inside `registerUser`:
1. POST `/v1/users` — handler returns 202 and launches a background goroutine.
2. `testApp.wg.Wait()` — blocks until that goroutine completes (the mailer is instant, so this returns immediately).
3. `testMailer.pop(t)` — returns the captured email; the activation token is in `email.data["activationToken"]`.

---

## Test isolation with `cleanDB`

Each integration test that writes to the database calls `cleanDB(t)` at the top. This registers a `t.Cleanup` callback that runs after the test completes:

```go
func cleanDB(t *testing.T) {
    t.Cleanup(func() {
        testApp.wg.Wait()
        testDB.Exec(ctx, "TRUNCATE movies, users CASCADE")
        testMailer.drain()
    })
}
```

- `TRUNCATE movies, users CASCADE` removes all test data. The `CASCADE` propagates to `tokens` and `users_permissions` via their foreign keys. The `permissions` table is left intact because it was seeded by the migration (`movies:read`, `movies:write`) and must not be removed between tests.
- `testMailer.drain()` discards any unsent emails so they don't bleed into the next test.
- `testApp.wg.Wait()` is called first to ensure all background goroutines have finished writing before truncation.

`t.Cleanup` runs after the test body, in LIFO order relative to other cleanups registered in the same test. Because tests in a package run sequentially by default, the cleanup from one test always finishes before the next test starts.

---

## `doRequest` helper

Integration tests never construct an `*http.Server`. Instead, `doRequest` feeds a synthetic `*http.Request` directly into the router and captures the response in memory:

```go
func doRequest(t *testing.T, method, path string, body io.Reader, headers map[string]string) *httptest.ResponseRecorder {
    req := httptest.NewRequest(method, path, body)
    req.Header.Set("Content-Type", "application/json")
    for k, v := range headers {
        req.Header.Set(k, v)
    }
    rr := httptest.NewRecorder()
    testRouter.ServeHTTP(rr, req)
    return rr
}
```

`httptest.NewRequest` creates a request with a synthetic remote address (`192.0.2.1:1234`). `httptest.NewRecorder` captures the status code, headers, and body written by the handler. The full middleware stack (authentication, rate limiting, CORS, metrics) runs exactly as it would in production because we go through the same `http.Handler`.
