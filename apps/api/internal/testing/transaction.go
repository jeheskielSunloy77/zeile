package testing

import (
	"context"
	"fmt"

	"gorm.io/gorm"
)

// TxFn represents a function that executes within a transaction
type TxFn func(tx *gorm.DB) error

// WithTransaction runs a function within a transaction and rolls it back afterward
func WithTransaction(ctx context.Context, db *TestDB, fn TxFn) error {
	if db == nil || db.DB == nil {
		return fmt.Errorf("database is not initialized")
	}

	// Begin transaction
	tx := db.DB.WithContext(ctx).Begin()
	if tx.Error != nil {
		return fmt.Errorf("failed to begin transaction: %w", tx.Error)
	}

	// Ensure rollback happens if commit doesn't occur
	defer tx.Rollback()

	// Run the function within the transaction
	if err := fn(tx); err != nil {
		return err
	}

	// Transaction was successful, commit it
	if err := tx.Commit().Error; err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// WithRollbackTransaction runs a function within a transaction and always rolls it back
// Useful for tests where you want to execute operations but never persist them
func WithRollbackTransaction(ctx context.Context, db *TestDB, fn TxFn) error {
	if db == nil || db.DB == nil {
		return fmt.Errorf("database is not initialized")
	}

	// Begin transaction
	tx := db.DB.WithContext(ctx).Begin()
	if tx.Error != nil {
		return fmt.Errorf("failed to begin transaction: %w", tx.Error)
	}

	// Always rollback at the end
	defer tx.Rollback()

	// Run the function within the transaction
	return fn(tx)
}
