package plugin

import (
	"net/http"

	"github.com/lahickey/quire/internal/session"
)

func Session(r *http.Request) string { return session.ID(r) }
