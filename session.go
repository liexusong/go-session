// Copyright 2019 liexusong. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

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
	SessionGet(string) ([]byte, error)
	SessionSet(string, []byte) error
	SessionDel(string) error
	SessionDestory() error
	SessionGC()
}

type Session struct {
	response http.ResponseWriter
	request  *http.Request
	sid      string
	handlers SessionHandlers
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

	sid := cookie.Value
	if sid == "" {
		sid = createSid()
		http.SetCookie(w, &http.Cookie{
			Name:   config.SessionName,
			Value:  sid,
			Domain: config.CookieDomain,
			MaxAge: config.CookieLifetime,
		})
	}

	handlers := defaultNewSessionHandlersFunc()

	err = handlers.SessionStart(config, sid)
	if err != nil {
		return nil, err
	}

	ret := &Session{
		response: w,
		request:  r,
		sid:      sid,
		handlers: handlers,
	}

	return ret, nil
}

func (s *Session) Get(name interface{}, value interface{}) error {
	buffer, err := s.handlers.SessionGet(encodeName(name))
	if err != nil {
		return err
	}

	return decodeValue(buffer, value)
}

func (s *Session) Set(name interface{}, value interface{}) error {
	realValue, err := encodeValue(value)
	if err != nil {
		return err
	}

	return s.handlers.SessionSet(encodeName(name), realValue)
}

func (s *Session) Del(name interface{}) error {
	return s.handlers.SessionDel(encodeName(name))
}

func (s *Session) Destory() error {
	return s.handlers.SessionDestory()
}

func (s *Session) GC() {
	s.handlers.SessionGC()
}

func (s *Session) GetSid() string {
	return s.sid
}

func encodeName(name interface{}) string {
	return fmt.Sprintf("%v", name)
}

func encodeValue(value interface{}) ([]byte, error) {
	writer := &bytes.Buffer{}

	err := gob.NewEncoder(writer).Encode(value)
	if err != nil {
		return nil, err
	}

	return writer.Bytes(), nil
}

func decodeValue(buffer []byte, value interface{}) error {
	return gob.NewDecoder(bytes.NewReader(buffer)).Decode(value)
}

func createSid() string {
	randGen := rand.New(rand.NewSource(time.Now().Unix()))

	v1 := randGen.Int63n(9999999999)
	v2 := randGen.Int63n(9999999999)

	randNum := fmt.Sprintf("%d.%d", v1|v2, time.Now().UnixNano())

	return string(md5.New().Sum([]byte(randNum)))
}
