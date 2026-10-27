//go:build component

// Package tests holds the component tests: the whole service (every module, real Postgres) served
// in-process and driven only through its HTTP API with the generated clients. A test calls one
// endpoint and checks the result through another. Each test signs in as a user of its own, so the
// tests run in parallel and never clean up. Run them with `task test:component`, which creates the
// test database (TEST_DB_SOURCE).
package tests

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/crypto/bcrypt"

	"github.com/erdira/homegym/backend"
	catalogclient "github.com/erdira/homegym/backend/modules/catalog/api/http/client"
	identitydb "github.com/erdira/homegym/backend/modules/identity/adapters/db"
	identityclient "github.com/erdira/homegym/backend/modules/identity/api/http/client"
	planningclient "github.com/erdira/homegym/backend/modules/planning/api/http/client"
	trackingclient "github.com/erdira/homegym/backend/modules/tracking/api/http/client"
	whoopclient "github.com/erdira/homegym/backend/modules/whoop/api/http/client"
)

var (
	// server serves the service under test.
	server *httptest.Server
	// pool is the test database, only for creating users (there is no sign-up).
	pool *pgxpool.Pool
)

func TestMain(m *testing.M) {
	os.Exit(run(m))
}

func run(m *testing.M) int {
	dbSource := os.Getenv("TEST_DB_SOURCE")
	if dbSource == "" {
		fmt.Fprintln(os.Stderr, "TEST_DB_SOURCE is not set; run the component tests with `task test:component`")
		return 1
	}
	ctx := context.Background()
	var err error
	pool, err = backend.NewPool(ctx, dbSource)
	if err != nil {
		fmt.Fprintln(os.Stderr, "connect to the test database:", err)
		return 1
	}
	defer pool.Close()

	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
	svc, err := backend.New(ctx, pool, logger, backend.Config{
		JWTSecret:   "component-test-secret",
		CORSOrigins: []string{"http://localhost"},
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, "start the service:", err)
		return 1
	}
	server = httptest.NewServer(svc.Handler())
	defer server.Close()
	return m.Run()
}

// testPassword is every test user's password.
const testPassword = "component-test"

// passwordHash is testPassword's bcrypt hash, computed once (bcrypt is slow on purpose).
var passwordHash = func() string {
	h, err := bcrypt.GenerateFromPassword([]byte(testPassword), bcrypt.MinCost)
	if err != nil {
		panic(err)
	}
	return string(h)
}()

// user is a test user: its id, an HTTP client carrying its session cookie and a client of each
// module's API on top of it.
type user struct {
	ID       uuid.UUID
	Email    string
	HTTP     *http.Client
	Identity *identityclient.ClientWithResponses
	Catalog  *catalogclient.ClientWithResponses
	Planning *planningclient.ClientWithResponses
	Tracking *trackingclient.ClientWithResponses
	Whoop    *whoopclient.ClientWithResponses
}

// newUser creates a user of the test's own (random email) and signs them in.
func newUser(t *testing.T) user {
	t.Helper()
	u := newUserSignedOut(t)
	resp, err := u.Identity.LoginWithResponse(t.Context(), identityclient.LoginJSONRequestBody{
		Email: u.Email, Password: testPassword,
	})
	if err != nil || resp.StatusCode() != http.StatusOK {
		t.Fatalf("login: %v %s", err, body(resp.Body))
	}
	return u
}

// newUserSignedOut creates a user without signing them in; its client has an empty cookie jar.
func newUserSignedOut(t *testing.T) user {
	t.Helper()
	email := "user-" + uuid.NewString() + "@example.com"
	id, err := identitydb.NewUsersRepo(pool).CreateUser(t.Context(), email, "Test", "User", passwordHash)
	if err != nil {
		t.Fatalf("create user: %v", err)
	}
	jar, _ := cookiejar.New(nil)
	httpClient := &http.Client{Jar: jar}
	u := user{ID: id, Email: email, HTTP: httpClient}
	if u.Identity, err = identityclient.NewClientWithResponses(apiURL(), identityclient.WithHTTPClient(httpClient)); err != nil {
		t.Fatal(err)
	}
	if u.Catalog, err = catalogclient.NewClientWithResponses(apiURL(), catalogclient.WithHTTPClient(httpClient)); err != nil {
		t.Fatal(err)
	}
	if u.Planning, err = planningclient.NewClientWithResponses(apiURL(), planningclient.WithHTTPClient(httpClient)); err != nil {
		t.Fatal(err)
	}
	if u.Tracking, err = trackingclient.NewClientWithResponses(apiURL(), trackingclient.WithHTTPClient(httpClient)); err != nil {
		t.Fatal(err)
	}
	if u.Whoop, err = whoopclient.NewClientWithResponses(apiURL(), whoopclient.WithHTTPClient(httpClient)); err != nil {
		t.Fatal(err)
	}
	return u
}

// body is a response body for failure messages.
func body(b []byte) string {
	if len(b) > 300 {
		return string(b[:300]) + "…"
	}
	return string(b)
}

// apiURL is the base URL the generated clients take (the specs' server is /api).
func apiURL() string { return server.URL + "/api" }
