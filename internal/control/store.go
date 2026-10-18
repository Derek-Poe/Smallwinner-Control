package control

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
	"syscall"
)

var ErrConflict = errors.New("configuration revision conflict")

// Store owns one Linux/WSL local directory. Flock prevents multiple cooperating
// processes from replacing config.json; the mutex serializes readers and CAS.
type Store struct {
	mu     sync.Mutex
	dir    string
	lock   *os.File
	config Config
	write  func(string, []byte) error
}

func OpenStore(dir string) (*Store, error) {
	if err := os.MkdirAll(dir, 0700); err != nil {
		return nil, err
	}
	f, err := os.OpenFile(filepath.Join(dir, ".lock"), os.O_CREATE|os.O_RDWR, 0600)
	if err != nil {
		return nil, err
	}
	if err = syscall.Flock(int(f.Fd()), syscall.LOCK_EX|syscall.LOCK_NB); err != nil {
		f.Close()
		return nil, errors.New("state directory is already owned or does not support locking")
	}
	s := &Store{dir: dir, lock: f, config: DefaultConfig(), write: atomicWrite}
	fail := func(e error) (*Store, error) { s.Close(); return nil, e }
	configFile, err := os.Open(filepath.Join(dir, "config.json"))
	var b []byte
	if err == nil {
		b, err = io.ReadAll(io.LimitReader(configFile, 16385))
		_ = configFile.Close()
	}
	if errors.Is(err, os.ErrNotExist) {
		b, _ = json.MarshalIndent(s.config, "", "  ")
		if err := s.write(dir, b); err != nil {
			return fail(err)
		}
	} else if err != nil {
		return fail(err)
	} else {
		if len(b) > 16384 {
			return fail(errors.New("configuration too large"))
		}
		s.config = Config{}
		if err := DecodeJSON(bytes.NewReader(b), &s.config); err != nil {
			return fail(errors.New("invalid persisted configuration JSON"))
		}
		if err := s.config.Validate(); err != nil {
			return fail(err)
		}
	}
	return s, nil
}
func (s *Store) Snapshot() Config { s.mu.Lock(); defer s.mu.Unlock(); return cloneConfig(s.config) }
func (s *Store) Update(c Config) (Config, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.lock == nil {
		return Config{}, errors.New("store closed")
	}
	if err := c.Validate(); err != nil {
		return Config{}, err
	}
	if c.Revision != s.config.Revision {
		return Config{}, ErrConflict
	}
	if c.Revision == MaxRevision {
		return Config{}, errors.New("configuration revision exhausted")
	}
	c = cloneConfig(c)
	c.Revision++
	b, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return Config{}, err
	}
	if err := s.write(s.dir, b); err != nil {
		return Config{}, fmt.Errorf("persist configuration: %w", err)
	}
	s.config = c
	return cloneConfig(c), nil
}
func (s *Store) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.lock == nil {
		return nil
	}
	err := s.lock.Close()
	s.lock = nil
	return err
}
func atomicWrite(dir string, data []byte) error {
	f, err := os.CreateTemp(dir, ".config-*")
	if err != nil {
		return err
	}
	name := f.Name()
	defer os.Remove(name)
	if _, err = f.Write(append(data, '\n')); err != nil {
		f.Close()
		return err
	}
	if err = f.Sync(); err != nil {
		f.Close()
		return err
	}
	if err = f.Close(); err != nil {
		return err
	}
	// Same-directory rename is atomic on the supported local Linux filesystem.
	if err = os.Rename(name, filepath.Join(dir, "config.json")); err != nil {
		return err
	}
	// Directory fsync is best effort after commit: reporting a failed transaction
	// after rename would incorrectly leave memory on the preceding revision.
	if d, e := os.Open(dir); e == nil {
		_ = d.Sync()
		_ = d.Close()
	}
	return nil
}

// DecodeJSON is shared by strict public requests and tests; trailing JSON is
// rejected instead of silently accepting a second document.
func DecodeJSON(r io.Reader, target any) error {
	d := json.NewDecoder(r)
	d.DisallowUnknownFields()
	if err := d.Decode(target); err != nil {
		return err
	}
	var extra any
	if err := d.Decode(&extra); err != io.EOF {
		return errors.New("expected exactly one JSON object")
	}
	return nil
}
