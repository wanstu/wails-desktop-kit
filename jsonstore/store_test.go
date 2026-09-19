package jsonstore

import (
	"errors"
	"path/filepath"
	"testing"
)

type sampleSettings struct {
	Name string `json:"name"`
	N    int    `json:"n"`
}

func TestStoreDefaultSaveUpdate(t *testing.T) {
	path := filepath.Join(t.TempDir(), "settings.json")
	store := New(path, Options[sampleSettings]{
		Default: func() sampleSettings { return sampleSettings{Name: "default"} },
		Normalize: func(v *sampleSettings) {
			if v.Name == "" {
				v.Name = "default"
			}
		},
		Validate: func(v sampleSettings) error {
			if v.N < 0 {
				return errors.New("negative")
			}
			return nil
		},
	})
	got, err := store.Load()
	if err != nil || got.Name != "default" {
		t.Fatalf("Load() = %#v, %v", got, err)
	}
	if err := store.Save(sampleSettings{Name: "saved", N: 1}); err != nil {
		t.Fatal(err)
	}
	got, err = store.Update(func(v *sampleSettings) error {
		v.N++
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if got.Name != "saved" || got.N != 2 {
		t.Fatalf("Update() = %#v", got)
	}
}
