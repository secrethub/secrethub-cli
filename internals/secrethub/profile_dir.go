package secrethub

import (
	"os"
	"path/filepath"

	homedir "github.com/mitchellh/go-homedir"
)

const (
	// defaultProfileDirName is the default name for the secrethub profile directory.
	defaultProfileDirName = ".secrethub"
	// defaultCredentialFilename is the name of the credential file.
	defaultCredentialFilename = "credential"
	// defaultCredentialFileMode is the filemode to assign to the credential file.
	defaultCredentialFileMode = os.FileMode(0600)
	// defaultProfileDirFileMode is the filemode to assign to the configuration directory.
	defaultProfileDirFileMode = os.FileMode(0700)

	// oldConfigFilename defines the filename for the file containing old configuration options.
	oldConfigFilename = "config"
	// oldConfigFileMode is the filemode to assign to the old configuration file.
	oldConfigFileMode os.FileMode = 0600
)

// ProfileDir points to the account's directory used for storing credentials and configuration.
type ProfileDir string

// NewProfileDir constructs the account's profile directory location, defaulting to ~/.secrethub
// when no path is given. Given paths must be absolute. Note that while the returned path is absolute,
// it is not guaranteed that the returned path actually exists.
func NewProfileDir(path string) (ProfileDir, error) {
	if path == "" {
		home, err := homedir.Dir()
		if err != nil {
			return "", ErrCannotFindHomeDir(err)
		}
		path = filepath.Join(home, defaultProfileDirName)
	}

	if !filepath.IsAbs(path) {
		return "", ErrInvalidConfigDirFlag
	}

	return ProfileDir(path), nil
}

// CredentialPath returns the path to the credential file.
func (d ProfileDir) CredentialPath() string {
	return filepath.Join(string(d), defaultCredentialFilename)
}

// IsOldConfiguration detects whether an old configuration exists in the profile directory.
func (d ProfileDir) IsOldConfiguration() bool {
	_, err := os.Stat(d.CredentialPath())
	if !os.IsNotExist(err) {
		return false
	}

	_, err = os.Stat(d.oldConfigFile())

	return err == nil
}

// oldConfigFile returns the path to the old config file. Note
// that it is not guaranteed that the old config file exists.
// Use ProfileDir.IsOldConfiguration for that.
func (d ProfileDir) oldConfigFile() string {
	return filepath.Join(d.String(), oldConfigFilename)
}

// FileMode returns the file mode used for the profile directory.
func (d ProfileDir) FileMode() os.FileMode {
	return defaultProfileDirFileMode
}

// CredentialFileMode returns the file mode used for credential files.
func (d ProfileDir) CredentialFileMode() os.FileMode {
	return defaultCredentialFileMode
}

func (d ProfileDir) String() string { return string(d) }
