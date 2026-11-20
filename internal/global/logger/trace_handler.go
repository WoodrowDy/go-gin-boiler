package logger

import (
	"context"
	"log/slog"

	"github.com/WoodrowDy/go-gin-boiler/internal/global/requestctx"
)

// traceHandler 는 다른 핸들러를 감싸서, ctx 에 traceId 가 있으면 로그 줄에 붙인다.
//
// nest-boiler 에서는 winston 포맷이 AsyncLocalStorage 에서 traceId 를 꺼내 붙였다.
// 여기서는 호출하는 쪽이 ctx 를 넘겨야 한다 — slog.InfoContext(ctx, ...) 처럼.
// ctx 없이 부른 slog.Info(...) 에는 traceId 가 붙지 않는다(기동·종료 로그가 그렇다).
//
// slog.Handler 는 메서드 4개짜리 인터페이스다. Handle 만 바꾸고 나머지는 그대로 넘긴다.
type traceHandler struct {
	next slog.Handler
}

func (h traceHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return h.next.Enabled(ctx, level)
}

func (h traceHandler) Handle(ctx context.Context, record slog.Record) error {
	if traceID := requestctx.TraceID(ctx); traceID != "" {
		record.AddAttrs(slog.String("traceId", traceID))
	}

	return h.next.Handle(ctx, record)
}

// WithAttrs · WithGroup 은 새 핸들러를 돌려준다. 안쪽 것만 돌려주면 이 층이 벗겨져서
// logger.With(...) 로 만든 로거에는 traceId 가 붙지 않게 된다. 그래서 다시 감싼다.
func (h traceHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return traceHandler{next: h.next.WithAttrs(attrs)}
}

func (h traceHandler) WithGroup(name string) slog.Handler {
	return traceHandler{next: h.next.WithGroup(name)}
}
