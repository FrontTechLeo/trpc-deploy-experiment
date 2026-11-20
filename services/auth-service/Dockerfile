FROM golang:1.24-alpine AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY gen/ gen/
COPY pkg/ pkg/
COPY proto/ proto/
COPY services/auth-service/ services/auth-service/

RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /app/auth-service ./services/auth-service/cmd/server

FROM alpine:3.20

RUN apk --no-cache add ca-certificates tzdata
WORKDIR /app

COPY --from=builder /app/auth-service /app/auth-service
COPY services/auth-service/conf/ /app/conf/
COPY services/auth-service/conf/polaris.yaml /app/polaris.yaml

EXPOSE 10001 10002

ENTRYPOINT ["/app/auth-service", "-conf", "/app/conf/trpc_go.yaml"]
