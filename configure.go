package rbac

import (
	"github.com/jmoiron/sqlx"
	"github.com/zenazn/goji/web"
)

type Opts struct {
	Dbx     *sqlx.DB
	Mux     *web.Mux
	MuxBase string
}

var opts *Opts
var dbx *sqlx.DB

func Configure(options *Opts) {
	opts = options
	dbx = opts.Dbx
}
