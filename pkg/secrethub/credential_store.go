package secrethub

import (
	"io/ioutil"
	"os"
	"time"

	"github.com/keylockerbv/secrethub-cli/pkg/ui"
	"github.com/keylockerbv/secrethub-cli/pkg/posix"
	"github.com/secrethub/secrethub-go/internals/errio"
	"github.com/secrethub/secrethub-go/pkg/secrethub"
)

// CredentialStore handles storing a shclient.Credential.
type CredentialStore interface {
	// CredentialExists returns whether there is already a credential stored
	// in the configured location. When this is the case, a write will
	// override this credential.
	CredentialExists() (bool, error)
	// TODO SHDEV-1026: Remove from interface once config upgrade uses CredentialStore
	NewProfileDir() (ProfileDir, error)
	IsPassphraseSet() bool
	SetPassphrase(string)
	// Set stores a credential that is returned on Get.
	// The credential is only stored in memory. Save
	// should be called to persist the credential.
	Set(credential secrethub.Credential)
	// Save persists the credential.
	Save() error
	// Get retrieves a credential from the store.
	Get() (secrethub.Credential, error)
	Register(FlagRegisterer)
}

// NewCredentialStore creates a new CredentialStore.
func NewCredentialStore(io ui.IO) CredentialStore {
	return &credentialStore{
		io: io,
	}
}

type credentialStore struct {
	ConfigDir                    string
	AccountCredential            string
	credentialPassphrase         string
	CredentialPassphraseCacheTTL time.Duration
	credential                   secrethub.Credential
	io                           ui.IO
}

// Register registers the flags for configuring the store on the provided Registerer.
func (store *credentialStore) Register(r FlagRegisterer) {
	r.Flag("config-dir", "The absolute path to a custom configuration directory. Defaults to $HOME/.secrethub").StringVar(&store.ConfigDir)
	r.Flag("credential", "Use a specific account credential to authenticate to the API. This overrides the credential stored in the configuration directory.").StringVar(&store.AccountCredential)
	r.Flag("credential-passphrase", "The passphrase to unlock your credential file. When set, it will not prompt for the passphrase, nor cache it in the OS keyring. Please only use this if you know what you're doing and ensure your passphrase doesn't end up in bash history.").Short('p').StringVar(&store.credentialPassphrase)
	r.Flag("credential-passphrase-cache-ttl", "Cache the credential passphrase in the OS keyring for this duration. The cache is automatically cleared after the timer runs out. Each time the passphrase is read from the cache the timer is reset. Passphrase caching is turned on by default for 5 minutes. Turn it off by setting the duration to 0.").Default("5m").DurationVar(&store.CredentialPassphraseCacheTTL)
}

// IsPassphraseSet returns whether a passphrase is configured.
// This can be because it is already set using SetPassphrase,
// or by using the credential-passphrase flag in the CLI.
func (store *credentialStore) IsPassphraseSet() bool {
	return store.credentialPassphrase != ""
}

// SetPassphrase sets the passphrase used to encrypt the credential.
func (store *credentialStore) SetPassphrase(passphrase string) {
	store.credentialPassphrase = passphrase
}

// Set stores a credential that is returned on Get.
// The credential is not persisted yet in a file.
func (store *credentialStore) Set(credential secrethub.Credential) {
	store.credential = credential
}

// CredentialExists returns whether there is already a credential stored
// in the configured location. When this is the case, a write will
// override this credential.
func (store *credentialStore) CredentialExists() (bool, error) {
	profileDir, err := store.NewProfileDir()
	if err != nil {
		return false, errio.Error(err)
	}
	credentialPath := profileDir.CredentialPath()
	_, err = os.Stat(credentialPath)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, errio.Error(err)
}

// Get retrieves a credential from the store.
// When a credential is set, that credential is returned,
// otherwise the credential is read from the configured file.
func (store *credentialStore) Get() (secrethub.Credential, error) {
	if store.credential != nil {
		return store.credential, nil
	}

	profileDir, err := store.NewProfileDir()
	if err != nil {
		return nil, errio.Error(err)
	}
	return NewCredentialReader(store.io, profileDir, store.AccountCredential, store.newPassphraseReader()).Read()
}

// Save encrypts the credential using the configured passphrase and
// persists it in the configured file.
func (store *credentialStore) Save() error {
	profileDir, err := store.NewProfileDir()
	if err != nil {
		return errio.Error(err)
	}

	err = os.MkdirAll(profileDir.String(), profileDir.FileMode())
	if err != nil {
		return errio.Error(err)
	}

	encoded, err := store.encodeCredential(store.credential, store.credentialPassphrase)
	if err != nil {
		return errio.Error(err)
	}

	err = ioutil.WriteFile(profileDir.CredentialPath(), posix.AddNewLine([]byte(encoded)), profileDir.CredentialFileMode())
	if err != nil {
		return ErrCannotWrite(profileDir.CredentialPath(), err)
	}

	return nil
}

func (store *credentialStore) encodeCredential(credential secrethub.Credential, passphrase string) (string, error) {
	if passphrase != "" {
		armorer, err := secrethub.NewPassBasedKey([]byte(passphrase))
		if err != nil {
			return "", errio.Error(err)
		}
		return secrethub.EncodeEncryptedCredential(credential, armorer)

	}
	return secrethub.EncodeCredential(credential)
}

// newPassphraseReader returns a PassphraseReader configured by the flags.
func (store *credentialStore) newPassphraseReader() PassphraseReader {
	return NewPassphraseReader(store.io, store.credentialPassphrase, store.CredentialPassphraseCacheTTL)
}

// NewProfileDir returns a new ProfileDir from the flag configuration.
// TODO SHDEV-1026: Make private once config upgrade uses CredentialStore
func (store *credentialStore) NewProfileDir() (ProfileDir, error) {
	return NewProfileDir(store.ConfigDir)
}
