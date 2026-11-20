// Package dberror 는 PostgreSQL 제약 위반을 판별한다.
//
// DB 에러는 의미에 따라 다르게 다룬다.
//   - 업무 규칙 위반(중복, 참조 중 삭제)은 잡은 자리에서 apperror 로 바꾼다.
//   - 그 외(연결 끊김, 타임아웃)는 잡지 않고 그대로 올린다.
//
// 그 자리는 service 다. 어떤 제약에 걸릴 수 있는지는 그 쓰기를 부른 쪽만 안다:
//
//	user, err := s.repo.InsertUser(ctx, nil, payload)
//	if dberror.IsUniqueViolation(err) {
//		return nil, apperror.Conflict(errDuplicateEmail)
//	}
//	if err != nil {
//		return nil, fmt.Errorf("insert user: %w", err)
//	}
package dberror

import (
	"errors"

	"github.com/jackc/pgx/v5/pgconn"
)

// PostgreSQL 에러 코드(SQLSTATE). https://www.postgresql.org/docs/current/errcodes-appendix.html
const (
	codeUniqueViolation     = "23505"
	codeForeignKeyViolation = "23503"
)

// IsUniqueViolation 은 유니크 제약 위반인지 본다. 중복을 409 로 바꿀 때 쓴다.
func IsUniqueViolation(err error) bool {
	return hasCode(err, codeUniqueViolation)
}

// IsForeignKeyViolation 은 FK 제약 위반인지 본다. 참조 중인 행의 삭제를 "사용 중" 으로 바꿀 때 쓴다.
func IsForeignKeyViolation(err error) bool {
	return hasCode(err, codeForeignKeyViolation)
}

func hasCode(err error, code string) bool {
	// 드라이버(pgx)가 돌려주는 에러 타입. GORM 이나 위 레이어가 %w 로 감싸도 errors.As 가 찾아낸다.
	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) {
		return false
	}

	return pgErr.Code == code
}
