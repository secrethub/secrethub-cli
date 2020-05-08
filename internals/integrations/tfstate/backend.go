package tfstate

import (
	"bytes"
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

func New(client secrethub.ClientInterface, port uint16, logger io.Writer) *backend {
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
	isChild, err := connectionFromChildProcess(os.Getpid(), r)
	if err != nil {
		w.WriteHeader(http.StatusForbidden)
		return
	}
	if !isChild {
		fmt.Println("can only be reached from a process spawned with secrethub run")
		w.WriteHeader(http.StatusForbidden)
		return
	}

	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		fmt.Printf("Errors: %v\n", err)
		return
	}

	path, password, ok := r.BasicAuth()
	if !ok {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintln(b.logger, "set the SecretHub path to the state as the username")
		return
	}

	if secretpath.Count(path) < 2 {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(b.logger, "set user to a valid repository or directory. Got: %s\n", path)
		return
	}

	statePath := secretpath.Join(path, "state")
	lockPath := secretpath.Join(path, "lock")
	passwordPath := secretpath.Join(path, "password")

	secret, err := b.client.Secrets().ReadString(passwordPath)
	if err != nil && !api.IsErrNotFound(err) {
		w.WriteHeader(http.StatusInternalServerError)
		return
	} else if err == nil {
		if password == "" {
			w.WriteHeader(http.StatusUnauthorized)
			fmt.Fprintf(w, "password stored at %s should be set as auth password", passwordPath)
			return
		}
		if password != secret {
			w.WriteHeader(http.StatusForbidden)
			fmt.Fprintf(w, "provided password does not password stored at %s", passwordPath)
			return
		}
	}

	switch r.Method {
	case http.MethodGet:
		secret, err := b.client.Secrets().Read(statePath)
		if api.IsErrNotFound(err) {
			w.WriteHeader(http.StatusNotFound)
			return
		} else if err != nil {
			fmt.Fprintf(b.logger, "%v\n", err)

			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		w.Write(secret.Data)
	case http.MethodPost:
		_, err = b.client.Secrets().Write(statePath, body)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Fprintf(w, err.Error())
			return
		}
	case "LOCK":
		currentLock, err := b.client.Secrets().Versions().GetWithData(lockPath + ":1")
		if err == nil {
			w.WriteHeader(http.StatusLocked)
			fmt.Fprint(w, string(currentLock.Data))
			return
		} else if !api.IsErrNotFound(err) {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		res, err := b.client.Secrets().Write(lockPath, body)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Fprintf(w, err.Error())
			return
		}
		if res.Version != 1 {
			w.WriteHeader(http.StatusLocked)
		}

	case "UNLOCK":
		secret, err := b.client.Secrets().Read(lockPath)
		if api.IsErrNotFound(err) {
			w.WriteHeader(http.StatusOK)
			fmt.Fprintf(w, "not locked")
			return
		}

		if len(body) > 0 && !bytes.Equal(body, secret.Data) {
			w.WriteHeader(http.StatusBadRequest)
			fmt.Fprintf(w, "incorrect lock")
			return
		}

		err = b.client.Secrets().Delete(lockPath)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
	default:
		w.WriteHeader(http.StatusNotImplemented)
	}
}

type prefixWriter struct {
	io.Writer
	prefix string
}

func (l prefixWriter) Write(p []byte) (int, error) {
	_, err := fmt.Fprintf(l.Writer, "%s%s", l.prefix, p)
	return len(p), err
}
