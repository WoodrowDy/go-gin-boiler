package middleware

import (
	"log/slog"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

// sensitiveHeaders 는 값을 로그에 남기지 않는 헤더다.
// 허용 목록이 아니라 차단 목록이다 — 허용 목록이면 디버깅에 필요한 헤더가 조용히 사라진다.
// 키는 http.CanonicalHeaderKey 형태로 적는다(http.Header 의 키가 그 형태다).
var sensitiveHeaders = map[string]struct{}{
	"Authorization":       {},
	"Proxy-Authorization": {},
}

const maskedValue = "***"

// AccessLog 는 요청마다 로그 두 줄을 남긴다.
//
//   - HTTP REQ — 들어올 때. 메서드, 경로, 클라이언트 IP, 헤더(자격증명은 가린다).
//   - HTTP RES — 나갈 때. 상태 코드와 소요 시간. 레벨은 상태로 정한다: 5xx error, 4xx warn, 그 외 info.
//
// traceId 는 여기서 붙이지 않는다. ctx 를 넘기면 로거의 핸들러가 붙인다.
// 그래서 TraceID 미들웨어 뒤에 등록해야 한다.
//
// skipPaths 에 적은 경로는 남기지 않는다(헬스체크처럼 주기적으로 들어와 로그를 덮는 것).
func AccessLog(skipPaths ...string) gin.HandlerFunc {
	skip := make(map[string]struct{}, len(skipPaths))
	for _, path := range skipPaths {
		skip[path] = struct{}{}
	}

	return func(c *gin.Context) {
		if _, ok := skip[c.Request.URL.Path]; ok {
			c.Next()
			return
		}

		start := time.Now()
		ctx := c.Request.Context()
		method := c.Request.Method
		path := c.Request.URL.RequestURI() // 쿼리 문자열 포함

		slog.InfoContext(ctx, "HTTP REQ",
			"method", method,
			"path", path,
			"clientIp", c.ClientIP(),
			headersAttr(c.Request.Header),
		)

		// c.Next() 는 뒤에 등록된 미들웨어와 핸들러를 전부 실행하고 돌아온다.
		// 이 줄 위는 "요청이 들어올 때", 아래는 "응답이 나갈 때" 다.
		c.Next()

		status := c.Writer.Status()

		slog.Log(ctx, levelForStatus(status), "HTTP RES",
			"method", method,
			"path", path,
			"status", status,
			"durationMs", time.Since(start).Milliseconds(),
		)
	}
}

func levelForStatus(status int) slog.Level {
	switch {
	case status >= http.StatusInternalServerError:
		return slog.LevelError
	case status >= http.StatusBadRequest:
		return slog.LevelWarn
	default:
		return slog.LevelInfo
	}
}

// headersAttr 는 헤더를 "headers" 그룹 하나로 묶는다.
// 텍스트 로그에서는 headers.User-Agent=..., JSON 에서는 {"headers":{"User-Agent":...}} 로 찍힌다.
func headersAttr(header http.Header) slog.Attr {
	// map 은 순회 순서가 매번 다르다. 로그가 실행마다 달라지지 않게 키를 정렬한다.
	keys := make([]string, 0, len(header))
	for key := range header {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	attrs := make([]slog.Attr, 0, len(keys))
	for _, key := range keys {
		value := strings.Join(header[key], ", ")
		if _, ok := sensitiveHeaders[key]; ok {
			value = maskHeaderValue(value)
		}
		attrs = append(attrs, slog.String(key, value))
	}

	return slog.Attr{Key: "headers", Value: slog.GroupValue(attrs...)}
}

// maskHeaderValue 는 스킴(Bearer, Basic)만 남기고 가린다. 스킴은 비밀이 아니다.
// 스킴이 없으면 앞부분이 곧 비밀이라 통째로 가린다.
func maskHeaderValue(value string) string {
	scheme, _, found := strings.Cut(value, " ")
	if !found || scheme == "" {
		return maskedValue
	}

	return scheme + " " + maskedValue
}
