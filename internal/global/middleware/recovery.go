package middleware

import (
	"fmt"
	"log/slog"
	"net/http"
	"runtime/debug"

	"github.com/gin-gonic/gin"
)

// Recovery 는 핸들러에서 난 panic 을 잡아 500 으로 바꾼다.
//
// Go 에는 try/catch 가 없다. 보통의 실패는 error 값으로 돌려주고, panic 은 "여기 올 리 없다" 는
// 버그(nil 역참조, 범위 밖 인덱스)에서만 난다. 그런 버그 하나가 서버 전체를 내리지 않게 막는 층이다.
//
// 스택은 로그에만 남기고 응답에는 싣지 않는다. 둘을 잇는 것은 traceId 다.
//
// AccessLog 뒤에 등록한다. 그래야 여기서 500 으로 바꾼 뒤에 AccessLog 가 그 상태를 읽어 HTTP RES 를 남긴다.
func Recovery() gin.HandlerFunc {
	return func(c *gin.Context) {
		// defer 로 건 함수는 이 함수가 끝날 때 — 정상 종료든 panic 이든 — 반드시 실행된다.
		// recover() 는 defer 안에서만 뜻이 있고, panic 중이면 그 값을, 아니면 nil 을 돌려준다.
		defer func() {
			recovered := recover()
			if recovered == nil {
				return
			}

			slog.ErrorContext(c.Request.Context(), "panic recovered",
				"panic", fmt.Sprint(recovered),
				"stack", string(debug.Stack()),
			)

			// TODO(step-4): 본문을 에러 응답 규약({ message, error, statusCode, meta })으로.
			c.AbortWithStatus(http.StatusInternalServerError)
		}()

		c.Next()
	}
}
