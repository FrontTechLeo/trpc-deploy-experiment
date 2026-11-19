# 文档中心 (Documentation Center)

本目录为 **tRPC-Go 微服务与生产级治理体系本地部署项目** 的核心知识与技术方案中心。

---

## 目录索引

### 1. 核心架构规划
- [**本地全套微服务治理体系实施方案**](trpc_local_microservice_governance_plan.md)  
  *涵盖系统全景架构、需求生命周期规范、Protobuf 契约驱动（Schema as Code）、灰度控制分流、质量监控（SLO）、OpenTelemetry 分布式链路追踪、Grafana Loki 结构化日志、业务运营大盘、CI/CD 自动化流水线及 16GB 本地硬件资源预算与 ADR 决策矩阵。*
- [**本地部署“偏向生产环境”的技术选型决策指南 (Local-to-Production Parity)**](local_production_parity_tech_selection.md)  
  *针对“在本地运行但架构全面向生产级标准看齐（生产同构、规范一致、零改动平滑上线）”的目标，深度权衡并提供 K3s、PolarisMesh、trpc-gateway、Loki、Jaeger 等全套生产级轻量化技术选型与 3.5~4.5GB 内存预算。*
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
