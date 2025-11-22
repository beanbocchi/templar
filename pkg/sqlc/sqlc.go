package sqlc

import (
	"context"
	"database/sql"
	"errors"
	"log/slog"

	"github.com/beanbocchi/templar/internal/db"
)

type DBTX interface {
	ExecContext(context.Context, string, ...interface{}) (sql.Result, error)
	PrepareContext(context.Context, string) (*sql.Stmt, error)
	QueryContext(context.Context, string, ...interface{}) (*sql.Rows, error)
	QueryRowContext(context.Context, string, ...interface{}) *sql.Row

	Begin() (*sql.Tx, error)
}

func NewStorage(dbtx DBTX) *Storage {
	return &Storage{
		dbtx:    dbtx,
		Queries: db.New(dbtx),
	}
}

type Storage struct {
	dbtx DBTX
	*db.Queries
}

func (s *Storage) BeginTx() (*TxStorage, error) {
	tx, err := s.dbtx.Begin()
	if err != nil {
		return nil, err
	}

	return &TxStorage{tx: tx, Queries: s.Queries.WithTx(tx)}, nil
}

type TxStorage struct {
	tx *sql.Tx
	*db.Queries
}

func (s *TxStorage) Commit() error {
	return s.tx.Commit()
}

func (s *TxStorage) Rollback() {
	if err := s.tx.Rollback(); !errors.Is(err, sql.ErrTxDone) && err != nil {
		slog.Error("failed to rollback transaction", "error", err)
	}
}
