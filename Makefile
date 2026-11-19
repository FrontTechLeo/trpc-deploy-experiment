.PHONY: help infra-up infra-down monitor-up monitor-down analytics-up analytics-down all-up all-down proto-lint proto-gen migrate-up migrate-down duckdb-test clean

export PATH := $(PATH):$(HOME)/Softwares/gopath/bin

help:
	@echo "tRPC-Go 本地部署环境快捷管理指令:"
	@echo "  make infra-up      - 启动基础治理套件 (Polaris, Redis, MySQL, RabbitMQ, MinIO, MongoDB)"
	@echo "  make infra-down    - 停止基础治理套件"
	@echo "  make monitor-up    - 启动可观测套件 (Prometheus, Jaeger, Loki, Grafana)"
	@echo "  make monitor-down  - 停止可观测套件"
	@echo "  make analytics-up  - 启动复杂报表与数仓套件 (TiDB + ClickHouse)"
	@echo "  make analytics-down- 停止复杂报表与数仓套件"
	@echo "  make all-up        - 一键拉起全套治理、观测与基础环境"
	@echo "  make all-down      - 停止全套环境"
	@echo "  make proto-lint    - 执行 Protobuf 契约门禁检查 (buf lint)"
	@echo "  make proto-gen     - 生成 Protobuf Go 代码 (buf generate)"
	@echo "  make migrate-up    - 执行数据库版本迁移 (goose up)"
	@echo "  make migrate-down  - 回滚上一个数据库版本迁移 (goose down)"
	@echo "  make duckdb-test   - 测试本地 DuckDB 分析引擎"

infra-up:
	docker compose -f deploy/docker-compose.infra.yml up -d

infra-down:
	docker compose -f deploy/docker-compose.infra.yml down

monitor-up:
	docker compose -f deploy/docker-compose.monitor.yml up -d

monitor-down:
	docker compose -f deploy/docker-compose.monitor.yml down

analytics-up:
	docker compose -f deploy/docker-compose.analytics.yml up -d

analytics-down:
	docker compose -f deploy/docker-compose.analytics.yml down

all-up: infra-up monitor-up
	@echo ">>> 全套基础环境拉起完成: Polaris(8090), RabbitMQ(15672), MinIO(9001), Grafana(3000), Jaeger(16686)"

all-down: monitor-down infra-down
	@echo ">>> 全套基础环境已停止"

proto-lint:
	cd proto && buf lint

proto-gen:
	cd proto && buf generate

migrate-up:
	goose -dir deploy/migrations mysql "root:root@tcp(127.0.0.1:3306)/gametool?parseTime=true" up

migrate-down:
	goose -dir deploy/migrations mysql "root:root@tcp(127.0.0.1:3306)/gametool?parseTime=true" down

duckdb-test:
	duckdb -c "SELECT 'DuckDB Local Engine' as engine, 'Ready for MinIO Parquet queries' as status;"
