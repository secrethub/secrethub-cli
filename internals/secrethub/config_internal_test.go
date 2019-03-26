package secrethub

// This file solely exists for backwards compatibility.

import (
	"github.com/secrethub/secrethub-go/internals/assert"
	"os"
	"testing"

	"net/url"

	"github.com/keylockerbv/secrethub-cli/internals/cli/configuration"
	"github.com/keylockerbv/secrethub-cli/internals/cli/ui"
	"github.com/secrethub/secrethub-go/internals/api/uuid"
)

var testPaths = []string{
	"testconfig.yml",
	"testconfig.json",
	".testconfig",
}

func TestLoadConfig_User(t *testing.T) {
	for _, path := range testPaths {
		t.Logf("testing path: %s", path)
		testLoadConfigUser(t, path)
	}
}

func testLoadConfigUser(t *testing.T, path string) {
	url := &url.URL{}

	// Arrange
	expected, err := newUserConfig("TESTUSER", "key.pem", url)
	if err != nil {
		t.Fatalf("newUserConfig gave error: %s", err)
	}

	err = configuration.WriteToFile(expected, path, oldConfigFileMode)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		err := os.Remove(path)
		assert.OK(t, err)
	}()

	// Act
	actual, err := LoadConfig(ui.NewFakeIO(), path)
	if err != nil {
		t.Fatalf("LoadConfig gave error: %s", err)
	}

	// Assert
	testConfigResult(t, *actual, *expected)

	if actual.User.Username != expected.User.Username {
		t.Errorf("unexpected username:\n\t %v (actual) != %v (expected)",
			actual.User.Username, expected.User.Username)
	}

	if actual.User.KeyFile != expected.User.KeyFile {
		t.Errorf("unexpected key file:\n\t %v (actual) != %v (expected)",
			actual.User.KeyFile, expected.User.KeyFile)
	}
}

func TestLoadConfig_Service(t *testing.T) {
	for _, path := range testPaths {
		t.Logf("testing path: %s", path)
		testLoadConfigService(t, path)
	}
}

func testLoadConfigService(t *testing.T, path string) {
	// Arrange
	// configTestPath := "serviceTestConfig.yaml"
	expected := newServiceConfig(uuid.New(), uuid.New(),
		"VERYLONGKEY", "https://localhost:8080")

	err := configuration.WriteToFile(expected, path, oldConfigFileMode)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		err := os.Remove(path)
		assert.OK(t, err)
	}()

	// Act
	actual, err := LoadConfig(ui.NewFakeIO(), path)
	if err != nil {
		t.Fatalf("LoadConfig gave error: %s", err)
	}

	// Assert
	testConfigResult(t, *actual, expected)

	if !uuid.Equal(actual.Service.RepoID, expected.Service.RepoID) {
		t.Errorf("unexpected repoID:\n\t %v (actual) != %v (expected)",
			actual.Service.RepoID, expected.Service.RepoID)
	}

	if !uuid.Equal(actual.Service.AccountID, expected.Service.AccountID) {
		t.Errorf("unexpected AccountID:\n\t %v (actual) != %v (expected)",
			actual.Service.AccountID, expected.Service.AccountID)
	}

	if actual.Service.PrivateKey != expected.Service.PrivateKey {
		t.Errorf("unexpected PrivateKey:\n\t %v (actual) != %v (expected)",
			actual.Service.PrivateKey, expected.Service.PrivateKey)
	}
}

func TestValidateUserConfig_NoUsername(t *testing.T) {
	// Arrange
	config := Config{
		User: &userConfig{
			KeyFile: "path/to/key",
		},
	}

	// Act
	err := (*config.User).validate()

	// Assert
	if err == nil {
		t.Error("validating a config with no username should error but did not")
	}
}

func TestValidateUserConfig_NoSSHKey(t *testing.T) {
	// Arrange
	config := Config{
		User: &userConfig{
			Username: "JDoe",
		},
	}

	// Act
	err := (*config.User).validate()

	// Assert
	if err == nil {
		t.Error("validating a config with no key file should error but did not")
	}
}

func TestValidateServiceConfig_NoID(t *testing.T) {
	// Arrange
	config := Config{
		Service: &serviceConfig{
			PrivateKey: "TESTKEY",
		},
	}

	// Act
	err := (*config.Service).validate()

	// Assert
	if err == nil {
		t.Error("validating a config with no account id should error but did not")
	}
}

func TestValidateServiceConfig_NoKey(t *testing.T) {
	// Arrange
	config := Config{
		Service: &serviceConfig{
			AccountID: uuid.New(),
		},
	}

	// Act
	err := (*config.Service).validate()

	// Assert
	if err == nil {
		t.Error("validating a config with no key should error but did not")
	}
}

func testConfigResult(t *testing.T, actual, expected Config) {
	if actual.Version != expected.Version {
		t.Errorf("unexpected version:\n\t %v (actual) != %v (expected)",
			actual.Version, expected.Version)
	}

	if actual.Remote != expected.Remote {
		t.Errorf("unexpected remote:\n\t %v (actual) != %v (expected)",
			actual.Remote, expected.Remote)
	}
}

func TestValidateRemote(t *testing.T) {
	tests := []struct {
		url      string
		expected error
	}{
		{
			"localhost:8080",
			ErrInvalidRemote,
		},
		{
			"localhost",
			ErrInvalidRemote,
		},
		{
			"tcp://localhost:8080",
			ErrInvalidRemote,
		},
		{
			"httpbin.org",
			ErrInvalidRemote,
		},
		{
			"http://localhost:8080",
			nil,
		},
		{
			"http://127.0.0.1",
			nil,
		},
		{
			"https://localhost:8080",
			nil,
		},
		{
			"https://127.0.0.1",
			nil,
		},
	}

	for _, test := range tests {
		err := ValidateRemote(test.url)
		if err != test.expected {
			t.Errorf("unepected error: %v (actual) != %v (expected)", err, test.expected)
		}
	}
}
