package secretspec

import (
	"github.com/secrethub/secrethub-go/internals/assert"
	"os"
	"testing"
)

func TestPresenter_Parse_Success(t *testing.T) {

	p, err := NewPresenter("./", true, DefaultParsers...)
	if err != nil {
		t.Fatal(err)
	}

	_, err = os.Create("inject_test_file")
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		err := os.Remove("inject_test_file")
		assert.OK(t, err)
	}()

	spec := []byte(
		`
secrets:
    - inject:
        source: "inject_test_file"
        target: "inject_target"
        filemode: "0777"
    - inject:
        source: "inject_test_file"
        target: "inject_target2"
        filemode: "0777"
    - file:
        source: "user/repo/secret"
        target: "file_target"
        filemode: "0777"
    - file:
        source: "user/repo/secret2"
        target: "file_target2"
        filemode: "0777"
    - env:
        vars:
            TEST: user/repo/secret3
    - env:
        name: "environment_name"
        vars:
            TEST: user/repo/secret`)

	err = p.Parse(spec)
	if err != nil {
		t.Error(err)
	}
}

func TestPresenter_Parse_DuplicateEnvConsumable(t *testing.T) {

	p, err := NewPresenter("./", true, DefaultParsers...)
	if err != nil {
		t.Fatal(err)
	}

	spec := []byte(
		`
secrets:
    - env:
        name: "environment_name"
        vars:
            TEST: user/repo/secret3
    - env:
        name: "environment_name"
        vars:
            TEST: user/repo/secret`)

	err = p.Parse(spec)
	if err == nil {
		t.Error("did not get a ErrDuplicateConsumable for duplicate Env consumable")
	}
}

func TestPresenter_Parse_DuplicateFileConsumable(t *testing.T) {

	p, err := NewPresenter("./", true, DefaultParsers...)
	if err != nil {
		t.Fatal(err)
	}

	_, err = os.Create("inject_test_file")
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		err := os.Remove("inject_test_file")
		assert.OK(t, err)
	}()

	spec := []byte(
		`
secrets:
    - file:
        source: "user/repo/secret"
        target: "file_target"
        filemode: "0777"
    - file:
        source: "user/repo/secret"
        target: "file_target"
        filemode: "0777"`)

	err = p.Parse(spec)
	if err == nil {
		t.Error("did not get a ErrDuplicateConsumable for duplicate File consumable")
	}
}

func TestPresenter_Parse_DuplicateInjectConsumable(t *testing.T) {

	p, err := NewPresenter("./", true, DefaultParsers...)
	if err != nil {
		t.Fatal(err)
	}

	spec := []byte(
		`
secrets:
    - inject:
        source: "inject_test_file"
        target: "inject_target"
        filemode: "0777"
    - inject:
        source: "inject_test_file"
        target: "inject_target"
        filemode: "0777"`)

	err = p.Parse(spec)
	if err == nil {
		t.Error("did not get a ErrDuplicateConsumable for duplicate Inject consumable")
	}
}
