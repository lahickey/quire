package baketest

import (
	"io/fs"
	"os"

	"github.com/lahickey/quire/internal/bake"
	"github.com/lahickey/quire/internal/bake/picture"
	"github.com/lahickey/quire/model"
)

func BakeMasters(s model.BakeSpec) error {
	b, err := bake.Start(s)
	if err != nil {
		return err
	}
	if s.Content != "" {
		err = fs.WalkDir(os.DirFS(s.Content), ".", func(name string, d fs.DirEntry, err error) error {
			if err != nil || d.IsDir() || !picture.IsMaster(name) {
				return err
			}
			_, err = b.Master(name)
			return err
		})
		if err != nil {
			return err
		}
	}
	return b.Finish()
}
