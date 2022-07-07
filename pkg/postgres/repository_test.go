package postgres

import (
	"github.com/jackc/pgconn"
	"net/url"
	"testing"
	"time"
)

// psql -h localhost -U postgres -p 5432 -d homework
const defaultPostgresURL = "postgres://postgres:password@localhost:5432/homework?sslmode=disable"

func TestConstructURI(t *testing.T) {
	cases := [...]struct {
		desc           string
		user           string
		password       string
		host           string
		port           uint16
		dbname         string
		sslmode        string
		connectTimeout int
		want           string
	}{
		{
			desc:           "ssl mode required with connection timeout",
			user:           "postgres",
			password:       "password",
			host:           "my.host.com",
			port:           5555,
			dbname:         "postgres",
			sslmode:        "require",
			connectTimeout: 5,
			want:           "postgres://postgres:password@my.host.com:5555/postgres?connect_timeout=5&sslmode=require",
		},
		{
			desc:           "no connection timeout",
			user:           "postgres",
			password:       "password",
			host:           "my.host.com",
			port:           5555,
			dbname:         "postgres",
			sslmode:        "require",
			connectTimeout: 0,
			want:           "postgres://postgres:password@my.host.com:5555/postgres?sslmode=require",
		},
		{
			desc:           "no ssl mode",
			user:           "postgres",
			password:       "password",
			host:           "my.host.com",
			port:           5555,
			dbname:         "postgres",
			sslmode:        "",
			connectTimeout: 5,
			want:           "postgres://postgres:password@my.host.com:5555/postgres?connect_timeout=5",
		},
		{
			desc:           "no password",
			user:           "postgres",
			password:       "",
			host:           "my.host.com",
			port:           5555,
			dbname:         "postgres",
			sslmode:        "",
			connectTimeout: 5,
			want:           "postgres://postgres@my.host.com:5555/postgres?connect_timeout=5",
		},
	}

	for _, tst := range cases {
		t.Run(tst.desc, func(t *testing.T) {
			connDetails := pgconn.Config{
				Host:           tst.host,
				Port:           tst.port,
				Database:       tst.dbname,
				User:           tst.user,
				Password:       tst.password,
				ConnectTimeout: time.Second * time.Duration(tst.connectTimeout),
			}
			got := ConstructURI(connDetails, tst.sslmode)
			if tst.want != got {
				t.Errorf("constructURI() got = %v, want %v", got, tst.want)
				return
			}
			if _, err := url.Parse(got); err != nil {
				t.Errorf("constructURI() got = %v, not valid: %v", got, err)
				return
			}
		})
	}
}
