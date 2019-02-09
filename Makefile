.PHONY: docker-build docker-push test
IMG ?= "reg.taproom.us/weters/sqmgrserver:latest"
BUILD_NUMBER ?= `date "+%y%m%d%H%M%S"`

docker-build:
	docker build -t ${IMG} --build-arg BUILD_NUMBER=${BUILD_NUMBER} .

docker-push: docker-build
	docker push ${IMG}

test:
	go test ./...
