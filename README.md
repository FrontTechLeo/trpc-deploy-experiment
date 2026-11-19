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

- [**本地部署全套 tRPC-Go 微服务与生产级治理体系最终实施方案 (Final Master Plan)**](docs/final_execution_plan_trpc_microservices.md)
  - 核心架构全景：K3s + PolarisMesh + trpc-gateway + 可观测套件 + 双大盘体系
  - 最终确定技术选型全景矩阵、端口规划与 16GB 宿主机资源预算（总计 ~4.7GB 内存）
  - 双大盘全流程闭环：代码提交 -> Drone CI 网页点选构建 -> 镜像自动入库 -> argocd-image-updater 自动更新 -> Argo Rollouts 10% 灰度切流 -> Prometheus 自动熔断自愈
  - 即将启动的五阶段实施路线图 (Phase 1 ~ Phase 5)
- [本地全套 tRPC-Go 微服务治理体系实施方案](docs/trpc_local_microservice_governance_plan.md)
  - 总体架构拓扑与数据流
  - 需求管理、版本迭代与语义化版本规范
  - 基于 Protobuf 的接口契约驱动与前后端协同 (Schema as Code)
  - 灰度发布控制与流量染色路由机制
  - 质量监控（SRE 黄金信号）、分布式链路追踪（OTel+Jaeger）、轻量结构化日志（Loki+Vector）
  - 本地环境硬件资源预算（3.5~4.5GB 内存占用评估）
  - 核心架构决策矩阵 (ADR) 与分阶段推进路线图 (Phase 1 ~ Phase 5)
- [本地部署“偏向生产环境”的技术选型决策指南 (Local-to-Production Parity)](docs/local_production_parity_tech_selection.md)
  - 生产同构原则：杜绝“本地能跑线上崩掉”，本地全面采用生产级技术规范
  - 集群底座：K3s（标准 Kubernetes 资源对象、Pod 健康探针、Deployment 滚动发布）
  - 服务治理与网关：PolarisMesh（真实注册心跳、配置热更、流量染色） + trpc-gateway（Protobuf 契约解包）
  - 可观测性：Prometheus + Grafana + OpenTelemetry + Jaeger + Loki（实现生产级 Trace-to-Log 联动）
  - 单机硬件预算：单机 16GB 物理机实测仅需占用 3.2~4.6GB 内存，兼顾生产级规范与单机自闭环
- [前端容器化打包构建、自动化部署与版本管理方案 (Frontend DevOps & CD Guide)](docs/frontend_containerization_and_cd_plan.md)
  - 生产级多阶段 Docker 构建（极小 Nginx 镜像 < 25MB）
  - 生产级 Nginx 配置：HTML `no-cache` 与静态带 Hash 资源 1 年强缓存
  - K3s 滚动更新、就绪探针、一键秒级回滚 (`kubectl rollout undo`)
  - 反向代理配置彻底解决浏览器跨域（CORS）问题
- [市面主流前端可视化平台评测：构建、发布与工程治理 (Frontend WebUI Platforms & Governance)](docs/frontend_webui_platforms_and_governance.md)
  - 告别纯脚本化：开箱即用 Web UI 构建、发布与多版本治理平台评测
  - 开源自建 Vercel 标杆：Coolify（自动检测框架、PR 独立 Preview 预览环境、版本历史树一键回滚）
  - 云原生 K8s 事实标准控制台：ArgoCD + Argo Rollouts Dashboard（前端可视化拓扑大盘、金丝雀灰度流量切分大盘）
  - 微服务环境治理专家：Zadig（前端多套隔离联调自测环境动态拉起与回收）
- [最佳工程闭环：可视化前端构建 (Visual CI) + ArgoCD/Rollouts 灰度治理](docs/visual_ci_with_argocd_rollouts_pipeline.md)
  - 彻底告别手动打包镜像：解耦“可视化构建 (CI)”与“GitOps 灰度治理 (CD)”
  - 极轻量 Web CI 引擎选型：Drone CI / Woodpecker / GitHub Actions
  - 关键无感桥梁：`argocd-image-updater` 实现新镜像秒级自动感知与灰度触发
  - 端到端全景：Web 界面点选构建 -> 自动多阶段打包 -> 自动金丝雀切流 -> 异常自动熔断
- [消息队列中间件选型与 Redis 替代性深度评估 (Message Queue Selection & Redis Evaluation)](docs/message_queue_technology_selection.md)
  - Redis 替代性深度剖析：List/PubSub/Stream 局限性与消息积压 OOM/丢消息风险
  - 为什么生产不建议全靠 Redis：削峰填谷内存瓶颈、持久化可靠性、缺少死信与延迟队列
  - 四大专业 MQ 评测：NATS JetStream (首选推荐，纯Go生态极轻 ~50MB)、RabbitMQ (带管理后台)、Kafka、RocketMQ
  - 本项目双轨落地建议：初期轻量选 Redis Stream / 生产严谨选 NATS
- [行业现状深度调研：目前主流项目究竟使用什么消息队列 (MQ) 比较多？](docs/mainstream_mq_industry_analysis.md)
  - 市场占有率宏观大盘：两大霸主 (Kafka 55% + RabbitMQ 25%) + 国内电商顶流 (RocketMQ 12%) + 云原生新贵 (NATS 5%)
  - 为什么 Kafka 是大数据与日志流的唯一事实标准
  - 为什么 RabbitMQ 是中小型企业与通用微服务业务的首选主力（开箱即用 Web 管理后台）
  - 针对 tRPC-Go + 本地 16GB 生产同构的最佳匹配建议
- [深度对比：Coolify 与 ArgoCD + Argo Rollouts Dashboard](docs/coolify_vs_argocd_rollouts_deep_dive.md)
  - 功能差异与本质区别：全包型应用PaaS（Coolify） vs 专业级云原生交付与灰度引擎（ArgoCD+Rollouts）
  - 功能强大度分析：灰度分流控制（10%->100%）、Prometheus 自动分析熔断（无人值守回滚）、GitOps 状态防篡改
  - 针对当前微服务体系的最佳协同落地建议
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
