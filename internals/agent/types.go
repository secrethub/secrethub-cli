package agent

import "github.com/secrethub/secrethub-go/internals/api"

type ErrorResponse struct {
	Error string `json:"error"`
}

type UnlockRequest struct {
	Passphrase string `json:"passphrase"`
	//TTL        time.Duration `json:"ttl,omitempty"`
}

type SignRequest struct {
	Payload []byte `json:"payload"`
}

type SignResponse struct {
	Signature []byte `json:"signature"`
}

type DecryptRequest struct {
	EncryptedData api.EncryptedData `json:"encrypted"`
}

type DecryptResponse struct {
	Decrypted []byte `json:"decrypted"`
}

type FingerprintResponse struct {
	Fingerprint string `json:"fingerprint"`
}
