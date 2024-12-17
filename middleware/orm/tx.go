package orm

import (
	"accumulation/pkg/log"
	"context"
	"database/sql"
	"fmt"

	"runtime"

	"gorm.io/gorm"
)

func DoTransaction(ctx context.Context, db *gorm.DB, fn func(ctx context.Context) error, opts ...*sql.TxOptions) error {
	tx := db.Begin(opts...)
	defer func() {
		if v := recover(); v != nil {
			var stack []string
			for i := 1; ; i++ {
				_, file, line, ok := runtime.Caller(i)
				if !ok {
					break
				}
				stack = append(stack, fmt.Sprintln(fmt.Sprintf("%s:%d", file, line)))
			}
			log.Errorf(ctx, "db panic %v", stack)
			tx.Rollback()

		}
	}()
	nexNew := NewTxContext(ctx, tx)
	if err := fn(nexNew); err != nil {
		if rerr := tx.Rollback(); rerr != nil {
			err = fmt.Errorf("%w: rolling back transaction: %v", err, rerr)
		}
		return err
	}
	if ormDb := tx.Commit(); ormDb.Error != nil {
		return fmt.Errorf("committing transaction: %v", ormDb.Error)
	}
	return nil
}

type transactionKey struct{}

// NewTxContext creates a new context
func NewTxContext(ctx context.Context, db *gorm.DB) context.Context {

	return context.WithValue(ctx, transactionKey{}, db)
}

// GetTxContext creates a new context
func GetTxContext(ctx context.Context) (*gorm.DB, bool) {
	tx, ok := ctx.Value(transactionKey{}).(*gorm.DB)
	return tx, ok
}
