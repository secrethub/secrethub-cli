package secrethub_test

// This file solely exists for backwards compatibility.

import (
	"testing"

	"github.com/keylockerbv/secrethub-cli/internals/cli/ui"
	"github.com/keylockerbv/secrethub-cli/internals/secrethub"
	"github.com/keylockerbv/secrethub-cli/internals/cli/configuration"
	"github.com/kylelemons/godebug/pretty"
)

func TestConfigMigrations_UsingLatestVersion(t *testing.T) {

	highestVersion := 1
	for _, m := range secrethub.ConfigMigrations {
		if m.VersionTo > highestVersion {
			highestVersion = m.VersionTo
		}
	}

	if highestVersion > secrethub.ConfigVersion {
		t.Errorf("Not using the latest config version, using: %d, latest: %d", secrethub.ConfigVersion, highestVersion)
	}

}

func TestConfigMigrations_MigrationPathAvailable(t *testing.T) {

	_, err := configuration.MigrateConfigTo(ui.NewFakeIO(), configuration.ConfigMap{}, 1, secrethub.ConfigVersion, secrethub.ConfigMigrations, true)

	if err == configuration.ErrVersionNotReachable {
		t.Errorf("No migration path found from config version 1 to the used version %d", secrethub.ConfigVersion)
	}

}

func TestConfigMigrations_FullPath(t *testing.T) {
	configIn := []byte(`{"remote":"https://localhost:8081",` +
		`"credentials":{"auth_id":"auth-id",` +
		`"auth_token":"auth-token"},` +
		`"rpc":{"protocol":"tcp","address":"localhost:1234"},"key_file":"secrethub"}`)

	expectedOut := []byte(`{"remote":"https://localhost:8081",` +
		`"rpc":{"protocol":"tcp","address":"localhost:1234"},` +
		`"type":"user",` +
		`"version":3,` +
		`"user": {"key_file":"secrethub", "username":"test"}}`)

	currentVersion := 1
	goalVersion := secrethub.ConfigVersion

	secrethub.GetUsername = func(ui.IO) (string, error) {
		return "test", nil
	}

	testMigration(t, configIn, expectedOut, currentVersion, goalVersion)
}

func TestConfigMigrations_v1_to_v2(t *testing.T) {

	configIn := []byte(`{"remote":"https://localhost:8081",` +
		`"credentials":{"auth_id":"auth-id",` +
		`"auth_token":"auth-token"},` +
		`"rpc":{"protocol":"tcp","address":"localhost:1234"},"key_file":"secrethub"}`)

	expectedOut := []byte(`{"remote":"https://localhost:8081",` +
		`"credentials":{"auth_id":"auth-id",` +
		`"auth_token":"auth-token"},` +
		`"rpc":{"protocol":"tcp","address":"localhost:1234"},` +
		`"type":"user",` +
		`"version":2,` +
		`"user": {"key_file":"secrethub"}}`)

	currentVersion := 1
	goalVersion := 2

	testMigration(t, configIn, expectedOut, currentVersion, goalVersion)

}

func TestConfigMigrations_v2_to_v3_ServiceNotAvailable(t *testing.T) {

	_, err := configuration.MigrateConfigTo(ui.NewFakeIO(), configuration.ConfigMap{"type": "service"}, 2, 3, secrethub.ConfigMigrations, false)

	if err != secrethub.ErrServiceMigrationNotSupported {
		t.Error("did not throw an error that migration for ServiceConfig is not supported from version 2 to version 3")
	}

}

func TestConfigMigrations_v2_to_v3_CannotParseUser(t *testing.T) {

	secrethub.GetUsername = func(ui.IO) (string, error) {
		return "test", nil
	}

	_, err := configuration.MigrateConfigTo(ui.NewFakeIO(), configuration.ConfigMap{"type": "user", "user": "not-a-map"}, 2, 3, secrethub.ConfigMigrations, false)

	if err != secrethub.ErrCannotParseUserField {
		t.Error("did not throw an error for supplying an illegal value for `user` in config")
	}
}

func TestConfigMigrations_v2_to_v3(t *testing.T) {

	configIn := []byte(`{"remote":"https://localhost:8081",` +
		`"credentials":{"auth_id":"auth-id",` +
		`"auth_token":"auth-token"},` +
		`"rpc":{"protocol":"tcp","address":"localhost:1234"},` +
		`"type":"user",` +
		`"version":2,` +
		`"user": {"key_file":"secrethub"}}`)

	expectedOut := []byte(`{"remote":"https://localhost:8081",` +
		`"rpc":{"protocol":"tcp","address":"localhost:1234"},` +
		`"type":"user",` +
		`"version":3,` +
		`"user": {"key_file":"secrethub", "username":"test"}}`)

	currentVersion := 2
	goalVersion := 3

	secrethub.GetUsername = func(ui.IO) (string, error) {
		return "test", nil
	}

	testMigration(t, configIn, expectedOut, currentVersion, goalVersion)

}

func testMigration(t *testing.T, configIn, expectedOut []byte, currentVersion, goalVersion int) {

	configInMap, err := configuration.ReadMap(configIn)

	if err != nil {
		t.Fatal(err)
	}

	readVersion, err := configInMap.GetVersion()
	if err != nil {
		t.Error(err)
	}

	expectedMap, err := configuration.ReadMap(expectedOut)
	if err != nil {
		t.Fatal(err)
	}

	if readVersion != currentVersion {
		t.Errorf("Returned wrong version for configuration, %d (expected) != %d (actual)", currentVersion, readVersion)
	}

	configOutMap, err := configuration.MigrateConfigTo(ui.NewFakeIO(), configInMap, readVersion, goalVersion, secrethub.ConfigMigrations, false)
	if err != nil {
		t.Error(err)
	}

	if configOutMap["version"] != goalVersion {
		t.Errorf("New config has wrong version number, %d (expected) != %d (actual)", goalVersion, configOutMap["version"])
	}

	diff := pretty.Compare(expectedMap, configOutMap)
	if diff != "" {
		t.Errorf("New config is not as expected. Differences: \n%s", diff)
	}
}
