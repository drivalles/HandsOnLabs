package ldconn_test

import (
	"fmt"
	ldconn "ld-backend/internal/impl/back/db/conn"
	"testing"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/stretchr/testify/assert"
	"gorm.io/gorm"
)

func TestIsRetryable(t *testing.T) {
	assert.True(t, ldconn.IsRetryable(&pgconn.PgError{Code: "40001"}))
	assert.True(t, ldconn.IsRetryable(fmt.Errorf("wrapped: %w", &pgconn.PgError{Code: "40001"})))
	assert.False(t, ldconn.IsRetryable(&pgconn.PgError{Code: "23505"}))
	assert.False(t, ldconn.IsRetryable(gorm.ErrRecordNotFound))
	assert.False(t, ldconn.IsRetryable(nil))
}

func TestCockroachUrl(t *testing.T) {
	assert.Equal(t, "host=h dbname=d serial_normalization=sql_sequence", ldconn.CockroachUrl("host=h dbname=d"))
	assert.Equal(t, "postgres://h/d?serial_normalization=sql_sequence", ldconn.CockroachUrl("postgres://h/d"))
	assert.Equal(t, "postgres://h/d?sslmode=require&serial_normalization=sql_sequence", ldconn.CockroachUrl("postgres://h/d?sslmode=require"))
	assert.Equal(t, "host=h serial_normalization=virtual_sequence", ldconn.CockroachUrl("host=h serial_normalization=virtual_sequence"))
}
