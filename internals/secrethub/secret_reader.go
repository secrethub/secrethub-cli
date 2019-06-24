package secrethub

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
		return "", err
	}

	return string(secret.Data), nil
}
