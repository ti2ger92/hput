package discsaver

import (
	"context"
	"encoding/json"
	"hput"
	"net/url"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.etcd.io/bbolt"
)

type TestLogger struct{}

func (t *TestLogger) Debugf(msg string, args ...interface{}) {}

func (t *TestLogger) Debug(msg string) {}

func (t *TestLogger) Errorf(msg string, args ...interface{}) {}

// Test_SaveText verifies that we can save text
func Test_SaveText(t *testing.T) {
	tt := []struct {
		name     string
		s        string
		p        url.URL
		res      hput.PutResult
		runnable *hput.Runnable
	}{
		{
			name: "save basic string",
			s:    "saved text",
			p:    url.URL{Path: "/pth"},
			runnable: &hput.Runnable{
				Type: hput.Text,
				Text: "saved text",
			},
		},
		{
			name: "save empty string",
			s:    "",
			p:    url.URL{Path: "/pth"},
			runnable: &hput.Runnable{
				Type: hput.Text,
				Text: "",
			},
		},
		{
			name: "save multi-line string",
			s:    "line1\nline2\nline3",
			p:    url.URL{Path: "/pth"},
			runnable: &hput.Runnable{
				Type: hput.Text,
				Text: "line1\nline2\nline3",
			},
		},
	}
	for _, test := range tt {
		t.Run(test.name, func(t *testing.T) {
			os.Remove("unit_test.db")
			defer os.Remove("unit_test.db")
			sa, err := New(&TestLogger{}, "unit_test.db")
			defer sa.Shutdown()
			assert.NoError(t, err)
			r := &hput.PutResult{}
			sa.SaveText(context.Background(), test.s, test.p, r)
			assert.Equal(t, test.res, *r)
			var foundBytes []byte
			sa.Db.View(func(tx *bbolt.Tx) error {
				bu := tx.Bucket(bucketName)
				foundBytes = bu.Get([]byte(test.p.Path))
				return nil
			})
			foundRunnable := &hput.Runnable{}
			err = json.Unmarshal(foundBytes, foundRunnable)
			assert.NoError(t, err)
			assert.Equal(t, test.runnable, foundRunnable)
		})
	}
}

func Test_SaveCode(t *testing.T) {
	tt := []struct {
		name     string
		c        string
		p        url.URL
		res      hput.PutResult
		runnable *hput.Runnable
	}{
		{
			name: "save basic code",
			c:    "var a=1",
			p:    url.URL{Path: "/pth"},
			runnable: &hput.Runnable{
				Type: hput.Js,
				Text: "var a=1",
			},
		},
		{
			name: "save empty string",
			c:    "",
			p:    url.URL{Path: "/pth"},
			runnable: &hput.Runnable{
				Type: hput.Js,
				Text: "",
			},
		},
		{
			name: "save multi-line string",
			c:    "var a=1\nvar b=a\nb",
			p:    url.URL{Path: "/pth"},
			runnable: &hput.Runnable{
				Type: hput.Js,
				Text: "var a=1\nvar b=a\nb",
			},
		},
	}
	for _, test := range tt {
		t.Run(test.name, func(t *testing.T) {
			os.Remove("unit_test.db")
			defer os.Remove("unit_test.db")
			sa, err := New(&TestLogger{}, "unit_test.db")
			defer sa.Shutdown()
			assert.NoError(t, err)
			r := &hput.PutResult{}
			sa.SaveCode(context.Background(), test.c, test.p, r)
			assert.Equal(t, test.res, *r)
			var foundBytes []byte
			sa.Db.View(func(tx *bbolt.Tx) error {
				bu := tx.Bucket(bucketName)
				foundBytes = bu.Get([]byte(test.p.Path))
				return nil
			})
			foundRunnable := &hput.Runnable{}
			err = json.Unmarshal(foundBytes, foundRunnable)
			assert.NoError(t, err)
			assert.Equal(t, test.runnable, foundRunnable)
		})
	}
}

// Test_SaveBinary verify we can save a binary file
func Test_SaveBinary(t *testing.T) {
	tt := []struct {
		name     string
		b        []byte
		p        url.URL
		res      hput.PutResult
		runnable *hput.Runnable
	}{
		{
			name: "save basic binary",
			b:    []byte{255, 0, 155},
			p:    url.URL{Path: "/pth"},
			runnable: &hput.Runnable{
				Type:   hput.Binary,
				Binary: []byte{255, 0, 155},
			},
		},
		{
			name: "save empty binary",
			p:    url.URL{Path: "/pth"},
			runnable: &hput.Runnable{
				Type: hput.Binary,
			},
		},
	}
	for _, test := range tt {
		t.Run(test.name, func(t *testing.T) {
			os.Remove("unit_test.db")
			defer os.Remove("unit_test.db")
			sa, err := New(&TestLogger{}, "unit_test.db")
			defer sa.Shutdown()
			assert.NoError(t, err)
			r := &hput.PutResult{}
			sa.SaveBinary(context.Background(), test.b, test.p, r)
			assert.Equal(t, test.res, *r)
			var foundBytes []byte
			sa.Db.View(func(tx *bbolt.Tx) error {
				bu := tx.Bucket(bucketName)
				foundBytes = bu.Get([]byte(test.p.Path))
				return nil
			})
			foundRunnable := &hput.Runnable{}
			err = json.Unmarshal(foundBytes, foundRunnable)
			assert.NoError(t, err)
			assert.Equal(t, test.runnable, foundRunnable)
		})
	}
}

func Test_GetRunnable(t *testing.T) {
	tt := []struct {
		name        string
		p           url.URL
		dbRunnable  hput.Runnable
		expRunnable hput.Runnable
	}{
		{
			name: "get text runnable",
			p:    url.URL{Path: "/pth"},
			dbRunnable: hput.Runnable{
				Type: hput.Text,
				Text: "sample text",
			},
			expRunnable: hput.Runnable{
				Path: "/pth",
				Type: hput.Text,
				Text: "sample text",
			},
		},
		{
			name: "get binary runnable",
			p:    url.URL{Path: "/pth"},
			dbRunnable: hput.Runnable{
				Type:   hput.Binary,
				Binary: []byte{255, 255, 0, 0, 255, 0, 1},
			},
			expRunnable: hput.Runnable{
				Path:   "/pth",
				Type:   hput.Binary,
				Binary: []byte{255, 255, 0, 0, 255, 0, 1},
			},
		},
		{
			name: "get js runnable",
			p:    url.URL{Path: "/pth/complex/many/parts"},
			dbRunnable: hput.Runnable{
				Type: hput.Js,
				Text: "var a = 1",
			},
			expRunnable: hput.Runnable{
				Path: "/pth/complex/many/parts",
				Type: hput.Js,
				Text: "var a = 1",
			},
		},
	}
	for _, test := range tt {
		t.Run(test.name, func(t *testing.T) {
			os.Remove("unit_test.db")
			defer os.Remove("unit_test.db")
			sa, err := New(&TestLogger{}, "unit_test.db")
			dbBytes, err := json.Marshal(test.dbRunnable)
			assert.NoError(t, err)
			sa.Db.Update(func(tx *bbolt.Tx) error {
				b := tx.Bucket(bucketName)
				err := b.Put([]byte(test.p.Path), dbBytes)
				assert.NoError(t, err)
				return err
			})
			foundRunnable, err := sa.GetRunnable(context.Background(), test.p)
			assert.NoError(t, err)
			assert.Equal(t, test.expRunnable, foundRunnable)
		})
	}
}

// Test_SendRunnables verify we can send runnables as a to a channel
func Test_SendRunnables(t *testing.T) {
	tt := []struct {
		name         string
		dbRunnables  map[string]hput.Runnable
		p            string // path to gather runnables from
		expRunnables map[string]hput.Runnable
	}{
		{
			name: "Return a single text runnable",
			dbRunnables: map[string]hput.Runnable{
				"/pth": {
					Type: hput.Text,
					Text: "Some Text",
				},
			},
			expRunnables: map[string]hput.Runnable{
				"/pth": {
					Path: "/pth",
					Type: hput.Text,
					Text: "Some Text",
				},
			},
			p: "/pth",
		},
		{
			name: "Return a single empty runnable",
			dbRunnables: map[string]hput.Runnable{
				"/pth": {},
			},
			expRunnables: map[string]hput.Runnable{
				"/pth": {
					Path: "/pth",
				},
			},
			p: "/pth",
		},
		{
			name: "Return a 3 different runnables",
			dbRunnables: map[string]hput.Runnable{
				"/some/pth": {
					Type: hput.Text,
					Text: "Some Text",
				},
				"/some/other/pth": {
					Type: hput.Js,
					Text: "var a=1",
				},
				"/some/binary/pth": {
					Type:   hput.Binary,
					Binary: []byte{255, 255, 0},
				},
			},
			expRunnables: map[string]hput.Runnable{
				"/some/pth": {
					Path: "/some/pth",
					Type: hput.Text,
					Text: "Some Text",
				},
				"/some/other/pth": {
					Path: "/some/other/pth",
					Type: hput.Js,
					Text: "var a=1",
				},
				"/some/binary/pth": {
					Path:   "/some/binary/pth",
					Type:   hput.Binary,
					Binary: []byte{255, 255, 0},
				},
			},
			p: "/some",
		},
	}
	for _, test := range tt {
		t.Run(test.name, func(t *testing.T) {
			os.Remove("unit_test.db")
			defer os.Remove("unit_test.db")
			sa, err := New(&TestLogger{}, "unit_test.db")
			assert.NoError(t, err)
			// setup the database with elements
			for key, dbRunnable := range test.dbRunnables {
				dbBytes, err := json.Marshal(dbRunnable)
				assert.NoError(t, err)
				sa.Db.Update(func(tx *bbolt.Tx) error {
					b := tx.Bucket(bucketName)
					err := b.Put([]byte(key), dbBytes)
					assert.NoError(t, err)
					return err
				})
			}

			runnablesChan := make(chan hput.Runnable, 1)
			doneChan := make(chan bool, 1)
			go func() {
				sa.SendRunnables(context.Background(), test.p, runnablesChan, doneChan)
			}()
			var done bool
			for done == false {
				select {
				case run := <-runnablesChan:
					val, ok := test.expRunnables[run.Path]
					assert.True(t, ok, "runnable %+v not found", run)
					assert.Equal(t, val, run)
					delete(test.dbRunnables, run.Path)
				case <-doneChan:
					done = true
				}
			}
			assert.Empty(t, test.dbRunnables)
		})
	}
}
