//go:build quire_nobake

package bake

import (
	"errors"

	"github.com/lahickey/quire/internal/journal"
	"github.com/lahickey/quire/model"
)

const Enabled = false

type Run struct{}

func Start(model.BakeSpec) (*Run, error) {
	return nil, errors.New("quire: built with quire_nobake, so this binary serves a bundle and cannot bake one")
}

func (*Run) Master(string) (journal.Record, error) { return journal.Record{}, nil }

func (*Run) Diagram(string) (string, error) { return "", nil }

func (*Run) Documents(string, []string) ([]journal.Document, error) { return nil, nil }

func (*Run) Data(string) ([]byte, error) { return nil, nil }

func (*Run) Card(string, model.ShareCard) error { return nil }

func (*Run) Finish() error { return nil }
