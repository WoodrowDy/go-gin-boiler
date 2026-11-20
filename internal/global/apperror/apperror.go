// Package apperror 는 "예상한 실패" 를 나타내는 에러 타입이다.
//
// 에러는 출처가 아니라 의미로 나눈다.
//
//   - 예상한 것(업무 규칙) — 없음, 잘못된 상태, 중복. service 가 apperror 를 돌려준다.
//     에러 미들웨어가 그 상태 코드와 메시지를 그대로 응답에 싣는다.
//   - 예상 못 한 것(장애)  — 연결 끊김, 타임아웃, 버그. apperror 로 감싸지 않고 그대로 올린다.
//     에러 미들웨어가 500 으로 바꾸고, 내용은 로그에만 남긴다.
//
// nest-boiler 에서 service 가 던지던 NotFoundException · ConflictException 에 해당한다.
// Go 에는 throw 가 없다. 에러는 반환값이고, 받는 쪽이 if err != nil 로 확인해 위로 넘긴다.
package apperror

import "net/http"

// Error 는 HTTP 상태 코드와 클라이언트에게 보여줄 메시지를 가진 에러다.
type Error struct {
	Status  int
	Message string
}

// Error 메서드가 있으면 error 인터페이스를 만족한다. "implements" 선언은 없다.
func (e *Error) Error() string {
	return e.Message
}

// New 는 상태 코드를 직접 정해 만든다. 아래 이름 붙은 함수에 없는 코드에 쓴다.
func New(status int, message string) *Error {
	return &Error{Status: status, Message: message}
}

// BadRequest 는 400 — 요청이 잘못됐다.
func BadRequest(message string) *Error {
	return New(http.StatusBadRequest, message)
}

// Unauthorized 는 401 — 누구인지 모른다.
func Unauthorized(message string) *Error {
	return New(http.StatusUnauthorized, message)
}

// Forbidden 은 403 — 누구인지는 알지만 권한이 없다.
func Forbidden(message string) *Error {
	return New(http.StatusForbidden, message)
}

// NotFound 는 404 — 대상이 없다.
func NotFound(message string) *Error {
	return New(http.StatusNotFound, message)
}

// Conflict 는 409 — 지금 상태와 충돌한다(중복, 잘못된 상태 전이, 참조 중 삭제).
func Conflict(message string) *Error {
	return New(http.StatusConflict, message)
}
