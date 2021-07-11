package focus

import (
	"bytes"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"time"

	"github.com/pterm/pterm"
	bolt "go.etcd.io/bbolt"
)

const (
	dbFile = "focus.db"
)

var store Store

type Store struct {
	conn *bolt.DB
}

func init() {
	err := store.init()
	if err != nil {
		if errors.Is(err, bolt.ErrDatabaseOpen) || errors.Is(err, bolt.ErrTimeout) {
			pterm.Error.Println("Only one instance of Focus can be active at a time")
		} else {
			pterm.Error.Println(err)
		}

		os.Exit(1)
	}
}

func (s *Store) init() error {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return err
	}

	appRoot := filepath.Join(homeDir, configPath)
	pathToDB := filepath.Join(appRoot, dbFile)

	var fileMode fs.FileMode = 0600

	db, err := bolt.Open(pathToDB, fileMode, &bolt.Options{Timeout: 1 * time.Second})
	if err != nil {
		return err
	}

	s.conn = db

	err = s.conn.Update(func(tx *bolt.Tx) error {
		_, err = tx.CreateBucketIfNotExists([]byte("sessions"))
		if err != nil {
			return err
		}

		_, err = tx.CreateBucketIfNotExists([]byte("timer"))
		return err
	})

	return err
}

func (s *Store) updateSession(key, value []byte) error {
	err := s.conn.Update(func(tx *bolt.Tx) error {
		return tx.Bucket([]byte("sessions")).Put(key, value)
	})

	return err
}

// saveTimerState persists the current timer settings,
// and the key of the paused session to the database.
func (s *Store) saveTimerState(timer, sessionKey []byte) error {
	err := s.conn.Update(func(tx *bolt.Tx) error {
		err := tx.Bucket([]byte("timer")).Put([]byte("timer"), timer)
		if err != nil {
			return err
		}

		return tx.Bucket([]byte("timer")).Put([]byte("paused_session_key"), sessionKey)
	})

	return err
}

// getTimerState retrieves the stored timer and session key.
func (s *Store) getTimerState() (timer, session []byte, err error) {
	err = s.conn.View(func(tx *bolt.Tx) error {
		timer = tx.Bucket([]byte("timer")).Get([]byte("timer"))

		sessionKey := tx.Bucket([]byte("timer")).Get([]byte("paused_session_key"))

		session = tx.Bucket([]byte("sessions")).Get(sessionKey)

		return nil
	})

	return timer, session, err
}

// deleteTimerState removes the stored timer and session key.
func (s *Store) deleteTimerState() error {
	return s.conn.Update(func(tx *bolt.Tx) error {
		err := tx.Bucket([]byte("timer")).Delete([]byte("timer"))
		if err != nil {
			return err
		}

		return tx.Bucket([]byte("timer")).Delete([]byte("paused_session_key"))
	})
}

// getSessions retrieves the saved pomodoro sessions
// within the specified time period.
func (s *Store) getSessions(startTime, endTime time.Time) ([][]byte, error) {
	var b [][]byte

	err := s.conn.View(func(tx *bolt.Tx) error {
		c := tx.Bucket([]byte("sessions")).Cursor()
		min := []byte(startTime.Format(time.RFC3339))
		max := []byte(endTime.Format(time.RFC3339))

		for k, v := c.Seek(min); k != nil && bytes.Compare(k, max) <= 0; k, v = c.Next() {
			b = append(b, v)
		}

		return nil
	})

	return b, err
}
