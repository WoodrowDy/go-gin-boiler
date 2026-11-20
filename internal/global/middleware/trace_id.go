// Package middleware 는 모든 요청이 지나가는 Gin 미들웨어를 모은다.
package middleware

import (
	"crypto/rand"
	"fmt"

	"github.com/gin-gonic/gin"

	"github.com/WoodrowDy/go-gin-boiler/internal/global/requestctx"
)

// HeaderRequestID 는 traceId 를 주고받는 헤더 이름이다.
const HeaderRequestID = "X-Request-Id"

// maxTraceIDLength 는 밖에서 들어온 traceId 를 그대로 쓸 수 있는 최대 길이다.
const maxTraceIDLength = 128

// TraceID 는 요청마다 traceId 를 정해 context 에 싣는다.
//
//   - 들어온 X-Request-Id 가 쓸 만하면 그대로 쓴다(게이트웨이·프론트가 넘긴 값을 잇는다). 아니면 새로 만든다.
//   - 응답 헤더 X-Request-Id 로 돌려준다.
//   - 요청의 context 에 싣는다 → 이후 핸들러·서비스·레포가 ctx 를 넘기는 한 어디서든 꺼낼 수 있다.
//
// 다른 미들웨어보다 먼저 등록한다. 뒤에 등록된 것만 traceId 가 실린 ctx 를 본다.
func TraceID() gin.HandlerFunc {
	return func(c *gin.Context) {
		traceID := c.GetHeader(HeaderRequestID)
		if !isUsableTraceID(traceID) {
			traceID = newTraceID()
		}

		// 헤더는 c.Next() 전에 쓴다. 핸들러가 본문을 쓰기 시작하면 헤더는 이미 나간 뒤다.
		c.Header(HeaderRequestID, traceID)

		// context 는 불변이라 "요청의 ctx 에 값을 추가" 할 수 없다.
		// traceId 를 실은 새 ctx 를 만들고, 그 ctx 를 가진 새 *http.Request 로 갈아 끼운다.
		ctx := requestctx.WithTraceID(c.Request.Context(), traceID)
		c.Request = c.Request.WithContext(ctx)

		c.Next()
	}
}

// isUsableTraceID 는 밖에서 온 값을 로그와 응답 헤더에 그대로 실어도 되는지 본다.
// 비어 있지 않고, 너무 길지 않고, 공백·제어 문자가 없는 ASCII 여야 한다.
func isUsableTraceID(traceID string) bool {
	if traceID == "" || len(traceID) > maxTraceIDLength {
		return false
	}

	for i := 0; i < len(traceID); i++ {
		if traceID[i] <= ' ' || traceID[i] > '~' {
			return false
		}
	}

	return true
}

// newTraceID 는 UUID v4 형식의 문자열을 만든다. 표준 라이브러리만 쓴다.
func newTraceID() string {
	var b [16]byte
	// crypto/rand.Read 는 Go 1.24 부터 에러를 돌려주지 않는다(실패하면 프로그램이 멈춘다).
	_, _ = rand.Read(b[:])

	b[6] = (b[6] & 0x0f) | 0x40 // 버전 4
	b[8] = (b[8] & 0x3f) | 0x80 // RFC 4122 variant

	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}
