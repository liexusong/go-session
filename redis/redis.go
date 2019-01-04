package redis

import (
	"errors"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/garyburd/redigo/redis"
	session "github.com/liexusong/go-session"
)

type RedisSessionManager struct {
	config         session.Config
	redisConn      redis.Conn
	locker         *sync.RWMutex
	reconnectTimes int64
}

type RedisSession struct {
	sid           string
	manager       *RedisSessionManager
	GCProbability int
	GCDivisor     int
	GCMaxLifetime int
}

// redis connection options
var redisOptions = []redis.DialOption{
	redis.DialConnectTimeout(time.Duration(5) * time.Second),
	redis.DialReadTimeout(time.Duration(5) * time.Second),
	redis.DialWriteTimeout(time.Duration(5) * time.Second),
}

func init() {
	session.SessionRegisterHandlers(NewSessionManagerHandlers)
}

func NewSessionManagerHandlers(config session.Config) (session.SessionManagerHandlers, error) {
	idx := strings.Index(config.SavePath, "://")
	if idx < 0 {
		return nil, errors.New("configure is invalid")
	}

	network := config.SavePath[:idx]
	address := config.SavePath[idx+3:]

	redisConn, err := redis.Dial(network, address, redisOptions...)
	if err != nil {
		return nil, err
	}

	manager := &RedisSessionManager{
		config:    config,
		redisConn: redisConn,
		locker:    new(sync.RWMutex),
	}

	go manager.checkRedisConnectAlive()

	return manager, nil
}

func (m *RedisSessionManager) reconnect() {
	atomic.AddInt64(&m.reconnectTimes, int64(1))

	idx := strings.Index(m.config.SavePath, "://")

	network := m.config.SavePath[:idx]
	address := m.config.SavePath[idx+3:]

	redisConn, err := redis.Dial(network, address, redisOptions...)
	if err != nil {
		return
	}

	m.locker.Lock()

	m.redisConn.Close()
	m.redisConn = redisConn

	m.locker.Unlock()
}

func (m *RedisSessionManager) checkRedisConnectAlive() {
	for {
		m.locker.RLock()

		retries := 0

	tryAgain:
		_, err := redis.String(m.redisConn.Do("PING"))
		if err != nil {
			retries++
			if retries > 10 {
				m.locker.RUnlock()
				m.reconnect()
				goto nextCheck
			}
			goto tryAgain
		}

		m.locker.RUnlock()

	nextCheck:
		time.Sleep(time.Second)
	}
}

func (m *RedisSessionManager) CreateSession(sid string) session.SessionHandlers {
	return &RedisSession{
		sid:           sid,
		manager:       m,
		GCProbability: m.config.GCProbability,
		GCDivisor:     m.config.GCDivisor,
		GCMaxLifetime: m.config.GCMaxLifetime,
	}
}

func (m *RedisSessionManager) GetReconnects() int64 {
	return atomic.LoadInt64(&m.reconnectTimes)
}

func (m *RedisSessionManager) doCommand(cmd string, args ...interface{}) (interface{}, error) {
	m.locker.RLock()
	defer m.locker.RUnlock()

	return m.redisConn.Do(cmd, args...)
}

func (s *RedisSession) updateSessionGCMaxLifetime() error {
	_, err := s.manager.doCommand("EXPIRE", s.sid, s.GCMaxLifetime)
	return err
}

func (s *RedisSession) SessionGet(name string) ([]byte, error) {
	buffer, err := redis.Bytes(s.manager.doCommand("HGET", s.sid, name))
	if err != nil {
		return nil, err
	}

	_ = s.updateSessionGCMaxLifetime()

	return buffer, nil
}

func (s *RedisSession) SessionSet(name string, value []byte) error {
	_, err := s.manager.doCommand("HSET", s.sid, name, value)
	if err != nil {
		return err
	}

	_ = s.updateSessionGCMaxLifetime()

	return nil
}

func (s *RedisSession) SessionDel(name string) error {
	_, err := s.manager.doCommand("HDEL", s.sid, name)
	if err != nil {
		return err
	}

	_ = s.updateSessionGCMaxLifetime()

	return nil
}

func (s *RedisSession) SessionDestory() error {
	_, err := s.manager.doCommand("DEL", s.sid)
	return err
}

func (s *RedisSession) SessionGC() {
}
