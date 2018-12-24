package deployment

import (
	"bytes"

	"github.com/keylockerbv/secrethub-cli/pkg/winrm"
	"github.com/secrethub/secrethub-go/internals/errio"
)

// WindowsDeployer deploy a secrets service to a Windows host.
type WindowsDeployer struct {
	conn *winrm.Client
	path string
}

// NewWindowsDeployer creates a WindowsDeployer using a WinRM connection.
func NewWindowsDeployer(conn *winrm.Client, path string) (Deployer, error) {
	wd := WindowsDeployer{
		conn: conn,
		path: path,
	}

	return wd, nil
}

// Configure copies a service credential to a Windows host.
func (wd WindowsDeployer) Configure(token []byte) error {
	r := bytes.NewBuffer(token)
	copyProgress := make(chan int)

	err := wd.conn.CopyFile(r, wd.path, copyProgress)
	if err != nil {
		return errio.Error(err)
	}

	return nil
}
