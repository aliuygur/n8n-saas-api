package db

import (
	"errors"

	"github.com/jackc/pgx/v5"
)

func IsNotFoundError(err error) bool {
	return errors.Is(err, pgx.ErrNoRows)
}
