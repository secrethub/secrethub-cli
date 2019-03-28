package secrethub

// This file solely exists for backwards compatibility.

import (
	"net/url"
	"path/filepath"
	"strings"

	"fmt"

	"github.com/asaskevich/govalidator"
	"github.com/secrethub/secrethub-cli/internals/cli/ui"
	"github.com/secrethub/secrethub-cli/internals/cli/configuration"
	homedir "github.com/mitchellh/go-homedir"
	"github.com/secrethub/secrethub-go/internals/api/uuid"
	"github.com/secrethub/secrethub-go/internals/errio"
)

const (
	// ConfigServiceType denotes an serviceConfig
	ConfigServiceType = "service"
	// ConfigUserType denotes an userConfig
	ConfigUserType = "user"
	// ConfigVersion defines the currently wanted version of config
	ConfigVersion = 3
)

// Errors
var (
	errConfig = errio.Namespace("config")

	ErrCannotSetConfigField = errConfig.Code("cannot_set_field").ErrorPref("cannot set config field `%s`")
	ErrInvalidConfigValue   = errConfig.Code("invalid_value").ErrorPref("invalid config value: %v")
	ErrKeyFileNotFound      = errConfig.Code("key_file_not_found").Error("cannot find the key file at configured path")

	ErrInvalidRemote = errConfig.Code("invalid_remote").Error("remote url must be of the form http[s]://hostname[:port]")
	// ErrConfigFileNotFound means the config file could not be found.
	ErrConfigNotFound = errConfig.Code("not_found").ErrorPref(
		"%s\n\n" +
			"Usually, this means you haven't configured a secrethub account yet. " +
			"You can sign up for a SecretHub account with:\n\n" +
			"\t" +
			fmt.Sprintf("%s signup", ApplicationName),
	)

	// ErrConfigReadFailed means reading the config failed.
	ErrConfigReadFailed = errConfig.Code("read_failed").ErrorPref(
		"failed to read the config file: %s\n\n" +
			"Usually, this means the config file got corrupted. " +
			"You can manually check the file at %s" +
			"or remove it (CAUTION: remember to save your credentials)")

	// ErrConfigUserNotSet means the username field is not set in the config.
	ErrConfigUserNotSet = errConfig.Code("username_not_set").Errorf(
		"username field not set in the config.\n\n"+
			"Please add this by using:\n\n"+
			"\t%s config set user.username <username>",
		ApplicationName,
	)

	// ErrConfigKeyNotSet means the key_file field is not set in the config.
	ErrConfigKeyNotSet = errConfig.Code("key_file_not_set").Errorf(
		"key_file field not set in the config.\n\n"+
			"Please add this by using:\n\n"+
			"\t%s config set user.key_file <path>",
		ApplicationName,
	)

	// ErrConfigAccountNotSet means the account_id field is not set in the config.
	ErrConfigAccountNotSet = errConfig.Code("account_not_set").Error(
		"app account_id not set in the config.\n\n" +
			"Please create a new service config.")

	// ErrConfigServiceKeyNotSet means the private_key field is not set in the config.
	ErrConfigServiceKeyNotSet = errConfig.Code("app_key_not_set").Error(
		"service private_key field not set in the config.\n\n" +
			"Please create a new service config.")

	// ErrConfigTypeUnknown means parsing failed because config type was not set.
	ErrConfigTypeUnknown = errConfig.Code("unknown_type").ErrorPref("could not parse your config: unknown config type\n\n" +
		"Usually, this means your config file got corrupted." +
		"You can manually check the file at %s")

	// ErrConfigWriteFailed means writing to the config file failed.
	ErrConfigWriteFailed = errConfig.Code("write_failed").ErrorPref("could not write your updated config: %s\n\n" +
		"Usually this means you don't have write access to the config file." +
		"Check your access to the file at %s")
)

// Config contains all the configuration for Secrets.
type Config struct {
	Version int            `json:"version" yaml:"version"`
	Remote  string         `json:"remote,omitempty" yaml:"remote,omitempty"`
	Type    string         `json:"type" yaml:"type"`
	User    *userConfig    `json:"user,omitempty" yaml:"user,omitempty"`
	Service *serviceConfig `json:"app,omitempty" yaml:"app,omitempty"`
}

// userConfig contains the required configuration fields for this application.
type userConfig struct {
	Username string `json:"username,omitempty" yaml:"username,omitempty"`
	KeyFile  string `json:"key_file" yaml:"key_file"`
}

// newConfig creates a new base Config.
func newConfig(remote string) Config {
	config := Config{
		Version: ConfigVersion,
		Remote:  remote,
	}

	return config
}

// newUserConfig returns a new prepared Config with UserType.
func newUserConfig(username, keyPath string, remoteURL *url.URL) (*Config, error) {
	if remoteURL == nil {
		return nil, ErrInvalidRemote
	}

	remote := remoteURL.String()

	config := newConfig(remote)

	// Replace ~/ with the home directory
	keyPathExpanded, err := homedir.Expand(keyPath)
	if err != nil {
		return &config, errio.Error(err)
	}

	absKeyPath, err := filepath.Abs(keyPathExpanded)
	if err != nil {
		return &config, errio.Error(err)
	}

	config.Type = ConfigUserType
	config.User = &userConfig{
		Username: username,
		KeyFile:  absKeyPath,
	}

	return &config, nil
}

// parseUserConfig parses a ConfigMap and migrates the config if necessary.
func parseUserConfig(io ui.IO, mapConfig configuration.ConfigMap, path string) (Config, error) {
	var config = Config{}
	var err error

	currentVersion, err := mapConfig.GetVersion()
	if err != nil {
		return config, errio.Error(err)
	}

	updated := false
	if currentVersion != ConfigVersion {
		logger.Debugf("Updating user configuration from version %d to %d",
			currentVersion,
			ConfigVersion)
		mapConfig, err = configuration.MigrateConfigTo(io, mapConfig, currentVersion, ConfigVersion, ConfigMigrations, false)
		if err != nil {
			return config, err
		}

		updated = true
	}

	err = configuration.ParseMap(&mapConfig, &config)
	if err != nil {
		return config, ErrConfigReadFailed(err, path)
	}

	err = config.User.validate()
	if err != nil {
		return config, err
	}

	if updated {
		logger.Debugf("Writing the updated config to %s", path)
		err = configuration.WriteToFile(config, path, oldConfigFileMode)
		if err != nil {
			return config, ErrConfigWriteFailed(err, path)
		}
	}

	return config, nil
}

// validateUserConfig validates specific rules for userConfig
func (c userConfig) validate() error {
	if c.Username == "" {
		return ErrConfigUserNotSet
	}

	if c.KeyFile == "" {
		return ErrConfigKeyNotSet

	}

	return nil
}

// serviceConfig contains the fields required for an app.
type serviceConfig struct {
	RepoID     *uuid.UUID `json:"repo_id" yaml:"repo_id"`
	AccountID  *uuid.UUID `json:"account_id" yaml:"account_id"`
	PrivateKey string     `json:"private_key" yaml:"private_key"`
}

func newServiceConfig(repoID, accountID *uuid.UUID, privateKey, remote string) Config {
	config := newConfig(remote)
	config.Type = ConfigServiceType
	config.Service = &serviceConfig{
		RepoID:     repoID,
		AccountID:  accountID,
		PrivateKey: privateKey,
	}

	return config
}

func parseServiceConfig(configData []byte) (Config, error) {
	config := Config{Service: &serviceConfig{}}

	err := configuration.Read(configData, &config)
	if err != nil {
		return config, err
	}

	err = config.Service.validate()
	return config, err
}

// validate validates specific rules for serviceConfig.
func (c serviceConfig) validate() error {
	if c.AccountID == nil {
		return ErrConfigAccountNotSet
	}

	if c.PrivateKey == "" {
		return ErrConfigServiceKeyNotSet
	}

	return nil
}

// LoadConfig loads the config from the a given path, validating and setting some defaults.
func LoadConfig(io ui.IO, path string) (*Config, error) {
	mapConfig, configData, err := configuration.ReadConfigurationDataFromFile(path)

	if err == configuration.ErrFileNotFound {
		logger.Debugf("cannot find config at path: %s", path)
		return nil, ErrConfigNotFound(err)
	} else if err != nil {
		return nil, err
	}

	var config Config

	t, err := mapConfig.GetType()
	if err != nil {
		return &config, errio.Error(err)
	}

	if t == ConfigUserType {
		logger.Debug("parsing user config")
		config, err = parseUserConfig(io, mapConfig, path)
	} else if t == ConfigServiceType {
		logger.Debug("parsing service config")
		config, err = parseServiceConfig(configData)
	} else {
		return &config, ErrConfigTypeUnknown(path)
	}
	if err != nil {
		return &config, err
	}

	return &config, nil
}

// ValidateRemote validates a remote url and enforces it is
// of the form http[s]://hostname[:port]
func ValidateRemote(url string) error {
	if !govalidator.IsURL(url) || (!strings.HasPrefix(url, "http://") && !strings.HasPrefix(url, "https://")) {
		return ErrInvalidRemote
	}
	return nil
}
