FROM golang:1.12 AS build-go
WORKDIR /build
COPY go.* ./
COPY cmd/ cmd/
COPY internal/ internal/
COPY pkg/ pkg/
RUN CGO_ENABLED=0 GOOS=linux go build -a -o sqmgrserver github.com/weters/sqmgr/cmd/sqmgrserver

FROM node:latest AS build-node
WORKDIR /build/
COPY web/ web/
RUN cd web \
	&& npm i \
	&& npm run build

FROM alpine:latest
EXPOSE 8080
WORKDIR /app
COPY --from=build-go /build/sqmgrserver /bin/sqmgrserver
COPY --from=build-go /usr/share/zoneinfo/America/New_York /usr/share/zoneinfo/America/New_York
COPY --from=build-go /etc/ssl/certs/ /etc/ssl/certs/
COPY --from=build-node /build/web/static/ web/static/
COPY web/templates/ web/templates/
ARG BUILD_NUMBER
ENV BUILD_NUMBER=${BUILD_NUMBER}
ENTRYPOINT [ "/bin/sqmgrserver" ]
