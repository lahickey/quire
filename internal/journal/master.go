package journal

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
)

const MasterCap = 2560

func Hash(b []byte) string {
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:16])
}

type Format struct {
	Ext      string
	Type     string
	Quality  int
	Alpha    bool
	Lossless bool
}

type MasterMeta struct {
	Hash  string `json:"h"`
	W     int    `json:"w"`
	H     int    `json:"y"`
	SrcW  int    `json:"sw,omitzero"`
	SrcH  int    `json:"sh,omitzero"`
	Alpha bool   `json:"a,omitzero"`
}

func WriteAtomic(path string, body []byte, perm os.FileMode) error {
	f, err := os.CreateTemp(filepath.Dir(path), "."+filepath.Base(path)+".*")
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
	if err := os.Chmod(tmp, perm); err != nil {
		return err
	}
	if err := os.Rename(tmp, path); err != nil {
		return err
	}
	ok = true
	return nil
}
