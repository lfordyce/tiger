package postgres

import (
	"context"
	"errors"
	"fmt"
	"github.com/jackc/pgconn"
	_ "github.com/jackc/pgconn"
	"github.com/jackc/pgx/v4"
	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/lfordyce/tiger/internal/domain"
	"github.com/spf13/pflag"
	"math"
	"net/url"
	"strconv"
	"time"
)

type Repository struct {
	Conn *pgxpool.Pool
}

func GetConfig(flags *pflag.FlagSet) (*DBDetails, error) {
	host, err := flags.GetString("host")
	if err != nil {
		return nil, err
	}

	port, err := flags.GetUint16("port")
	if err != nil {
		return nil, err
	}

	database, err := flags.GetString("database")
	if err != nil {
		return nil, err
	}

	password, err := flags.GetString("password")
	if err != nil {
		return nil, err
	}

	user, err := flags.GetString("user")
	if err != nil {
		return nil, err
	}

	return &DBDetails{
		Host:     host,
		Port:     port,
		DBName:   database,
		Password: password,
		User:     user,
	}, nil
}

type DBDetails struct {
	Host     string
	DBName   string
	User     string
	Password string
	Port     uint16
}

func (d *DBDetails) OpenConnection(ctx context.Context) (domain.Handler, func(), error) {
	connDetails := pgconn.Config{
		Host:           d.Host,
		Port:           d.Port,
		Database:       d.DBName,
		User:           d.User,
		Password:       d.Password,
		ConnectTimeout: time.Second * time.Duration(5),
	}

	dsn := ConstructURI(connDetails, "disable")

	pool, err := pgxpool.Connect(ctx, dsn)
	if err != nil {
		return Repository{}, func() {}, fmt.Errorf("postgres.OpenConnection: failed to establish postgres connection %w", err)
	}

	if err := MigrationManager(dsn, MigrationUp); err != nil {
		return nil, nil, fmt.Errorf("postgres.OpenConnection: failed to run db migrations %w", err)
	}

	return Repository{Conn: pool}, pool.Close, nil
}

func (r Repository) Process(req domain.Request) (float64, error) {
	var elapsed float64
	err := r.Conn.QueryRow(context.Background(), "SELECT * FROM bench($1::TEXT, $2::TIMESTAMPTZ, $3::TIMESTAMPTZ)", req.HostID, req.StartTime, req.EndTime).Scan(&elapsed)
	if errors.Is(err, pgx.ErrNoRows) {
		return math.NaN(), fmt.Errorf("postgres.Process: elapsed data not found")
	}
	if err != nil {
		return math.NaN(), fmt.Errorf("postgres.Process: failed to query events table %w", err)
	}
	return elapsed, nil
}

func ConstructURI(connDetails pgconn.Config, sslmode string) string {
	c := new(url.URL)
	c.Scheme = "postgres"
	c.Host = fmt.Sprintf("%s:%d", connDetails.Host, connDetails.Port)
	if connDetails.Password != "" {
		c.User = url.UserPassword(connDetails.User, connDetails.Password)
	} else {
		c.User = url.User(connDetails.User)
	}
	c.Path = connDetails.Database
	q := c.Query()
	if sslmode != "" {
		q.Set("sslmode", sslmode)
	}

	if connDetails.ConnectTimeout != 0 {
		q.Set("connect_timeout", strconv.Itoa(int(connDetails.ConnectTimeout.Seconds())))
	}
	c.RawQuery = q.Encode()
	return c.String()
}
