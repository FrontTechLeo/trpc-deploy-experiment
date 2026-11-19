# 03. 轻量化可观测性套件部署与资源控制经验

> **记录时间**：2026-11-19  
> **核心组件**：Prometheus + Alertmanager + Jaeger + Grafana Loki + Vector + Grafana  
> **设计目标**：控制整套监控内存占用在 1.5GB 以内，实现生产同构的全链路追踪与日志联动  

---

## 一、 为什么在本地舍弃 ELK 选择 Loki + Vector？

在生产环境中，大型集群常使用 Elasticsearch 处理海量全文检索，但在单机/本地部署场景中：
- **Elasticsearch 开销极大**：JVM 堆内存至少需要 2~4GB 才能稳定运行，容器启动耗时 1~2 分钟。
- **Grafana Loki 极致轻量**：仅对日志标签建立索引，日志原文压缩存储在本地文件系统中，单容器内存消耗仅需 **150~300MB**。
- **Vector 高效日志收集**：Rust 编写的超高性能收集代理，替代 Java 的 Logstash 或 Python 脚本，单核 CPU 消耗 < 5%，内存 < 50MB。

---

## 二、 监控套件 Compose 配置 (`deploy/docker-compose.monitor.yml`)

```yaml
version: '3.8'

services:
  prometheus:
    image: prom/prometheus:v2.52.0
    container_name: prometheus
    restart: always
    ports:
      - "9090:9090"
    volumes:
      - ./config/prometheus.yml:/etc/prometheus/prometheus.yml
      - prom_data:/prometheus
    command:
      - '--config.file=/etc/prometheus/prometheus.yml'
      - '--storage.tsdb.retention.time=7d' # 本地保留7天指标，避免占用过多磁盘

  jaeger:
    image: jaegertracing/all-in-one:1.56
    container_name: jaeger
    restart: always
    environment:
      - COLLECTOR_OTLP_ENABLED=true
    ports:
      - "16686:16686" # Jaeger Web UI
      - "4317:4317"   # OTLP gRPC 接入
      - "4318:4318"   # OTLP HTTP 接入

  loki:
    image: grafana/loki:2.9.8
    container_name: loki
    restart: always
    ports:
      - "3100:3100"
    command: -config.file=/etc/loki/local-config.yaml
    volumes:
      - loki_data:/tmp/loki

  grafana:
    image: grafana/grafana:11.0.0
    container_name: grafana
    restart: always
    ports:
      - "3000:3000"
    environment:
      - GF_SECURITY_ADMIN_PASSWORD=admin
    volumes:
      - grafana_data:/var/lib/grafana

volumes:
  prom_data:
  loki_data:
  grafana_data:
```

---

## 三、 tRPC-Go 接入可观测性拦截器

### 1. Prometheus 指标暴露
在 `trpc_go.yaml` 中配置：
```yaml
plugins:
  metrics:
    prometheus:
      port: 19090
      path: /metrics
```
Prometheus 会自动抓取每个微服务的 RPC 耗时、QPS 及成功率分位数据。

### 2. OpenTelemetry 链路追踪
通过引入 `trpc.group/trpc-go/trpc-filter/opentelemetry`：
```go
import (
	_ "trpc.group/trpc-go/trpc-filter/opentelemetry"
)
```
配置文件开启 filter：
```yaml
server:
  filter:
    - opentelemetry # 自动拦截请求并上报 Span 至本地 Jaeger (4317 端口)
```

### 3. Trace-to-Log 联动技巧
在 Grafana 的 Loki 数据源设置中：
1. 添加 `Derived fields`。
2. Name 设为 `TraceID`。
3. Regex 设为 `"trace_id":"([0-9a-fA-F]+)"`。
4. Internal link 勾选并关联至 `Jaeger` 数据源。
5. **效果**：在 Grafana 日志查看器中，点击日志行中的 TraceID，页面右侧直接滑出对应的完整 Jaeger 调用链瀑布图。
