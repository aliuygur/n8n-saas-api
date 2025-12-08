package db

import (
	"database/sql"
	"errors"
)

func IsNotFoundError(err error) bool {
	if errors.Is(err, sql.ErrNoRows) {
		return true
	}

	return false
}
