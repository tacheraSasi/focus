// Package store connects to the data store and manages timers and sessions
package store

import (
	"bytes"
	"encoding/json"
	"errors"
	"io/fs"
	"slices"
	"time"

	bolt "go.etcd.io/bbolt"
	bolterr "go.etcd.io/bbolt/errors"

	"github.com/ayoisaiah/focus/internal/config"
	"github.com/ayoisaiah/focus/internal/models"
	"github.com/ayoisaiah/focus/internal/timeutil"
)

// Client is a BoltDB database client.
type Client struct {
	*bolt.DB
}

const (
	sessionBucket = "sessions"
	focusBucket   = "focus"
)

var errFocusRunning = errors.New(
	"is Focus already running? Only one instance can be active at a time",
)

func (c *Client) UpdateSessions(sessions map[time.Time]*models.Session) error {
	for k, v := range sessions {
		key := timeutil.ToKey(k)

		b, err := json.Marshal(v)
		if err != nil {
			return err
		}

		return c.Update(func(tx *bolt.Tx) error {
			return tx.Bucket([]byte(sessionBucket)).Put(key, b)
		})
	}

	return nil
}

func (c *Client) DeleteSessions(startTimes []time.Time) error {
	return c.Update(func(tx *bolt.Tx) error {
		for i := range startTimes {
			key := timeutil.ToKey(startTimes[i])

			err := tx.Bucket([]byte(sessionBucket)).Delete(key)
			if err != nil {
				return err
			}
		}

		return nil
	})
}

func (c *Client) Open() error {
	db, err := openDB(config.DBFilePath())
	if err != nil {
		return err
	}

	*c = Client{
		db,
	}

	return nil
}

func (c *Client) GetSessions(
	since, until time.Time,
	tags []string,
) ([]*models.Session, error) {
	var result []*models.Session

	err := c.View(func(tx *bolt.Tx) error {
		c := tx.Bucket([]byte(sessionBucket)).Cursor()
		min := []byte(since.Format(time.RFC3339))
		max := []byte(until.Format(time.RFC3339))

		//nolint:ineffassign,staticcheck // due to how boltdb works
		sk, sv := c.Seek(min)
		// get the previous session so as to check if
		// it was ended within the specified time bounds
		pk, pv := c.Prev()
		if pk != nil {
			var sess models.Session

			err := json.Unmarshal(pv, &sess)
			if err != nil {
				return err
			}

			// include session in results if it was ended
			// in the bounds of the specified time period
			if sess.EndTime.After(since) {
				sk, sv = pk, pv
			} else {
				sk, sv = c.Next()
			}
		} else {
			sk, sv = c.Seek(min)
		}

		for k, v := sk, sv; k != nil && bytes.Compare(k, max) <= 0; k, v = c.Next() {
			var sess models.Session

			err := json.Unmarshal(v, &sess)
			if err != nil {
				return err
			}

			// Filter out tags that don't match
			if len(tags) != 0 {
				for _, t := range sess.Tags {
					if slices.Contains(tags, t) {
						result = append(result, &sess)
					}
				}
			} else {
				result = append(result, &sess)
			}
		}

		return nil
	})

	return result, err
}

// openDB creates or opens a database.
func openDB(dbFilePath string) (*bolt.DB, error) {
	var fileMode fs.FileMode = 0o600

	db, err := bolt.Open(
		dbFilePath,
		fileMode,
		&bolt.Options{Timeout: 1 * time.Second},
	)

	if err != nil && errors.Is(err, bolterr.ErrTimeout) {
		return nil, errFocusRunning
	}

	return db, nil
}

// NewClient returns a wrapper to a BoltDB connection.
func NewClient(dbFilePath string) (*Client, error) {
	db, err := openDB(dbFilePath)
	if err != nil {
		return nil, err
	}
	// Create the necessary buckets for storing data if they do not exist already
	err = db.Update(func(tx *bolt.Tx) error {
		_, err = tx.CreateBucketIfNotExists([]byte(sessionBucket))
		if err != nil {
			return err
		}

		_, err = tx.CreateBucketIfNotExists([]byte(focusBucket))
		if err != nil {
			return err
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	c := &Client{
		db,
	}

	err = db.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte(focusBucket))
		version := string(bucket.Get([]byte("version")))

		// prior to v1.4.0, no version info was stored in the database
		// if upgrading from earlier version, run migrations to meet
		// current database format
		// Does nothing for new users
		if version == "" {
			err = c.migrateV1_4_0(tx)
			if err != nil {
				return err
			}
		}

		if version != config.Version {
			return bucket.Put([]byte("version"), []byte(config.Version))
		}

		return nil
	})

	return c, err
}
