package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	gormPostgres "gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

const (
	dbMaxIdleConn     = 10
	dbMaxOpenConn     = 100
	dbConnMaxLifetime = time.Hour

	postgresDriverName = "pgx"
)

type IPostgresInstance interface {
	Database() *gorm.DB
	SQLDatabase() *sql.DB
	Close() error
	Ping(ctx context.Context) error
}

type Options struct {
	MaxIdleConns int
	MaxOpenConns int
}

type postgresInstance struct {
	db    *gorm.DB
	sqlDB *sql.DB
}

func NewClient(dsn string, isDebug bool, opts Options) (IPostgresInstance, error) {
	sqlDB, err := sql.Open(postgresDriverName, dsn)
	if err != nil {
		return nil, fmt.Errorf("open sql db: %w", err)
	}

	return NewClientFromSQLDB(sqlDB, isDebug, opts)
}

func NewClientFromSQLDB(sqlDB *sql.DB, isDebug bool, opts Options) (IPostgresInstance, error) {
	if err := sqlDB.Ping(); err != nil {
		return nil, fmt.Errorf("ping sql db: %w", err)
	}

	if opts.MaxIdleConns <= 0 {
		opts.MaxIdleConns = dbMaxIdleConn
	}
	if opts.MaxOpenConns <= 0 {
		opts.MaxOpenConns = dbMaxOpenConn
	}

	sqlDB.SetConnMaxLifetime(dbConnMaxLifetime)
	sqlDB.SetMaxOpenConns(opts.MaxOpenConns)
	sqlDB.SetMaxIdleConns(opts.MaxIdleConns)

	logLevel := logger.Error
	if isDebug {
		logLevel = logger.Info
	}

	db, err := gorm.Open(gormPostgres.New(gormPostgres.Config{
		Conn: sqlDB,
	}), &gorm.Config{
		Logger: logger.Default.LogMode(logLevel),
	})
	if err != nil {
		return nil, fmt.Errorf("open gorm db: %w", err)
	}

	return &postgresInstance{
		db:    db,
		sqlDB: sqlDB,
	}, nil
}

func (p *postgresInstance) Database() *gorm.DB {
	return p.db
}

func (p *postgresInstance) SQLDatabase() *sql.DB {
	return p.sqlDB
}

func (p *postgresInstance) Close() error {
	return p.sqlDB.Close()
}

func (p *postgresInstance) Ping(ctx context.Context) error {
	return p.sqlDB.PingContext(ctx)
}
