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

.PHONY: run
run: .keys/private.key .keys/public.key
	# these keys MUST never be used outside of a dev environment
	SESSION_AUTH_KEY=dev-session-auth-key---X2xr5nJgD2eetKHZoYOoh00otckwU8mmB3jEvTBhc \
	SESSION_ENC_KEY=dev-session-enc-key---Bgvp9YxwQT \
	JWT_PRIVATE_KEY=.keys/private.key \
	JWT_PUBLIC_KEY=.keys/public.key \
	OPAQUE_SALT=V45ixWTj \
	go run cmd/sqmgrserver/*.go -dev

.PHONY: docker-build
docker-build:
	docker build -t ${IMG} --build-arg BUILD_NUMBER=${BUILD_NUMBER} .
	docker build -t ${LB_IMG} -f Dockerfile-liquibase .

.PHONY: docker-push
docker-push: docker-build
	docker push ${IMG}
	docker push ${LB_IMG}

.PHONY: k8s-deploy
k8s-deploy: docker-push
	kubectl set image deploy ${DEPLOY_NAME} sqmgr=$(shell docker inspect --format='{{index .RepoDigests 0}}' reg.taproom.us/weters/sqmgrserver:latest) --record

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

.PHONY: integration-db
integration-db: dev-db clean-integration
	docker exec -it sqmgr-postgres createdb -Upostgres integration
	-docker rm -f -v sqmgr-postgres

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

