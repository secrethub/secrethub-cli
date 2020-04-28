package agent

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"github.com/go-chi/chi"
)

type controller struct {
	sm *sessionManager
}

func newController() *controller {
	return &controller{
		sm: newSessionManager(),
	}
}

func (c *controller) handler() http.Handler {
	r := chi.NewRouter()

	r.Get("/ping", c.Ping)
	r.Post("/unlock", c.Unlock)

	r.Group(func(r chi.Router) {
		r.Use(c.sm.middleware)

		r.Get("/fingerprint", c.Fingerprint)
		r.Post("/decrypt", c.Decrypt)
		r.Post("/lock", c.Lock)
		r.Post("/sign", c.Sign)
	})

	return r
}

func (c *controller) Unlock(w http.ResponseWriter, r *http.Request) {
	var unlockReq UnlockRequest
	err := json.NewDecoder(r.Body).Decode(&unlockReq)
	if err != nil {
		respondError(w, http.StatusBadRequest, err)
		return
	}

	ppid, ok := r.Context().Value(ppidKey).(int)
	if !ok {
		respondError(w, http.StatusInternalServerError, errors.New("no peer ppid found"))
		return
	}
	session, err := c.sm.get(ppid)
	if err != nil {
		session = newSession()
		if err := session.unlock(unlockReq.Passphrase); err != nil {
			respondError(w, http.StatusForbidden, err)
			return
		}
		c.sm.add(ppid, session)
	}

	session.updateSoftTTL(softTimeout)

	respondJSON(w, http.StatusOK, nil)
}

func (c *controller) Ping(_ http.ResponseWriter, _ *http.Request) {
	return
}

func (c *controller) Lock(w http.ResponseWriter, r *http.Request) {
	c.withSession(w, r, func(w http.ResponseWriter, r *http.Request, sess *session) {
		sess.credential = nil
		respondJSON(w, http.StatusOK, nil)
	})
}

func (c *controller) Fingerprint(w http.ResponseWriter, r *http.Request) {
	c.withSession(w, r, func(w http.ResponseWriter, r *http.Request, sess *session) {
		fingerprint, err := sess.credential.ID()
		if err != nil {
			respondError(w, http.StatusInternalServerError, err)
			return
		}
		respondJSON(w, http.StatusOK, FingerprintResponse{
			Fingerprint: fingerprint,
		})
	})
}

func (c *controller) Sign(w http.ResponseWriter, r *http.Request) {
	c.withSession(w, r, func(w http.ResponseWriter, r *http.Request, sess *session) {
		var signReq SignRequest
		err := json.NewDecoder(r.Body).Decode(&signReq)
		if err != nil {
			respondError(w, http.StatusBadRequest, err)
			return
		}
		signature, err := sess.credential.Sign(signReq.Payload)
		if err != nil {
			respondError(w, http.StatusInternalServerError, err)
			return
		}
		respondJSON(w, http.StatusOK, SignResponse{
			Signature: signature,
		})
	})
}

func (c *controller) Decrypt(w http.ResponseWriter, r *http.Request) {
	c.withSession(w, r, func(w http.ResponseWriter, r *http.Request, sess *session) {
		var signReq DecryptRequest
		err := json.NewDecoder(r.Body).Decode(&signReq)
		if err != nil {
			respondError(w, http.StatusBadRequest, err)
			return
		}
		decrypted, err := sess.credential.Unwrap(&signReq.EncryptedData)
		if err != nil {
			respondError(w, http.StatusInternalServerError, err)
			return
		}
		respondJSON(w, http.StatusOK, DecryptResponse{
			Decrypted: decrypted,
		})
	})
}

func (c *controller) withSession(w http.ResponseWriter, r *http.Request, f func(w http.ResponseWriter, r *http.Request, sess *session)) {
	sess, err := c.getReqSession(r)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err)
		return
	}
	f(w, r, sess)
}

func (c *controller) getReqSession(r *http.Request) (*session, error) {
	ppid, ok := r.Context().Value(ppidKey).(int)
	if !ok {
		return nil, errors.New("no peer ppid found")
	}
	return c.sm.get(ppid)
}

func respondJSON(w http.ResponseWriter, code int, data interface{}) {
	w.Header().Set("Content-Type", "text/javascript")
	js, err := json.Marshal(data)
	if err != nil {
		//BadRequestError(w, fmt.Sprintf("JSON serialization error: %v", err))
		return
	}
	w.Header().Set("Content-Length", strconv.Itoa(len(js)+1))
	w.WriteHeader(code)
	w.Write(js)
	w.Write([]byte("\n"))
}

func respondError(w http.ResponseWriter, code int, err error) {
	respondJSON(w, code, ErrorResponse{
		Error: err.Error(),
	})
}
