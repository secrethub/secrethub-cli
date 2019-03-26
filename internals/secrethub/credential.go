package secrethub

import (
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"io/ioutil"
	"os"

	"github.com/keylockerbv/secrethub-cli/internals/cli/ui"
	"github.com/secrethub/secrethub-go/internals/crypto"
	"github.com/secrethub/secrethub-go/internals/errio"
	"github.com/secrethub/secrethub-go/pkg/secrethub"
)

// Errors
var (
	ErrCannotDecryptCredential       = errMain.Code("cannot_decrypt_credential").Error("passphrase is incorrect or the credential has been corrupted")
	ErrIncorrectPassphraseNotCleared = errMain.Code("incorrect_passphrase_not_cleared").ErrorPref("%s. The passphrase could not be cleared from the passphrase cache: %s")
	ErrWrongKeyFormat                = errMain.Code("wrong_key_format").ErrorPref("%s. Please use a different key")
	ErrCredentialNotExist            = errMain.Code("credential_not_exist").Error("could not find credential file. Run `secrethub signup` to create an account.")
)

// CredentialReader reads a credential.
type CredentialReader interface {
	Read() (secrethub.Credential, error)
}

// CredentialReaderFunc is a helper function that implements the CredentialReader interface.
type CredentialReaderFunc func() (secrethub.Credential, error)

// Read reads a credential.
func (fn CredentialReaderFunc) Read() (secrethub.Credential, error) {
	return fn()
}

// NewCredentialReader creates a new credential reader.
func NewCredentialReader(io ui.IO, profileDir ProfileDir, flagValue string, passReader PassphraseReader) CredentialReader {
	fn := func() (secrethub.Credential, error) {
		if flagValue != "" {
			return parseCredential(flagValue, passReader)
		}

		if profileDir.IsOldConfiguration() {
			logger.Debug("falling back to old configuration")
			return readOldCredential(io, profileDir, passReader)
		}

		_, err := os.Stat(profileDir.CredentialPath())
		if os.IsNotExist(err) {
			return nil, ErrCredentialNotExist
		}

		bytes, err := ioutil.ReadFile(profileDir.CredentialPath())
		if err != nil {
			return nil, errio.Error(err)
		}

		return parseCredential(string(bytes), passReader)
	}

	return CredentialReaderFunc(fn)
}

// parseCredential parses and decodes a credential, optionally unarmoring it.
func parseCredential(raw string, reader PassphraseReader) (secrethub.Credential, error) {
	parser := secrethub.NewCredentialParser(secrethub.DefaultCredentialDecoders)

	encoded, err := parser.Parse(raw)
	if err != nil {
		return nil, errio.Error(err)
	}

	if encoded.IsEncrypted() {
		id := getEncryptedCredentialID([]byte(raw))

		passphrase, err := reader.Get(id)
		if err != nil {
			return nil, errio.Error(err)
		}

		passBasedKey, err := secrethub.NewPassBasedKey(passphrase)
		if err != nil {
			return nil, err
		}

		credential, err := encoded.DecodeEncrypted(passBasedKey)
		if crypto.IsWrongKey(err) {
			incorrectErr := reader.IncorrectPassphrase(id)
			if incorrectErr != nil {
				return nil, ErrIncorrectPassphraseNotCleared(err, incorrectErr)
			}

			return nil, ErrCannotDecryptCredential
		} else if err != nil {
			return nil, errio.Error(err)
		}

		return credential, nil
	}

	return encoded.Decode()
}

// getEncryptedCredentialID returns an encoded hash of an encrypted credential.
// This can be useful for e.g. identifiying the credential to read a passphrase
// for. Note that this should NOT be used for plaintext credentials!
func getEncryptedCredentialID(encrypted []byte) string {
	return base64.StdEncoding.WithPadding(base64.NoPadding).EncodeToString(sha256.New().Sum(encrypted))
}

// readOldCredential reads an old configuration into a Credential.
func readOldCredential(io ui.IO, profileDir ProfileDir, passReader PassphraseReader) (secrethub.Credential, error) {
	config, err := LoadConfig(io, profileDir.oldConfigFile())
	if err != nil {
		return nil, errio.Error(err)
	}

	return config.toCredential(passReader)
}

// Credential ports the old configuration to a Credential.
func (c *Config) toCredential(passReader PassphraseReader) (secrethub.Credential, error) {
	if c.Type == ConfigUserType {
		key, err := ioutil.ReadFile(c.User.KeyFile)
		if err != nil {
			return nil, ErrCannotReadFile(c.User.KeyFile)
		}

		return loadPEMKey(key, passReader)
	} else if c.Type == ConfigServiceType {
		return loadPEMKey([]byte(c.Service.PrivateKey), passReader)
	}

	return nil, errors.New("cannot fallback to old config type")
}

// loadPEMKey loads a PEM encoded key, optionally decrypting it when necessary.
func loadPEMKey(key []byte, passReader PassphraseReader) (secrethub.Credential, error) {
	pemKey, err := crypto.ReadPEM(key)
	if err != nil {
		return nil, errio.Error(err)
	}

	var clientKey crypto.RSAPrivateKey
	if pemKey.IsEncrypted() {
		id := getEncryptedCredentialID(key)

		passphrase, err := passReader.Get(id)
		if err != nil {
			return nil, errio.Error(err)
		}

		clientKey, err = pemKey.Decrypt(passphrase)
		if err == crypto.ErrIncorrectPassphrase {
			incorrectErr := passReader.IncorrectPassphrase(id)
			if incorrectErr != nil {
				return nil, ErrIncorrectPassphraseNotCleared(err, incorrectErr)
			}
			return nil, errio.Error(err)
		}
		if err == crypto.ErrNotPKCS1Format {
			return nil, ErrWrongKeyFormat(err)
		} else if err != nil {
			return nil, errio.Error(err)
		}
	} else {
		clientKey, err = pemKey.Decode()
		if err == crypto.ErrNotPKCS1Format {
			return nil, ErrWrongKeyFormat(err)
		} else if err != nil {
			return nil, errio.Error(err)
		}
	}

	return secrethub.RSACredential{RSAPrivateKey: clientKey}, nil
}
