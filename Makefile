IMG ?= "weters/sqmgr-api"
LB_IMG ?= "weters/sqmgr-lb"
VERSION ?= $(shell git describe --always)
PG_HOST ?= "localhost"
PG_PORT ?= "5432"
PG_USERNAME ?= "postgres"
PG_DATABASE ?= "postgres"
PG_PASSWORD ?= ""
ROLLBACK_COUNT ?= "1"
DEPLOY_NAME ?= "sqmgr-api"

.keys/private.pem:
	-mkdir .keys
	openssl genrsa -out .keys/private.pem 2048

.keys/public.pem: .keys/private.pem
	openssl pkey -in .keys/private.pem -pubout -out .keys/public.pem

.PHONY: run
run: .keys/private.pem .keys/public.pem
	go run cmd/sqmgr-api/*.go -dev

.PHONY: docker-build
docker-build: test-integration
	docker build -t ${IMG}:${VERSION} --build-arg VERSION=${VERSION} .
	docker tag ${IMG}:${VERSION} ${IMG}:latest
	docker build -t ${LB_IMG}:${VERSION} -f Dockerfile-liquibase .
	docker tag ${LB_IMG}:${VERSION} ${LB_IMG}:latest

.PHONY: docker-push
docker-push:
	docker push ${IMG}:${VERSION}
	docker push ${IMG}:latest
	docker push ${LB_IMG}:${VERSION}
	docker push ${LB_IMG}:latest

.PHONY: k8s-deploy
k8s-deploy:
	kubectl set image deploy ${DEPLOY_NAME} sqmgr=$(IMG):${VERSION} --record
	kubectl rollout status deploy ${DEPLOY_NAME}

.PHONY: test
test:
	golint ./...
	go vet ./...
	go test -coverprofile=coverage.out ./...

.PHONY: clean-integration
clean-integration:
	-docker exec -it sqmgr-postgres dropdb -Upostgres integration

.PHONY: test-integration
test-integration: PG_DATABASE=integration
test-integration: integration-db migrations
	golint ./...
	go vet ./...
	INTEGRATION=1 go test -v -coverprofile=coverage.out ./...

.PHONY: cover
cover: test
	go tool cover -html coverage.out

.PHONY: cover-integration
cover-integration: test-integration
	go tool cover -html coverage.out

.git/hooks/pre-commit:
	ln -s ../../git-hooks/pre-commit .git/hooks/pre-commit

.PHONY: git-hooks
git-hooks: .git/hooks/pre-commit

.PHONY: dev-db
dev-db: git-hooks
	-docker run --name sqmgr-postgres --detach --publish 5432:5432 postgres:11

.PHONY: dev-db-delete
dev-db-delete:
	-docker rm -f -v sqmgr-postgres

.PHONY: integration-db
integration-db: dev-db clean-integration
	docker exec -it sqmgr-postgres createdb -Upostgres integration

.PHONY: dev-db-reset
dev-db-reset: dev-db-delete dev-db wait migrations

.PHONY: migrations
migrations:
	liquibase \
		--changeLogFile ./sql/migrations.sql \
		--driver org.postgresql.Driver \
		--classpath ./third_party/jdbc/postgresql/postgresql-42.2.5.jar \
		--url "jdbc:postgresql://${PG_HOST}:${PG_PORT}/${PG_DATABASE}" \
		--username ${PG_USERNAME} \
		--password ${PG_PASSWORD} \
		update

.PHONY: migrations-down
migrations-down:
	liquibase \
		--changeLogFile ./sql/migrations.sql \
		--driver org.postgresql.Driver \
		--classpath ./third_party/jdbc/postgresql/postgresql-42.2.5.jar \
		--url "jdbc:postgresql://${PG_HOST}:${PG_PORT}/${PG_DATABASE}" \
		--username ${PG_USERNAME} \
		--password ${PG_PASSWORD} \
		rollbackCount ${ROLLBACK_COUNT}

.PHONY: testdata
testdata:
	go run hack/testdata/*.go

.PHONY: wait
wait:
	sleep 1
