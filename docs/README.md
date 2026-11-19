# 文档中心 (Documentation Center)

本目录为 **tRPC-Go 微服务与生产级治理体系本地部署项目** 的核心知识与技术方案中心。

---

## 目录索引

### 1. 核心架构规划
- [**本地部署全套 tRPC-Go 微服务与生产级治理体系最终实施方案 (Final Master Plan)**](final_execution_plan_trpc_microservices.md)  
  *【最终定稿执行方案 v2.2.0】确立以 K3s 为底座，PolarisMesh 为治理控制面，trpc-gateway 为接入网关，Prometheus/Loki/Jaeger 为可观测平台，“Drone CI 构建大盘 + ArgoCD/Rollouts 灰度自愈大盘”双大盘闭环，以及多模存储（MySQL/Redis/RabbitMQ/MinIO/MongoDB）与复杂报表套件（TiDB + ClickHouse）的六阶段实施路线图。*
- [**现代实时数仓与 OLAP 引擎全景图：除了 Doris 与 ClickHouse，还有哪些更好的选择？**](modern_data_warehouse_and_olap_landscape.md)  
  *深入探索现代大数据与分析领域的四大主流阵营：嵌入式分析神作 DuckDB (进程内零后台/直查MinIO Parquet)、新一代全场景 MPP 湖仓 StarRocks (原Doris团队进化体/多表Join天花板)、面向终端高并发交互式报表 Apache Pinot、跨异构库联邦查询 Trino (Presto)，并给出单机与生产环境选型建议。*
- [**需不需要引入 Apache Doris 或 ClickHouse 配合 Grafana 做业务数据统计上报？ (Doris vs ClickHouse & Reporting Evaluation)**](doris_vs_clickhouse_business_analytics_guide.md)  
  *全面剖析业务统计上报的演进路径：何时无需引入额外数仓（利用现有 MySQL 直连 Grafana 或 Prometheus 业务打点 0 内存开销）、何时必须引入专业 OLAP、Doris 与 ClickHouse 的极限性能/单机内存开销 (400MB vs 3~5GB)/多表 Join/Grafana 插件集成全景对比与选型决策。*
- [**TiDB + ClickHouse 复杂报表与实时数仓架构设计方案 (Analytics & Reporting Architecture)**](tidb_and_clickhouse_reporting_architecture.md)  
  *针对高并发事务与复杂报表解耦诉求，深度剖析 TiDB (分布式 HTAP / TiFlash 列存毫秒级聚合) 与 ClickHouse (极致列式 OLAP / 亿级行为埋点大宽表秒级查询) 的分工定位、双引擎协同拓扑与单机容器编排验证。*
- [**本地全套微服务治理体系实施方案**](trpc_local_microservice_governance_plan.md)  
  *涵盖系统全景架构、需求生命周期规范、Protobuf 契约驱动（Schema as Code）、灰度控制分流、质量监控（SLO）、OpenTelemetry 分布式链路追踪、Grafana Loki 结构化日志、业务运营大盘、CI/CD 自动化流水线及 16GB 本地硬件资源预算与 ADR 决策矩阵。*
- [**非关系型数据库 (NoSQL) 与扩展中间件完整盘点及必要性评估**](nosql_and_storage_middleware_evaluation.md)  
  *全面评估是否缺失非关系型数据库：深度解析分布式对象存储 MinIO (解决容器多副本文件共享丢失必选 ~150MB)、文档数据库 MongoDB (游戏道具背包动态Schema选型 ~300MB)、ES搜索与时序数据库，给出包含全套组件的 5.4GB 资源承载预算。*
- [**本地部署“偏向生产环境”的技术选型决策指南 (Local-to-Production Parity)**](local_production_parity_tech_selection.md)  
  *针对“在本地运行但架构全面向生产级标准看齐（生产同构、规范一致、零改动平滑上线）”的目标，深度权衡并提供 K3s、PolarisMesh、trpc-gateway、Loki、Jaeger 等全套生产级轻量化技术选型与 3.5~4.5GB 内存预算。*
- [**最佳工程闭环：可视化前端构建 (Visual CI) + ArgoCD/Rollouts 灰度治理**](visual_ci_with_argocd_rollouts_pipeline.md)  
  *彻底告别手动打包：采用“轻量 Web CI (Drone CI / Actions) 负责可视化点击与流式构建打包 + argocd-image-updater 自动联动 + Argo Rollouts 负责金丝雀灰度与 Prometheus 自动熔断”，实现 0 手动干预的端到端可视化生产闭环。*
- [**终极对决：RabbitMQ vs Apache Kafka 深度对比与选型决策**](rabbitmq_vs_kafka_deep_dive.md)  
  *针对偏向 RabbitMQ 与 Kafka 的纠结，从设计哲学、开箱即用 Web 管理后台、单机资源开销（200MB vs 1.5GB）、动态路由与适用场景全面横向打擂台，给出明确首选结论。*
- [**前端容器化打包构建、自动化部署与版本管理方案 (Frontend DevOps & CD Guide)**](frontend_containerization_and_cd_plan.md)  
  *涵盖前端多阶段 Dockerfile 极简打包（< 25MB）、Nginx SPA 防缓存与长期静态缓存配置、K3s 滚动更新与秒级回滚（kubectl rollout undo）、前后端反向代理防跨域全套现成实战方案。*
- [**市面主流前端可视化平台评测：构建、发布与工程治理 (Frontend WebUI Platforms & Governance)**](frontend_webui_platforms_and_governance.md)  
  *针对告别纯脚本化诉求，深度评测开源自建版 Vercel (Coolify/Dokploy)、云原生标准控制台 (ArgoCD + Rollouts Dashboard) 以及环境治理专家 (Zadig)，提供带 Web UI 的构建、版本树回滚、PR预览环境与灰度看板全套选型对比。*
- [**消息队列中间件选型与 Redis 替代性深度评估 (Message Queue Selection & Redis Evaluation)**](message_queue_technology_selection.md)  
  *深入剖析能否直接用 Redis 代替消息队列（List/PubSub/Stream 对比与内存/丢数据瓶颈），评测 NATS JetStream (纯Go极轻量 ~50MB 首选)、RabbitMQ (带Web控制台)、Kafka、RocketMQ，给出清晰的落地建议。*
- [**行业现状深度调研：目前主流项目究竟使用什么消息队列 (MQ) 比较多？**](mainstream_mq_industry_analysis.md)  
  *宏观市场占有率与行业格局深度分析：Kafka (大数据日志霸主 60%)、RabbitMQ (通用业务与中小微企业主力 25%)、RocketMQ (国内电商交易金融事务 12%)、NATS (云原生Go微服务新星 5%)，按业务场景精准定位选型。*
- [**深度对比：Coolify 与 ArgoCD + Argo Rollouts Dashboard**](coolify_vs_argocd_rollouts_deep_dive.md)  
  *深入剖析两者的本质定位与功能上限：轻量一体化源码PaaS (Coolify) vs 工业级云原生持续交付与自动灰度分析引擎 (ArgoCD + Rollouts)，给出功能强大度评测与生产同构落地建议。*
- [**生产环境微服务与治理体系技术选型决策指南 (Production ADR Guide)**](production_technology_selection_guide.md)  
  *面向真实生产环境高并发、容灾高可用、弹性扩缩容、合规审计与成本控制诉求，提供容器编排、服务治理、API网关、日志监控、链路追踪、GitOps 流水线等八大维度的全景对比与落地方案。*

### 2. 实施知识与实操经验库 (Knowledge Base)
实施过程中的踩坑记录、环境搭建心得、配置样例与故障排查经验统一归档于 [`knowledge-base/`](knowledge-base/README.md)：
- [**01. 本地开发环境配置与依赖避坑指南**](knowledge-base/01-environment-setup.md)：Ubuntu 环境 Docker、Docker Compose、Go 1.24 配置与国内镜像加速。
- [**02. 北极星单机版部署与 tRPC 接入实践**](knowledge-base/02-polaris-standalone-setup.md)：PolarisMesh 单机容器拉起、服务注册发现、配置中心热更新及灰度路由规则验证。
- [**03. 轻量化可观测体系搭建经验**](knowledge-base/03-observability-quickstart.md)：Loki + Vector + Prometheus + Jaeger 内存精细控制（< 1.5GB）及 Trace-to-Log 联动配置。
- [**04. 生产环境选型分析：K3s vs Docker Compose 深度对比**](knowledge-base/04-k3s-vs-docker-compose-production.md)：生产环境下容器集群高可用、零停机平滑升级、弹性伸缩（HPA）及双模演进路线。

---

## 知识沉淀与迭代机制

后续在项目推进、代码编写、中间件联调与上线演练过程中，团队成员应遵循以下沉淀机制：
1. **方案变更记录**：若架构决策发生调整（如中间件变更、协议扩展），同步更新 `trpc_local_microservice_governance_plan.md` 的 ADR 章节。
2. **踩坑即沉淀**：凡是遇到耗时超过 15 分钟的环境配置、网络通信、插件兼容或端口冲突问题，及时在 `knowledge-base/` 目录下新增排坑文档。
3. **可复现性原则**：所有文档中的命令与配置均应保持可复制直接执行，并标注对应软件的具体版本。
