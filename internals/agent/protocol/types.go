package protocol

import "github.com/secrethub/secrethub-go/internals/api"

const (
	SocketName  = "agent.sock"
	PIDFileName = "agent.pid"
)

type VersionResponse struct {
	Version string `json:"version"`
}

type ErrorResponse struct {
	Error string `json:"error"`
}

type UnlockRequest struct {
	Passphrase string `json:"passphrase"`
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
