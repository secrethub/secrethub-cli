package client

import (
	"bufio"
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
	"syscall"
	"time"

	"github.com/secrethub/secrethub-cli/internals/agent"
	"github.com/secrethub/secrethub-go/internals/api"
)

type Client interface {
	Sign(ctx context.Context, payload []byte) ([]byte, error)
	Decrypt(ctx context.Context, payload api.EncryptedData) ([]byte, error)
	Lock(ctx context.Context) error
	Fingerprint(ctx context.Context) (string, error)
}

type client struct {
	http *http.Client
}

func New(configDir string) Client {
	socketPath := filepath.Join(configDir, agent.SocketName)
	return &client{
		http: &http.Client{
			Transport: &http.Transport{
				DialContext: func(context.Context, string, string) (net.Conn, error) {
					return net.Dial("unix", socketPath)
				},
			},
			Timeout: 1 * time.Minute,
		},
	}
}

func (c *client) checkForAgent(ctx context.Context) error {
	if err := c.ping(ctx); err == nil {
		return nil
	}

	if err := c.spawnAgent(); err != nil {
		return fmt.Errorf("could not start agent: %v", err)
	}

	if err := c.waitForAgent(ctx); err != nil {
		return fmt.Errorf("could not start agent: %v", err)
	}
	return nil
}

func (c *client) spawnAgent() error {
	bin, err := os.Executable()
	if err != nil {
		return err
	}
	cmd := exec.Command(bin, "agent")
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Setpgid: true,
	}
	return cmd.Start()

}

func (c *client) ping(ctx context.Context) error {
	pc := &http.Client{
		Transport: c.http.Transport,
		Timeout:   1 * time.Second,
	}
	req, err := http.NewRequest("GET", "http://unix/ping", nil)
	if err != nil {
		return err
	}
	resp, err := pc.Do(req.WithContext(ctx))
	if err != nil {
		return err
	}
	_ = resp.Body.Close()
	return nil
}

func (c *client) waitForAgent(ctx context.Context) error {
	backoffPeriod := time.Millisecond
	for backoffPeriod < 10*time.Second {
		err := c.ping(ctx)
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
	return errors.New("could not reach agent")
}

func (c *client) do(ctx context.Context, method string, path string, in, out interface{}) error {
	err := c.checkForAgent(ctx)
	if err != nil {
		return fmt.Errorf("cannot find agent: %v", err)
	}

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
	if resp.StatusCode == http.StatusUnauthorized {
		err = c.unlock(ctx)
		if err != nil {
			return err
		}
		resp, err = c.http.Do(req.WithContext(ctx))
		if err != nil {
			return err
		}
	}

	respBody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("read response: %v", err)
	}

	switch resp.StatusCode {
	case http.StatusOK:
	default:
		var errResp agent.ErrorResponse
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
	reader := bufio.NewReader(os.Stdin)
	fmt.Print("Enter passphrase: ")
	passphrase, _ := reader.ReadString('\n')

	var resp string

	err := c.do(ctx, "POST", "/unlock", agent.UnlockRequest{
		Passphrase: strings.TrimSuffix(passphrase, "\n"),
	}, &resp)

	if err != nil {
		return err
	}
	return nil
}

func (c *client) Sign(ctx context.Context, payload []byte) ([]byte, error) {
	var resp agent.SignResponse
	err := c.do(ctx, "POST", "/sign", agent.SignRequest{
		Payload: payload,
	}, &resp)
	if err != nil {
		return nil, err
	}
	return resp.Signature, nil
}

func (c *client) Decrypt(ctx context.Context, payload api.EncryptedData) ([]byte, error) {
	var resp agent.DecryptResponse
	err := c.do(ctx, "POST", "/decrypt", agent.DecryptRequest{
		EncryptedData: payload,
	}, &resp)
	if err != nil {
		return nil, err
	}
	return resp.Decrypted, nil
}

func (c *client) Lock(ctx context.Context) error {
	var resp string
	err := c.do(ctx, "POST", "/lock", nil, &resp)
	if err != nil {
		return err
	}
	return nil
}

func (c *client) Fingerprint(ctx context.Context) (string, error) {
	var resp agent.FingerprintResponse
	err := c.do(ctx, "GET", "/fingerprint", nil, &resp)
	if err != nil {
		return "", err
	}
	return resp.Fingerprint, nil
}
