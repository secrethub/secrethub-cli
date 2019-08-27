package secrethub

import (
	"testing"
	"time"

	"github.com/secrethub/secrethub-go/internals/assert"

	libkeyring "github.com/zalando/go-keyring"
)

var (
	password        = "test-password"
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

func (c *TestKeyringCleaner) Cleanup() error {
	c.cleanupCalled = true
	return nil
}

// FakePassphraseReader is a helper type that implements the PassphraseReader interface.
type FakePassphraseReader struct {
	pass []byte
	err  error
}

func (fp FakePassphraseReader) Get() ([]byte, error) {
	return fp.pass, fp.err
}

func TestPassphraseReaderGet_Flag(t *testing.T) {
	// Arrange
	reader := passphraseReader{
		FlagValue: password,
		Cache:     NewPassphraseCache(testTTL, &TestKeyringCleaner{}, newTestKeyring()),
	}

	// Act
	actual, err := reader.get()

	// Assert
	assert.OK(t, err)
	assert.Equal(t, actual, password)
}

func TestPassphraseReaderGet_Keystore(t *testing.T) {
	// Arrange
	cache := NewPassphraseCache(testTTL, &TestKeyringCleaner{}, newTestKeyring())
	err := cache.Set(password)
	assert.OK(t, err)
	reader := passphraseReader{
		FlagValue: "",
		Cache:     cache,
	}

	// Act
	actual, err := reader.get()

	// Assert
	assert.OK(t, err)
	assert.Equal(t, actual, password)
}

func TestPassphraseCacheSetSuccess(t *testing.T) {
	// Arrange
	cache := NewPassphraseCache(testTTL, &TestKeyringCleaner{}, newTestKeyring())

	// Act
	err := cache.Set(password)

	// Assert
	assert.OK(t, err)
}

func TestPassphraseCacheSet_CleanupCalled(t *testing.T) {
	// Arrange
	cleaner := &TestKeyringCleaner{}
	cache := NewPassphraseCache(testTTL, cleaner, newTestKeyring())

	// Act
	err := cache.Set(password)

	// Assert
	assert.OK(t, err)
	if !cleaner.cleanupCalled {
		t.Errorf("keyring cleaner not called")
	}
}

func TestPassphraseCacheGet_Success(t *testing.T) {
	// Arrange
	cache := NewPassphraseCache(testTTL, &TestKeyringCleaner{}, newTestKeyring())
	err := cache.Set(password)
	assert.OK(t, err)

	// Act
	actual, err := cache.Get()

	// Assert
	assert.OK(t, err)
	assert.Equal(t, actual, password)
}

func TestPassphraseCacheGet_UpdatedAfterRead(t *testing.T) {
	// Arrange
	keyring := newTestKeyring()
	cache := NewPassphraseCache(testTTL, &TestKeyringCleaner{}, keyring)
	err := cache.Set(password)
	assert.OK(t, err)

	expected, err := keyring.Get()
	assert.OK(t, err)

	time.Sleep(20 * time.Millisecond)

	// Act
	_, err = cache.Get()
	assert.OK(t, err)

	// Assert
	actual, err := keyring.Get()
	assert.OK(t, err)
	if !actual.ExpiresAt.After(expected.ExpiresAt) {
		t.Errorf("password last read not updated")
	}
}

func TestPassphraseCacheGet_NonExisting(t *testing.T) {
	// Arrange
	cache := NewPassphraseCache(testTTL, &TestKeyringCleaner{}, newTestKeyring())

	// Act
	_, err := cache.Get()

	// Assert
	assert.Equal(t, err, ErrKeyringItemNotFound)
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

	err := keyring.Set(item)
	assert.OK(t, err)

	// Act
	actual, err := cache.Get()

	// Assert
	assert.Equal(t, actual, "")
	assert.Equal(t, err, ErrKeyringItemNotFound)

	_, err = keyring.Get()
	assert.Equal(t, err, ErrKeyringItemNotFound)
}

func TestKeyringSet_Success(t *testing.T) {
	// Arrange
	keyring := newTestKeyring()

	// Act
	err := keyring.Set(testKeyringItem)

	// Assert
	assert.OK(t, err)
}

func TestKeyringSet_Twice(t *testing.T) {
	// Arrange
	keyring := newTestKeyring()

	err := keyring.Set(&KeyringItem{
		Passphrase: "first",
	})
	assert.OK(t, err)

	// Act
	err = keyring.Set(testKeyringItem)

	// Assert
	assert.OK(t, err)
}

func TestKeyring_Get(t *testing.T) {
	// Arrange
	keyring := newTestKeyring()
	err := keyring.Set(testKeyringItem)
	assert.OK(t, err)

	// Act
	actual, err := keyring.Get()

	// Assert
	assert.OK(t, err)
	assert.Equal(t, actual, testKeyringItem)
}

func TestKeyring_Get_NonExisting(t *testing.T) {
	// Arrange
	keyring := newTestKeyring()

	// Act
	_, err := keyring.Get()

	// Assert
	assert.Equal(t, err, ErrKeyringItemNotFound)
}

func TestKeyring_Delete(t *testing.T) {
	// Arrange
	keyring := newTestKeyring()
	err := keyring.Set(testKeyringItem)
	assert.OK(t, err)

	// Act
	err = keyring.Delete()

	// Assert
	assert.OK(t, err)
}

func TestKeyring_Delete_NonExisting(t *testing.T) {
	// Arrange
	keyring := newTestKeyring()

	// Act
	err := keyring.Delete()

	// Assert
	assert.Equal(t, err, ErrKeyringItemNotFound)
}
