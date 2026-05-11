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
	"context"
	"os"
	"strings"
	"sync"
	"testing"

	"github.com/udisondev/veziizi/backend/e2e/setup"
)

var (
	suitesMu sync.Mutex
	suites   = map[string]*setup.Suite{}
)

func TestMain(m *testing.M) {
	code := m.Run()
	setup.ShutdownShared()
	// Tear down all per-class suites manually — we used NewSuiteUnmanaged so
	// nothing else will stop their watermill routers / HTTP servers.
	suitesMu.Lock()
	for _, s := range suites {
		s.Shutdown()
	}
	suitesMu.Unlock()
	// Stop the package-wide Postgres container (per-suite DBs lived inside it).
	if err := setup.StopSharedPostgres(context.Background()); err != nil {
		// Test exit code is what matters; just log.
		_, _ = os.Stderr.WriteString("StopSharedPostgres: " + err.Error() + "\n")
	}
	os.Exit(code)
}

// getSuite returns an isolated test suite scoped to the current Test*Suite
// class (`Test<Name>Suite`), not to each individual t.Run subtest.
//
// One Postgres container + watermill router per Test class means subtests
// inside the same suite see the same state (members_lookup, event bus), while
// parallel Test*Suite classes stay fully isolated. Shared infrastructure
// across classes is forbidden — it hides race bugs in async pipelines (see
// feedback_e2e_isolation memory).
func getSuite(t *testing.T) *setup.Suite {
	t.Helper()

	// t.Name() returns "TestXxxSuite/TestYy" for subtests; first segment is
	// the top-level test that owns the lifecycle.
	className := strings.SplitN(t.Name(), "/", 2)[0]

	suitesMu.Lock()
	defer suitesMu.Unlock()

	if s, ok := suites[className]; ok {
		return s
	}
	// Build a fresh suite on first request for this class. We can't use
	// setup.NewSuite here because it ties the suite lifetime to t.Cleanup of
	// the FIRST subtest that requested it — later subtests would then see a
	// closed pool. Lifecycle is owned by TestMain instead.
	s := setup.NewSuiteUnmanaged(t)
	suites[className] = s
	return s
}

