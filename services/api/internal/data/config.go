package data

import "time"

type Database struct {
	Driver          string // postgres|sqlite3
	Source          string // DSN
	MaxOpenConns    int
	MaxIdleConns    int
	ConnMaxLifetime time.Duration
	Debug           bool
	AutoMigrate     bool
}

type Data struct {
	Database Database
}
