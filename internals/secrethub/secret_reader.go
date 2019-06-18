package secrethub

import "github.com/secrethub/secrethub-go/pkg/secrethub"

type secretReader struct {
	client secrethub.Client
}

func newSecretReader(client secrethub.Client) secretReader {
	return secretReader{client: client}
}

func (sr secretReader) ReadSecret(path string) (string, error) {
	secret, err := sr.client.Secrets().Versions().GetWithData(path)
	if err != nil {
		return "", err
	}
	return string(secret.Data), nil
}
