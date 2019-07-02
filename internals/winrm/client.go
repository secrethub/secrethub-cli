package winrm

import (
	"bytes"
	"crypto/sha1"
	"crypto/tls"
	"crypto/x509"
	"encoding/hex"
	"encoding/pem"
	"fmt"
	"io"
	"net/http"

	"github.com/secrethub/secrethub-cli/internals/cli/ui"

	"github.com/secrethub/secrethub-go/internals/errio"

	"github.com/masterzen/winrm"
)

// Errors
var (
	errWinRM = errio.Namespace("winrm")

	ErrCannotReachHost   = errWinRM.Code("cannot_reach_host").ErrorPref("Cannot reach the Client host: %s")
	ErrNoPeerCertificate = errWinRM.Code("no_peer_cert").Error("Could not get the peer certificate")
	ErrCertNotTrusted    = errWinRM.Code("peer_not_trusted").Error("Peer certificate is not trusted")
	ErrExecutingCommand  = errWinRM.Code("executing_command").ErrorPref("An error occurred while executing the command : %s")

	ErrCertAuthNoClientKey  = errWinRM.Code("no_client_key_certificate_authentication").Error("No client key was supplied to use for certificate authentication")
	ErrCertAuthNoClientCert = errWinRM.Code("no_client_certificate_certificate_authentication").Error("No client certificate was supplied to use for certificate authentication")

	ErrBasicAuthNoUsername = errWinRM.Code("no_username_basic_authentication").Error("No username was supplied to use for basic authentication")
	ErrBasicAuthNoPassword = errWinRM.Code("no_password_basic_authentication").Error("No password was supplied to use for basic authentication")
)

// Config contains all configuration options for a WinRM client connection except
// for the authentication options
type Config struct {
	Host string
	Port int

	HTTPS bool

	SkipVerifyCert bool
	CaCert         []byte
}

/// authMethod gives an interface to retrieve the valid credentials or defaults.
// The Client always needs both values for certificates authentication and basic authentication.
// So we use implementations of this interface to give us the values without having to know the implementation.
// The necessity of this is because of the weird interface giving by our dependency on the masterzen/winrm library.
type authMethod interface {
	// Returns the Client cert and key or empty arrays.
	getClientCert() ([]byte, []byte)
	// Returns the username and password or empty strings.
	getBasic() (string, string)
}

// authBasic implements authMethod for basic authentication.
type authBasic struct {
	username string
	password string
}

// getClientCert returns sane empty defaults for the Client to use.
func (ab *authBasic) getClientCert() ([]byte, []byte) {
	return []byte{}, []byte{}
}

// getBasic returns the username and password.
func (ab *authBasic) getBasic() (string, string) {
	return ab.username, ab.password
}

// NewBasicClient returns a Client with basic authentication using username and password.
// This validates if a proper username and password is given.
// It uses a authBasic implementation for authMethod.
func NewBasicClient(config *Config, username, password string) (*Client, error) {
	if username == "" {
		return nil, ErrBasicAuthNoUsername
	}

	if password == "" {
		return nil, ErrBasicAuthNoPassword
	}

	authMethod := &authBasic{
		username: username,
		password: password,
	}

	client, err := newWinRMClient(config, authMethod)

	return &Client{
		Client: client,
		auth:   authMethod,
		config: config,
	}, err

}

// authCert implements authMethod for certificate authentication.
type authCert struct {
	clientCert []byte
	clientKey  []byte
}

// getClientCert returns the clientCert and clientKey for the Client to use.
func (ac *authCert) getClientCert() ([]byte, []byte) {
	return ac.clientCert, ac.clientKey
}

// getBasic returns sane empty defaults for the Client to use.
func (ac *authCert) getBasic() (string, string) {
	return "", ""
}

// NewCertClient returns a Client with certificate authentication using client certificate and key.
// This validates if a clientKey and clientCert is given.
// It uses a authCert implementation for authMethod.
func NewCertClient(config *Config, clientCert, clientKey []byte) (*Client, error) {
	if clientKey == nil {
		return nil, ErrCertAuthNoClientKey
	}

	if clientCert == nil {
		return nil, ErrCertAuthNoClientCert
	}

	authMethode := &authCert{
		clientCert: clientCert,
		clientKey:  clientKey,
	}

	client, err := newWinRMClient(config, authMethode)

	return &Client{
		Client: client,
		auth:   authMethode,
		config: config,
	}, err

}

// newWinRMClient creates a WinRM client based upon the config and the authMethod.
func newWinRMClient(config *Config, auth authMethod) (*winrm.Client, error) {
	clientCert, clientKey := auth.getClientCert()
	username, password := auth.getBasic()
	endpoint := winrm.NewEndpoint(config.Host, config.Port, config.HTTPS, config.SkipVerifyCert, config.CaCert, clientKey, clientCert, 0)
	client, err := winrm.NewClient(endpoint, username, password)
	return client, err
}

// Client contains all necessary data for a Client connection.
type Client struct {
	*winrm.Client
	auth   authMethod
	config *Config
}

// GetServerCert retrieves the server TLS certificate of a Client-host.
// If the certificate is not trusted, the user is asked whether to trust the certificate fingerprint.
// If the user trusts the fingerprint, the server certificate is set as the Client CA certificate,
// so a following TLS connection will succeed.
func (c *Client) GetServerCert(io ui.IO) error {
	if !c.config.HTTPS || c.config.SkipVerifyCert || c.config.CaCert != nil {
		return nil
	}
	url := fmt.Sprintf("https://%s:%d/wsman", c.config.Host, c.config.Port)
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{
			// We use this as default as the self signed certificate does not have a chain leading up to a CA.
			InsecureSkipVerify: true,
		},
	}
	httpClient := &http.Client{Transport: tr}

	resp, err := httpClient.Get(url)
	if err != nil {
		return ErrCannotReachHost(err)
	}

	if len(resp.TLS.PeerCertificates) < 1 {
		return ErrNoPeerCertificate
	}

	cert := resp.TLS.PeerCertificates[0]

	_, err = cert.Verify(x509.VerifyOptions{})
	if err != nil {
		_, unknownAuthority := err.(x509.UnknownAuthorityError)
		if unknownAuthority {
			h := sha1.New()
			_, err := h.Write(cert.Raw)
			if err != nil {
				return err
			}
			fingerprint := formatHex(h.Sum(nil))

			question := fmt.Sprintf("Certificate Authority is not currently in the truststore. Do you trust certificate with fingerprint %s?", fingerprint)

			answer, err := ui.AskYesNo(io, question, ui.DefaultNo)
			if err != nil {
				return err
			}

			if !answer {
				return ErrCertNotTrusted
			}

			c.config.CaCert = pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: cert.Raw})
			// Reinitialise the client with the CaCert.
			c.Client, err = newWinRMClient(c.config, c.auth)
			if err != nil {
				return err
			}
			return nil
		}
		return err
	}
	return nil
}

// CopyFile copies the data from a io.Reader to a file at the destination path on the host.
// The progress (in amount of bytes transferred) of the copy is published on the progress channel.
func (c *Client) CopyFile(file io.Reader, dest string, progress chan int) error {
	return doCopy(c.Client, file, dest, progress)
}

// RunCommand executes a command on the host.
// StdOut and StdErr of the command are written to the respective writers.
// If an ErrExecutingCommand is returned, then the stdErr should be checked.
func (c *Client) RunCommand(command string, out, stdErr *bytes.Buffer) error {
	exitCode, err := c.Client.Run(command, out, stdErr)
	if exitCode != 0 || err != nil {
		if err != nil {
			return ErrExecutingCommand(err)
		}
		// No error is given yet exit code is 0, so wrap the stdErr in an error to give a meaningful error back.
		return ErrExecutingCommand(stdErr.String())
	}

	return nil
}

// formatHex formats a slice of bytes as a hex-string. Every 2 bytes are separated by a colon.
// For example: 11:f0:b2
func formatHex(b []byte) string {
	buf := make([]byte, 0, 3*len(b))
	x := buf[1*len(b) : 3*len(b)]
	hex.Encode(x, b)
	for i := 0; i < len(x); i += 2 {
		buf = append(buf, x[i], x[i+1], ':')
	}
	return string(buf[:len(buf)-1])
}
