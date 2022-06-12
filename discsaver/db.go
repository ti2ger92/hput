package discsaver

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"hput"
	"net/url"

	bolt "go.etcd.io/bbolt"
)

var bucketName = []byte("hput")

// Logger logs out.
type Logger interface {
	Debug(msg string)
	Debugf(msg string, args ...interface{})
	// Warnf(msg string, args ...interface{})
	Errorf(msg string, args ...interface{})
}

// Saver can save save and retrieve for hput
type Saver struct {
	Db     *bolt.DB
	Logger Logger
	Path   string
}

// New create a new saver
func New(l Logger, f string) (*Saver, error) {
	db, err := bolt.Open(f, 0600, nil)
	if err != nil {
		l.Errorf("discsaver.New(): Could not create database %+v", err)
		return nil, err
	}
	l.Debugf("discsaver.New():created db: %+v", db)
	err = db.Update(func(tx *bolt.Tx) error {
		_, err := tx.CreateBucketIfNotExists(bucketName)
		if err != nil {
			return fmt.Errorf("create bucket: %s", err)
		}
		return nil
	})
	if err != nil {
		l.Errorf("discsaver.New():Could not create bucket %+v", err)
		db.Close()
		return nil, err
	}
	return &Saver{
		Db:     db,
		Logger: l,
	}, nil
}

// Shutdown gracefully clean this up
// TODO: invoke this on graceful shutdown
func (s *Saver) Shutdown() {
	s.Db.Close()
}

// SaveText saves a text value to a path
func (sa *Saver) SaveText(_ context.Context, s string, p url.URL, r *hput.PutResult) error {
	ru := hput.Runnable{
		Type: hput.Text,
		Text: s,
	}
	return sa.saveRunnable(ru, p, r)
}

func (sa *Saver) SaveCode(_ context.Context, s string, p url.URL, r *hput.PutResult) error {
	ru := hput.Runnable{
		Type: hput.Js,
		Text: s,
	}
	return sa.saveRunnable(ru, p, r)
}

// SaveBinary saves a binary value to a path
func (sa *Saver) SaveBinary(_ context.Context, b []byte, p url.URL, r *hput.PutResult) error {
	ru := hput.Runnable{
		Type:   hput.Binary,
		Binary: b,
	}
	return sa.saveRunnable(ru, p, r)
}

// saveRunnable saves a runnable and reports if the runnable was replaced
func (sa *Saver) saveRunnable(ru hput.Runnable, p url.URL, r *hput.PutResult) error {
	sa.Logger.Debugf("discsaver.saveRunnable(): retrieving runnable %+v", ru)
	v, err := json.Marshal(ru)
	if err != nil {
		sa.Logger.Errorf("discsaver.saveRunnable(): could not prepare saved record: %v", err)
		return err
	}
	err = sa.Db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket(bucketName)
		existing := b.Get([]byte(p.Path))
		if len(existing) > 0 {
			r.Overwrote = true
		}
		err = b.Put([]byte(p.Path), v)
		if err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		sa.Logger.Errorf("discsaver.saveRunnable(): error saving text to database %s", err)
		return fmt.Errorf("error saving text to database %w", err)
	}
	return nil
}

// GetRunnable returns the runnable from a path
func (sa *Saver) GetRunnable(_ context.Context, p url.URL) (hput.Runnable, error) {
	var runnableBytes []byte
	sa.Logger.Debugf("discsaver.GetRunnable(): retrieving runnable at url %+v", p)
	err := sa.Db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket(bucketName)
		runnableBytes = b.Get([]byte(p.Path))
		return nil
	})
	if err != nil {
		sa.Logger.Errorf("discsaver.GetRunnable(): error retrieving runnable from database %v", err)
		return hput.Runnable{}, fmt.Errorf("error retrieving runnable from database: %w", err)
	}
	if len(runnableBytes) == 0 {
		sa.Logger.Debug("discsaver.GetRunnable(): got no runnable")
		return hput.Runnable{}, nil
	}
	runnable := &hput.Runnable{}
	err = json.Unmarshal(runnableBytes, runnable)
	if err != nil {
		sa.Logger.Errorf("discsaver.GetRunnable(): error unmarshaling runnable from database %v", err)
		return hput.Runnable{}, fmt.Errorf("error unmarshaling runnable from database %w", err)
	}
	runnable.Path = p.Path
	sa.Logger.Debugf("discsaver.GetRunnable(): returning runnable %+v", *runnable)
	return *runnable, nil
}

// SendRunnables returns all runnables from the database
func (sa *Saver) SendRunnables(_ context.Context, p string, runnables chan<- hput.Runnable, done chan<- bool) error {
	defer func() { done <- true }()
	err := sa.Db.View(func(tx *bolt.Tx) error {
		c := tx.Bucket(bucketName).Cursor()

		prefix := []byte(p)
		for k, v := c.Seek(prefix); k != nil && bytes.HasPrefix(k, prefix); k, v = c.Next() {
			runnable := &hput.Runnable{}
			err := json.Unmarshal(v, runnable)
			if err != nil {
				sa.Logger.Errorf("discsaver.SendRunnables(): error marshaling runnable from database scan %v", err)
				return fmt.Errorf("error marshaling runnable from database scan %w", err)
			}
			runnable.Path = string(k)
			runnables <- *runnable
		}
		return nil
	})
	if err != nil {
		sa.Logger.Errorf("discsaver.SendRunnables() could not iterate through runnables: %+v", err)
		return fmt.Errorf("could not iterate through runnables: %w", err)
	}
	return nil
}
