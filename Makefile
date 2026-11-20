# 자주 쓰는 명령 모음. nest-boiler 의 pnpm scripts 자리.
.PHONY: run test vet db-up db-down db-reset

# 개발 서버 (APP_ENV=local → envs/.env.local)
run: ; APP_ENV=local go run ./cmd/api
test: ; go test ./...
vet: ; go vet ./...

# 로컬 Postgres (Docker, host 포트 5435)
db-up: ; bash scripts/db-up.sh
# 중지. 데이터 볼륨은 남는다
db-down: ; bash scripts/db-down.sh
# 볼륨까지 지우고 다시 띄운다
db-reset: ; bash scripts/db-reset.sh
