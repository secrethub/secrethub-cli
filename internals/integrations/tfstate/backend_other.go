// +build !windows !386

package tfstate

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"

	"github.com/secrethub/secrethub-go/internals/api"
	"github.com/secrethub/secrethub-go/pkg/secrethub"
	"github.com/secrethub/secrethub-go/pkg/secretpath"
)

type backend struct {
	client secrethub.ClientInterface
	port   uint16
	logger io.Writer
}

func New(client secrethub.ClientInterface, port uint16, logger io.Writer) Backend {
	return &backend{
		client: client,
		port:   port,
		logger: prefixWriter{
			Writer: logger,
			prefix: "[SecretHub]: ",
		},
	}
}

func (b *backend) Serve() error {
	server := &http.Server{
		Addr:    fmt.Sprintf("127.0.0.1:%d", b.port),
		Handler: http.HandlerFunc(b.Handle),
	}
	err := server.ListenAndServe()
	if err != nil && err != http.ErrServerClosed {
		return err
	}

	return nil
}

func (b *backend) Handle(w http.ResponseWriter, r *http.Request) {
	resp, err := b.handle(r)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(b.logger, "Encountered an unexpected error: %s\n", err)
		return
	}
	w.WriteHeader(resp.code)
	fmt.Fprintf(w, resp.body)
}

type statusResponse struct {
	code int
	body string
}

func (b *backend) respondError(statusCode int, format string, a ...interface{}) *statusResponse {
	msg := fmt.Sprintf(format, a...)
	fmt.Fprintf(b.logger, "%s\n", msg)
	return &statusResponse{
		code: statusCode,
		body: msg,
	}
}

func (b *backend) handle(r *http.Request) (*statusResponse, error) {
	isChild, err := connectionFromChildProcess(os.Getpid(), r)
	if err != nil {
		return nil, err
	}
	if !isChild {
		return b.respondError(http.StatusForbidden, "can only be reached from a process spawned with secrethub run"), nil
	}

	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		return nil, fmt.Errorf("reading request body: %s", err)
	}

	path, password, ok := r.BasicAuth()
	if !ok {
		return b.respondError(http.StatusBadRequest, "`username` field must be set to the SecretHub directory where the Terraform state should be stored (for example `<org>/<repo>/terraform-state`)"), nil
	}

	if secretpath.Count(path) < 2 {
		return b.respondError(http.StatusBadRequest, "`username` field must be set to the SecretHub directory where the Terraform state should be stored (for example `<org>/<repo>/terraform-state`), got: %s", path), nil
	}

	statePath := secretpath.Join(path, "state")
	lockPath := secretpath.Join(path, "lock")
	passwordPath := secretpath.Join(path, "password")

	if _, err := b.client.Dirs().GetTree(path, 0, false); api.IsErrNotFound(err) {
		return b.respondError(http.StatusBadRequest, "`%s` is not a directory on SecretHub", path), nil
	}

	secret, err := b.client.Secrets().ReadString(passwordPath)
	if err != nil && !api.IsErrNotFound(err) {
		return nil, err
	} else if err == nil {
		if password == "" {
			return b.respondError(http.StatusUnauthorized, "password stored at %s should be set as auth password", passwordPath), nil
		}
		if password != secret {
			return b.respondError(http.StatusForbidden, "provided password does not match password stored at %s", passwordPath), nil
		}
	}

	switch r.Method {
	case http.MethodGet:
		secret, err := b.client.Secrets().Read(statePath)
		if api.IsErrNotFound(err) {
			return &statusResponse{code: http.StatusNotFound}, nil
		} else if err != nil {
			return nil, err
		}

		return &statusResponse{
			code: http.StatusOK,
			body: string(secret.Data),
		}, nil
	case http.MethodPost:
		_, err = b.client.Secrets().Write(statePath, body)
		if api.IsErrNotFound(err) {
			return b.respondError(http.StatusNotFound, err.Error()), nil
		} else if err != nil {
			return nil, err
		}
		return &statusResponse{
			code: http.StatusOK,
		}, nil
	case "LOCK":
		currentLock, err := b.client.Secrets().Versions().GetWithData(lockPath + ":1")
		if err == nil {
			return &statusResponse{
				code: http.StatusLocked,
				body: string(currentLock.Data),
			}, nil
		} else if !api.IsErrNotFound(err) {
			return nil, err
		}

		res, err := b.client.Secrets().Write(lockPath, body)
		if api.IsErrNotFound(err) {
			return b.respondError(http.StatusNotFound, err.Error()), nil
		} else if err != nil {
			return nil, err
		}
		if res.Version != 1 {
			return &statusResponse{
				code: http.StatusLocked,
			}, nil
		}
		return &statusResponse{
			code: http.StatusOK,
		}, nil

	case "UNLOCK":
		secret, err := b.client.Secrets().Read(lockPath)
		if api.IsErrNotFound(err) {
			return &statusResponse{
				code: http.StatusOK,
				body: "not locked",
			}, nil
		}

		if len(body) > 0 && !bytes.Equal(body, secret.Data) {
			return b.respondError(http.StatusBadRequest, "incorrect lock"), nil
		}

		err = b.client.Secrets().Delete(lockPath)
		if err != nil {
			return nil, err
		}
		return &statusResponse{
			code: http.StatusOK,
		}, nil
	default:
		return nil, errors.New("received an unexpected request")
	}
}
