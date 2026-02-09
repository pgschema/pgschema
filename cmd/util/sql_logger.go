package util

import (
	"context"
	"database/sql"

	"github.com/pgplex/pgschema/internal/logger"
)

// ExecContextWithLogging executes SQL with debug logging if debug mode is enabled.
// It logs the SQL statement before execution and the result/error after execution.
func ExecContextWithLogging(ctx context.Context, db *sql.DB, sqlStmt string, description string) (sql.Result, error) {
	isDebug := logger.IsDebug()
	if isDebug {
		logger.Get().Debug("Executing SQL", "description", description, "sql", sqlStmt)
	}

	result, err := db.ExecContext(ctx, sqlStmt)

	if isDebug {
		if err != nil {
			logger.Get().Debug("SQL execution failed", "description", description, "error", err)
		} else {
			logger.Get().Debug("SQL execution succeeded", "description", description)
		}
	}

	return result, err
}
