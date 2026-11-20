FROM golang:1.24-alpine AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY gen/ gen/
COPY pkg/ pkg/
COPY proto/ proto/
COPY services/gateway/ services/gateway/

RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /app/gateway ./services/gateway/cmd/server

FROM alpine:3.20

RUN apk --no-cache add ca-certificates tzdata
WORKDIR /app

COPY --from=builder /app/gateway /app/gateway
COPY services/gateway/conf/ /app/conf/
COPY services/gateway/conf/polaris.yaml /app/polaris.yaml

EXPOSE 8080

ENTRYPOINT ["/app/gateway", "-conf", "/app/conf/trpc_go.yaml", "-router", "/app/conf/router.yaml"]
