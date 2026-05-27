package repositories

import (
	"database/sql"
	"sync"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
)

var (
	psqlDB *sql.DB
	mu     sync.Mutex
)

func CreateAndCheckPSQLCon(DBDSN string) (*sql.DB, error) {
	mu.Lock()
	defer mu.Unlock()

	if psqlDB != nil {
		return psqlDB, nil
	}

	db, err := sql.Open("pgx", DBDSN)
	if err != nil {
		return nil, err
	}

	err = db.Ping()
	if err != nil {
		db.Close()
		return nil, err
	}

	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(25)
	db.SetConnMaxLifetime(5 * time.Minute)

	psqlDB = db
	return psqlDB, nil
}
