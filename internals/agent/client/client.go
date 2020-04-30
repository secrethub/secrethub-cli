package client

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/secrethub/secrethub-go/internals/api"

	"github.com/secrethub/secrethub-cli/internals/agent/protocol"
)

var (
	errAgentLocked     = errors.New("agent is locked")
	errWrongPassphrase = errors.New("wrong passphrase")
)

type Client interface {
	Sign(ctx context.Context, payload []byte) ([]byte, error)
	Decrypt(ctx context.Context, payload api.EncryptedData) ([]byte, error)
	Lock(ctx context.Context) error
	Fingerprint(ctx context.Context) (string, error)
}

type Prompter func(string) (string, error)

type client struct {
	prompter   func(string) (string, error)
	http       *http.Client
	cliVersion string
}

func New(configDir, version string, prompter Prompter) Client {
	socketPath := filepath.Join(configDir, protocol.SocketName)
	return &client{
		http: &http.Client{

			Transport: &http.Transport{
				DisableKeepAlives: true,
				DialContext: func(ctx context.Context, _ string, _ string) (net.Conn, error) {
					dialer := net.Dialer{}
					return dialer.DialContext(ctx, "unix", socketPath)
				},
			},
			Timeout: 10 * time.Second,
		},
		cliVersion: version,
		prompter:   prompter,
	}
}

func (c *client) checkForAgent(ctx context.Context) error {
	if version, err := c.version(ctx); err == nil && version == c.cliVersion {
		return nil
	}
	if err := c.stopAgent(); err != nil {
		return fmt.Errorf("could not stop: %v", err)
	}

	if err := c.startAgent(); err != nil {
		return fmt.Errorf("could not start: %v", err)
	}

	if err := c.waitForAgent(ctx); err != nil {
		return fmt.Errorf("could not reach agent: %v", err)
	}
	return nil
}

func (c *client) stopAgent() error {
	bin, err := os.Executable()
	if err != nil {
		return err
	}

	args := []string{"agent", "--kill"}
	cmd := exec.Command(bin, args...)
	return cmd.Run()
}

func (c *client) startAgent() error {
	bin, err := os.Executable()
	if err != nil {
		return err
	}

	args := []string{"agent", "--daemon"}
	cmd := exec.Command(bin, args...)
	return cmd.Run()
}

func (c *client) version(ctx context.Context) (string, error) {
	var versionResp protocol.VersionResponse
	err := c.do(ctx, "GET", "/version", nil, &versionResp)
	if err != nil {
		return "", err
	}
	return versionResp.Version, nil
}

func (c *client) waitForAgent(ctx context.Context) error {
	backoffPeriod := time.Millisecond
	for backoffPeriod < 10*time.Second {
		_, err := c.version(ctx)
		if err == nil {
			return nil
		}
		if ctx.Err() != nil {
			return ctx.Err()
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(backoffPeriod):
		}
		backoffPeriod *= 2
	}
	return errors.New("timeout")
}

func (c *client) doWithCheck(ctx context.Context, method string, path string, in, out interface{}) error {
	err := c.checkForAgent(ctx)
	if err != nil {
		return err
	}
	err = c.do(ctx, method, path, in, out)
	if err == errAgentLocked {
		err := c.unlock(ctx)
		if err != nil {
			return fmt.Errorf("unlock: %v", err)
		}
		err = c.do(ctx, method, path, in, out)
		if err != nil {
			return err
		}
	} else if err != nil {
		return err
	}
	return nil
}

func (c *client) do(ctx context.Context, method string, path string, in, out interface{}) error {
	bodyJson, err := json.Marshal(in)
	if err != nil {
		return fmt.Errorf("encode body: %v", err)
	}

	req, err := http.NewRequest(method, "http://unix"+path, bytes.NewBuffer(bodyJson))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.http.Do(req.WithContext(ctx))
	if err != nil {
		return err
	}

	respBody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("read response: %v", err)
	}

	switch resp.StatusCode {
	case http.StatusOK:
	case http.StatusUnauthorized:
		return errAgentLocked
	case http.StatusForbidden:
		return errWrongPassphrase
	default:
		var errResp protocol.ErrorResponse
		err = json.Unmarshal(respBody, &errResp)
		if err != nil {
			return fmt.Errorf("unpack response (error %s): %v", resp.Status, err)
		}
		return fmt.Errorf("unexpected status %s: %s", resp.Status, errResp.Error)
	}

	err = json.Unmarshal(respBody, out)
	if err != nil {
		return fmt.Errorf("unpack response: %v", err)
	}
	return nil
}

func (c *client) unlock(ctx context.Context) error {
	for i := 0; i < 3; i++ {
		passphrase, err := c.prompter("Enter passphrase: ")
		if err != nil {
			return fmt.Errorf("prompt password: %v", err)
		}

		var resp string

		err = c.do(ctx, "POST", "/unlock", protocol.UnlockRequest{
			Passphrase: strings.TrimSuffix(passphrase, "\n"),
		}, &resp)
		if err == errWrongPassphrase {
			continue
		} else if err != nil {
			return err
		}
		return nil
	}
	return errWrongPassphrase
}

func (c *client) Sign(ctx context.Context, payload []byte) ([]byte, error) {
	var resp protocol.SignResponse
	err := c.doWithCheck(ctx, "POST", "/sign", protocol.SignRequest{
		Payload: payload,
	}, &resp)
	if err != nil {
		return nil, err
	}
	return resp.Signature, nil
}

func (c *client) Decrypt(ctx context.Context, payload api.EncryptedData) ([]byte, error) {
	var resp protocol.DecryptResponse
	err := c.doWithCheck(ctx, "POST", "/decrypt", protocol.DecryptRequest{
		EncryptedData: payload,
	}, &resp)
	if err != nil {
		return nil, err
	}
	return resp.Decrypted, nil
}

func (c *client) Lock(ctx context.Context) error {
	var resp string
	err := c.doWithCheck(ctx, "POST", "/lock", nil, &resp)
	if err != nil {
		return err
	}
	return nil
}

func (c *client) Fingerprint(ctx context.Context) (string, error) {
	var resp protocol.FingerprintResponse
	err := c.doWithCheck(ctx, "GET", "/fingerprint", nil, &resp)
	if err != nil {
		return "", err
	}
	return resp.Fingerprint, nil
}
