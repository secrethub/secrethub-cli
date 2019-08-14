package fakes

import "errors"

// FakeSecretReader implements tpl.SecretReader.
type FakeSecretReader struct {
	Secrets map[string]string
}

// ReadSecret implements tpl.SecretReader.ReadSecret.
func (fsr FakeSecretReader) ReadSecret(path string) (string, error) {
	secret, ok := fsr.Secrets[path]
	if ok {
		return secret, nil
	}
	return "", errors.New("secret not found")
}
