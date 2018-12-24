package secrethub

import (
	"log"
	"os"
	"testing"

	"github.com/keylockerbv/secrethub/testutil"
)

var (
	// testdata is the package 'global' directory for storing temporary test data.
	testdata *testutil.TestDataDir
)

func TestMain(m *testing.M) {
	// Setup the testdata directory
	var err error
	testdata, err = testutil.NewTestDataDir()
	if err != nil {
		log.Fatal(err)
	}

	// Run the tests.
	code := m.Run()

	// Cleanup if everything went well, otherwise leave testdata for inspection.
	if code != 0 {
		err = testdata.Cleanup()
		if err != nil {
			log.Fatal(err)
		}
	}

	os.Exit(code)
}
