package logger

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"strings"
	"testing"

	"github.com/WoodrowDy/go-gin-boiler/internal/global/config"
	"github.com/WoodrowDy/go-gin-boiler/internal/global/requestctx"
)

func TestNewHandler_Format(t *testing.T) {
	tests := []struct {
		name     string
		env      string
		wantJSON bool
	}{
		{name: "local 은 텍스트", env: config.EnvLocal, wantJSON: false},
		{name: "test 는 텍스트", env: config.EnvTest, wantJSON: false},
		{name: "dev 는 JSON", env: config.EnvDev, wantJSON: true},
		{name: "prod 는 JSON", env: config.EnvProd, wantJSON: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			slog.New(newHandler(&buf, tt.env, false)).Info("hello", "addr", ":8080")

			if gotJSON := json.Valid(buf.Bytes()); gotJSON != tt.wantJSON {
				t.Errorf("JSON 여부 = %v, want %v\n출력: %s", gotJSON, tt.wantJSON, buf.String())
			}
		})
	}
}

func TestNewHandler_Level(t *testing.T) {
	tests := []struct {
		name      string
		env       string
		wantDebug bool
	}{
		{name: "local 은 Debug 를 찍는다", env: config.EnvLocal, wantDebug: true},
		{name: "prod 는 Debug 를 버린다", env: config.EnvProd, wantDebug: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			slog.New(newHandler(&buf, tt.env, false)).Debug("detail")

			if gotDebug := buf.Len() > 0; gotDebug != tt.wantDebug {
				t.Errorf("Debug 출력 여부 = %v, want %v", gotDebug, tt.wantDebug)
			}
		})
	}
}

func TestNewHandler_TraceID(t *testing.T) {
	ctx := requestctx.WithTraceID(context.Background(), "abc-123")

	tests := []struct {
		name string
		log  func(l *slog.Logger)
		want bool
	}{
		{name: "ctx 를 넘기면 붙는다", log: func(l *slog.Logger) { l.InfoContext(ctx, "hello") }, want: true},
		{name: "ctx 없이 부르면 붙지 않는다", log: func(l *slog.Logger) { l.Info("hello") }, want: false},
		{
			name: "With 로 만든 로거에도 붙는다",
			log:  func(l *slog.Logger) { l.With("domain", "users").InfoContext(ctx, "hello") },
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			tt.log(slog.New(newHandler(&buf, config.EnvLocal, false)))

			if got := strings.Contains(buf.String(), "traceId=abc-123"); got != tt.want {
				t.Errorf("traceId 포함 여부 = %v, want %v\n출력: %s", got, tt.want, buf.String())
			}
		})
	}
}
