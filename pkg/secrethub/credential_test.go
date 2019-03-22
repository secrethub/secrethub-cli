package secrethub

import (
	"github.com/secrethub/secrethub-go/internals/assert"
	"io/ioutil"
	"net/url"
	"path/filepath"
	"testing"

	"github.com/secrethub/secrethub-go/pkg/secrethub"

	"github.com/keylockerbv/secrethub-cli/pkg/configuration"
	"github.com/keylockerbv/secrethub-cli/pkg/ui"
	"github.com/secrethub/secrethub-go/internals/api/uuid"
	"github.com/secrethub/secrethub-go/internals/crypto"
)

// TestOldConfigToCredential tests whether older config structs can be successfully migrated to a Credential.
func TestOldConfigToCredential(t *testing.T) {
	// Arrange
	dir, cleanup := testdata.TempDir(t)
	defer cleanup()

	url, err := url.Parse("https://some.remote.com")
	assert.OK(t, err)

	key, err := crypto.GenerateRSAPrivateKey(1024)
	assert.OK(t, err)

	// Plaintext User
	exportedPlaintext, err := key.ExportPEM()
	assert.OK(t, err)
	plaintextPath := filepath.Join(dir, "user_plain_key_file")
	err = ioutil.WriteFile(plaintextPath, exportedPlaintext, 0770)
	assert.OK(t, err)
	userConfigPlain, err := newUserConfig("user1", plaintextPath, url)
	assert.OK(t, err)

	// Encrypted User
	pass := "password123"
	exportedCiphertext, err := key.ExportPrivateKeyWithPassphrase(pass)
	assert.OK(t, err)

	ciphertextPath := filepath.Join(dir, "user_enc_key_file")
	err = ioutil.WriteFile(ciphertextPath, exportedCiphertext, 0770)
	assert.OK(t, err)
	userConfigCiphertext, err := newUserConfig("user1", ciphertextPath, url)
	assert.OK(t, err)

	// Service
	serviceConfig := newServiceConfig(
		uuid.New(),
		uuid.New(),
		string(exportedPlaintext),
		url.String(),
	)

	cases := map[string]struct {
		config             Config
		passphrase         string
		expectedCredential secrethub.Credential
	}{
		"user_plain": {
			config:     *userConfigPlain,
			passphrase: "",
			expectedCredential: secrethub.RSACredential{
				RSAPrivateKey: key,
			},
		},
		"user_enc": {
			config:     *userConfigCiphertext,
			passphrase: pass,
			expectedCredential: secrethub.RSACredential{
				RSAPrivateKey: key,
			},
		},
		"service": {
			config:     serviceConfig,
			passphrase: "",
			expectedCredential: secrethub.RSACredential{
				RSAPrivateKey: key,
			},
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			passReader := FakePassphraseReader{
				pass: []byte(tc.passphrase),
				err:  nil,
			}

			// Act
			credential, err := tc.config.toCredential(passReader)
			assert.OK(t, err)

			// Assert
			assert.Equal(t, credential, tc.expectedCredential)
		})
	}
}

func TestReadOldCredential(t *testing.T) {
	// Arrange
	dir, cleanup := testdata.TempDir(t)
	defer cleanup()

	url, err := url.Parse("https://some.remote.com")
	assert.OK(t, err)

	key, err := crypto.GenerateRSAPrivateKey(1024)
	assert.OK(t, err)

	exported, err := key.ExportPEM()
	assert.OK(t, err)

	keyPath := filepath.Join(dir, "key")
	err = ioutil.WriteFile(keyPath, exported, 0770)
	assert.OK(t, err)

	config, err := newUserConfig("user1", keyPath, url)
	assert.OK(t, err)

	err = configuration.WriteToFile(config, filepath.Join(dir, oldConfigFilename), oldConfigFileMode)
	assert.OK(t, err)

	profileDir := ProfileDir(dir)

	passReader := FakePassphraseReader{}

	expectedCredential := secrethub.RSACredential{
		RSAPrivateKey: key,
	}

	// Act
	credential, err := readOldCredential(ui.NewFakeIO(), profileDir, passReader)
	assert.OK(t, err)

	// Assert
	assert.Equal(t, credential, expectedCredential)
}

// TestCredentialReader tests the credential reading logic
// based on flag values, old configuration or new settings.
func TestCredentialReader(t *testing.T) {
	// Arrange
	key, err := crypto.GenerateRSAPrivateKey(1024)
	assert.OK(t, err)

	flagKey, err := crypto.GenerateRSAPrivateKey(1024)
	assert.OK(t, err)

	flagValue, err := secrethub.EncodeCredential(secrethub.RSACredential{RSAPrivateKey: flagKey})
	assert.OK(t, err)

	// Setup an old configuration dir
	oldDir, cleanupOld := testdata.TempDir(t)
	defer cleanupOld()

	url, err := url.Parse("https://some.remote.com")
	assert.OK(t, err)

	pemKey, err := key.ExportPEM()
	assert.OK(t, err)

	keyPath := filepath.Join(oldDir, "key")
	err = ioutil.WriteFile(keyPath, pemKey, 0770)
	assert.OK(t, err)

	userConfig, err := newUserConfig("user1", keyPath, url)
	assert.OK(t, err)

	err = configuration.WriteToFile(userConfig, filepath.Join(oldDir, oldConfigFilename), oldConfigFileMode)
	assert.OK(t, err)

	// Setup a configuration dir with credential
	credentialDir, cleanupSettings := testdata.TempDir(t)
	defer cleanupSettings()

	encoded, err := secrethub.EncodeCredential(secrethub.RSACredential{RSAPrivateKey: key})
	assert.OK(t, err)

	err = ioutil.WriteFile(filepath.Join(credentialDir, defaultCredentialFilename), []byte(encoded), defaultCredentialFileMode)
	assert.OK(t, err)

	cases := map[string]struct {
		dir       string
		flagValue string
		expected  secrethub.Credential
		err       error
	}{
		"empty_flag_with_old_config": {
			dir:       oldDir,
			flagValue: "",
			expected:  secrethub.RSACredential{RSAPrivateKey: key},
			err:       nil,
		},
		"flag_with_old_config": {
			dir:       oldDir,
			flagValue: flagValue,
			expected:  secrethub.RSACredential{RSAPrivateKey: flagKey},
			err:       nil,
		},
		"empty_flag_with_credential": {
			dir:       credentialDir,
			flagValue: "",
			expected:  secrethub.RSACredential{RSAPrivateKey: key},
			err:       nil,
		},
		"flag_with_credential": {
			dir:       credentialDir,
			flagValue: flagValue,
			expected:  secrethub.RSACredential{RSAPrivateKey: flagKey},
			err:       nil,
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			// Arrange
			reader := NewCredentialReader(
				ui.NewFakeIO(),
				ProfileDir(tc.dir),
				tc.flagValue,
				FakePassphraseReader{},
			)

			// Act
			actual, err := reader.Read()

			// Assert
			assert.Equal(t, err, tc.err)
			assert.Equal(t, actual, tc.expected)
		})
	}
}

func TestParseCredential(t *testing.T) {
	// Arrange
	key, err := crypto.GenerateRSAPrivateKey(1024)
	assert.OK(t, err)

	credential := secrethub.RSACredential{RSAPrivateKey /**/ : key}

	plain, err := secrethub.EncodeCredential(credential)
	assert.OK(t, err)

	passphrase := "wachtwoord123"
	armorer, err := secrethub.NewPassBasedKey([]byte(passphrase))
	assert.OK(t, err)

	armored, err := secrethub.EncodeEncryptedCredential(credential, armorer)
	assert.OK(t, err)

	newReader := func(pass string) PassphraseReader {
		return passphraseReader{
			FlagValue: pass,
			Cache:     NewPassphraseCache(0, &TestKeyringCleaner{}, newTestKeyring()),
		}
	}

	cases := map[string]struct {
		raw      string
		reader   PassphraseReader
		expected secrethub.Credential
		err      error
	}{
		"invalid_credential": {
			raw:      "some_invalid_token_string",
			reader:   newReader(""),
			expected: nil,
			err:      secrethub.ErrInvalidNumberOfCredentialSegments(1),
		},
		"plain": {
			raw:      plain,
			reader:   newReader(""),
			expected: credential,
			err:      nil,
		},
		"armored": {
			raw:      armored,
			reader:   newReader(passphrase),
			expected: credential,
			err:      nil,
		},
		"armored_wrong_pass": {
			raw:      armored,
			reader:   newReader("wrong passphrase"),
			expected: nil,
			err:      ErrCannotDecryptCredential,
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			// Act
			actual, err := parseCredential(tc.raw, tc.reader)

			// Assert
			assert.Equal(t, err, tc.err)
			assert.Equal(t, actual, tc.expected)
		})
	}
}
