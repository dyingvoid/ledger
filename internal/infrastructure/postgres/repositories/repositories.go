package repositories

import "github.com/jackc/pgx/v5/pgxpool"

type Repositories struct {
	pool *pgxpool.Pool
}

func New(pool *pgxpool.Pool) *Repositories {
	return &Repositories{pool: pool}
}
