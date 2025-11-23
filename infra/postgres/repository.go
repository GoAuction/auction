package postgres

import (
	"database/sql"
	"fmt"

	_ "github.com/lib/pq"
)

type PgRepository struct {
	db *sql.DB
}

func NewPgRepository(host string, database string, user string, password string, port string) *PgRepository {
	db, err := sql.Open("postgres", fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable", host, port, user, password, database))
	if err != nil {
		panic(err)
	}
	return &PgRepository{
		db: db,
	}
}

func (r *PgRepository) Close() error {
	return r.db.Close()
}