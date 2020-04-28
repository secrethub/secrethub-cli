package credential

import (
	"context"

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
	return c.agentClient.Fingerprint(context.Background())
}

func (c credential) Sign(bytes []byte) ([]byte, error) {
	return c.agentClient.Sign(context.Background(), bytes)
}

func (c credential) SignMethod() string {
	return credentials.RSACredential{}.SignMethod()
}

func (c credential) Unwrap(ciphertext *api.EncryptedData) ([]byte, error) {
	return c.agentClient.Decrypt(context.Background(), *ciphertext)
}

type Provider credential

func (p Provider) Provide(_ *http.Client) (auth.Authenticator, credentials.Decrypter, error) {
	return auth.NewHTTPSigner(credential(p)), credential(p), nil
}

func New(configDir string) Provider {
	return Provider(credential{
		agentClient: client.New(configDir),
	})
}
