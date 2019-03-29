package secrethub

// This file solely exists for backwards compatibility.

import (
	"fmt"

	"github.com/secrethub/secrethub-cli/internals/cli/configuration"
	"github.com/secrethub/secrethub-cli/internals/cli/ui"
)

var (
	// ConfigMigrations contains all the configuration migrations for secrethub
	ConfigMigrations = []configuration.Migration{
		{VersionFrom: 1, VersionTo: 2, UpdateFunc: configMigrationV1toV2},
		{VersionFrom: 2, VersionTo: 3, UpdateFunc: configMigrationV2toV3},
	}

	// GetUsername returns the username of the current user as this is needed for a migration
	GetUsername = askForUsername
)

// Errors
var (
	// ErrServiceMigrationNotSupported is given when user tries to migrate a version 1 service config, which is not supported
	ErrServiceMigrationNotSupported = errConfig.Code("service_migration_not_supported").Error(
		"service migration not supported to this version. Please remove this service and create a new one")

	// ErrCannotParseUserField is given when the `user` field in a config could not be parsed correctly
	ErrCannotParseUserField = errConfig.Code("cannot_parse_user").Error("could not correctly parse field `user` for config migration")
)

func configMigrationV1toV2(_ ui.IO, src configuration.ConfigMap) (configuration.ConfigMap, error) {

	keyFile := src["key_file"]

	userConfig := map[interface{}]interface{}{
		"key_file": keyFile,
	}

	remote := src["remote"]
	credentials := src["credentials"]
	rpc := src["rpc"]

	newConfig := configuration.ConfigMap{
		"remote":      remote,
		"credentials": credentials,
		"rpc":         rpc,
		"type":        "user",
		"user":        userConfig,
	}

	return newConfig, nil
}

func configMigrationV2toV3(io ui.IO, src configuration.ConfigMap) (configuration.ConfigMap, error) {

	if src["type"] != "user" {
		return src, ErrServiceMigrationNotSupported
	}

	username, err := GetUsername(io)

	if err != nil {
		return src, err
	}

	user, ok := src["user"].(map[interface{}]interface{})
	if !ok {
		return src, ErrCannotParseUserField
	}
	user["username"] = username

	delete(src, "credentials")

	return src, nil
}

func askForUsername(io ui.IO) (string, error) {
	for {
		username, err := ui.Ask(io, "To migrate user configuration, please enter username: ")
		if err != nil {
			return "", err
		}

		confirmed, err := ui.AskYesNo(io, fmt.Sprintf("Is `%s` the correct username?", username), ui.DefaultNone)
		if err != nil {
			return "", err
		}

		if confirmed {
			return username, nil
		}
	}
}
