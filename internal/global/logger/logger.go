// Package logger 는 환경에 맞는 slog 로거를 만든다.
package logger

import (
	"log/slog"
	"os"

	"github.com/WoodrowDy/go-gin-boiler/internal/global/config"
)

// New 는 env 에 맞는 핸들러로 *slog.Logger 를 만들어 돌려준다.
// local·test → TextHandler(사람이 읽기 좋게), 그 외 → JSONHandler(수집기가 파싱).
//
// config 전체가 아니라 env 문자열만 받는다 — 로거가 아는 것은 "어느 환경인가" 하나면 충분하다.
func New(env string) *slog.Logger {
	readable := env == config.EnvLocal || env == config.EnvTest

	// 최소 로그 레벨. local·test 는 Debug 까지, 그 외는 Info 부터.
	level := slog.LevelInfo
	if readable {
		level = slog.LevelDebug
	}

	opts := &slog.HandlerOptions{Level: level}

	// 구체 타입(*slog.TextHandler / *slog.JSONHandler)이 서로 달라서,
	// 둘 다 담을 수 있는 인터페이스 타입으로 선언한다.
	var handler slog.Handler
	if readable {
		handler = slog.NewTextHandler(os.Stdout, opts)
	} else {
		handler = slog.NewJSONHandler(os.Stdout, opts)
	}

	return slog.New(handler)
}
