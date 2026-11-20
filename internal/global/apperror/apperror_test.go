package apperror

import (
	"errors"
	"fmt"
	"net/http"
	"testing"
)

func TestConstructors(t *testing.T) {
	tests := []struct {
		name       string
		err        *Error
		wantStatus int
	}{
		{name: "BadRequest", err: BadRequest("m"), wantStatus: http.StatusBadRequest},
		{name: "Unauthorized", err: Unauthorized("m"), wantStatus: http.StatusUnauthorized},
		{name: "Forbidden", err: Forbidden("m"), wantStatus: http.StatusForbidden},
		{name: "NotFound", err: NotFound("m"), wantStatus: http.StatusNotFound},
		{name: "Conflict", err: Conflict("m"), wantStatus: http.StatusConflict},
		{name: "New", err: New(http.StatusTeapot, "m"), wantStatus: http.StatusTeapot},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.err.Status != tt.wantStatus {
				t.Errorf("Status = %d, want %d", tt.err.Status, tt.wantStatus)
			}
			if tt.err.Error() != "m" {
				t.Errorf("Error() = %q, want m", tt.err.Error())
			}
		})
	}
}

// 위 레이어가 %w 로 감싸도 errors.As 로 꺼낼 수 있어야 한다 — 에러 미들웨어가 이렇게 찾는다.
func TestErrorsAs_ThroughWrapping(t *testing.T) {
	var err error = NotFound("사용자를 찾을 수 없습니다.")
	wrapped := fmt.Errorf("read user: %w", fmt.Errorf("service: %w", err))

	var appErr *Error
	if !errors.As(wrapped, &appErr) {
		t.Fatal("감싼 에러에서 *Error 를 찾지 못했다")
	}
	if appErr.Status != http.StatusNotFound {
		t.Errorf("Status = %d, want 404", appErr.Status)
	}
}

func TestErrorsAs_PlainError(t *testing.T) {
	var appErr *Error
	if errors.As(errors.New("connection refused"), &appErr) {
		t.Error("보통 에러가 *Error 로 잡혔다")
	}
}
