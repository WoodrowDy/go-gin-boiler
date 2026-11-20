package logger

import (
	"bytes"
	"context"
	"log/slog"
	"strings"
	"testing"

	"github.com/WoodrowDy/go-gin-boiler/internal/global/config"
	"github.com/WoodrowDy/go-gin-boiler/internal/global/requestctx"
)

func TestPrettyHandler_Lines(t *testing.T) {
	ctx := requestctx.WithTraceID(context.Background(), "abc-123")

	tests := []struct {
		name string
		log  func(l *slog.Logger)
		want []string // 출력에 이 조각들이 모두 있어야 한다
	}{
		{
			name: "보통 로그는 메시지와 key=value",
			log:  func(l *slog.Logger) { l.Info("starting server", "addr", ":8080") },
			want: []string{"INFO ", "starting server addr=:8080"},
		},
		{
			name: "공백이 있는 값은 따옴표로 감싼다",
			log:  func(l *slog.Logger) { l.Error("listen failed", "err", "address already in use") },
			want: []string{"ERROR", `err="address already in use"`},
		},
		{
			name: "그룹은 점으로 편다",
			log: func(l *slog.Logger) {
				l.Info("hello", slog.Group("headers", slog.String("Accept", "*/*")))
			},
			want: []string{"headers.Accept=*/*"},
		},
		{
			name: "접근 로그는 한 줄로 세운다",
			log: func(l *slog.Logger) {
				l.InfoContext(ctx, MsgHTTPResponse,
					"method", "GET", "path", "/ping", "status", 200, "durationMs", 0.157958, "clientIp", "::1")
			},
			want: []string{"200 |     158µs |             ::1 | GET     /ping", "traceId=abc-123"},
		},
		{
			name: "밀리초 단위 응답 시간",
			log: func(l *slog.Logger) {
				l.Warn(MsgHTTPResponse,
					"method", "DELETE", "path", "/users/1", "status", 404, "durationMs", 12.345678, "clientIp", "10.0.0.1")
			},
			want: []string{"WARN ", "404 |   12.35ms |        10.0.0.1 | DELETE  /users/1"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			tt.log(slog.New(newHandler(&buf, config.EnvLocal, false)))

			got := buf.String()
			if strings.Count(got, "\n") != 1 {
				t.Errorf("한 줄이어야 한다\n출력: %q", got)
			}
			for _, want := range tt.want {
				if !strings.Contains(got, want) {
					t.Errorf("%q 가 없다\n출력: %s", want, got)
				}
			}
			if strings.Contains(got, "\x1b[") {
				t.Errorf("색을 껐는데 색 코드가 있다\n출력: %q", got)
			}
		})
	}
}

func TestPrettyHandler_Color(t *testing.T) {
	tests := []struct {
		name   string
		status int
		want   string
	}{
		{name: "2xx 는 초록", status: 200, want: ansiGreen + "200" + ansiReset},
		{name: "4xx 는 노랑", status: 404, want: ansiYellow + "404" + ansiReset},
		{name: "5xx 는 빨강", status: 500, want: ansiRed + "500" + ansiReset},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			slog.New(newHandler(&buf, config.EnvLocal, true)).
				Info(MsgHTTPResponse, "method", "GET", "path", "/", "status", tt.status, "durationMs", 1.0, "clientIp", "::1")

			if got := buf.String(); !strings.Contains(got, tt.want) {
				t.Errorf("상태 코드 색이 다르다\n출력: %q", got)
			}
		})
	}
}

func TestPrettyHandler_WithAttrsAndGroup(t *testing.T) {
	var buf bytes.Buffer
	base := slog.New(newHandler(&buf, config.EnvLocal, false))

	base.With("domain", "users").WithGroup("req").Info("hello", "id", 7)

	got := buf.String()
	for _, want := range []string{"domain=users", "req.id=7"} {
		if !strings.Contains(got, want) {
			t.Errorf("%q 가 없다\n출력: %s", want, got)
		}
	}

	// With 로 만든 로거가 원래 로거의 필드를 오염시키지 않는다.
	buf.Reset()
	base.Info("plain")
	if strings.Contains(buf.String(), "domain=") {
		t.Errorf("원래 로거에 필드가 새어 들어갔다\n출력: %s", buf.String())
	}
}
