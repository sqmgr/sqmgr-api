FROM golang:1.12 AS build-go
WORKDIR /build
COPY go.* ./
COPY cmd/ cmd/
COPY internal/ internal/
COPY pkg/ pkg/
RUN CGO_ENABLED=0 GOOS=linux go build -a -o sqmgr-api github.com/weters/sqmgr-api/cmd/sqmgr-api

FROM alpine:latest
EXPOSE 5000
WORKDIR /app
RUN apk add --no-cache ca-certificates
COPY --from=build-go /build/sqmgr-api /bin/sqmgr-api
COPY --from=build-go /usr/share/zoneinfo/America/New_York /usr/share/zoneinfo/America/New_York
ARG VERSION
ENV SQMGR_VERSION=${VERSION}
ENTRYPOINT [ "/bin/sqmgr-api" ]
