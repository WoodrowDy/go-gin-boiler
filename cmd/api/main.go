package main

// import는 표준 라이브러리 / 서드파티를 빈 줄로 나누는 게 관례다.
// gofmt가 각 블록 안을 알파벳순으로 정렬해준다.
import (
	"context"
	"errors"
	"log"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/WoodrowDy/go-gin-boiler/internal/database"
	"github.com/WoodrowDy/go-gin-boiler/internal/global/config"
	"github.com/WoodrowDy/go-gin-boiler/internal/global/logger"
	"github.com/WoodrowDy/go-gin-boiler/internal/global/middleware"
	"github.com/gin-gonic/gin"
)

// newRouter는 라우터를 조립해서 반환한다.
// main()과 테스트가 둘 다 이 함수를 부른다 — 함수로 뽑은 유일한 이유.
func newRouter(env string) *gin.Engine {
	// ★ Step 3-3: gin.Default() 대신 gin.New() — 미들웨어가 하나도 없는 엔진에서 시작해 직접 붙인다.
	//    Default() 가 붙여 주는 Logger 는 traceId 를 모르고 비구조화 텍스트를 뿌린다.
	r := gin.New()

	// 등록 순서가 곧 실행 순서다. 요청은 위에서 아래로 들어가고, 응답은 아래에서 위로 나온다.
	//   TraceID   — 맨 앞. 뒤의 모든 것이 traceId 가 실린 ctx 를 본다.
	//   AccessLog — 접근 로그. Recovery 바깥이라 panic 이 500 으로 바뀐 뒤의 상태를 읽는다.
	//               local 은 HTTP RES 한 줄만(터미널에서 읽기 좋게), 그 외는 HTTP REQ 도 남긴다(헤더 포함).
	//   Recovery  — panic 을 잡아 500 으로 바꾼다.
	//   ErrorHandler — 핸들러에 가장 가깝게. 핸들러가 c.Error(err) 로 남긴 에러를 응답 규약으로 바꾼다.
	r.Use(
		middleware.TraceID(),
		middleware.AccessLog(middleware.AccessLogOptions{
			LogRequest: env != config.EnvLocal,
		}),
		middleware.Recovery(),
		middleware.ErrorHandler(),
	)

	// 등록되지 않은 경로의 404 도 같은 응답 규약으로.
	r.NoRoute(middleware.NoRoute())

	// /ping 은 응답 규약({ row, meta }) 밖에 둔다 — 의도한 예외다.
	// 로드밸런서·오케스트레이터가 찌르는 probe 용이라, 받는 쪽은 상태 코드만 본다.
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
	defer stop() // 해제해야 기본 동작이 돌아온다 → shutdown이 늦어질 때 Ctrl+C를 다시 눌러 강제 종료 가능

	// ★ Step 2: 설정을 먼저 로드한다.
	//    config.Load()는 "값"을 돌려주는 문장이라, 아래 http.Server{...} 리터럴 "안"에는 못 들어간다.
	//    struct 리터럴의 { } 는 코드 블록이 아니라 "필드: 값" 목록이기 때문. 그래서 여기 밖에서 먼저 부른다.
	cfg, err := config.Load()
	if err != nil {
		// 여긴 defer가 아직 안 걸린 지점이라 무해하지만 원칙 유지
		log.Printf("config: %v", err)
		os.Exit(1)
	}

	// ★ Step 3-1: 로거는 config 다음에 만든다 — 포맷을 고르려면 cfg.App.Env 가 있어야 한다.
	//    그래서 위의 config 실패 로그만 표준 log 로 남는다(그 시점엔 아직 slog 가 없다).
	//    SetDefault 로 등록해 두면 이후 어디서든 slog.Info(...) 가 이 로거를 쓴다.
	slog.SetDefault(logger.New(cfg.App.Env))

	// Gin 은 기본이 debug 모드라 기동할 때 경고와 라우트 목록을 찍는다. local 에서만 그대로 둔다.
	if cfg.App.Env != config.EnvLocal {
		gin.SetMode(gin.ReleaseMode)
	}

	// ★ Step 5: DB 에 연결한다. 닿지 않으면 뜨지 않는다 — 접속 정보가 틀린 채로 떠서 첫 요청에서 실패하는 것보다 낫다.
	//    db 는 커넥션 하나가 아니라 풀이다. Step 7 에서 도메인을 조립할 때 repository · service 에 넘긴다.
	db, err := database.Open(ctx, cfg.DB)
	if err != nil {
		slog.Error("database", "err", err)
		os.Exit(1)
	}
	slog.Info("database connected", "host", cfg.DB.Host, "port", cfg.DB.Port, "name", cfg.DB.Name)

	// ② 서버를 직접 만든다. router.Run()을 쓰지 않는 이유는 shutdown 훅이 없기 때문.
	//    이 { } 안에는 "필드: 값" 만 온다.
	srv := &http.Server{
		Addr: cfg.Server.Addr(), // Step 1의 하드코딩 ":8080" → 이제 config가 소유

		// *gin.Engine이 http.Handler 인터페이스(= ServeHTTP 메서드 하나)를 만족해서
		// 여기 그대로 들어간다. gin 쪽에 "implements" 선언 같은 건 없다.
		Handler: newRouter(cfg.App.Env),

		// 타임아웃 4종. ShutdownTimeout은 http.Server의 필드가 아니라서 여기 없고, 아래 ⑤에서 쓴다.
		ReadHeaderTimeout: cfg.Server.ReadHeaderTimeout,
		ReadTimeout:       cfg.Server.ReadTimeout,
		WriteTimeout:      cfg.Server.WriteTimeout,
		IdleTimeout:       cfg.Server.IdleTimeout,
	}

	slog.Info("starting server", "addr", cfg.Server.Addr())

	// ③ 서버를 별도 고루틴에서 띄운다.
	//    ListenAndServe()는 서버가 죽을 때까지 리턴하지 않는 블로킹 호출이라,
	//    main에서 직접 부르면 아래 shutdown 코드에 영원히 도달하지 못한다.
	go func() {
		err := srv.ListenAndServe()

		// ErrServerClosed는 "Shutdown()이 불려서 정상 종료했다"는 신호값이지 에러가 아니다.
		// == 대신 errors.Is를 쓰는 이유: Go는 fmt.Errorf("...: %w", err)로 에러를 감싸는 게
		// 관례라, 감싸인 에러는 ==로 못 잡는다. errors.Is는 껍질을 벗겨가며 비교한다.
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			slog.Error("listen failed", "err", err)
		}
	}() // ← 이 () 가 "지금 호출한다". go 뒤에는 반드시 호출이 와야 한다.

	// ④ main은 여기서 멈춰 선다.
	//    ctx.Done()은 채널을 반환하는 "메서드"이고, 기다리는 일은 <- 연산자가 한다.
	//    Go는 main 고루틴이 리턴하면 다른 고루틴이 뭘 하든 프로그램이 즉사하므로,
	//    main이 반드시 어딘가에서 블로킹하고 있어야 한다.
	<-ctx.Done()

	slog.Info("shutting down")

	// ⑤ 종료 대기용 context를 새로 판다.
	//    ⚠️ 위의 ctx를 넘기면 안 된다. 방금 시그널로 이미 취소된 상태라
	//       넘기는 즉시 데드라인 초과로 취급돼 아무것도 안 기다린다. 가장 흔한 버그.
	//    타임아웃 값은 이제 config가 소유한다 (Step 1의 const shutdownTimeout → cfg.Server.ShutdownTimeout).
	shutdownCtx, cancel := context.WithTimeout(context.Background(), cfg.Server.ShutdownTimeout)
	defer cancel()

	// ⑥ Shutdown이 하는 일:
	//    1) 리스너를 닫아 새 요청을 안 받는다
	//    2) 놀고 있는 keep-alive 커넥션을 정리한다
	//    3) 처리 중인 요청이 끝나기를 기다린다  ← 이게 핵심
	if err := srv.Shutdown(shutdownCtx); err != nil {
		// 여기 오면 ShutdownTimeout 안에 요청들이 안 끝났다는 뜻이다.
		slog.Error("forced shutdown", "err", err)
	}

	// ⑦ DB 는 서버 다음에 닫는다. 순서를 바꾸면 처리 중이던 요청이 닫힌 풀을 만난다.
	if err := database.Close(db); err != nil {
		slog.Error("close database", "err", err)
	}

	slog.Info("server stopped")
}
