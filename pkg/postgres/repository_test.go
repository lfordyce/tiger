package postgres

import (
	"context"
	"github.com/jackc/pgconn"
	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/lfordyce/tiger/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"math"
	"net/url"
	"testing"
	"time"
)

func TestRunMigrations(t *testing.T) {
	if testing.Short() {
		t.SkipNow()
	}

	connDetails := pgconn.Config{
		Host:           "localhost",
		Port:           5432,
		Database:       "homework",
		User:           "postgres",
		Password:       "password",
		ConnectTimeout: time.Second * time.Duration(5),
	}
	dsn := ConstructURI(connDetails, "disable")

	require.NoError(t, MigrationManager(dsn, MigrationUp))

	ctx := context.Background()
	pool, err := pgxpool.Connect(ctx, dsn)
	require.NoError(t, err)

	r := Repository{Conn: pool}
	t.Run("it can execute the newly migrated batch function", func(t *testing.T) {

		start, err := time.Parse("2006-01-02 15:04:05", "2017-01-02 13:02:02")
		assert.NoError(t, err)

		end, err := time.Parse("2006-01-02 15:04:05", "2017-01-02 14:02:02")
		assert.NoError(t, err)

		elapsed, err := r.Process(domain.Request{
			HostID:    "host_000001",
			StartTime: start,
			EndTime:   end,
		})
		assert.NoError(t, err)
		assert.True(t, elapsed != math.NaN())
	})
	r.Conn.Close()
	require.NoError(t, MigrationManager(dsn, MigrationDown))
}

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
