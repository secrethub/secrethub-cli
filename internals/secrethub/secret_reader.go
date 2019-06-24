package secrethub

import "github.com/secrethub/secrethub-go/pkg/secrethub"

type secretReader struct {
	_client   secrethub.Client
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
	client, err := sr.client()
	if err != nil {
		return "", err
	}

	secret, err := client.Secrets().Versions().GetWithData(path)
	if err != nil {
		return "", err
	}

	return string(secret.Data), nil
}

func (sr secretReader) client() (secrethub.Client, error) {
	if sr._client == nil {
		var err error
		sr._client, err = sr.newClient()
		if err != nil {
			return nil, err
		}
	}
	return sr._client, nil
}
