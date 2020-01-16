package secrethub

import (
	"errors"
	"fmt"

	"github.com/secrethub/secrethub-cli/internals/secrethub/tpl"
	"github.com/secrethub/secrethub-go/internals/api"
)

type ErrSecretNotFound struct {
	path string
}

func (err ErrSecretNotFound) Error() string {
	return fmt.Sprintf("secret %s not found", err.path)
}

type secretReader struct {
	newClient newClientFunc
}

// newSecretReader wraps a client to implement tpl.SecretReader.
func newSecretReader(newClient newClientFunc) *secretReader {
	return &secretReader{
		newClient: newClient,
	}
}

// ReadSecret reads the secret using the provided client.
func (sr secretReader) ReadSecret(path string) (string, error) {
	client, err := sr.newClient()
	if err != nil {
		return "", err
	}

	secret, err := client.Secrets().Versions().GetWithData(path)
	if err != nil {
		return "", ErrSecretNotFound{path: path}
	}

	return string(secret.Data), nil
}

type bufferedSecretReader struct {
	secretReader tpl.SecretReader
	secretsRead  []string
}

// newBufferedSecretReader wraps a secret reader and stores the retrieved
// secret values for retrieval with the Values function.
func newBufferedSecretReader(sr tpl.SecretReader) *bufferedSecretReader {
	return &bufferedSecretReader{
		secretReader: sr,
		secretsRead:  []string{},
	}
}

// ReadSecret uses the underlying secret reader to read the secret
// and stores the result for retrieval with the Values function.
func (sr *bufferedSecretReader) ReadSecret(path string) (string, error) {
	secret, err := sr.secretReader.ReadSecret(path)

	if err == nil {
		sr.secretsRead = append(sr.secretsRead, secret)
	}

	return secret, err
}

// Values returns a list of values read with this secret reader.
func (sr bufferedSecretReader) Values() []string {
	return sr.secretsRead
}

type ignoreMissingSecretReader struct {
	secretReader tpl.SecretReader
}

func newIgnoreMissingSecretReader(sr tpl.SecretReader) *ignoreMissingSecretReader {
	return &ignoreMissingSecretReader{
		secretReader: sr,
	}
}

// ReadSecret uses the underlying secret reader to read the secret, but ignores
// errors for non-existing secrets. Instead, it returns the empty string.
func (sr *ignoreMissingSecretReader) ReadSecret(path string) (string, error) {
	secret, err := sr.secretReader.ReadSecret(path)
	if isErrNotFound(err) {
		return "", nil
	}
	return secret, err
}

// isErrNotFound returns whether the given error is caused by a un-existing resource.
func isErrNotFound(err error) bool {
	if api.IsErrNotFound(err) {
		return true
	}
	if errors.As(err, &ErrSecretNotFound{}) {
		return true
	}
	return false
}
