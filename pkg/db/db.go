package db

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"

	trmpgx "github.com/avito-tech/go-transaction-manager/drivers/pgxv5/v2"
	"github.com/avito-tech/go-transaction-manager/trm/v2/manager"
)

type DB struct {
	pool   *pgxpool.Pool
	getter *trmpgx.CtxGetter
}

func NewDB(pool *pgxpool.Pool) *DB {
	return &DB{
		pool:   pool,
		getter: trmpgx.DefaultCtxGetter,
	}
}

func (db *DB) Conn(ctx context.Context) trmpgx.Tr {
	return db.getter.DefaultTrOrDB(ctx, db.pool)
}

type TransactionManagerInterface interface {
	Do(ctx context.Context, fn func(ctx context.Context) error) error
}

type TransactionManager struct {
	manager *manager.Manager
}

func NewTransactionManager(pool *pgxpool.Pool) (*TransactionManager, error) {
	trManager, err := manager.New(trmpgx.NewDefaultFactory(pool))

	if err != nil {
		return nil, err
	}

	return &TransactionManager{manager: trManager}, nil
}

func (tm *TransactionManager) Do(ctx context.Context, fn func(ctx context.Context) error) error {
	return tm.manager.Do(ctx, fn)
}
