# tRPC-Go 微服务与生产级治理体系本地部署试验库

本项目用于在本地环境完整落地基于 **tRPC-Go** 的微服务架构全套生产级治理体系。通过单机容器化与轻量化中间件，实现与生产环境 **100% 架构同构**，涵盖需求管理、接口契约驱动、版本控制、灰度分流、可观测性（质量/链路/日志/运营指标）、前后端迭代协同与自动化 CI/CD 流水线。

---

## 目录结构规划

```text
.
├── docs/                       # 核心架构方案与技术设计文档
│   ├── trpc_local_microservice_governance_plan.md # 本地全套微服务治理体系规划方案
│   └── knowledge-base/         # 实施过程中的知识经验、踩坑记录与实操指南
│       ├── README.md           # 知识经验库说明与沉淀规范
│       ├── 01-environment-setup.md       # 本地开发环境配置与依赖避坑
│       ├── 02-polaris-standalone-setup.md # 北极星单机版部署与 tRPC 接入实践
│       └── 03-observability-quickstart.md # 轻量可观测体系 (Loki+Prometheus+Jaeger) 搭建
├── deploy/                     # 本地化部署基础设施编排 (Docker Compose / K3s)
│   ├── docker-compose.infra.yml # 基础中间件 (Polaris, MySQL, Redis)
│   └── docker-compose.monitor.yml # 观测套件 (Loki, Prometheus, Jaeger, Grafana)
├── proto/                      # 统一 Protobuf IDL 接口契约仓 (Schema as Code)
├── services/                   # tRPC-Go 微服务源码目录
│   ├── auth-service/           # 用户鉴权服务
│   ├── game-service/           # 核心业务服务
│   └── tool-service/           # 运营工具服务
└── Makefile                    # 一键构建、生成代码、运行与测试工具链
```

---

## 核心方案文档索引

- [本地全套 tRPC-Go 微服务治理体系实施方案](docs/trpc_local_microservice_governance_plan.md)
  - 总体架构拓扑与数据流
  - 需求管理、版本迭代与语义化版本规范
  - 基于 Protobuf 的接口契约驱动与前后端协同 (Schema as Code)
  - 灰度发布控制与流量染色路由机制
  - 质量监控（SRE 黄金信号）、分布式链路追踪（OTel+Jaeger）、轻量结构化日志（Loki+Vector）
  - 本地环境硬件资源预算（3.5~4.5GB 内存占用评估）
  - 核心架构决策矩阵 (ADR) 与分阶段推进路线图 (Phase 1 ~ Phase 5)
- [生产环境微服务与治理体系技术选型决策指南 (Production ADR Guide)](docs/production_technology_selection_guide.md)
  - 真实生产高可用（HA）、多节点容灾、故障秒级自愈与弹性扩缩容（HPA）
  - 容器集群（托管K8s vs 高可用K3s）、服务治理（PolarisMesh vs Nacos）、网关层（APISIX+trpc-gateway）
  - 结构化日志（Loki vs ES）、链路追踪（OTel+Tempo）、监控告警（VictoriaMetrics）
  - 生产安全交付范式：GitOps (ArgoCD) 声明式持续交付与一键 Git 回滚
  - 推荐生产落地套餐（公有云主流套餐 vs 私有化自主可控套餐）

---

## 知识经验与实施踩坑沉淀 (Knowledge Base)

在项目推进与实施过程中，所有环境配置细节、排错排障经历、架构权衡均统一沉淀至 [`docs/knowledge-base/`](docs/knowledge-base/README.md)：
1. [01. 本地环境准备与依赖安装避坑指南](docs/knowledge-base/01-environment-setup.md)
2. [02. PolarisMesh 北极星单机版部署与 tRPC-Go 接入经验](docs/knowledge-base/02-polaris-standalone-setup.md)
3. [03. 轻量化可观测性套件部署与资源控制经验](docs/knowledge-base/03-observability-quickstart.md)

---

## 快速开始 (快速概览)

1. **环境准备**：
   - 操作系统：Linux / macOS
   - Go 语言：>= 1.20 (推荐 1.24)
   - 容器底座：Docker & Docker Compose
2. **查阅规划方案**：
   - 阅读 [本地部署全套微服务实施方案](docs/trpc_local_microservice_governance_plan.md) 了解整体架构与技术决策。
