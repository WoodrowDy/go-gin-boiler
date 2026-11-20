# go-gin-boiler

Go + Gin + GORM + PostgreSQL 백엔드 보일러플레이트.
[nest-boiler](https://github.com/WoodrowDy/nest-boiler)와 **같은 형태(아키텍처·규약)** 를 Go로 옮기는 것이 목표다.

> 이 레포는 학습용으로 진행한다: 각 스텝을 가이드 문서를 보고 **직접 작성**하고,
> 스텝이 끝날 때마다 리뷰를 받는다. (진행 상태는 맨 아래 로드맵 체크박스로 관리)

---

## 스택 매핑 (nest-boiler → go-gin-boiler)

| 역할 | nest-boiler (TS) | go-gin-boiler (Go) | 메모 |
|---|---|---|---|
| 런타임/언어 | Node + TypeScript | Go 1.25 | 컴파일 언어, 단일 바이너리 배포 |
| 웹 프레임워크 | NestJS 11 | **Gin v1.12** | DI 컨테이너 없음 → 생성자 주입을 손으로 |
| ORM | TypeORM 1.x | **GORM v2 (제네릭 API)** | `gorm.G[T](db)` — ctx 가 모든 실행 메서드의 첫 인자 |
| DB | PostgreSQL (Docker, 5434) | PostgreSQL (Docker, **5435**) | nest-boiler 와 동시에 띄울 수 있게 |
| 마이그레이션 생성 | typeorm migration:generate | **Atlas + atlas-provider-gorm** (`atlas migrate diff`) | 엔티티 diff 로 SQL 파일 생성, 리뷰 후 커밋 |
| 마이그레이션 이름·린트 | scripts/_migration-name.sh, migration-lint.sh | 같은 스크립트를 SQL 파일용으로 이식 | `동사_대상`, 파괴적 구문 경고 |
| 부팅 가드 | assertNoPendingMigrations | 같은 가드를 `main` 에서 서버 기동 전에 | 스키마가 코드보다 뒤쳐지면 뜨지 않는다 |
| 검증 | class-validator (필드 계약에서 파생) | **go-playground/validator** (Gin binding) | 규칙은 커스텀 태그로 한 곳에 정의 |
| 입력 정규화 | class-transformer @Transform | 값 타입의 `UnmarshalJSON` | 예: `type Phone string` |
| Swagger | @nestjs/swagger + @ApiDoc | **swaggo/swag** (주석 기반) | local 밖에서는 Basic auth |
| 로깅 | winston + nest-winston | **slog** (표준 라이브러리) | 모든 줄에 traceId |
| traceId 컨텍스트 | AsyncLocalStorage | **context.Context** | ⭐ TS→Go 최대 개념 전환 포인트 |
| 응답 래핑 | ObjectResponse / ListResponse | 같은 규약을 제네릭 구조체로 | `{ row, meta }` / `{ rows, count, meta }` |
| 전역 예외 처리 | HttpExceptionFilter | 에러 미들웨어 + `apperror` | `{ message, error, statusCode, meta }` |
| env 관리 | envs/.env.* + Joi 검증 | **godotenv + caarlos0/env** | `APP_ENV=local\|dev\|prod\|test` |
| 스크립트 러너 | pnpm scripts | **Makefile** | `make run`, `make db-up`, `make test` |
| 핫 리로드 | nest --watch | **air** | 개발 편의 |
| lint/format | eslint + prettier | **golangci-lint + gofumpt** | import 방향은 depguard, 이름 규칙은 구조 테스트 |
| git 훅 | husky + lint-staged | **lefthook** | pre-commit: lint, pre-push: test |

## 아키텍처

nest-boiler와 동일한 **package-by-domain + 레이어드** (DDD-lite).
단, Go 관례를 따라 레이어를 폴더가 아닌 **파일**로 나눈다
(Go는 폴더=패키지 단위라, 도메인 하나를 잘게 쪼개면 순환 import 지옥이 된다).

```
go-gin-boiler/
├── cmd/
│   ├── api/main.go              # 부트스트랩 + 수동 조립 (nest 의 main.ts + app.module.ts)
│   └── seed/main.go             # 시드 실행 (local 전용)
├── internal/                    # 외부 import 불가 (Go 컴파일러가 강제)
│   ├── domain/
│   │   └── template/            # 예시 도메인. 새 도메인은 users 를 복사해서 시작한다
│   │       ├── users/           # ★ 기본 CRUD
│   │       │   ├── handler.go       # = controller (라우팅 + 바인딩 + 응답 변환)
│   │       │   ├── service.go       # = service (없으면 던진다, 트랜잭션 경계)
│   │       │   ├── repository.go    # = repository (쿼리만, 엔티티 in/out)
│   │       │   ├── entity.go        # = entity (GORM 태그만)
│   │       │   ├── dto.go           # = dtos (request + response)
│   │       │   ├── mapper.go        # = mapper (Entity → Response, 무상태)
│   │       │   ├── constants.go     # = *.constants.ts (에러 메시지 등)
│   │       │   └── seed.go          # = seeds (시더 + 팩토리)
│   │       └── boards/          # ★ users 와의 관계, 쓰기 2개 트랜잭션, 업무 액션(publish)
│   ├── global/
│   │   ├── config/              # env 로딩 + 검증
│   │   ├── logger/              # slog 설정
│   │   ├── middleware/          # traceId, 요청/응답 로거, 에러 처리
│   │   ├── response/            # ObjectResponse / ListResponse / Meta
│   │   ├── apperror/            # 도메인 에러 타입 (NotFound, Conflict → HTTP 매핑)
│   │   ├── dberror/             # 제약 위반 판별 (23505, 23503)
│   │   └── validation/          # 커스텀 validator 태그, 값 타입(Phone 등)
│   └── database/
│       ├── database.go          # GORM 연결
│       ├── entity.go            # CoreHard / CoreSoft 공통 필드
│       ├── pending.go           # 적용 안 된 마이그레이션이 있으면 부팅 거부
│       └── migrations/          # Atlas 가 만든 SQL 파일 (적용된 파일은 고치지 않는다)
├── envs/                        # .env.local 등 (git-ignored) + .example
├── scripts/                     # db-up/down/reset, migration-new, migration-lint, migrate
├── test/                        # e2e (httptest) — 도메인별 fixtures / mocks / scenarios
├── atlas.hcl
├── docker-compose.yml
├── Makefile
├── .golangci.yml
└── lefthook.yml
```

`users` 는 기본 CRUD 를, `boards` 는 도메인이 다른 도메인을 건드릴 때 더해지는 것을 보여준다.
`boards` 는 `users.Service` 만 import 한다 — 반대 방향은 컴파일러가 순환 import 로 막는다.

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

- **entity.go = 영속성 전용.** GORM 태그만. `json:` · `binding:` 태그를 두지 않는다 (응답 계약은 dto.go 소유).
- **DTO 가 검증과 직렬화를 소유한다.** 요청 구조체에 `binding:` 태그, 응답 구조체에 `json:` 태그.
  엔티티에서 DTO 를 파생하지 않는다.
  - Go 에는 `PickType` 이 없어서 요청 구조체를 엔드포인트마다 직접 쓴다. 대신 **검증 규칙은 필드 이름당 하나**만 둔다:
    규칙이 둘 이상의 구조체에 필요하면 커스텀 태그(`binding:"required,phone"`)로 `global/validation` 에 정의한다.
  - PATCH 페이로드는 포인터 필드로 "보내지 않음" 과 "빈 값" 을 구분한다.
  - 입력 정규화는 값 타입이 맡는다 (`type Phone string` 의 `UnmarshalJSON`). 바인딩이든 시드든 같은 경로를 탄다.
- **응답 모양은 엔드포인트당 하나.** `UserSaveResponse` / `UserListItemResponse` / `UserDetailResponse` 처럼 나누고,
  매퍼는 모양마다 함수 하나(`toSaveResponse`, `toListItem`, `toDetail`)를 둔다.
- **매퍼는 옮기기만 한다.** 구조체 리터럴에 나가는 필드를 한 줄씩 적는다 — 적지 않은 것은 나가지 않는다.
  값을 만들어야 하면 서비스가 만들어 넘긴다.
- **service 는 엔티티를 반환한다.** HTTP 응답 DTO 를 모른다. Entity → Response 변환은 handler 가 매퍼로 한다.
  - handler 메서드 하나에 서비스 호출 하나. 둘이면 handler 가 조율하고 있는 것이니 서비스로 접는다.
- **repository 는 쿼리만.** 조회는 엔티티를, 단건이 없으면 `nil, nil` 을, 쓰기는 영향받은 행 수를 돌려준다.
  에러를 만들지 않고 감싸지 않는다. 404 판단과 트랜잭션 경계는 service 다.
  - 단건 조회의 `id` 조건은 분기 없이 항상 붙인다. 목록 필터를 만드는 쪽에는 `id` 분기가 없다.
  - 사용자 입력이 들어가는 `LIKE` / `ILIKE` 는 `%` `_` `\` 를 이스케이프하는 헬퍼를 거친다.
- **도메인은 서비스로만 들어간다.** 다른 도메인의 repository 를 직접 부르지 않는다.
- **인터페이스는 최소화.** 레이어마다 인터페이스를 미리 깔지 않는다.
  구체 타입으로 시작하고, mock 이 실제로 필요해지는 순간에 **사용하는 쪽에** 인터페이스를 정의한다.
  → 근거: [Go 공식 코드리뷰 가이드 — Interfaces](https://go.dev/wiki/CodeReviewComments#interfaces),
  [Standard Package Layout (Ben Johnson)](https://medium.com/@benbjohnson/standard-package-layout-7cdbc8391fc1)
- **강제 수단.** import 방향은 golangci-lint `depguard` 로, 아래 이름 규칙처럼 문법을 봐야 하는 것은
  `go/ast` 로 선언을 읽는 구조 테스트(`go test`)로 막는다.

## 네이밍 (CRUD)

메서드 이름만 보고 레이어를 알 수 있게, 레이어마다 동사를 다른 곳에서 가져온다.
**handler 는 HTTP, repository 는 SQL, service 는 둘 다 쓰지 않는 CRUD 동사.**

| 레이어 | 생성 | 단건 조회 | 목록 조회 | 수정 | 삭제 |
|---|---|---|---|---|---|
| Handler | `PostX` | `GetX` | `GetXs` | `PatchX` | `DeleteX` |
| Service | `CreateX` | `ReadX` | `ReadXList` · `ReadXListAndCount` | `ModifyX` | `RemoveX` |
| Repository | `InsertX` | `SelectX` | `SelectXList` · `SelectXListAndCount` | `UpdateX` | `SoftDeleteX` (`HardDeleteX`) |

```
POST   /users      → PostUser   → CreateUser           → InsertUser
GET    /users/:id  → GetUser    → ReadUser             → SelectUser
GET    /users      → GetUsers   → ReadUserListAndCount → SelectUserListAndCount
PATCH  /users/:id  → PatchUser  → ModifyUser           → UpdateUser
DELETE /users/:id  → DeleteUser → RemoveUser           → SoftDeleteUser
```

- 타입 이름에는 도메인을 반복하지 않는다 — 패키지가 이미 말해준다: `users.Handler`, `users.Service`, `users.Repository`.
- 요청 DTO 는 엔드포인트 이름을 딴다: `PostUserPayload`, `PatchUserPayload`, `GetUserQuery`.
- 업무 액션은 업무 동사를 쓴다: `POST /boards/:id/publish` → `PostPublishBoard` → `PublishBoard`.
  상태 규칙은 그 액션의 본문에만 둔다. `ModifyX` 는 필드만 바꾸고 상태를 모른다.
- 미스가 정상인 호출자를 위해 `ReadXOrNil` 을 두고, `ReadX` 는 그것을 불러 404 만 더한다.
- **쓰기 직후 같은 요청에서 다시 읽지 않는다.** 에러 없이 돌아왔으면 성공이다. 저장된 값이나 이미 가진 값으로 응답한다.

## 트랜잭션

경계는 service 다. repository 는 합류만 한다.
Go 에는 선택 인자가 없어서 `tx *gorm.DB` 를 명시적으로 받고, `nil` 이면 자기 커넥션을 쓴다.

```go
// repository — tx 가 오면 그 트랜잭션으로, 없으면 자기 커넥션으로
func (r *Repository) conn(tx *gorm.DB) *gorm.DB {
	if tx != nil {
		return tx
	}
	return r.db
}

func (r *Repository) UpdateUserBoardCount(ctx context.Context, tx *gorm.DB, userID int64, delta int) (int, error) {
	return gorm.G[User](r.conn(tx)).
		Where("id = ?", userID).
		Update(ctx, "board_count", gorm.Expr("board_count + ?", delta))
}
```

```go
// service — 쓰기가 둘 이상이라 하나로 묶어야 하는 메서드가, 거기서만 연다
func (s *Service) CreateBoard(ctx context.Context, payload PostBoardPayload) (*Board, error) {
	var board *Board

	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// "없음" 을 알려줄 수 있는 쓰기를 먼저 한다
		affected, err := s.users.AddBoardCount(ctx, tx, payload.UserID, 1)
		if err != nil {
			return err
		}
		if affected == 0 {
			return apperror.NotFound(errWriterNotFound)
		}

		board, err = s.repo.InsertBoard(ctx, tx, payload)
		return err
	})

	return board, err
}
```

- 문장 하나짜리 쓰기는 트랜잭션을 열지 않는다. 존재 확인은 쓰기에 접는다 — `affected == 0` 이 404 다.
- **트랜잭션을 여는 메서드는 `tx` 를 받지 않는다.** 그 밖의 service 메서드는 받아서 repository 와 다른 service 로 그대로 넘긴다.
- 다른 도메인이 자기 트랜잭션에 끼워 넣는 쓰기는, 0 행의 뜻을 호출자만 알 때 에러 대신 `affected` 를 돌려준다
  (`users.Service.AddBoardCount`).
- TypeORM 의 QueryRunner try/catch/finally 는 `Transaction` 클로저 하나가 된다. 에러를 반환하면 롤백, `nil` 이면 커밋.

## 에러 처리

에러는 **출처가 아니라 의미**로 나눈다.

- **예상한 것(업무 규칙)** — 없음, 잘못된 상태, 중복, 참조 중 삭제. service 가 `apperror` 를 반환한다.
  메시지는 도메인의 `constants.go` 에 둔다.
- **예상 못 한 것(장애)** — 연결 끊김, 타임아웃, 쿼리 버그. 잡지 않고 `%w` 로 감싸 그대로 올린다.
- DB 제약 위반은 **그 자리에서** 옮긴다: `dberror.IsUniqueViolation(err)`(23505) → `apperror.Conflict(...)`.
- HTTP 경계에서는 에러 미들웨어 하나가 전부 받는다. 응답 모양은 nest-boiler 와 같다:
  `{ message, error, statusCode, meta: { traceId, timestamp } }`
  - 아무도 옮기지 않은 제약 위반은 409 로 떨어뜨린다.
  - 모르는 에러의 내용은 응답에 싣지 않는다. 상세는 로그에, 둘을 잇는 것은 `meta.traceId`.
  - 5xx 는 미들웨어가 직접 로그를 남긴다. 4xx 는 접근 로그의 상태 코드로 충분하다.

## 응답과 관측

- 성공 응답은 `ObjectResponse[T]`(`{ row, meta }`) / `ListResponse[T]`(`{ rows, count, meta }`).
- **쓰기는 상태 코드로 답한다.** POST 는 저장된 리소스를, PATCH · DELETE · 업무 액션은 본문 없이 200 을 돌려준다.
  이때 traceId 는 `X-Request-Id` 헤더에 있다.
- traceId: 들어온 `X-Request-Id` 를 쓰거나 새로 만들어 `context.Context` 에 싣고, 응답 헤더로 돌려준다.
- 로그: slog. 모든 줄에 traceId 가 붙는다. 접근 로그는 `HTTP REQ` / `HTTP RES`(상태 + 소요 시간, 상태에 따라 레벨).

## 마이그레이션

- 엔티티를 바꾸면 `make migration-new name=add_board_status` — 적용 안 된 것을 먼저 적용한 뒤,
  Atlas 가 엔티티와 마이그레이션 디렉터리의 차이로 SQL 파일을 만든다. **커밋 전에 생성된 SQL 을 읽는다.**
- 이름은 `동사_대상`, snake_case. 스키마: `create | add | alter | rename | drop`, 데이터: `insert | update | delete`.
- `make migration-lint` 는 이름 규칙을 막고, 되돌리기 어려운 구문(`DROP`, 타입 변경 등)은 보여준다.
- 적용된 마이그레이션 파일은 고치지 않는다. 새 파일을 얹는다.
- **prod 는 되돌리지 않는다 — roll forward.**
- 앱은 적용 안 된 마이그레이션이 있으면 뜨지 않는다. 순서는 항상 마이그레이션 먼저, 배포 다음.

---

## 학습 로드맵 (스텝별로 직접 작성)

각 스텝은 "무엇을 / 왜 / TS 개발자용 학습 포인트" 가이드를 받고 직접 구현한다.
한 스텝 = 한 커밋(또는 몇 개) 단위로 끊는다.

- [x] **Step 0 — 환경 준비**: Go 툴체인 정리, golangci-lint·air·swag CLI 설치, `.gitignore`, 첫 커밋
      · 학습 포인트: GOPATH는 잊어라, go.mod가 package.json이다. `go install`과 `go get`의 차이
- [x] **Step 1 — Gin Hello World + 프로젝트 뼈대**: `cmd/api/main.go`, health check 엔드포인트, graceful shutdown
      · 학습 포인트: 고루틴 + 시그널 채널(`signal.NotifyContext`) — Node의 `process.on('SIGINT')`과 비교
- [x] **Step 2 — 설정(config)**: envs/ 구조 이식, `APP_ENV`별 .env 로딩, 구조체 기반 파싱+필수값 검증
      · 학습 포인트: 구조체 태그가 데코레이터의 자리를 대신한다. zero value 개념
- [x] **Step 3 — 로깅 + traceId**: slog 설정, traceId 미들웨어(`X-Request-Id`), `HTTP REQ` / `HTTP RES` 접근 로그, 로그에서 자격증명 지우기
      · 학습 포인트: ⭐ `context.Context` — AsyncLocalStorage처럼 "암묵적 전파"가 아니라 **명시적으로 첫 인자로 흘려보낸다**. Go에서 가장 중요한 관례
- [x] **Step 4 — 응답 규약 + 에러 처리**: 제네릭 `ObjectResponse[T]` / `ListResponse[T]`, `apperror`, 에러 미들웨어(응답 모양, 5xx 로그, 내용 비노출)
      · 학습 포인트: `error`는 값이다. `errors.Is/As`, 래핑(`fmt.Errorf("%w")`). 제네릭 타입 파라미터
- [x] **Step 5 — Docker Postgres + GORM 연결**: docker-compose(포트 5435), scripts/db-*.sh 이식, GORM 연결, CoreHard / CoreSoft 공통 엔티티
      · 학습 포인트: `gorm.DeletedAt` = TypeORM `@DeleteDateColumn`. 커넥션 풀 설정은 Go에서 명시적. 제네릭 API `gorm.G[T]`
- [ ] **Step 6 — 마이그레이션**: Atlas + atlas-provider-gorm, 이름 규칙·린트 스크립트 이식, 부팅 가드, 배포 환경 적용 경로(`migrate.sh`)
      · 학습 포인트: 엔티티 diff 로 만든 SQL 을 리뷰하는 흐름은 nest-boiler 와 같다. 달라지는 것은 산출물이 TS 클래스가 아니라 SQL 파일이라는 점
- [ ] **Step 7 — users 도메인**: handler/service/repository/entity/dto/mapper 풀 세트, CRUD + 페이지네이션 + 소프트삭제, 네이밍·쓰기 규약, 중복(409) 변환
      · 학습 포인트: DI 컨테이너 없이 main.go에서 손으로 조립(wire-up). 포인터 필드로 PATCH 표현하기. 커스텀 validator 태그와 `UnmarshalJSON`
- [ ] **Step 8 — boards 도메인**: users 와의 관계, 쓰기 2개 트랜잭션(`CreateBoard` / `RemoveBoard`), 업무 액션(`PublishBoard`), 엔드포인트별 응답 모양
      · 학습 포인트: 클로저 기반 트랜잭션과 `tx` 전달. 패키지 간 의존 방향을 컴파일러가 강제한다
- [ ] **Step 9 — Swagger**: swaggo/swag 주석, `/api-docs` 서빙, 표준 에러 응답 문서화, local 밖에서는 Basic auth(자격증명 없으면 부팅 거부)
- [ ] **Step 10 — 테스트**: httptest 기반 e2e(test/ — fixtures / mocks / scenarios), 테스트 DB 분리, 테이블 주도 테스트
      · 학습 포인트: Go 테스트 문화의 핵심 = 표준 `testing` + 테이블 주도. jest 없이도 충분한 이유
- [ ] **Step 11 — 품질 게이트**: .golangci.yml(depguard 레이어 규칙), 이름 규칙 구조 테스트, lefthook(pre-commit: lint, pre-push: test), Makefile 정리
      · 학습 포인트: `go/ast` 로 코드를 데이터처럼 읽기
- [ ] **Step 12 (선택) — 시드 + JWT**: `cmd/seed`(local 전용, 팩토리 데이터), nest-boiler의 jwt 도메인 이식

## 실행 (완성 후 기준 — 스텝 진행하며 채워짐)

```bash
make db-up          # Postgres 컨테이너 기동
make migrate-up     # 마이그레이션 적용
make run            # air 핫리로드 개발 서버 (APP_ENV=local)
make test           # e2e 포함 테스트
make lint           # golangci-lint
```
