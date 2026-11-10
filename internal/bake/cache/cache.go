package cache

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/lahickey/quire/internal/journal"
)

type Dir string

func (d Dir) path(name string) string { return filepath.Join(string(d), name) }

func (d Dir) link(out Dir, name string) error {
	if os.Link(d.path(name), out.path(name)) == nil {
		return nil
	}
	body, err := d.Read(name)
	if err != nil {
		return fmt.Errorf("quire: %s: %w", name, err)
	}
	return out.Write(name, body)
}

func (d Dir) Read(name string) ([]byte, error) { return os.ReadFile(d.path(name)) }

func (d Dir) Write(name string, body []byte) error {
	f, err := os.CreateTemp(string(d), "."+name+".*")
	if err != nil {
		return err
	}
	tmp := f.Name()
	ok := false
	defer func() {
		if !ok {
			_ = os.Remove(tmp)
		}
	}()
	if _, err := f.Write(body); err != nil {
		_ = f.Close()
		return err
	}
	if err := f.Close(); err != nil {
		return err
	}
	if err := os.Chmod(tmp, 0o644); err != nil {
		return err
	}
	if err := os.Rename(tmp, d.path(name)); err != nil {
		return err
	}
	ok = true
	return nil
}

func (d Dir) Size(name string) (int64, bool) {
	st, err := os.Stat(d.path(name))
	if err != nil || st.IsDir() {
		return 0, false
	}
	return st.Size(), true
}

func (d Dir) Journal() map[string]journal.Record {
	recs, err := journal.LoadJournal(os.DirFS(string(d)))
	if err != nil {
		return map[string]journal.Record{}
	}
	return recs
}

func (d Dir) CopyOut(dest string, records []journal.Record) error {
	if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
		return err
	}
	tmp, err := os.MkdirTemp(filepath.Dir(dest), "."+filepath.Base(dest)+".*")
	if err != nil {
		return err
	}
	ok := false
	defer func() {
		if !ok {
			_ = os.RemoveAll(tmp)
		}
	}()
	// MkdirTemp is 0700, and this directory becomes dest.
	if err := os.Chmod(tmp, 0o755); err != nil {
		return err
	}
	out := Dir(tmp)
	for _, r := range records {
		for _, v := range r.Files() {
			if err := d.link(out, v.Name); err != nil {
				return err
			}
		}
	}
	if err := out.Write(journal.JournalName, journal.WriteJournal(records)); err != nil {
		return err
	}
	if err := os.RemoveAll(dest); err != nil {
		return err
	}
	if err := os.Rename(tmp, dest); err != nil {
		return err
	}
	ok = true
	return nil
}
