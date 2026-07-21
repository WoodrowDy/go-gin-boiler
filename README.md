# go-gin-boiler

Go + Gin + GORM + PostgreSQL 백엔드 보일러플레이트.
[nest-boiler](../nest-boiler)와 **같은 형태(아키텍처·규약)** 를 Go로 이식하는 것이 목표다.

> 이 레포는 학습용으로 진행한다: 각 스텝을 가이드 문서를 보고 **직접 작성**하고,
> 스텝이 끝날 때마다 리뷰를 받는다. (진행 상태는 맨 아래 로드맵 체크박스로 관리)

---

## 스택 매핑 (nest-boiler → go-gin-boiler)

| 역할 | nest-boiler (TS) | go-gin-boiler (Go) | 메모 |
|---|---|---|---|
| 런타임/언어 | Node + TypeScript | Go 1.24+ | 컴파일 언어, 단일 바이너리 배포 |
| 웹 프레임워크 | NestJS 10 | **Gin v1.12** | Nest 같은 DI 컨테이너 없음 → 생성자 주입을 손으로 |
| ORM | TypeORM 0.3 | **GORM v2** | 엔티티=구조체+태그, 소프트삭제 내장 |
| DB | PostgreSQL (Docker, 5433) | PostgreSQL (Docker, **5434**) | nest-boiler(5433)와 충돌 방지 |
| 마이그레이션 | typeorm migration:generate | **golang-migrate** (SQL 파일 직접 작성) | GORM AutoMigrate는 개발 초기용, 운영 규약은 SQL 마이그레이션 |
| 검증 | class-validator (DTO 데코레이터) | **go-playground/validator** (Gin binding 내장) | `binding:"required,min=2"` 태그 |
| 직렬화/변환 | class-transformer @Transform | DTO 메서드/생성자에서 수동 변환 | Go에는 데코레이터가 없다 |
| Swagger | @nestjs/swagger + @ApiDoc | **swaggo/swag** (주석 기반) | 핸들러 위 주석 → `swag init` |
| 로깅 | winston + nest-winston | **slog** (표준 라이브러리) | 외부 의존성 없이 구조화 로깅 |
| traceId 컨텍스트 | AsyncLocalStorage | **context.Context** | ⭐ TS→Go 최대 개념 전환 포인트 |
| 응답 래핑 | ObjectResponse / ListResponse | 동일 규약을 제네릭 구조체로 | `{ row, meta }` / `{ rows, count, meta }` |
| 전역 예외 처리 | (P1 TODO였음) | 에러 미들웨어 + `error` 반환 관례 | Go는 try/catch가 없다 → 여기서 제대로 |
| env 관리 | envs/.env.* + Joi 검증 | **godotenv + caarlos0/env** (구조체 파싱+검증) | `APP_ENV=local\|dev\|prod\|test` |
| 스크립트 러너 | pnpm scripts | **Makefile** | `make run`, `make db-up`, `make test` |
| 핫 리로드 | nest --watch | **air** | 개발 편의 |
| lint/format | eslint + prettier | **golangci-lint + gofmt(gofumpt)** | 레이어 규칙 강제는 depguard로 |
| git 훅 | husky + lint-staged | **lefthook** | pre-commit: lint, pre-push: test |

## 아키텍처

nest-boiler와 동일한 **package-by-domain + 레이어드** (DDD-lite).
단, Go 관례를 따라 레이어를 폴더가 아닌 **파일**로 나눈다
(Go는 폴더=패키지 단위라, 도메인 하나를 잘게 쪼개면 순환 import 지옥이 된다).

```
go-gin-boiler/
├── cmd/
│   └── api/
│       └── main.go              # 부트스트랩만 (nest의 main.ts)
├── internal/                    # 외부 import 불가 (Go 컴파일러가 강제)
│   ├── domain/
│   │   └── staticboard/         # ★ 레퍼런스 도메인 (nest의 static-board 이식)
│   │       ├── handler.go       #   = controller (라우팅 + 바인딩 + 응답변환)
│   │       ├── service.go       #   = service (검증, 트랜잭션 경계)
│   │       ├── repository.go    #   = repository (쿼리만, 엔티티 in/out)
│   │       ├── entity.go        #   = entity (GORM 태그만, JSON/검증 지식 없음)
│   │       ├── dto.go           #   = dtos (request/response + validator 태그)
│   │       ├── mapper.go        #   = mapper (Entity → Response, 무상태)
│   │       └── constants.go     #   = *.constants.ts (에러 메시지 등)
│   ├── global/
│   │   ├── config/              # env 로딩 + 검증 (envs/env.ts + Joi 대응)
│   │   ├── logger/              # slog 설정 (winston.config 대응)
│   │   ├── middleware/          # traceId, 요청/응답 로거, (나중에) jwt
│   │   ├── response/            # ObjectResponse/ListResponse + 에러 응답
│   │   └── apperror/            # 도메인 에러 타입 (NotFound 등 → HTTP 매핑)
│   └── database/
│       ├── database.go          # GORM 연결 (typeorm.config 대응)
│       ├── entity.go            # CoreHard/CoreSoft 대응 공통 필드
│       └── migrations/          # golang-migrate SQL 파일 (up/down 쌍)
├── envs/                        # .env.local 등 (git-ignored) + .example
├── scripts/                     # db-up.sh 등
├── test/                        # e2e (httptest 기반)
├── docker-compose.yml
├── Makefile
├── .golangci.yml
└── lefthook.yml
```

### 왜 이 구조인가 — Go와 레이어드 아키텍처

nest-boiler의 레이어드 아키텍처를 Go로 옮기기 전에 조사한 배경. **결론: "의존성 레이어링"은 Go가 강제하는 것이라 완벽히 맞고, "레이어별 폴더"는 Go와 충돌해서 도메인 패키지 방식으로 변형했다.**

- **의존성이 한 방향으로 흐르는 것(개념적 레이어링)은 Go와 완벽히 맞는다 — 선택이 아니라 강제.**
  Go 컴파일러는 순환 import를 금지하므로 모든 Go 프로그램은 DAG가 될 수밖에 없다.
  "레이어링과 싸우지 말고 그 위에 아키텍처를 지어라"가 Go의 태도.
  handler → service → repository 단방향 의존은 TS에선 eslint로 막았지만 Go에선 컴파일러가 공짜로 보장한다.
  → 근거: [Layered Design in Go (jerf.org)](https://www.jerf.org/iri/post/2025/go_layered_design/)

- **레이어를 폴더(패키지)로 나누는 것은 Go와 안 맞는다.**
  `controllers/`, `services/` 식 레이어별 패키지는 (1) 패키지가 Go의 유일한 캡슐화 단위(대문자=export)라
  도메인 내부 구현을 숨길 곳이 없어지고, (2) 도메인 간 참조가 얽히는 순간 순환 import로 컴파일이 거부된다.
  커뮤니티가 수렴한 답 = **package-by-feature**(도메인당 패키지 하나, 그 안에 레이어 파일). 이 레포 구조의 근거.
  → 근거: [Standard Package Layout (Ben Johnson)](https://medium.com/@benbjohnson/standard-package-layout-7cdbc8391fc1),
  [Why Clean Architecture Struggles in Golang](https://dev.to/lucasdeataides/why-clean-architecture-struggles-in-golang-and-what-works-better-m4g)

- **Clean Architecture식 관행을 그대로 이식하면 마찰이 나는 지점 3가지.**
  ① 모든 레이어에 인터페이스 + DI 컨테이너 → Go는 "인터페이스는 사용하는 쪽에, 필요할 때만" + 생성자 수동 조립이 관례.
  ② 레이어별 mock 단위 테스트 중심 문화 → Go는 통합/e2e 테스트 선호.
  ③ 과도한 추상화 자체가 Go의 단순성 철학과 충돌.
  → 근거: [Flat structure vs. layered architecture (LogRocket)](https://blog.logrocket.com/flat-structure-vs-layered-architecture-structuring-your-go-app/),
  [Why Clean Architecture Struggles in Golang](https://dev.to/lucasdeataides/why-clean-architecture-struggles-in-golang-and-what-works-better-m4g)

## 레이어 규칙 (nest-boiler와 동일, 도구만 다름)

- **entity.go = 영속성 전용.** GORM 태그만. `json:` 태그도 두지 않는다 (응답 계약은 dto.go 소유).
- **DTO가 검증+직렬화를 소유.** request DTO에 `binding:` 태그, response DTO에 `json:` 태그.
  엔티티에서 DTO를 파생하지 않는다.
- **service는 엔티티를 반환**, HTTP 응답 DTO를 모른다. Entity→Response 변환은 handler에서 mapper 호출.
- **repository는 쿼리만.** 404 판단·트랜잭션 경계는 service.
- **트랜잭션 경계는 service.** repo 메서드는 `db *gorm.DB`를 받아서 동작
  → 외부 트랜잭션(`tx`)이 들어오면 합류, 아니면 기본 커넥션. nest-boiler의 `transactionManager?` 패턴과 동일:
  ```go
  // service에서:
  err := s.db.Transaction(func(tx *gorm.DB) error {
      if err := s.repo.Create(ctx, tx, entity); err != nil { return err }
      return s.repo.UpdateCount(ctx, tx, boardID)
  })
  ```
- 규칙의 기계적 강제는 golangci-lint **depguard/gomodguard** 로 (eslint no-restricted-imports 대응).
- **인터페이스는 최소화.** 레이어마다 인터페이스를 미리 깔지 않는다(NestJS 습관 금지).
  구체 타입으로 시작하고, mock이 실제로 필요해지는 순간에 **사용하는 쪽에** 인터페이스를 정의한다.
  → 근거: [Go 공식 코드리뷰 가이드 — Interfaces](https://go.dev/wiki/CodeReviewComments#interfaces),
  [Standard Package Layout (Ben Johnson)](https://medium.com/@benbjohnson/standard-package-layout-7cdbc8391fc1)

---

## 학습 로드맵 (스텝별로 직접 작성)

각 스텝은 "무엇을 / 왜 / TS 개발자용 학습 포인트" 가이드를 받고 직접 구현한다.
한 스텝 = 한 커밋(또는 몇 개) 단위로 끊는다.

- [x] **Step 0 — 환경 준비**: Go 툴체인 정리, golangci-lint·air·swag·migrate CLI 설치, `.gitignore`, 첫 커밋
      · 학습 포인트: GOPATH는 잊어라, go.mod가 package.json이다. `go install`과 `go get`의 차이
- [x] **Step 1 — Gin Hello World + 프로젝트 뼈대**: `cmd/api/main.go`로 이동, health check 엔드포인트, graceful shutdown
      · 학습 포인트: 고루틴 + 시그널 채널(`signal.NotifyContext`) — Node의 `process.on('SIGINT')`과 비교
- [ ] **Step 2 — 설정(config)**: envs/ 구조 이식, `APP_ENV`별 .env 로딩, 구조체 기반 파싱+필수값 검증
      · 학습 포인트: 구조체 태그가 데코레이터의 자리를 대신한다. zero value 개념
- [ ] **Step 3 — 로깅 + traceId**: slog 설정, traceId 미들웨어(X-Request-Id), 요청/응답 로거 미들웨어
      · 학습 포인트: ⭐ `context.Context` — AsyncLocalStorage처럼 "암묵적 전파"가 아니라 **명시적으로 첫 인자로 흘려보낸다**. Go에서 가장 중요한 관례
- [ ] **Step 4 — 응답 규약 + 에러 처리**: 제네릭 `ObjectResponse[T]`/`ListResponse[T]`, apperror 패키지, 에러→HTTP 매핑 미들웨어
      · 학습 포인트: `error`는 값이다. `errors.Is/As`, 래핑(`fmt.Errorf("%w")`). nest에서 P1 TODO였던 전역 예외 필터를 Go에선 처음부터
- [ ] **Step 5 — Docker Postgres + GORM 연결**: docker-compose(포트 5434), scripts/db-up.sh 이식, GORM 연결, Core 공통 엔티티(soft delete 포함)
      · 학습 포인트: GORM의 `gorm.DeletedAt` = TypeORM `@DeleteDateColumn`. 커넥션 풀 설정은 Go에서 명시적
- [ ] **Step 6 — 마이그레이션**: golang-migrate 도입, Makefile 타깃(migrate-new/up/down), 첫 테이블 SQL 작성
      · 학습 포인트: TypeORM generate처럼 자동 diff가 없다 → SQL을 직접 쓴다 (nest-boiler에서 "생성된 SQL을 리뷰하라"던 규약이 여기선 기본값)
- [ ] **Step 7 — staticboard 도메인 이식**: handler/service/repository/entity/dto/mapper 풀 세트, CRUD + 페이지네이션 + 소프트삭제
      · 학습 포인트: DI 컨테이너 없이 main.go에서 손으로 조립(wire-up). 인터페이스는 사용하는 쪽에 정의
- [ ] **Step 8 — Swagger**: swaggo/swag 주석, /api-docs 서빙, 표준 에러 응답 문서화(@ApiDoc 대응)
- [ ] **Step 9 — 테스트**: httptest 기반 e2e(test/ 폴더), 테스트 DB 분리, 테이블 주도 테스트(table-driven test)
      · 학습 포인트: Go 테스트 문화의 핵심 = 표준 `testing` + 테이블 주도. jest 없이도 충분한 이유
- [ ] **Step 10 — 품질 게이트**: .golangci.yml(레이어 규칙 depguard 포함), lefthook(pre-commit: lint, pre-push: test), Makefile 정리
- [ ] **Step 11 (선택) — JWT 미들웨어**: nest-boiler의 jwt 도메인 이식

## 실행 (완성 후 기준 — 스텝 진행하며 채워짐)

```bash
make db-up      # Postgres 컨테이너 기동
make migrate-up # 마이그레이션 적용
make run        # air 핫리로드 개발 서버 (APP_ENV=local)
make test       # e2e 포함 테스트
make lint       # golangci-lint
```
