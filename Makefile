PG_HOST ?= "localhost"
PG_PORT ?= "5432"
PG_DATABASE ?= "postgres"
ROLLBACK_COUNT ?= "1"

.git/hooks/pre-commit:
	ln -s ../../git-hooks/pre-commit .git/hooks/pre-commit

.keys/private.pem:
	-mkdir .keys
	openssl genrsa -out .keys/private.pem 2048

.keys/public.pem: .keys/private.pem
	openssl pkey -in .keys/private.pem -pubout -out .keys/public.pem

bin/migrate:
	mkdir -p bin
	go build -o bin/migrate -tags postgres github.com/golang-migrate/migrate/v4/cmd/migrate

bin/golint:
	mkdir -p bin
	go build -o bin/golint golang.org/x/lint/golint

.PHONY: run
run: .keys/public.pem dev-db
	go run cmd/sqmgr-api/*.go -migrate

.PHONY: test
test: PG_DATABASE=integration
test: bin/golint integration-db migrations
	bin/golint -set_exit_status ./...
	./hack/gofmt-check.sh
	go vet ./...
	go vet ./...
	INTEGRATION=1 go test -v -coverprofile=coverage.out ./...

.PHONY: cover
cover: test
	go tool cover -html coverage.out

.PHONY: git-hooks
git-hooks: .git/hooks/pre-commit

.PHONY: dev-db
dev-db:
	-docker run --name sqmgr-postgres --detach --publish 5432:5432 postgres:11
	@docker exec sqmgr-postgres bash -c 'for i in {1..30}; do if /usr/bin/pg_isready>/dev/null 2>&1; then break; fi; sleep 0.1; done;'

.PHONY: integration-db
integration-db: dev-db
	-docker exec -it sqmgr-postgres dropdb -Upostgres integration
	docker exec -it sqmgr-postgres createdb -Upostgres integration

.PHONY: testdata
testdata:
	go run hack/testdata/*.go

.PHONY: migrations
migrations: bin/migrate
	bin/migrate -path ./sql -database postgres://postgres:@${PG_HOST}:${PG_PORT}/${PG_DATABASE}?sslmode=disable up

.PHONY: migrations-down
migrations-down: bin/migrate
	bin/migrate -path ./sql -database postgres://postgres:@${PG_HOST}:${PG_PORT}/${PG_DATABASE}?sslmode=disable down ${ROLLBACK_COUNT}

.PHONY: format
format:
	find . -type f -name '*.go' | xargs -L 1 gofmt -s -w

.PHONY:
clean:
	-docker rm -f -v sqmgr-postgres
	rm -rf bin/migrate
	rm -rf bin/golint
	rm -rf .keys/*.pem
