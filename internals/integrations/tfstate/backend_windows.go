// +build windows,386

package tfstate

import (
	"errors"
	"io"

	"github.com/secrethub/secrethub-go/pkg/secrethub"
)

type notSupportedBackend struct {
}

func New(client secrethub.ClientInterface, port uint16, logger io.Writer) Backend {
	return &notSupportedBackend{}
}

func (b *notSupportedBackend) Serve() error {
	return errors.New("tfstate backend currently not supported on Windows i386")
}
