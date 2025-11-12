package database

import (
	"context"
	"errors"

	"gorm.io/gorm"
)

// WithTransaction executes the provided fn within a transaction while propagating context.
// The transaction DB instance passed to fn already includes the context, so repository methods
// can use it directly. Calling WithContext again is optional (and safe) if you want to keep the
// repository signature uniform for both transactional and non-transactional DB handles.
//
// Usage:
//
//	err := WithTransaction(ctx, db, func(tx *gorm.DB) error {
//	    // tx already has context - just use it directly
//	    if err := repo.Create(ctx, tx, entity); err != nil {
//	        return err // rollback
//	    }
//	    return nil // commit
//	})
func WithTransaction(ctx context.Context, db *gorm.DB, fn func(*gorm.DB) error) error {
	if fn == nil {
		return errors.New("database: transaction function is nil")
	}

	if ctx == nil {
		ctx = context.Background()
	}

	return db.WithContext(ctx).Transaction(fn)
}
