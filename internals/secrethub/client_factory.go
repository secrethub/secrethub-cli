package secrethub

import (
	"github.com/spf13/cobra"
	"net/http"
	"net/url"
	"strings"

	"github.com/secrethub/secrethub-go/pkg/secrethub"
	"github.com/secrethub/secrethub-go/pkg/secrethub/configdir"
	"github.com/secrethub/secrethub-go/pkg/secrethub/credentials"
)

// Errors
var (
	ErrUnknownIdentityProvider = errMain.Code("unknown_identity_provider").ErrorPref("%s is not a supported identity provider. Valid options are `aws`, `gcp` and `key`.")
)

// ClientFactory handles creating a new client with the configured options.
type ClientFactory interface {
	// NewClient returns a new SecretHub client.
	NewClient() (secrethub.ClientInterface, error)
	NewClientWithCredentials(credentials.Provider) (secrethub.ClientInterface, error)
	NewUnauthenticatedClient() (secrethub.ClientInterface, error)
	Register(command *cobra.Command)
}

// NewClientFactory creates a new ClientFactory.
func NewClientFactory(store CredentialConfig) ClientFactory {
	return &clientFactory{
		store: store,
	}
}

type clientFactory struct {
	client           *secrethub.Client
	ServerURL        urlValue
	identityProvider string
	proxyAddress     urlValue
	store            CredentialConfig
}

// Register the flags for configuration on a cli application.
func (f *clientFactory) Register(r *cobra.Command) {

	r.PersistentFlags().Var(&f.ServerURL, "api-remote", "The SecretHub API address, don't set this unless you know what you're doing.")
	r.PersistentFlags().StringVar(&f.identityProvider,"identity-provider", "key","Enable native authentication with a trusted identity provider. Options are `aws` (IAM + KMS), `gcp` (IAM + KMS) and `key`. When you run the CLI on one of the platforms, you can leverage their respective identity providers to do native keyless authentication. Defaults to key, which uses the default credential sourced from a file, command-line flag, or environment variable. ")
	r.PersistentFlags().Var(&f.proxyAddress,"proxy-address", "Set to the address of a proxy to connect to the API through a proxy. The prepended scheme determines the proxy type (http, https and socks5 are supported). For example: `--proxy-address http://my-proxy:1234`")
}

// NewClient returns a new client that is configured to use the remote that
// is set with the flag.
func (f *clientFactory) NewClient() (secrethub.ClientInterface, error) {
	if f.client == nil {
		var credentialProvider credentials.Provider
		switch strings.ToLower(f.identityProvider) {
		case "aws":
			credentialProvider = credentials.UseAWS()
		case "gcp":
			credentialProvider = credentials.UseGCPServiceAccount()
		case "key":
			credentialProvider = f.store.Provider()
		default:
			return nil, ErrUnknownIdentityProvider(f.identityProvider)
		}

		options := f.baseClientOptions()
		options = append(options, secrethub.WithCredentials(credentialProvider))

		client, err := secrethub.NewClient(options...)
		if err == configdir.ErrCredentialNotFound {
			return nil, ErrCredentialNotExist
		} else if err != nil {
			return nil, err
		}
		f.client = client
	}
	return f.client, nil
}

func (f *clientFactory) NewClientWithCredentials(provider credentials.Provider) (secrethub.ClientInterface, error) {
	options := f.baseClientOptions()
	options = append(options, secrethub.WithCredentials(provider))

	client, err := secrethub.NewClient(options...)
	if err != nil {
		return nil, err
	}

	return client, nil
}

func (f *clientFactory) NewUnauthenticatedClient() (secrethub.ClientInterface, error) {
	options := f.baseClientOptions()

	client, err := secrethub.NewClient(options...)
	if err != nil {
		return nil, err
	}

	return client, nil
}

func (f *clientFactory) baseClientOptions() []secrethub.ClientOption {
	options := []secrethub.ClientOption{
		secrethub.WithConfigDir(f.store.ConfigDir()),
		secrethub.WithAppInfo(&secrethub.AppInfo{
			Name:    "secrethub-cli",
			Version: Version,
		}),
	}

	if f.proxyAddress.u != nil {
		transport := http.DefaultTransport.(*http.Transport)
		transport.Proxy = func(request *http.Request) (*url.URL, error) {
			return f.proxyAddress.u, nil
		}
		options = append(options, secrethub.WithTransport(transport))
	}

	if &f.ServerURL != nil {
		options = append(options, secrethub.WithServerURL(f.ServerURL.String()))
	}

	return options
}

type urlValue struct {
	u *url.URL
}

func (u *urlValue) String() string {
	return u.u.String()
}

func (u *urlValue) Set(s string) error {
	parsed, err := url.Parse(s)
	u.u = parsed
	if err != nil {
		return err
	}
	return nil
}

func (u *urlValue) Type() string {
	return "urlValue"
}

