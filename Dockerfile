FROM golang:latest AS build
WORKDIR /go/src/github.com/weters/sqmgr
COPY cmd/ cmd/
COPY internal/ internal/
COPY vendor vendor/
RUN CGO_ENABLED=0 GOOS=linux go build -a -o sqmgrserver github.com/weters/sqmgr/cmd/sqmgrserver

FROM busybox:latest
EXPOSE 8080
WORKDIR /app
COPY --from=build /go/src/github.com/weters/sqmgr/sqmgrserver /bin/sqmgrserver
COPY web/ web/
ARG BUILD_NUMBER
ENV BUILD_NUMBER=${BUILD_NUMBER}
ENTRYPOINT [ "/bin/sqmgrserver" ]
