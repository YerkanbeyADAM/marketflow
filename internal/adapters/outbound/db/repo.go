package db

import (
	"database/sql"
	"fmt"
	"log"

	"marketflow/internal/config"

	_ "github.com/jackc/pgx/v5/stdlib"
)

type Postgres struct {
	db *sql.DB
	Repo
}

func NewPostgres(cfg config.PostgresConfig) (*Postgres, error) {
	psqlInfo := fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		cfg.Host, cfg.Port, cfg.User, cfg.Password, cfg.DBName, cfg.SSLMode,
	)

	db, err := sql.Open("pgx", psqlInfo)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to PostgreSQL: %w", err)
	}

	return &Postgres{
		db:   db,
		Repo: Repo{Conn: db},
	}, nil
}

func (p *Postgres) Close() {
	if err := p.db.Close(); err != nil {
		log.Printf("Error closing the database connection: %v", err)
	}
}
