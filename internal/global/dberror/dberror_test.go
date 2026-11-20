package dberror

import (
	"errors"
	"fmt"
	"testing"

	"github.com/jackc/pgx/v5/pgconn"
)

func TestViolations(t *testing.T) {
	unique := &pgconn.PgError{Code: "23505", Message: "duplicate key value violates unique constraint"}
	foreignKey := &pgconn.PgError{Code: "23503", Message: "violates foreign key constraint"}

	tests := []struct {
		name           string
		err            error
		wantUnique     bool
		wantForeignKey bool
	}{
		{name: "유니크 위반", err: unique, wantUnique: true},
		{name: "FK 위반", err: foreignKey, wantForeignKey: true},
		{name: "감싸도 찾는다", err: fmt.Errorf("insert user: %w", unique), wantUnique: true},
		{name: "다른 DB 에러", err: &pgconn.PgError{Code: "57014", Message: "canceling statement"}},
		{name: "DB 에러가 아닌 것", err: errors.New("connection refused")},
		{name: "nil", err: nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsUniqueViolation(tt.err); got != tt.wantUnique {
				t.Errorf("IsUniqueViolation() = %v, want %v", got, tt.wantUnique)
			}
			if got := IsForeignKeyViolation(tt.err); got != tt.wantForeignKey {
				t.Errorf("IsForeignKeyViolation() = %v, want %v", got, tt.wantForeignKey)
			}
		})
	}
}
