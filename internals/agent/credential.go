package agent

import (
	"context"
	"fmt"

	"github.com/secrethub/secrethub-go/internals/api"
	"github.com/secrethub/secrethub-go/internals/auth"
	"github.com/secrethub/secrethub-go/pkg/secrethub/credentials"
	"github.com/secrethub/secrethub-go/pkg/secrethub/internals/http"

	"github.com/secrethub/secrethub-cli/internals/agent/client"
)

type credential struct {
	agentClient client.Client
}

func (c credential) ID() (string, error) {
	id, err := c.agentClient.Fingerprint(context.Background())
	if err != nil {
		return "", fmt.Errorf("agent: %v", err)
	}
	return id, nil
}

func (c credential) Sign(bytes []byte) ([]byte, error) {
	signature, err := c.agentClient.Sign(context.Background(), bytes)
	if err != nil {
		return nil, fmt.Errorf("agent: %v", err)
	}
	return signature, nil
}

func (c credential) SignMethod() string {
	return credentials.RSACredential{}.SignMethod()
}

func (c credential) Unwrap(ciphertext *api.EncryptedData) ([]byte, error) {
	decrypted, err := c.agentClient.Decrypt(context.Background(), *ciphertext)
	if err != nil {
		return nil, fmt.Errorf("agent: %v", err)
	}
	return decrypted, nil
}

type Provider credential

func (p Provider) Provide(_ *http.Client) (auth.Authenticator, credentials.Decrypter, error) {
	return auth.NewHTTPSigner(credential(p)), credential(p), nil
}

func CredentialProvider(configDir, version string, prompter client.Prompter) Provider {
	return Provider(credential{
		agentClient: client.New(configDir, version, prompter),
	})
}
