package session

import (
	"bytes"
	"crypto/md5"
	"encoding/gob"
	"fmt"
	"math/rand"
	"net/http"
	"time"
)

type Config struct {
	SavePath       string
	SessionName    string
	CookieDomain   string
	CookieLifetime int
	GCProbability  int
	GCDivisor      int
	GCMaxLifetime  int
}

type SessionHandlers interface {
	SessionStart(Config, string) error
	SessionGet(interface{}, interface{}) error
	SessionSet(interface{}, interface{}) error
	SessionDel(interface{}) error
	SessionDestory() error
	SessionGC()
}

type Session struct {
	response  http.ResponseWriter
	request   *http.Request
	sessionId string
	handlers  SessionHandlers
}

var defaultNewSessionHandlersFunc func() SessionHandlers

func SessionRegisterHandlers(newHandlersFunc func() SessionHandlers) {
	defaultNewSessionHandlersFunc = newHandlersFunc
}

func NewSession(w http.ResponseWriter, r *http.Request, config Config) (*Session, error) {
	cookie, err := r.Cookie(config.SessionName)
	if err != nil {
		return nil, err
	}

	sessionId := cookie.Value
	if sessionId == "" {
		sessionId = SessionCreateId()
		http.SetCookie(w, &http.Cookie{
			Name:   config.SessionName,
			Value:  sessionId,
			Domain: config.CookieDomain,
			MaxAge: config.CookieLifetime,
		})
	}

	handlers := defaultNewSessionHandlersFunc()

	err = handlers.SessionStart(config, sessionId)
	if err != nil {
		return nil, err
	}

	ret := &Session{
		response:  w,
		request:   r,
		sessionId: sessionId,
		handlers:  handlers,
	}

	return ret, nil
}

func (s *Session) Get(name interface{}, value interface{}) error {
	return s.handlers.SessionGet(name, value)
}

func (s *Session) Set(name interface{}, value interface{}) error {
	return s.handlers.SessionSet(name, value)
}

func (s *Session) Del(name interface{}) error {
	return s.handlers.SessionDel(name)
}

func (s *Session) Destory() error {
	return s.handlers.SessionDestory()
}

func (s *Session) GC() {
	s.handlers.SessionGC()
}

func SessionEncodeName(name interface{}) string {
	return fmt.Sprintf("%v", name)
}

func SessionEncodeValue(value interface{}) ([]byte, error) {
	writer := &bytes.Buffer{}

	err := gob.NewEncoder(writer).Encode(value)
	if err != nil {
		return nil, err
	}

	return writer.Bytes(), nil
}

func SessionDecodeValue(buffer []byte, value interface{}) error {
	return gob.NewDecoder(bytes.NewReader(buffer)).Decode(value)
}

func SessionCreateId() string {
	randGen := rand.New(rand.NewSource(time.Now().Unix()))

	v1 := randGen.Int63n(9999999999)
	v2 := randGen.Int63n(9999999999)

	randNum := fmt.Sprintf("%d.%d", v1|v2, time.Now().UnixNano())

	return string(md5.New().Sum([]byte(randNum)))
}
