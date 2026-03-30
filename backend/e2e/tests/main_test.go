// Package tests contains E2E tests for the veziizi API.
//
// Run tests with:
//
//	cd /path/to/veziizi4
//	make test-e2e
//
// Or manually:
//
//	TEST_DATABASE_URL=postgres://veziizi:veziizi@localhost:5432/veziizi_test?sslmode=disable go test -v ./backend/e2e/tests/...
package tests

import (
	"os"
	"testing"

	"github.com/udisondev/veziizi/backend/e2e/client"
	"github.com/udisondev/veziizi/backend/e2e/setup"
)

var (
	// baseURL is the base URL of the test server.
	baseURL string

	// apiClient is a shared client without authentication.
	apiClient *client.Client
)

func TestMain(m *testing.M) {
	// This is a package-level setup. Individual test files can use
	// setup.GetSharedSuite(t) to get the shared suite, or
	// setup.NewSuite(t) for isolated tests.

	// For now, we'll let individual tests set up their own suites
	// since shared suite setup requires a *testing.T which isn't
	// available in TestMain.

	code := m.Run()

	// Cleanup shared suite if it was created
	setup.ShutdownShared()

	os.Exit(code)
}

// getSuite returns a shared test suite for the current test.
// Use this for tests that can share infrastructure.
func getSuite(t *testing.T) *setup.Suite {
	return setup.GetSharedSuite(t)
}

// getClient returns a new API client connected to the test server.
func getClient(t *testing.T) *client.Client {
	suite := getSuite(t)
	return client.New(suite.BaseURL)
}
