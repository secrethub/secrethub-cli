package secrethub

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	libkeyring "github.com/zalando/go-keyring"

	"github.com/secrethub/secrethub-cli/internals/cli/cloneproc"
	"github.com/secrethub/secrethub-cli/internals/cli/ui"
	"github.com/secrethub/secrethub-go/pkg/secrethub/credentials"
)

// Errors
var (
	ErrKeyringItemNotFound           = errMain.Code("keyring_not_found").Error("item not found in keyring")
	ErrCannotGetKeyringItem          = errMain.Code("cannot_get_keyring").ErrorPref("cannot get passphrase from keyring: %s")
	ErrCannotSetKeyringItem          = errMain.Code("cannot_set_keyring").ErrorPref("cannot set passphrase in keyring: %s")
	ErrCannotDeleteKeyringItem       = errMain.Code("cannot_delete_keyring").ErrorPref("cannot delete passphrase from keyring: %s")
	ErrCannotClearExpiredKeyringItem = errMain.Code("cannot_clear_expired_keyring_item").ErrorPref("cannot clear expired keyring item: %s")
	ErrPassphraseFlagNotSet          = errMain.Code("passphrase_not_set").Error(
		fmt.Sprintf(
			"required --key-passphrase, -p flag has not been set.\n\n"+
				"When input or output is piped, the --key-passphrase flag (or %s_KEY_PASSPHRASE env var) is required. "+
				"Please only use this if you know what you're doing and ensure your passphrase doesn't end up in bash history.",
			strings.ToUpper(ApplicationName),
		),
	)
)

const (
	keyringServiceLabel = "secrethub"
	keyringKey          = "secrethub-passphrase"
)

// PassphraseReader can retrieve a password and be instructed if the password is incorrect.
// The implementation can determine to do some clean up if the password is incorrect.
type PassphraseReader interface {
	Get(username string) ([]byte, error)
	IncorrectPassphrase(username string) error
}

// passphraseReader provides passphrase reading capability to the CLI.
type passphraseReader struct {
	tries     int
	hasAsked  bool
	io        ui.IO
	FlagValue string
	Cache     *PassphraseCache
}

func (pr *passphraseReader) Read() ([]byte, error) {
	defer func() { pr.tries++ }()

	if pr.tries > 0 {
		_ = pr.Cache.Delete()
	}

	passphrase, err := pr.get()
	if err != nil {
		_ = pr.Cache.Delete()
		return nil, err
	}
	return []byte(passphrase), nil
}

// NewPassphraseReader constructs a new PassphraseReader using values in the CLI.
func NewPassphraseReader(io ui.IO, credentialPassphrase string, credentialPassphraseTTL time.Duration) credentials.PassphraseReader {
	ttl := credentialPassphraseTTL
	cleaner := NewKeyringCleaner()
	keyring := NewKeyring()

	return &passphraseReader{
		io:        io,
		FlagValue: credentialPassphrase,
		Cache:     NewPassphraseCache(ttl, cleaner, keyring),
	}
}

// Get returns the passphrase for the keyfile. When caching is enabled,
// it will cache the passphrase for later Get calls. It retrieves the
// passphrase from the following sources in order of preference:
//  1. The value provided by a flag.
//  2. PassphraseCache
//  3. Input typed in by the user.
func (pr *passphraseReader) get() (string, error) {
	if pr.FlagValue != "" {
		if pr.tries == 0 {
			return pr.FlagValue, nil
		}
		return "", nil
	}

	if pr.Cache.IsEnabled() {
		passphrase, err := pr.Cache.Get()
		if err != nil && err != ErrKeyringItemNotFound {
			return "", err
		} else if err == nil {
			return passphrase, nil
		}
	}
	var err error
	var passphrase string
	if pr.hasAsked {
		passphrase, err = ui.AskSecret(pr.io, "Incorrect passphrase, try again:")
	} else {
		passphrase, err = ui.AskSecret(pr.io, "Please put in the passphrase to unlock your credential:")
	}
	if err == ui.ErrCannotAsk {
		return "", ErrPassphraseFlagNotSet // if we cannot ask, users should use the --passphrase flag
	} else if err != nil {
		return "", err
	}

	pr.hasAsked = true

	if pr.Cache.IsEnabled() && passphrase != "" {
		err := pr.Cache.Set(passphrase)
		if err != nil {
			return "", err
		}
	}

	return passphrase, nil
}

// PassphraseCache caches passphrases in a keyring for a given time to live.
type PassphraseCache struct {
	keyring Keyring
	ttl     time.Duration
	cleaner KeyringCleaner
}

// NewPassphraseCache returns a PassphraseCache initialised with the given arguments.
func NewPassphraseCache(ttl time.Duration, cleaner KeyringCleaner, keyring Keyring) *PassphraseCache {
	return &PassphraseCache{
		keyring: keyring,
		ttl:     ttl,
		cleaner: cleaner,
	}
}

// IsEnabled determines whether passphrases can be cached.
func (c PassphraseCache) IsEnabled() bool {
	return c.ttl > 0 && c.keyring.IsAvailable()
}

// Set caches the passphrase for the configured time to live.
func (c PassphraseCache) Set(passphrase string) error {
	item, err := c.keyring.Get()
	if err == ErrKeyringItemNotFound {
		item = &KeyringItem{
			Passphrase: []byte(passphrase),
		}
	} else if err != nil {
		return err
	}

	if !item.RunningCleanupProcess {
		err = c.cleaner.Cleanup()
		if err != nil {
			return err
		}
	}

	item.ExpiresAt = c.ExpiresAt()

	return c.keyring.Set(item)
}

// Get returns a passphrase for the given username if it was cached.
// Every call to Get resets the time to live of the passphrase.
func (c PassphraseCache) Get() (string, error) {
	item, err := c.keyring.Get()
	if err != nil {
		return "", err
	}

	if item.IsExpired() {
		err := c.keyring.Delete()
		if err != nil && err != ErrKeyringItemNotFound {
			return "", ErrCannotClearExpiredKeyringItem(err)
		}
		return "", ErrKeyringItemNotFound
	}

	if !item.RunningCleanupProcess {
		err = c.cleaner.Cleanup()
		if err != nil {
			return "", err
		}
	}

	item.ExpiresAt = c.ExpiresAt()

	err = c.keyring.Set(item)
	if err != nil {
		return "", err
	}

	return string(item.Passphrase), nil
}

// Delete tries delete the stored passphrase for a given username.
func (c PassphraseCache) Delete() error {
	return c.keyring.Delete()
}

// ExpiresAt returns a timestamp to expire a keyring item at.
func (c PassphraseCache) ExpiresAt() time.Time {
	return time.Now().UTC().Add(c.ttl)
}

// KeyringItem wraps a passphrase with metadata to be stored the keyring.
type KeyringItem struct {
	RunningCleanupProcess bool      `json:"running_cleanup_process,omitempty"`
	ExpiresAt             time.Time `json:"expires_at"`
	Passphrase            []byte    `json:"passphrase"`
}

// IsExpired returns true when the item has expired.
func (ki KeyringItem) IsExpired() bool {
	return time.Now().After(ki.ExpiresAt)
}

// Keyring is an OS-agnostic interface for setting, getting and
// deleting secrets from the system keyring.
type Keyring interface {
	IsAvailable() bool
	Get() (*KeyringItem, error)
	Set(item *KeyringItem) error
	Delete() error
}

// keyring implements Keyring interface by using libkeyring.
type keyring struct {
	usernameMaxLen int
	label          string
}

// NewKeyring returns a new Keyring
// KeyRing only supports usernames up to 20 characters to ensure the maximum input for the macOS keyring is not achieved.
// There is also a limited on the maximum length of password about 900 characters, but this is ridiculously long.
// It is very unlikely that it is hit, and hard to fix for a system up for replacement.
func NewKeyring() Keyring {
	return &keyring{
		usernameMaxLen: 20,
		label:          keyringServiceLabel,
	}
}

// IsAvailable returns true when the OS keyring is available.
// On some operating systems it may not be installed.
func (kr keyring) IsAvailable() bool {
	_, err := libkeyring.Get(kr.label, "keyring_availability_check")
	return err == libkeyring.ErrNotFound || err == nil
}

// Get gets an item from the keyring for the given username.
// This should not be used outside this file!
func (kr keyring) Get() (*KeyringItem, error) {
	stored, err := libkeyring.Get(kr.label, keyringKey)
	if err == libkeyring.ErrNotFound {
		return nil, ErrKeyringItemNotFound
	} else if err != nil {
		return nil, ErrCannotGetKeyringItem(err)
	}

	item := &KeyringItem{}
	err = json.Unmarshal([]byte(stored), item)
	if err != nil {
		return nil, ErrCannotGetKeyringItem(err)
	}

	return item, nil
}

// Set sets an item for the given username in the keyring.
// This should not be used outside this file!
func (kr keyring) Set(item *KeyringItem) error {
	bytes, err := json.Marshal(item)
	if err != nil {
		return ErrCannotSetKeyringItem(err)
	}

	err = libkeyring.Set(kr.label, keyringKey, string(bytes))
	if err != nil {
		return ErrCannotSetKeyringItem(err)
	}

	return nil
}

// Delete deletes an item in the keyring for a given username.
func (kr keyring) Delete() error {
	err := libkeyring.Delete(kr.label, keyringKey)
	if err == libkeyring.ErrNotFound {
		return ErrKeyringItemNotFound
	} else if err != nil {
		return ErrCannotDeleteKeyringItem(err)
	}

	return nil
}

// KeyringCleaner is used to remove items from a keyring.
type KeyringCleaner interface {
	// Cleanup removes an item from the keyring when it expires.
	Cleanup() error
}

// keyringCleaner cleans up the credential by spawning a new CLI process that will take care of cleaning up the credential.
type keyringCleaner struct{}

// NewKeyringCleaner returns a new KeyringCleaner.
func NewKeyringCleaner() KeyringCleaner {
	return &keyringCleaner{}
}

// Cleanup starts a Cleanup process to clean up the cached passphrase when it expires.
func (kc keyringCleaner) Cleanup() error {
	err := cloneproc.Spawn("keyring-clear")
	if err != nil {
		return err
	}

	return nil
}
