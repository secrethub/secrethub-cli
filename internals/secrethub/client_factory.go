package secrethub

import (
	"net/url"

	"github.com/secrethub/secrethub-go/pkg/secrethub"
)

// ClientFactory handles creating a new client with the configured options.
type ClientFactory interface {
	// NewClient returns a new SecretHub client.
	NewClient() (secrethub.Client, error)
	Register(FlagRegisterer)
}

// NewClientFactory creates a new ClientFactory.
func NewClientFactory(store CredentialStore) ClientFactory {
	return &clientFactory{
		store: store,
	}
}

type clientFactory struct {
	client    secrethub.Client
	ServerURL *url.URL
	store     CredentialStore
}

// Register the flags for configuration on a cli application.
func (f *clientFactory) Register(r FlagRegisterer) {
	r.Flag("api-remote", "The SecretHub API address, don't set this unless you know what you're doing.").Hidden().URLVar(&f.ServerURL)
}

// NewClient returns a new client that is configured to use the remote that
// is set with the flag.
func (f *clientFactory) NewClient() (secrethub.Client, error) {
	if f.client == nil {
		credential, err := f.store.Get()
		if err != nil {
			return nil, err
		}

		f.client = secrethub.NewClient(credential, f.NewClientOptions())
	}
	return f.client, nil
}

// NewClientOptions returns the client options configured by the flags.
func (f *clientFactory) NewClientOptions() *secrethub.ClientOptions {
	var opts secrethub.ClientOptions

	if f.ServerURL != nil {
		opts.ServerURL = f.ServerURL.String()
	}
	return &opts
}
