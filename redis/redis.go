package redis

import (
	"errors"
	"strings"

	"github.com/garyburd/redigo/redis"
	session "github.com/liexusong/go-session"
)

type RedisSession struct {
	conn          redis.Conn
	isActive      bool
	sid           string
	GCProbability int
	GCDivisor     int
	GCMaxLifetime int
}

func init() {
	session.SessionRegisterHandlers(NewSessionHandlers)
}

func NewSessionHandlers() session.SessionHandlers {
	return &RedisSession{}
}

func (s *RedisSession) SessionStart(config session.Config, sid string) error {
	if s.isActive {
		return nil
	}

	idx := strings.Index(config.SavePath, "://")
	if idx < 0 {
		return errors.New("configure is invalid")
	}

	var redisConn redis.Conn

	network := config.SavePath[:idx]
	address := config.SavePath[idx+3:]

	redisConn, err := redis.Dial(network, address)
	if err != nil {
		return err
	}

	s.conn = redisConn
	s.isActive = true
	s.sid = sid
	s.GCMaxLifetime = config.GCMaxLifetime
	s.GCProbability = config.GCProbability
	s.GCDivisor = config.GCDivisor

	return nil
}

func (s *RedisSession) updateSessionGCMaxLifetime() error {
	_, err := s.conn.Do("EXPIRE", s.sid, s.GCMaxLifetime)
	return err
}

func (s *RedisSession) SessionGet(name string) ([]byte, error) {
	buffer, err := redis.Bytes(s.conn.Do("HGET", s.sid, name))
	if err != nil {
		return nil, err
	}

	_ = s.updateSessionGCMaxLifetime()

	return buffer, nil
}

func (s *RedisSession) SessionSet(name string, value []byte) error {
	_, err := s.conn.Do("HSET", s.sid, name, value)
	if err != nil {
		return err
	}

	_ = s.updateSessionGCMaxLifetime()

	return nil
}

func (s *RedisSession) SessionDel(name string) error {
	_, err := s.conn.Do("HDEL", s.sid, name)
	if err != nil {
		return err
	}

	_ = s.updateSessionGCMaxLifetime()

	return nil
}

func (s *RedisSession) SessionDestory() error {
	if !s.isActive {
		return nil
	}

	_, err := s.conn.Do("DEL", s.sid)
	if err != nil {
		return err
	}

	err = s.conn.Close()
	if err != nil {
		return err
	}

	s.conn = nil
	s.isActive = false

	return nil
}

func (s *RedisSession) SessionGC() {
}
