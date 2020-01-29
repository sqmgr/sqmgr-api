FROM golang:1.12 AS build-go
WORKDIR /build
COPY go.* ./
RUN go mod download
COPY cmd/ cmd/
COPY internal/ internal/
COPY pkg/ pkg/
RUN CGO_ENABLED=0 GOOS=linux go build -o sqmgr-api github.com/sqmgr/sqmgr-api/cmd/sqmgr-api \
 && CGO_ENABLED=0 GOOS=linux go build -o sqmgr-guest-user-cleanup github.com/sqmgr/sqmgr-api/cmd/sqmgr-guest-user-cleanup

FROM alpine:latest
EXPOSE 5000
WORKDIR /app
RUN apk add --no-cache ca-certificates
COPY --from=build-go /build/sqmgr-api /bin/sqmgr-api
COPY --from=build-go /build/sqmgr-guest-user-cleanup /bin/sqmgr-guest-user-cleanup
COPY --from=build-go /usr/share/zoneinfo/America/New_York /usr/share/zoneinfo/America/New_York
COPY ./sql ./
ARG VERSION
ENV SQMGR_VERSION=${VERSION}
ENTRYPOINT [ "/bin/sqmgr-api" ]
