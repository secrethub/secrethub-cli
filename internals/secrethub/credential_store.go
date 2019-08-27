package secrethub

import (
	"time"

	"github.com/secrethub/secrethub-go/pkg/secrethub/configdir"
	"github.com/secrethub/secrethub-go/pkg/secrethub/credentials"

	"github.com/secrethub/secrethub-cli/internals/cli/ui"
)

// CredentialStore handles storing a shclient.Credential.
type CredentialStore interface {
	IsPassphraseSet() bool
	Provider() credentials.Provider
	Import() (credentials.Key, error)
	ConfigDir() configdir.Dir
	PassphraseReader() credentials.Reader

	Register(FlagRegisterer)
}

// NewCredentialStore creates a new CredentialStore.
func NewCredentialStore(io ui.IO) CredentialStore {
	return &credentialStore{
		io: io,
	}
}

type credentialStore struct {
	configDir                    ConfigDir
	AccountCredential            string
	credentialPassphrase         string
	CredentialPassphraseCacheTTL time.Duration
	io                           ui.IO
}

func (store *credentialStore) ConfigDir() configdir.Dir {
	return store.configDir.Dir
}

func (store *credentialStore) IsPassphraseSet() bool {
	return store.credentialPassphrase != ""
}

// Register registers the flags for configuring the store on the provided Registerer.
func (store *credentialStore) Register(r FlagRegisterer) {
	r.Flag("config-dir", "The absolute path to a custom configuration directory. Defaults to $HOME/.secrethub").Default("").SetValue(&store.configDir)
	r.Flag("credential", "Use a specific account credential to authenticate to the API. This overrides the credential stored in the configuration directory.").StringVar(&store.AccountCredential)
	r.Flag("credential-passphrase", "The passphrase to unlock your credential file. When set, it will not prompt for the passphrase, nor cache it in the OS keyring. Please only use this if you know what you're doing and ensure your passphrase doesn't end up in bash history.").Short('p').StringVar(&store.credentialPassphrase)
	r.Flag("credential-passphrase-cache-ttl", "Cache the credential passphrase in the OS keyring for this duration. The cache is automatically cleared after the timer runs out. Each time the passphrase is read from the cache the timer is reset. Passphrase caching is turned on by default for 5 minutes. Turn it off by setting the duration to 0.").Default("5m").DurationVar(&store.CredentialPassphraseCacheTTL)
}

// Provider retrieves a credential from the store.
// When a credential is set, that credential is returned,
// otherwise the credential is read from the configured file.
func (store *credentialStore) Provider() credentials.Provider {
	return credentials.UseKey(store.getCredentialReader()).Passphrase(store.PassphraseReader())
}

func (store *credentialStore) Import() (credentials.Key, error) {
	return credentials.ImportKey(store.getCredentialReader(), store.PassphraseReader())
}

func (store *credentialStore) getCredentialReader() credentials.Reader {
	if store.AccountCredential != "" {
		return credentials.FromString(store.AccountCredential)
	}
	return store.configDir.Credential()
}

// PassphraseReader returns a PassphraseReader configured by the flags.
func (store *credentialStore) PassphraseReader() credentials.Reader {
	return NewPassphraseReader(store.io, store.credentialPassphrase, store.CredentialPassphraseCacheTTL)
}
