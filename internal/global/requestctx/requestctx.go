// Package requestctx 는 요청 하나에 딸린 값을 context.Context 에 싣고 꺼낸다.
//
// nest-boiler 의 RequestContext(AsyncLocalStorage)에 해당한다. 차이는 전파 방식이다.
// ALS 는 한 번 열어 두면 그 아래 비동기 호출 어디서든 "암묵적으로" 꺼낼 수 있지만,
// Go 에는 그런 장치가 없다. ctx 를 함수의 첫 인자로 계속 넘겨야 하고, 넘기지 않은 곳에서는 꺼낼 수 없다.
package requestctx

import "context"

// traceIDKey 는 context 에 값을 넣을 때 쓰는 키다.
// 내보내지 않는 전용 타입이라, 다른 패키지가 같은 키를 만들어 값을 덮어쓰거나 꺼낼 수 없다.
// 문자열 키("traceId")를 쓰면 누구든 같은 문자열로 충돌시킬 수 있다.
type traceIDKey struct{}

// WithTraceID 는 traceID 를 실은 새 context 를 돌려준다.
// context 는 불변이다 — 받은 ctx 를 고치지 않고, 그것을 감싼 새 값을 만든다.
func WithTraceID(ctx context.Context, traceID string) context.Context {
	return context.WithValue(ctx, traceIDKey{}, traceID)
}

// TraceID 는 ctx 에 실린 traceID 를 돌려준다. 없으면 빈 문자열.
func TraceID(ctx context.Context) string {
	// ctx.Value 는 any 를 돌려준다. ", _" 형태의 타입 단언은 실패해도 패닉 없이 zero value("")를 준다.
	traceID, _ := ctx.Value(traceIDKey{}).(string)

	return traceID
}
