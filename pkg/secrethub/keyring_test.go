package secrethub

import (
	"testing"
	"time"

	"github.com/keylockerbv/secrethub/testutil"
	logging "github.com/op/go-logging"
	libkeyring "github.com/zalando/go-keyring"
)

var (
	password        = "test-password"
	testLogger      = logging.MustGetLogger("keyring_test")
	testTTL         = 15 * time.Second
	testKeyringItem = &KeyringItem{
		RunningCleanupProcess: false,
		ExpiresAt:             time.Now().UTC().Add(testTTL),
		Passphrase:            password,
	}
)

func newTestKeyring() Keyring {
	libkeyring.MockInit()
	return NewKeyring()
}

type TestKeyringCleaner struct {
	cleanupCalled bool
}

func (c *TestKeyringCleaner) Cleanup(username string) error {
	c.cleanupCalled = true
	return nil
}

// FakePassphraseReader is a helper type that implements the PassphraseReader interface.
type FakePassphraseReader struct {
	pass []byte
	err  error
}

func (fp FakePassphraseReader) Get(username string) ([]byte, error) {
	return fp.pass, fp.err
}

func (fp FakePassphraseReader) IncorrectPassphrase(username string) error {
	return fp.err
}

func TestPassphraseReaderGet_Flag(t *testing.T) {
	// Arrange
	reader := passphraseReader{
		Logger:    testLogger,
		FlagValue: password,
		Cache:     NewPassphraseCache(testTTL, &TestKeyringCleaner{}, newTestKeyring()),
	}

	// Act
	actual, err := reader.Get(username1)

	// Assert
	testutil.OK(t, err)
	testutil.Compare(t, actual, []byte(password))
}

func TestPassphraseReaderGet_Keystore(t *testing.T) {
	// Arrange
	cache := NewPassphraseCache(testTTL, &TestKeyringCleaner{}, newTestKeyring())
	err := cache.Set(username1, password)
	testutil.OK(t, err)
	reader := passphraseReader{
		Logger:    testLogger,
		FlagValue: "",
		Cache:     cache,
	}

	// Act
	actual, err := reader.Get(username1)

	// Assert
	testutil.OK(t, err)
	testutil.Compare(t, actual, []byte(password))
}

func TestPassphraseCacheSetSuccess(t *testing.T) {
	// Arrange
	cache := NewPassphraseCache(testTTL, &TestKeyringCleaner{}, newTestKeyring())

	// Act
	err := cache.Set(username1, password)

	// Assert
	testutil.OK(t, err)
}

func TestPassphraseCacheSet_CleanupCalled(t *testing.T) {
	// Arrange
	cleaner := &TestKeyringCleaner{}
	cache := NewPassphraseCache(testTTL, cleaner, newTestKeyring())

	// Act
	err := cache.Set(username1, password)

	// Assert
	testutil.OK(t, err)
	if !cleaner.cleanupCalled {
		t.Errorf("keyring cleaner not called")
	}
}

func TestPassphraseCacheGet_Success(t *testing.T) {
	// Arrange
	cache := NewPassphraseCache(testTTL, &TestKeyringCleaner{}, newTestKeyring())
	err := cache.Set(username1, password)
	testutil.OK(t, err)

	// Act
	actual, err := cache.Get(username1)

	// Assert
	testutil.OK(t, err)
	testutil.Compare(t, actual, password)
}

func TestPassphraseCacheGet_UpdatedAfterRead(t *testing.T) {
	// Arrange
	keyring := newTestKeyring()
	cache := NewPassphraseCache(testTTL, &TestKeyringCleaner{}, keyring)
	err := cache.Set(username1, password)
	testutil.OK(t, err)

	expected, err := keyring.Get(username1)
	testutil.OK(t, err)

	time.Sleep(20 * time.Millisecond)

	// Act
	_, err = cache.Get(username1)
	testutil.OK(t, err)

	// Assert
	actual, err := keyring.Get(username1)
	testutil.OK(t, err)
	if !actual.ExpiresAt.After(expected.ExpiresAt) {
		t.Errorf("password last read not updated")
	}
}

func TestPassphraseCacheGet_NonExisting(t *testing.T) {
	// Arrange
	cache := NewPassphraseCache(testTTL, &TestKeyringCleaner{}, newTestKeyring())

	// Act
	_, err := cache.Get(username1)

	// Assert
	testutil.Compare(t, err, ErrKeyringItemNotFound)
}

func TestPassphraseCacheGet_Expired(t *testing.T) {
	// Arrange
	keyring := newTestKeyring()
	cache := NewPassphraseCache(testTTL, &TestKeyringCleaner{}, keyring)

	item := &KeyringItem{
		RunningCleanupProcess: false,
		ExpiresAt:             time.Now().Add(-10 * time.Millisecond),
		Passphrase:            password,
	}

	err := keyring.Set(username1, item)
	testutil.OK(t, err)

	// Act
	actual, err := cache.Get(username1)

	// Assert
	testutil.Compare(t, actual, "")
	testutil.Compare(t, err, ErrKeyringItemNotFound)

	_, err = keyring.Get(username1)
	testutil.Compare(t, err, ErrKeyringItemNotFound)
}

func TestKeyringSet_Success(t *testing.T) {
	// Arrange
	keyring := newTestKeyring()

	// Act
	err := keyring.Set(username1, testKeyringItem)

	// Assert
	testutil.OK(t, err)
}

func TestKeyringSet_Twice(t *testing.T) {
	// Arrange
	keyring := newTestKeyring()

	err := keyring.Set(username1, &KeyringItem{
		Passphrase: "first",
	})
	testutil.OK(t, err)

	// Act
	err = keyring.Set(username1, testKeyringItem)

	// Assert
	testutil.OK(t, err)
}

func TestKeyring_Get(t *testing.T) {
	// Arrange
	keyring := newTestKeyring()
	err := keyring.Set(username1, testKeyringItem)
	testutil.OK(t, err)

	// Act
	actual, err := keyring.Get(username1)

	// Assert
	testutil.OK(t, err)
	testutil.Compare(t, actual, testKeyringItem)
}

func TestKeyring_Get_NonExisting(t *testing.T) {
	// Arrange
	keyring := newTestKeyring()

	// Act
	_, err := keyring.Get(username1)

	// Assert
	testutil.Compare(t, err, ErrKeyringItemNotFound)
}

func TestKeyring_Delete(t *testing.T) {
	// Arrange
	keyring := newTestKeyring()
	err := keyring.Set(username1, testKeyringItem)
	testutil.OK(t, err)

	// Act
	err = keyring.Delete(username1)

	// Assert
	testutil.OK(t, err)
}

func TestKeyring_Delete_NonExisting(t *testing.T) {
	// Arrange
	keyring := newTestKeyring()

	// Act
	err := keyring.Delete(username1)

	// Assert
	testutil.Compare(t, err, ErrKeyringItemNotFound)
}
