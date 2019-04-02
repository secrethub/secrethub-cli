package secrethub

import (
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"testing"

	"github.com/secrethub/secrethub-go/internals/assert"
)

var (
	// testdata is the package 'global' directory for storing temporary test data.
	testdata *testDataDir
)

func TestMain(m *testing.M) {
	// Setup the testdata directory
	var err error
	testdata, err = newTestDataDir()
	if err != nil {
		log.Fatal(err)
	}

	// Run the tests.
	code := m.Run()

	// Cleanup if everything went well, otherwise leave testdata for inspection.
	if code != 0 {
		err = testdata.cleanup()
		if err != nil {
			log.Fatal(err)
		}
	}

	os.Exit(code)
}

// TestDataDir is a helper type to work with testdata directories.
// These directories are typically used to store state or perform
// tests using the real filesystem.
type testDataDir struct {
	root string
}

// NewTestDataDir creates a new testdata directory inside the current
// working directory and returns it. When running tests, Golang automatically
// sets the working directory to the package's directory. This makes
// the testdata directory appear inside the package directory, so
// make sure to add */testdata/* to your .gitignore file when using this.
func newTestDataDir() (*testDataDir, error) {
	workingDir, err := os.Getwd()
	if err != nil {
		return nil, err
	}

	path := filepath.Join(workingDir, "testdata")

	err = os.MkdirAll(path, 0770)
	if err != nil {
		return nil, err
	}

	return &testDataDir{
		root: path,
	}, nil
}

// Cleanup removes the testdata directory.
func (td testDataDir) cleanup() error {
	return os.RemoveAll(td.root)
}

// TempDir creates a temporary directory, returning its path and a cleanup function.
// It is the caller's responsibility to ensure the cleanup function is called when the
// temporary directory is no longer needed.
func (td testDataDir) tempDir(tb testing.TB) (string, func()) {
	tb.Helper()

	path, err := ioutil.TempDir(td.root, "")
	assert.OK(tb, err)

	// Log it to make debugging easier.
	tb.Logf("created tempdir at %s", path)

	return path, func() {
		tb.Helper()

		err := os.RemoveAll(path)
		assert.OK(tb, err)
	}
}
