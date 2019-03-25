IMG ?= "reg.taproom.us/weters/sqmgrserver:latest"
BUILD_NUMBER ?= `date "+%y%m%d%H%M%S"`
PG_HOST ?= "localhost"
PG_PORT ?= "5432"
PG_USERNAME ?= "postgres"
PG_DATABASE ?= "postgres"
PG_PASSWORD ?= ""
ROLLBACK_COUNT ?= "1"

run:
	# these keys MUST never be used outside of a dev environment
	SESSION_AUTH_KEY=dev-session-auth-key---X2xr5nJgD2eetKHZoYOoh00otckwU8mmB3jEvTBhc SESSION_ENC_KEY=dev-session-enc-key---Bgvp9YxwQT go run cmd/sqmgrserver/*.go

docker-build:
	docker build -t ${IMG} --build-arg BUILD_NUMBER=${BUILD_NUMBER} .

docker-push: docker-build
	docker push ${IMG}

test:
	go test ./...

dev-db:
	docker run --name sqmgr-postgres --detach --publish 5432:5432 postgres:11

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

git-hooks:
	ln -s ../../git-hooks/pre-commit .git/hooks/pre-commit

.PHONY: docker-build docker-push migrations migrations-down run test git-hooks dev-db
