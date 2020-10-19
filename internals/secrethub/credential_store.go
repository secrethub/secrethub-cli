package secrethub

import (
	"time"

	"github.com/secrethub/secrethub-go/pkg/secrethub/configdir"
	"github.com/secrethub/secrethub-go/pkg/secrethub/credentials"

	"github.com/secrethub/secrethub-cli/internals/cli/ui"
)

// Errors
var (
	ErrCredentialNotExist = errMain.Code("credential_not_exist").Error("could not find credential file. Go to https://signup.secrethub.io/ to create an account or run `secrethub init` to use an already existing account on this machine.")
)

// CredentialConfig handles the configuration necessary for local credentials.
type CredentialConfig interface {
	IsPassphraseSet() bool
	Provider() credentials.Provider
	Import() (credentials.Key, error)
	ConfigDir() configdir.Dir
	PassphraseReader() credentials.PassphraseReader

	Register(FlagRegisterer)
}

// NewCredentialConfig creates a new CredentialConfig.
func NewCredentialConfig(io ui.IO) CredentialConfig {
	return &credentialConfig{
		io: io,
	}
}

type credentialConfig struct {
	configDir                    ConfigDir
	AccountCredential            string
	credentialPassphrase         string
	CredentialPassphraseCacheTTL time.Duration
	io                           ui.IO
}

func (store *credentialConfig) ConfigDir() configdir.Dir {
	return store.configDir.Dir
}

func (store *credentialConfig) IsPassphraseSet() bool {
	return store.credentialPassphrase != ""
}

// Register registers the flags for configuring the store on the provided Registerer.
func (store *credentialConfig) Register(r FlagRegisterer) {
	r.Flag("config-dir", "The absolute path to a custom configuration directory. Defaults to $HOME/.secrethub").Default("").PlaceHolder("CONFIG-DIR").SetValue(&store.configDir)
	r.Flag("credential", "Use a specific account credential to authenticate to the API. This overrides the credential stored in the configuration directory.").StringVar(&store.AccountCredential)
	r.Flag("p", "").Short('p').Hidden().NoEnvar().StringVar(&store.credentialPassphrase) // Shorthand -p is deprecated. Use --credential-passphrase instead.
	r.Flag("credential-passphrase", "The passphrase to unlock your credential file. When set, it will not prompt for the passphrase, nor cache it in the OS keyring. Please only use this if you know what you're doing and ensure your passphrase doesn't end up in bash history.").StringVar(&store.credentialPassphrase)
	r.Flag("credential-passphrase-cache-ttl", "Cache the credential passphrase in the OS keyring for this duration. The cache is automatically cleared after the timer runs out. Each time the passphrase is read from the cache the timer is reset. Passphrase caching is turned on by default for 5 minutes. Turn it off by setting the duration to 0.").Default("5m").DurationVar(&store.CredentialPassphraseCacheTTL)
}

// Provider retrieves a credential from the store.
// When a credential is set, that credential is returned,
// otherwise the credential is read from the configured file.
func (store *credentialConfig) Provider() credentials.Provider {
	return credentials.UseKey(store.getKeyReader(), credentials.KeyDecoderWithPassphrase(store.PassphraseReader()))
}

func (store *credentialConfig) Import() (credentials.Key, error) {
	return store.getKeyReader().Read(credentials.KeyDecoderWithPassphrase(store.PassphraseReader()))
}

func (store *credentialConfig) getKeyReader() credentials.KeyReader {
	if store.AccountCredential != "" {
		return credentials.FromString(store.AccountCredential)
	}
	return store.configDir.Credential()
}

// PassphraseReader returns a PassphraseReader configured by the flags.
func (store *credentialConfig) PassphraseReader() credentials.PassphraseReader {
	return NewPassphraseReader(store.io, store.credentialPassphrase, store.CredentialPassphraseCacheTTL)
}
