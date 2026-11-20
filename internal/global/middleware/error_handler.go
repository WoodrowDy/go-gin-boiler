package middleware

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/WoodrowDy/go-gin-boiler/internal/global/apperror"
	"github.com/WoodrowDy/go-gin-boiler/internal/global/dberror"
	"github.com/WoodrowDy/go-gin-boiler/internal/global/response"
)

// 도메인 메시지가 없는 실패에 싣는 문구다. 원래 에러의 내용은 응답에 싣지 않는다.
const (
	messageInternalError       = "서버 오류가 발생했습니다."
	messageUniqueViolation     = "이미 존재하는 값입니다."
	messageForeignKeyViolation = "다른 데이터가 참조 중입니다."
)

// ErrorHandler 는 핸들러가 남긴 에러를 응답으로 바꾼다. nest-boiler 의 전역 예외 필터에 해당한다.
//
// 핸들러는 에러를 직접 응답으로 쓰지 않는다. c.Error(err) 로 남기고 return 한다:
//
//	user, err := h.service.ReadUser(ctx, id)
//	if err != nil {
//		_ = c.Error(err)
//		return
//	}
//
// 그러면 여기서 한 곳에서 모양을 맞춘다.
//
//   - apperror.Error (감싸여 있어도) → 그 상태 코드와 메시지.
//   - 아무도 옮기지 않은 DB 제약 위반 → 409. service 가 놓쳐도 500 이 아니라 말이 되는 상태로 떨어진다.
//     그래도 옮기는 자리는 service 가 먼저다 — 도메인 메시지는 거기에만 있다.
//   - 그 밖의 에러                   → 500. 내용은 로그에만 남기고 응답에는 정해진 문구만 싣는다.
//     스택이나 드라이버 메시지에는 테이블·컬럼 이름이 들어 있다. 추적은 meta.traceId 로 한다.
//
// 5xx 는 여기서 로그를 남긴다. 4xx 는 남기지 않는다 — 접근 로그에 상태 코드가 이미 있다.
//
// Recovery 뒤(핸들러에 가장 가깝게)에 등록한다.
func ErrorHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()

		// c.Next() 가 돌아왔다 = 핸들러가 끝났다. 남긴 에러가 없으면 할 일이 없다.
		if len(c.Errors) == 0 {
			return
		}

		// 여러 개를 남겼어도 응답은 하나다. 마지막 것을 쓴다.
		err := c.Errors.Last().Err
		status, message := resolveError(err)

		if status >= http.StatusInternalServerError {
			slog.ErrorContext(c.Request.Context(), "request failed", "err", err)
		}

		writeError(c, status, message)
	}
}

// NoRoute 는 등록되지 않은 경로의 404 를 같은 응답 규약으로 맞춘다.
// r.NoRoute(middleware.NoRoute()) 로 건다.
func NoRoute() gin.HandlerFunc {
	return func(c *gin.Context) {
		_ = c.Error(apperror.NotFound("Cannot " + c.Request.Method + " " + c.Request.URL.Path))
	}
}

// resolveError 는 에러를 상태 코드와 응답 문구로 바꾼다.
func resolveError(err error) (status int, message string) {
	// errors.As 는 %w 로 감싼 껍질을 벗겨 가며 *apperror.Error 를 찾는다.
	// service 가 돌려준 것을 handler 가 감싸서 넘겨도 잡힌다.
	var appErr *apperror.Error
	if errors.As(err, &appErr) {
		return appErr.Status, appErr.Message
	}

	// DB 에러는 의미가 있는 것만 바꾼다. 제약 위반은 업무 규칙 위반이라 409 로 옮기고,
	// 그 외(연결 끊김 · 타임아웃)는 손대지 않고 500 으로 둔다. 전부 409 로 뭉개면 원인이 사라진다.
	if dberror.IsUniqueViolation(err) {
		return http.StatusConflict, messageUniqueViolation
	}
	if dberror.IsForeignKeyViolation(err) {
		return http.StatusConflict, messageForeignKeyViolation
	}

	return http.StatusInternalServerError, messageInternalError
}

// writeError 는 실패 응답을 쓰고 뒤의 핸들러를 멈춘다.
func writeError(c *gin.Context, status int, message string) {
	// 본문을 이미 쓰기 시작했으면 상태 코드와 헤더는 나간 뒤다. 덧붙이면 깨진 JSON 이 된다.
	if c.Writer.Written() {
		c.Abort()
		return
	}

	c.AbortWithStatusJSON(status, response.Error(c.Request.Context(), status, message))
}
