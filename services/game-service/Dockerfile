FROM golang:1.24-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY gen/ gen/
COPY pkg/ pkg/
COPY proto/ proto/
COPY services/game-service/ services/game-service/

RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /app/game-service ./services/game-service/cmd/server

FROM alpine:3.20

RUN apk --no-cache add ca-certificates tzdata

WORKDIR /app

COPY --from=builder /app/game-service /app/game-service
COPY services/game-service/conf/ /app/conf/
COPY services/game-service/conf/polaris.yaml /app/polaris.yaml

EXPOSE 10003 10004

ENTRYPOINT ["/app/game-service", "-conf", "/app/conf/trpc_go.yaml"]
