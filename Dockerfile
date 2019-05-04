FROM golang:1.12 AS build
WORKDIR /build
COPY go.* ./
COPY cmd/ cmd/
COPY internal/ internal/
COPY pkg/ pkg/
RUN CGO_ENABLED=0 GOOS=linux go build -a -o sqmgrserver github.com/weters/sqmgr/cmd/sqmgrserver

FROM busybox:latest
EXPOSE 8080
WORKDIR /app
COPY --from=build /build/sqmgrserver /bin/sqmgrserver
COPY --from=build /usr/share/zoneinfo/America/New_York /usr/share/zoneinfo/America/New_York
COPY web/ web/
ARG BUILD_NUMBER
ENV BUILD_NUMBER=${BUILD_NUMBER}
ENTRYPOINT [ "/bin/sqmgrserver" ]
