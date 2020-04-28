package agent

import (
	"errors"
	"fmt"
	"net/http"
	"reflect"
	"time"
	"unsafe"

	"github.com/secrethub/secrethub-go/pkg/secrethub/configdir"
	"github.com/secrethub/secrethub-go/pkg/secrethub/credentials"
)

const (
	softTimeout = 5 * time.Minute
	hardTimeout = 30 * time.Minute
)

type session struct {
	credential *credentials.RSACredential
	//passphraseValidator *passphraseValidator
	softTTL time.Time
	hardTTL time.Time
}

func (s *session) valid() bool {
	if time.Until(s.softTTL) < 0 {
		return false
	}

	if time.Until(s.hardTTL) < 0 {
		return false
	}

	if s.credential == nil {
		return false
	}

	return true
}

func GetUnexportedField(field reflect.Value) interface{} {
	return reflect.NewAt(field.Type(), unsafe.Pointer(field.UnsafeAddr())).Elem().Interface()
}

func (s *session) unlock(passphrase string) error {
	//if s.passphraseValidator != nil && s.credential != nil {
	//	if err := s.passphraseValidator.validate([]byte(passphrase)); err != nil {
	//		return err
	//	}
	//	return nil
	//}

	config, err := configdir.Default()
	if err != nil {
		return fmt.Errorf("find credential: %v", err)
	}

	cred, err := credentials.ImportKey(config.Credential(), credentials.FromString(passphrase))
	if err != nil {
		return fmt.Errorf("unlocking credential: %v", err)
	}
	rsaCred := GetUnexportedField(reflect.ValueOf(&cred).Elem().FieldByName("key")).(*credentials.RSACredential)

	s.updateSoftTTL(softTimeout)
	s.credential = rsaCred
	//s.passphraseValidator = pv
	return nil
}

func (s *session) updateSoftTTL(duration time.Duration) {
	s.softTTL = time.Now().Add(duration)
}

type sessionManager struct {
	sessions map[int]*session
}

func newSessionManager() *sessionManager {
	return &sessionManager{
		sessions: make(map[int]*session),
	}
}

func newSession() *session {
	sess := &session{
		softTTL: time.Now().Add(softTimeout),
		hardTTL: time.Now().Add(hardTimeout),
	}
	return sess
}

func (am *sessionManager) add(ppid int, sess *session) {
	am.sessions[ppid] = sess
}

func (am *sessionManager) get(ppid int) (*session, error) {
	sess, ok := am.sessions[ppid]
	if !ok {
		return nil, errors.New("no session")
	}

	if !sess.valid() {
		delete(am.sessions, ppid)
		return nil, errors.New("session expired")
	}

	return sess, nil
}

func (am *sessionManager) middleware(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ppid, ok := r.Context().Value(ppidKey).(int)
		if !ok {
			respondError(w, http.StatusInternalServerError, errors.New("no peer ppid found"))
			return
		}

		sess, err := am.get(ppid)
		if err != nil {
			respondError(w, http.StatusUnauthorized, fmt.Errorf("require authentication: %v", err))
			return
		}

		sess.updateSoftTTL(softTimeout)

		h.ServeHTTP(w, r)
	})

}
