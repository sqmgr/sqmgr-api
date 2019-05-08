IMG ?= "reg.taproom.us/weters/sqmgrserver"
LB_IMG ?= "reg.taproom.us/weters/sqmgr-lb"
BUILD_NUMBER ?= `date "+%y%m%d%H%M%S"`
PG_HOST ?= "localhost"
PG_PORT ?= "5432"
PG_USERNAME ?= "postgres"
PG_DATABASE ?= "postgres"
PG_PASSWORD ?= ""
ROLLBACK_COUNT ?= "1"
DEPLOY_NAME ?= "sqmgr-dev"

.keys/private.key:
	-mkdir .keys
	openssl genrsa -out .keys/private.key 2048

.keys/public.key: .keys/private.key
	openssl pkey -in .keys/private.key -pubout -out .keys/public.key

run: .keys/private.key .keys/public.key
	# these keys MUST never be used outside of a dev environment
	SESSION_AUTH_KEY=dev-session-auth-key---X2xr5nJgD2eetKHZoYOoh00otckwU8mmB3jEvTBhc \
	SESSION_ENC_KEY=dev-session-enc-key---Bgvp9YxwQT \
	JWT_PRIVATE_KEY=.keys/private.key \
	JWT_PUBLIC_KEY=.keys/public.key \
	OPAQUE_SALT=V45ixWTj \
	go run cmd/sqmgrserver/*.go -dev

docker-build:
	docker build -t ${IMG} --build-arg BUILD_NUMBER=${BUILD_NUMBER} .
	docker build -t ${LB_IMG} -f Dockerfile-liquibase .

docker-push: docker-build
	docker push ${IMG}
	docker push ${LB_IMG}

k8s-deploy: docker-push
	kubectl set image deploy ${DEPLOY_NAME} sqmgr=$(shell docker inspect --format='{{index .RepoDigests 0}}' reg.taproom.us/weters/sqmgrserver:latest) --record

test:
	golint ./...
	go vet ./...
	go test -coverprofile=coverage.out ./...

clean-integration:
	-docker exec -it sqmgr-postgres dropdb -Upostgres integration

test-integration: PG_DATABASE=integration
test-integration: integration-db migrations
	golint ./...
	go vet ./...
	INTEGRATION=1 go test -v -coverprofile=coverage.out ./...

cover: test
	go tool cover -html coverage.out

cover-integration: test-integration
	go tool cover -html coverage.out

dev-db: git-hooks
	-docker run --name sqmgr-postgres --detach --publish 5432:5432 postgres:11

integration-db: dev-db clean-integration
	docker exec -it sqmgr-postgres createdb -Upostgres integration

.git/hooks/pre-commit:
	ln -s ../../git-hooks/pre-commit .git/hooks/pre-commit

git-hooks: .git/hooks/pre-commit

migrations:
	liquibase \
		--changeLogFile ./sql/migrations.sql \
		--driver org.postgresql.Driver \
		--classpath ./third_party/jdbc/postgresql/postgresql-42.2.5.jar \
		--url "jdbc:postgresql://${PG_HOST}:${PG_PORT}/${PG_DATABASE}" \
		--username ${PG_USERNAME} \
		--password ${PG_PASSWORD} \
		update

migrations-down:
	liquibase \
		--changeLogFile ./sql/migrations.sql \
		--driver org.postgresql.Driver \
		--classpath ./third_party/jdbc/postgresql/postgresql-42.2.5.jar \
		--url "jdbc:postgresql://${PG_HOST}:${PG_PORT}/${PG_DATABASE}" \
		--username ${PG_USERNAME} \
		--password ${PG_PASSWORD} \
		rollbackCount ${ROLLBACK_COUNT}

testdata:
	go run hack/testdata/*.go

dev-db-delete:
	-docker rm -f -v sqmgr-postgres

wait:
	sleep 1

dev-db-reset: dev-db-delete dev-db wait migrations

.PHONY: run docker-build docker-push test clean-integration test-integration cover cover-integration dev-db integration-db git-hooks migrations migrations-down wait dev-db-delete dev-db-reset testdata k8s-deploy
