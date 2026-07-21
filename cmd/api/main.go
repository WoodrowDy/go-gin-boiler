package main

// import는 표준 라이브러리 / 서드파티를 빈 줄로 나누는 게 관례다.
// gofmt가 각 블록 안을 알파벳순으로 정렬해준다.
import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
)

// TODO(step-2): addr을 config 구조체로 옮긴다 (APP_ENV별 .env)
const addr = ":8080"

// shutdown이 진행 중인 요청을 기다려주는 최대 시간.
// 이 시간을 넘기면 포기하고 강제 종료한다.
const shutdownTimeout = 10 * time.Second

// newRouter는 라우터를 조립해서 반환한다.
// main()과 테스트가 둘 다 이 함수를 부른다 — 함수로 뽑은 유일한 이유.
func newRouter() *gin.Engine {
	// TODO(step-3): gin.New() + Recovery + slog 로거 미들웨어로 교체.
	//               Default()가 붙여주는 Logger는 traceId를 모르고 비구조화 텍스트를 뿌린다.
	r := gin.Default() // = gin.New() + Logger + Recovery

	r.GET("/ping", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"message": "pong",
		})
	})

	return r
}

func main() {
	// ① 시그널을 context로 바꾼다.
	//    이 줄은 OS에 "SIGINT/SIGTERM은 내가 처리한다"고 등록하는 것이기도 하다.
	//    → 등록하는 순간 기본 동작(프로세스 즉시 종료)이 꺼진다.
	//    ctx  : 시그널이 오면 취소되는 context
	//    stop : 등록을 해제하는 함수
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)

	// defer stop()을 빼먹으면 안 된다.
	// 해제해야 기본 동작이 돌아온다 → shutdown이 늦어질 때 사용자가 Ctrl+C를 다시 눌러
	// 강제 종료할 수 있다.
	defer stop() // <- 이 줄이 없으면 shutdown이 끝나도 프로세스가 안 죽는다. (기본 동작이 꺼져 있으므로)

	// ② 서버를 직접 만든다. router.Run()을 쓰지 않는 이유는 shutdown 훅이 없기 때문.
	srv := &http.Server{
		Addr: addr,

		// *gin.Engine이 http.Handler 인터페이스(= ServeHTTP 메서드 하나)를 만족해서
		// 여기 그대로 들어간다. gin 쪽에 "implements" 선언 같은 건 없다.
		Handler: newRouter(),

		// TODO(step-2): 타임아웃 값들도 config로 뺄지 판단
		// Go의 기본값은 전부 0 = 무제한이다. 안 넣으면 느린 클라이언트가 커넥션을 무한정 붙잡는다.
		ReadHeaderTimeout: 5 * time.Second,  // 헤더까지 읽는 최대 시간 (Slowloris 방어. gosec가 이걸 콕 집어 검사한다)
		ReadTimeout:       5 * time.Second,  // 요청 전체를 읽는 최대 시간
		WriteTimeout:      10 * time.Second, // 응답을 쓰는 최대 시간
		IdleTimeout:       15 * time.Second, // keep-alive 커넥션을 놀린 채 유지할 시간
	}

	// TODO(step-3): log → slog
	log.Printf("starting server on %s", addr)

	// ③ 서버를 별도 고루틴에서 띄운다.
	//    ListenAndServe()는 서버가 죽을 때까지 리턴하지 않는 블로킹 호출이라,
	//    main에서 직접 부르면 아래 shutdown 코드에 영원히 도달하지 못한다.
	go func() {
		err := srv.ListenAndServe()

		// ErrServerClosed는 "Shutdown()이 불려서 정상 종료했다"는 신호값이지 에러가 아니다.
		// == 대신 errors.Is를 쓰는 이유: Go는 fmt.Errorf("...: %w", err)로 에러를 감싸는 게
		// 관례라, 감싸인 에러는 ==로 못 잡는다. errors.Is는 껍질을 벗겨가며 비교한다.
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			// log.Fatalf를 쓰면 안 된다. os.Exit(1)을 직접 불러서 위의 defer stop()이
			// 통째로 건너뛰어진다.
			log.Printf("listen: %v", err)
		}
	}() // ← 이 () 가 "지금 호출한다". go 뒤에는 반드시 호출이 와야 한다.

	// ④ main은 여기서 멈춰 선다.
	//    ctx.Done()은 채널을 반환하는 "메서드"이고, 기다리는 일은 <- 연산자가 한다.
	//    <- 를 빠뜨리면 컴파일은 되지만 아무것도 안 기다리고 그냥 지나간다.
	//    Go는 main 고루틴이 리턴하면 다른 고루틴이 뭘 하든 프로그램이 즉사하므로,
	//    main이 반드시 어딘가에서 블로킹하고 있어야 한다.
	<-ctx.Done()

	log.Println("shutting down...")

	// ⑤ 종료 대기용 context를 새로 판다.
	//    ⚠️ 위의 ctx를 넘기면 안 된다. 방금 시그널로 이미 취소된 상태라
	//       넘기는 즉시 데드라인 초과로 취급돼 아무것도 안 기다린다. 가장 흔한 버그.
	shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()

	// ⑥ Shutdown이 하는 일:
	//    1) 리스너를 닫아 새 요청을 안 받는다
	//    2) 놀고 있는 keep-alive 커넥션을 정리한다
	//    3) 처리 중인 요청이 끝나기를 기다린다  ← 이게 핵심
	if err := srv.Shutdown(shutdownCtx); err != nil {
		// 여기 오면 shutdownTimeout 안에 요청들이 안 끝났다는 뜻이다.
		log.Printf("forced shutdown: %v", err)
	}

	log.Println("server stopped")
}