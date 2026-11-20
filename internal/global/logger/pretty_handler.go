package logger

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log/slog"
	"strconv"
	"strings"
	"sync"
	"time"
)

// ANSI 색 코드. 터미널이 이 바이트열을 만나면 글자색을 바꾼다.
const (
	ansiReset  = "\x1b[0m"
	ansiDim    = "\x1b[2m"
	ansiRed    = "\x1b[31m"
	ansiGreen  = "\x1b[32m"
	ansiYellow = "\x1b[33m"
	ansiBlue   = "\x1b[34m"
	ansiCyan   = "\x1b[36m"
)

// prettyHandler 는 local 에서 사람이 읽기 좋은 한 줄을 찍는다.
//
//	16:05:27.032 INFO  starting server addr=:8080
//	16:05:29.309 INFO  200 |     157µs |             ::1 | GET     /ping  traceId=a7211c13-…
//	16:05:29.341 WARN  404 |      16µs |             ::1 | GET     /nope  traceId=c8587b53-…
//
// 접근 로그(HTTP RES)는 Gin 기본 로거처럼 상태·시간·IP·메서드·경로를 한 줄에 세운다.
// 찍는 값은 JSON 로그와 같다 — 모양만 다르다.
type prettyHandler struct {
	w     io.Writer
	mu    *sync.Mutex // WithAttrs 로 복제된 핸들러들이 같은 w 에 쓰므로 뮤텍스를 공유한다
	level slog.Leveler
	color bool

	attrs  []field // WithAttrs 로 미리 붙은 필드
	prefix string  // WithGroup 으로 쌓인 접두어 ("group." 형태)
}

type field struct {
	key   string
	value slog.Value
}

func newPrettyHandler(w io.Writer, level slog.Leveler, color bool) *prettyHandler {
	return &prettyHandler{w: w, mu: &sync.Mutex{}, level: level, color: color}
}

func (h *prettyHandler) Enabled(_ context.Context, level slog.Level) bool {
	return level >= h.level.Level()
}

func (h *prettyHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	clone := *h
	clone.attrs = append([]field{}, h.attrs...) // 원본 슬라이스를 건드리지 않게 복사
	for _, attr := range attrs {
		clone.attrs = appendField(clone.attrs, h.prefix, attr)
	}

	return &clone
}

func (h *prettyHandler) WithGroup(name string) slog.Handler {
	if name == "" {
		return h
	}

	clone := *h
	clone.prefix = h.prefix + name + "."

	return &clone
}

func (h *prettyHandler) Handle(_ context.Context, record slog.Record) error {
	fields := append([]field{}, h.attrs...)
	record.Attrs(func(attr slog.Attr) bool {
		fields = appendField(fields, h.prefix, attr)
		return true
	})

	var buf bytes.Buffer

	if !record.Time.IsZero() {
		buf.WriteString(h.paint(ansiDim, record.Time.Format("15:04:05.000")))
		buf.WriteByte(' ')
	}
	buf.WriteString(h.paint(levelColor(record.Level), fmt.Sprintf("%-5s", record.Level.String())))
	buf.WriteByte(' ')

	if record.Message == MsgHTTPResponse {
		fields = h.writeAccess(&buf, fields)
	} else {
		buf.WriteString(record.Message)
	}

	for _, f := range fields {
		buf.WriteByte(' ')
		buf.WriteString(h.paint(ansiDim, f.key+"="))
		buf.WriteString(formatValue(f.value))
	}
	buf.WriteByte('\n')

	// 한 줄을 다 만든 뒤에 한 번에 쓴다. 여러 고루틴의 줄이 중간에 섞이지 않는다.
	h.mu.Lock()
	defer h.mu.Unlock()
	_, err := h.w.Write(buf.Bytes())

	return err
}

// writeAccess 는 접근 로그의 정해진 필드를 한 줄로 세우고, 쓰지 않은 나머지 필드를 돌려준다.
func (h *prettyHandler) writeAccess(buf *bytes.Buffer, fields []field) []field {
	var (
		status   int64
		duration time.Duration
		method   string
		path     string
		clientIP string
		rest     []field
	)

	for _, f := range fields {
		switch f.key {
		case "status":
			status = toInt64(f.value)
		case "durationMs":
			duration = time.Duration(toFloat64(f.value) * float64(time.Millisecond))
		case "method":
			method = f.value.String()
		case "path":
			path = f.value.String()
		case "clientIp":
			clientIP = f.value.String()
		default:
			rest = append(rest, f)
		}
	}

	buf.WriteString(h.paint(statusColor(status), strconv.FormatInt(status, 10)))
	fmt.Fprintf(buf, " | %9s | %15s | %-7s %s ", roundDuration(duration), clientIP, method, path)

	return rest
}

func (h *prettyHandler) paint(color, text string) string {
	if !h.color || color == "" {
		return text
	}

	return color + text + ansiReset
}

// appendField 는 attr 하나를 필드로 편다. 그룹은 "그룹.키" 로 풀어 넣는다.
func appendField(fields []field, prefix string, attr slog.Attr) []field {
	// LogValuer 를 구현한 값은 여기서 실제 값으로 바뀐다.
	value := attr.Value.Resolve()

	if value.Kind() == slog.KindGroup {
		groupPrefix := prefix
		if attr.Key != "" {
			groupPrefix += attr.Key + "."
		}
		for _, inner := range value.Group() {
			fields = appendField(fields, groupPrefix, inner)
		}

		return fields
	}

	if attr.Key == "" {
		return fields
	}

	return append(fields, field{key: prefix + attr.Key, value: value})
}

// formatValue 는 값을 한 줄에 안전하게 넣는다. 공백·따옴표·등호가 있으면 따옴표로 감싼다.
func formatValue(value slog.Value) string {
	text := value.String()
	if text == "" || strings.ContainsAny(text, " \t\n\"=") {
		return strconv.Quote(text)
	}

	return text
}

// roundDuration 은 자릿수를 줄인다. 157.958µs → 158µs, 12.345678ms → 12.35ms.
func roundDuration(d time.Duration) time.Duration {
	switch {
	case d >= time.Second:
		return d.Round(10 * time.Millisecond)
	case d >= time.Millisecond:
		return d.Round(10 * time.Microsecond)
	default:
		return d.Round(time.Microsecond)
	}
}

func toInt64(value slog.Value) int64 {
	switch value.Kind() {
	case slog.KindInt64:
		return value.Int64()
	case slog.KindUint64:
		return int64(value.Uint64())
	case slog.KindFloat64:
		return int64(value.Float64())
	default:
		return 0
	}
}

func toFloat64(value slog.Value) float64 {
	switch value.Kind() {
	case slog.KindFloat64:
		return value.Float64()
	case slog.KindInt64:
		return float64(value.Int64())
	case slog.KindUint64:
		return float64(value.Uint64())
	default:
		return 0
	}
}

func levelColor(level slog.Level) string {
	switch {
	case level >= slog.LevelError:
		return ansiRed
	case level >= slog.LevelWarn:
		return ansiYellow
	case level >= slog.LevelInfo:
		return ansiBlue
	default:
		return ansiDim
	}
}

func statusColor(status int64) string {
	switch {
	case status >= 500:
		return ansiRed
	case status >= 400:
		return ansiYellow
	case status >= 300:
		return ansiCyan
	default:
		return ansiGreen
	}
}
