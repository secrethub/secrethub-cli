package configuration

import (
	"testing"

	"github.com/keylockerbv/secrethub-cli/pkg/ui"
)

var testMigrations = []Migration{
	{VersionFrom: 1, VersionTo: 2, UpdateFunc: testMigration1},
	{VersionFrom: 2, VersionTo: 3, UpdateFunc: testMigration2},
	{VersionFrom: 3, VersionTo: 4, UpdateFunc: testMigration3},
	{VersionFrom: 4, VersionTo: 5, UpdateFunc: testMigration3},
}

func TestConfigMigrater(t *testing.T) {

	config, err := getTestConfig()
	if err != nil {
		t.Fatal(err)
	}
	version, err := config.GetVersion()
	if err != nil {
		t.Error(err)
	}

	goalVersion := 4

	config, err = MigrateConfigTo(ui.NewFakeIO(), config, version, goalVersion, testMigrations, false)

	if err != nil {
		t.Error(err)
	}

	if !(config["applied_migration_1"] == true && config["applied_migration_2"] == true && config["applied_migration_3"] == true && config["migration_count"] == 3) {
		t.Errorf("Not all migrations were applied correctly. This was the result: %s", config)
	}

	if config["change"] == "not_changed" {
		t.Errorf("Migration did not correctly change `change`")
	}

	if config["not_change"] != "not_changed" {
		t.Errorf("Migration changed `not_change`")
	}

}

func TestConfigMigrater_NotReachable(t *testing.T) {
	config, err := getTestConfig()
	if err != nil {
		t.Fatal(err)
	}
	version, err := config.GetVersion()
	if err != nil {
		t.Error(err)
	}

	goalVersion := 6

	_, err = MigrateConfigTo(ui.NewFakeIO(), config, version, goalVersion, testMigrations, false)

	if err != ErrVersionNotReachable {
		t.Error("Did not throw a ErrVersionNotReachable for an unreachable version")
	}
}

func getTestConfig() (ConfigMap, error) {

	data := []byte(`{` +
		`"applied_migration_1": false,` +
		`"applied_migration_2": false,` +
		`"not_change": "not_changed",` +
		`"change": "not_changed",` +
		`"migration_count": 0` +
		`}`)

	return ReadMap(data)
}

func testMigration1(_ ui.IO, src ConfigMap) (ConfigMap, error) {

	src["applied_migration_1"] = true
	src["change"] = "changed"
	src["migration_count"] = src["migration_count"].(int) + 1

	return src, nil
}

func testMigration2(_ ui.IO, src ConfigMap) (ConfigMap, error) {

	src["applied_migration_2"] = true
	src["migration_count"] = src["migration_count"].(int) + 1

	return src, nil
}

func testMigration3(_ ui.IO, src ConfigMap) (ConfigMap, error) {

	src["applied_migration_3"] = true
	src["migration_count"] = src["migration_count"].(int) + 1

	return src, nil
}
