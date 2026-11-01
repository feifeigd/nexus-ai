package data

type Database struct {
	Driver       string // mysql|postgres|sqlite
	Source       string // DSN
	MaxOpenConns int
	MaxIdleConns int
	Debug        bool
	AutoMigrate  bool
}

type Data struct {
	Database Database
}
