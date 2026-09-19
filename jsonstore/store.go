package jsonstore

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/wanstu/wails-desktop-kit/atomicfile"
)

type Options[T any] struct {
	Default   func() T
	Normalize func(*T)
	Validate  func(T) error
	Mode      os.FileMode
}

type Store[T any] struct {
	mu      sync.Mutex
	path    string
	options Options[T]
}

func New[T any](path string, options Options[T]) *Store[T] {
	if options.Mode == 0 {
		options.Mode = 0o600
	}
	return &Store[T]{path: path, options: options}
}

func (s *Store[T]) Path() string { return s.path }
func (s *Store[T]) Dir() string  { return filepath.Dir(s.path) }

func (s *Store[T]) Load() (T, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.loadLocked()
}

func (s *Store[T]) Save(value T) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.saveLocked(value)
}

func (s *Store[T]) Update(fn func(*T) error) (T, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	value, err := s.loadLocked()
	if err != nil {
		return value, err
	}
	if fn != nil {
		if err := fn(&value); err != nil {
			return value, err
		}
	}
	if err := s.saveLocked(value); err != nil {
		return value, err
	}
	return value, nil
}

func (s *Store[T]) defaultValue() T {
	if s.options.Default != nil {
		return s.options.Default()
	}
	var zero T
	return zero
}

func (s *Store[T]) loadLocked() (T, error) {
	value := s.defaultValue()
	data, err := os.ReadFile(s.path)
	if errors.Is(err, os.ErrNotExist) {
		if s.options.Normalize != nil {
			s.options.Normalize(&value)
		}
		if s.options.Validate != nil {
			if err := s.options.Validate(value); err != nil {
				return value, fmt.Errorf("jsonstore: validate default: %w", err)
			}
		}
		return value, nil
	}
	if err != nil {
		return value, fmt.Errorf("jsonstore: read %s: %w", s.path, err)
	}
	if err := json.Unmarshal(data, &value); err != nil {
		return value, fmt.Errorf("jsonstore: decode %s: %w", s.path, err)
	}
	if s.options.Normalize != nil {
		s.options.Normalize(&value)
	}
	if s.options.Validate != nil {
		if err := s.options.Validate(value); err != nil {
			return value, fmt.Errorf("jsonstore: validate %s: %w", s.path, err)
		}
	}
	return value, nil
}

func (s *Store[T]) saveLocked(value T) error {
	if s.options.Normalize != nil {
		s.options.Normalize(&value)
	}
	if s.options.Validate != nil {
		if err := s.options.Validate(value); err != nil {
			return fmt.Errorf("jsonstore: validate %s: %w", s.path, err)
		}
	}
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return fmt.Errorf("jsonstore: encode %s: %w", s.path, err)
	}
	data = append(data, '\n')
	if err := atomicfile.Write(s.path, data, s.options.Mode); err != nil {
		return fmt.Errorf("jsonstore: save %s: %w", s.path, err)
	}
	return nil
}
