# syntax=docker/dockerfile:1

FROM golang:1.22-alpine AS build
WORKDIR /src

RUN apk add --no-cache git ca-certificates

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/twitch-miner ./cmd/miner

FROM alpine:3.20 AS runtime
RUN apk add --no-cache ca-certificates tzdata \
	&& addgroup -S miner \
	&& adduser -S -G miner -h /data -s /sbin/nologin miner

COPY --from=build /out/twitch-miner /usr/local/bin/twitch-miner

ENV TCPM_DATA_DIR=/data
WORKDIR /data
USER miner

VOLUME ["/data"]
ENTRYPOINT ["/usr/local/bin/twitch-miner"]
CMD ["-data-dir", "/data"]
