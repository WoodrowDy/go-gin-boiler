// Package logger 는 환경에 맞는 slog 로거를 만든다.
package logger

import (
	"io"
	"log/slog"
	"os"

	"github.com/WoodrowDy/go-gin-boiler/internal/global/config"
)

// 접근 로그의 메시지. 미들웨어가 이 이름으로 찍고, local 의 prettyHandler 가 이 이름을 보고 한 줄로 그린다.
const (
	MsgHTTPRequest  = "HTTP REQ"
	MsgHTTPResponse = "HTTP RES"
)

// New 는 env 에 맞는 핸들러로 *slog.Logger 를 만들어 돌려준다.
//
//   - local·test → 사람이 읽는 한 줄 포맷(prettyHandler). 터미널이면 색을 넣는다.
//   - 그 외       → JSONHandler. 수집기가 파싱한다.
//
// config 전체가 아니라 env 문자열만 받는다 — 로거가 아는 것은 "어느 환경인가" 하나면 충분하다.
func New(env string) *slog.Logger {
	return slog.New(newHandler(os.Stdout, env, useColor(os.Stdout)))
}

// newHandler 는 출력 대상을 인자로 받는다. 테스트가 os.Stdout 대신 버퍼를 넣어 출력을 확인한다.
func newHandler(w io.Writer, env string, color bool) slog.Handler {
	readable := env == config.EnvLocal || env == config.EnvTest

	// 최소 로그 레벨. local·test 는 Debug 까지, 그 외는 Info 부터.
	level := slog.LevelInfo
	if readable {
		level = slog.LevelDebug
	}

	// 구체 타입(*prettyHandler / *slog.JSONHandler)이 서로 달라서,
	// 둘 다 담을 수 있는 인터페이스 타입으로 선언한다.
	var handler slog.Handler
	if readable {
		handler = newPrettyHandler(w, level, color)
	} else {
		handler = slog.NewJSONHandler(w, &slog.HandlerOptions{Level: level})
	}

	// 포맷을 정한 핸들러를 한 겹 감싼다 — 모든 로그 줄에 traceId 를 붙이는 층.
	return traceHandler{next: handler}
}

// useColor 는 색을 넣어도 되는지 본다. 터미널에 직접 찍을 때만 넣는다.
// 파일이나 파이프로 보낼 때 색 코드는 깨진 글자로 남는다. NO_COLOR 가 있으면 끈다(https://no-color.org).
func useColor(f *os.File) bool {
	if os.Getenv("NO_COLOR") != "" {
		return false
	}

	info, err := f.Stat()
	if err != nil {
		return false
	}

	return info.Mode()&os.ModeCharDevice != 0
}
