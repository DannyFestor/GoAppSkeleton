package sessions

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
)

const cookieName = "session"

var errTampered = errors.New("session cookie signature mismatch")

type CookieStore struct {
	key []byte
}

func NewCookieStore(key []byte) *CookieStore {
	return &CookieStore{key: key}
}

func (s *CookieStore) Load(r *http.Request) (Session, error) {
	cookie, err := r.Cookie(cookieName)
	if err != nil {
		return Session{}, nil
	}
	payload, signature, _ := strings.Cut(cookie.Value, ".")
	if !hmac.Equal([]byte(signature), []byte(s.sign(payload))) {
		return nil, errTampered
	}
	raw, err := base64.RawURLEncoding.DecodeString(payload)
	if err != nil {
		return nil, err
	}
	var session Session
	return session, json.Unmarshal(raw, &session)
}

func (s *CookieStore) Save(w http.ResponseWriter, _ *http.Request, session Session) error {
	raw, err := json.Marshal(session)
	if err != nil {
		return err
	}
	payload := base64.RawURLEncoding.EncodeToString(raw)
	http.SetCookie(w, &http.Cookie{
		Name:     cookieName,
		Value:    payload + "." + s.sign(payload),
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})
	return nil
}

func (s *CookieStore) Destroy(w http.ResponseWriter, _ *http.Request) error {
	http.SetCookie(w, &http.Cookie{Name: cookieName, Path: "/", MaxAge: -1})
	return nil
}

func (s *CookieStore) sign(payload string) string {
	mac := hmac.New(sha256.New, s.key)
	mac.Write([]byte(payload))
	return base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
}
