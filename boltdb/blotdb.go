package boltdb

import (
	"errors"

	"github.com/boltdb/bolt"
	session "github.com/liexusong/go-session"
)

type BoltSessionManager struct {
	fd     *bolt.DB
	config session.Config
}

type BoltSession struct {
	manager *BoltSessionManager
	sid     string
}

func init() {
	session.SessionRegisterHandlers(NewSessionManagerHandlers)
}

func NewSessionManagerHandlers(config session.Config) (session.SessionManagerHandlers, error) {
	fd, err := bolt.Open(config.SavePath, 0600, nil)
	if err != nil {
		return nil, err
	}

	manager := &BoltSessionManager{
		fd:     fd,
		config: config,
	}

	return manager, nil
}

func (m *BoltSessionManager) CreateSession(sid string) session.SessionHandlers {
	return &BoltSession{
		manager: m,
		sid:     sid,
	}
}

func (m *BoltSessionManager) SessionGC() {
}

func (s *BoltSession) SessionGet(name string) ([]byte, error) {
	var (
		value []byte
		err   error
	)

	s.manager.fd.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte(s.sid))
		if bucket == nil {
			err = errors.New("not found bucket by session")
			return err
		}

		value = bucket.Get([]byte(name))

		return nil
	})

	return value, err
}

func (s *BoltSession) SessionSet(name string, value []byte) error {
	var err error

	s.manager.fd.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte(s.sid))
		if bucket == nil {
			bucket, err = tx.CreateBucket([]byte(s.sid))
			if err != nil {
				return err
			}
		}

		err = bucket.Put([]byte(name), value)

		return err
	})

	return err
}

func (s *BoltSession) SessionDel(name string) error {
	var err error

	s.manager.fd.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte(s.sid))
		if bucket == nil {
			err = errors.New("not found bucket by session")
			return err
		}

		err = bucket.Delete([]byte(name))

		return err
	})

	return err
}

func (s *BoltSession) SessionDestory() error {
	var err error

	s.manager.fd.Update(func(tx *bolt.Tx) error {
		err = tx.DeleteBucket([]byte(s.sid))
		return err
	})

	return err
}
