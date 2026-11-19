.PHONY: help infra-up infra-down monitor-up monitor-down analytics-up analytics-down all-up all-down clean

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
