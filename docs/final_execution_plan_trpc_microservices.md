# 本地部署全套 tRPC-Go 微服务与生产级治理体系最终实施方案 (Final Master Plan)

> **方案版本**：v2.1.0 (最终定稿执行版)  
> **定稿日期**：2026-11-19  
> **核心原则**：生产同构 (Production Parity)、单机自闭环 (Self-Contained)、契约驱动 (Schema as Code)、全可视化操作 (Dual-Dashboard GitOps)  
> **确定选型**：**K3s + PolarisMesh + trpc-gateway + Prometheus/Grafana + Loki + Jaeger + 双大盘 (Drone CI + ArgoCD/Rollouts) + RabbitMQ**

---

## 一、 系统全景架构与数据流拓扑

系统整体由 **“统一接入网关 + tRPC-Go 微服务网格 + 北极星治理控制面 + 一体化可观测底座 + 双大盘交付体系 + 消息队列中间件”** 构成：

```mermaid title="本地生产同构 tRPC-Go 全套微服务与治理体系架构全景图"
flowchart TD
    subgraph DevSpace["前端/研发交互与操作入口"]
        GitRepo["Git 源码仓 (前端/后端业务代码)"]
        DroneUI["【大盘一：可视化构建大盘】Drone CI\n• 提交代码自动触发 / 网页一键点选构建\n• 实时流式高亮日志展示\n• 极简 Docker 多阶段镜像打包 (< 25MB)"]
    end

    subgraph RegistrySpace["本地镜像存储与自动感知桥梁"]
        Registry["本地私有镜像仓 (Local Registry:5000)\n存放业务镜像 frontend-web:v1.0.x / svr:v1.0.x"]
        Updater["ArgoCD Image Updater\n• 秒级监听 Registry 新 Tag\n• 0 手动干预，自动提交 GitOps 清单变更"]
    end

    subgraph ClusterBase["K3s 单机轻量级 Kubernetes 集群底座 (本地 16GB 物理机)"]
        ArgoUI["【大盘二：可视化灰度大盘】ArgoCD + Argo Rollouts\n• 树状架构健康拓扑大盘 (Deployment/Service/Ingress)\n• 金丝雀灰度分流大盘 (10% -> 50% -> 100%)\n• Prometheus 指标异常 0.5s 自动熔断回退\n• 一键可视化秒级版本回滚 (Rollback)"]

        subgraph IngressGateway["接入与网关层"]
            Ingress["K3s Traefik Ingress (80/443 入口)"]
            GW["trpc-gateway 网关 Pod\n• HTTP/JSON 转 tRPC 二进制 RPC 协议\n• 自动透传 W3C TraceContext\n• 自动透传流量染色标签 (Canary Tag)"]
        end

        subgraph ServiceMesh["tRPC-Go 业务微服务集群 (Deployment 滚动多副本)"]
            AuthSvr["auth-service (用户鉴权 Pod)\n[健康就绪/存活探针 + 优雅停机]"]
            GameSvr["game-service (核心业务 Pod)\n[动态扩缩容 + 故障秒级自愈]"]
            ToolSvr["tool-service (运营工具 Pod)"]
        end

        subgraph GovernancePlane["服务治理控制面 (PolarisMesh 单机版)"]
            Polaris["PolarisMesh\n• 真实服务注册与动态心跳健康检查\n• 配置中心秒级热更新\n• 灰度权重分流规则引擎\n• Sentinel 熔断降级"]
        end

        subgraph ObservabilityStack["统一可观测平台 (Trace-to-Log 一键联动)"]
            Jaeger["Jaeger (分布式链路追踪瀑布图)"]
            Prometheus["Prometheus (SRE 黄金信号指标监控)"]
            Loki["Grafana Loki + Vector (轻量流式日志检索)"]
            Grafana["Grafana 统一监控大盘 (指标/日志/链路三位一体)"]
        end

        subgraph StorageLayer["基础存储与消息中间件"]
            MySQL["MySQL 8.0 (持久化数据卷)"]
            Redis["Redis 7.2 (缓存/分布式锁)"]
            RabbitMQ["RabbitMQ 3.13 (AMQP/Web控制台)"]
        end
    end

    %% 流程关系
    GitRepo --> DroneUI
    DroneUI -->|推送新镜像| Registry
    Registry -.->|自动感知新 Tag| Updater
    Updater -->|自动声明式更新| ArgoUI
    ArgoUI -->|渐进式灰度发布| IngressGateway

    Ingress --> GW
    GW -->|tRPC 二进制协议| AuthSvr
    AuthSvr -->|tRPC 二进制协议| GameSvr
    GameSvr -->|tRPC 二进制协议| ToolSvr

    AuthSvr & GameSvr & ToolSvr <--> Polaris
    GW <--> Polaris

    AuthSvr & GameSvr & ToolSvr -.->|OTel Tracing| Jaeger
    AuthSvr & GameSvr & ToolSvr -.->|/metrics| Prometheus
    AuthSvr & GameSvr & ToolSvr -.->|JSON Logs| Loki
    Prometheus & Loki & Jaeger --> Grafana
    Prometheus -.->|指标异常自动触发熔断| ArgoUI

    AuthSvr & GameSvr & ToolSvr --> MySQL & Redis & RabbitMQ
```

---

## 二、 最终确定的技术选型矩阵与规划

| 架构模块 | 最终确定选型 | 容器化/运行方式 | 端口规划 | 关键功能与生产同构特性 |
|---|---|---|---|---|
| **集群编排底座** | **K3s** | 单节点集群系统服务 | 6443 (API Server) | 完整的标准 Kubernetes 规范、Pod 探针、Service CoreDNS 与滚动发布 |
| **微服务框架** | **tRPC-Go** | Go 1.24 原生/容器 | 10001 ~ 10010 | 腾讯开源高性能 RPC 框架，Protobuf 契约驱动，支持多协议与拦截器插件 |
| **服务治理中心** | **PolarisMesh (北极星)** | Docker / K3s Pod | 8090 (UI), 8091 (gRPC) | 服务注册发现心跳、配置中心秒级热更新、动态灰度染色路由与熔断降级 |
| **微服务 API 网关** | **trpc-gateway** | K3s Deployment | 8080 (HTTP), 8443 (HTTPS) | 原生 Protobuf 契约解包，HTTP/JSON 转 tRPC 二进制，透传 Trace 与染色标签 |
| **消息队列中间件** | **RabbitMQ (带管理后台)** | 容器 (挂载持久卷) | 5672 (AMQP), 15672 (Web UI) | 业务异步解耦、任务派发、开箱即用 Web 管理控制台、灵活 Topic 路由与死信队列 |
| **【大盘一】构建大盘** | **Drone CI / Woodpecker** | Docker 单容器 | 8000 (Web UI) | **可视化源码构建**：接入 Git，点击按钮一键打包，流式查看多阶段镜像构建日志 |
| **自动化联动桥梁** | **argocd-image-updater** | K3s Controller | 内网运行 | **0 手动干预**：秒级自动监听镜像仓新 Tag，自动向 Git 提交更新触发灰度 |
| **【大盘二】灰度大盘** | **ArgoCD + Argo Rollouts** | K3s Pods | 8081 (ArgoCD), 31000 (Rollouts UI) | **可视化灰度与自愈**：树状拓扑大盘、10% 金丝雀分流、Prometheus 指标自动熔断、一键秒级回滚 |
| **分布式链路追踪** | **OpenTelemetry + Jaeger** | K3s Pod | 16686 (UI), 4317 (OTLP gRPC) | W3C TraceContext 全链路透传，全链路 Span 瀑布图与慢调用毫秒级定位 |
| **质量指标监控** | **Prometheus + Alertmanager** | K3s Pod | 9090 (Prometheus), 9093 | SRE 黄金信号指标采集（成功率 > 99.5%、P99 延时、QPS、Goroutine 数量） |
| **结构化日志系统** | **Vector + Grafana Loki** | K3s DaemonSet/Pod | 3100 (Loki API) | 结构化 JSON 流式日志存储，比传统 ELK 节约 80% 开销，支持 Trace-to-Log 一键跳转 |
| **统一监控可视化** | **Grafana 11.x** | K3s Pod | 3000 (Web 控制台) | 聚合质量大盘、业务运营大盘、Loki 日志探索与 Jaeger 追踪瀑布图 |
| **持久存储中间件** | **MySQL 8.0 + Redis 7.2** | 容器 (本地挂载持久化卷) | 3306 (MySQL), 6379 (Redis) | 真实关系数据库与分布式缓存环境，验证真实连接池、事务与分布式锁 |

---

## 三、 双大盘端到端交付运作机制

彻底告别命令行与手动打包，实现 **“提交代码 $\rightarrow$ 页面点选构建 $\rightarrow$ 自动触发灰度 $\rightarrow$ 指标自愈熔断”**：

### 1. 构建阶段（大盘一：Drone CI）
1. 开发者在本地 IDE 编写业务代码，执行 `git push`；
2. 登录 **Drone CI Web 控制台**，可在界面上看到流水线自动启动，或手动点击 **“Trigger Build”**；
3. 页面实时展示流式构建终端动画（依赖拉取 $\rightarrow$ 静态检查 $\rightarrow$ 极简多阶段打包），构建出 `frontend-web:v1.0.2` 并自动推入本地私有镜像仓库。

### 2. 自动衔接阶段（argocd-image-updater）
1. 运行在 K3s 内的 `argocd-image-updater` 在 10 秒内检测到镜像仓库出现新 Tag；
2. 自动向 GitOps 清单提交更新，将发布目标镜像更新为 `v1.0.2`。

### 3. 发布治理与熔断阶段（大盘二：ArgoCD + Argo Rollouts Dashboard）
1. 打开 **Argo Rollouts Web 控制台**，直观看到当前稳定版（90%）与灰度金丝雀版本（10%）的实时流量切分滑块；
2. **自动化自愈机制**：Rollouts 在灰度放量 10% 期间，自动调用 Prometheus 查询前端与接口健康指标：
   - **指标正常**：自动平滑推进至 50% $\rightarrow$ 100% 全量切流；
   - **指标恶化（如 5xx 突增 > 1%）**：**系统在 0.5 秒内自动判定熔断，秒级切回稳定旧版本**，无需运维熬夜盯盘！
3. **紧急一键回滚**：如遇特殊业务逻辑 Bug，在 ArgoCD 网页的“History and Rollback”列表点击任意上一稳定版本，一键秒级恢复。

---

## 四、 本地宿主机硬件资源开销预算

当前宿主机规格为 **Ubuntu 24.04、16GB 内存、300GB 磁盘**：

| 组件分类 | 具体组件 | 预估内存占用 | 预估 CPU 占用 | 资源可行性评估 |
|---|---|---|---|---|
| **底座与治理控制面** | K3s + PolarisMesh | ~1.1 GB | 0.4 ~ 0.8 核 | 运行极其轻量稳定 |
| **网关与业务服务** | trpc-gateway + 3个业务服务多副本 | ~0.7 GB | 0.4 ~ 1.0 核 | 满足微服务全链路调用 |
| **双大盘交付套件** | Drone CI + ArgoCD + Rollouts + Updater | ~0.7 GB | 0.3 ~ 0.8 核 | 全可视化构建与灰度控制 |
| **可观测平台** | Prometheus + Loki + Jaeger + Grafana + Vector | ~1.3 GB | 0.4 ~ 0.9 核 | 节约 80% 内存的高性能组合 |
| **存储与消息中间件** | MySQL 8.0 + Redis 7.2 + RabbitMQ 3.13 | ~1.15 GB | 0.3 ~ 0.6 核 | 带持久化数据卷，含 RabbitMQ Web 后台 |
| **全套汇总预算** | **全套生产同构微服务生态** | **约 4.95 GB** | **约 1.8 ~ 4.2 核** | **宿主机可用 12GB，余量高达 7GB+，从容运行** |

---

## 五、 五阶段落地实施路线图 (即将开始执行)

本实施方案分五个标准化阶段推进落地，每阶段产出明确且可验收：

```mermaid title="微服务治理体系五阶段落地执行路线图"
flowchart TD
    P1["阶段一：基础设施底座与单机自闭环 (Day 1~3)\n• 安装 Docker & K3s 单节点集群\n• 部署 PolarisMesh / MySQL / Redis / RabbitMQ\n• 搭建本地私有镜像仓库 (Local Registry:5000)"]
    P2["阶段二：微服务与网关最小链路跑通 (Day 4~6)\n• 搭建统一 Protobuf 契约仓 api/proto/ 引入 buf 门禁\n• 实现首个 tRPC-Go 服务注册到北极星并支持配置热更\n• 部署 trpc-gateway 实现 HTTP 转 tRPC 路由调用"]
    P3["阶段三：双大盘交付与灰度发布闭环 (Day 7~9)\n• 部署 Drone CI 构建大盘，调通多阶段极简前端打包\n• 部署 ArgoCD + Argo Rollouts 灰度大盘与 Updater 桥梁\n• 实现代码提交 -> Web构建 -> 自动灰度切流 -> 一键回滚"]
    P4["阶段四：可观测平台与生产级自动熔断联动 (Day 10~12)\n• 部署 Prometheus + Grafana + Loki + Jaeger\n• 接入 tRPC 拦截器，打通 Trace-to-Log 一键联动\n• 配置 Argo Rollouts 联动 Prometheus 异常自动熔断自愈"]
    P5["阶段五：全链路集成压测、应急演练与交付验收 (Day 13~15)\n• 模拟前端高并发与白屏故障，演练无人值守自动熔断\n• 沉淀操作排障手册与生产一键迁移 Helm 模板\n• 联合评审与交付验收"]

    P1 --> P2 --> P3 --> P4 --> P5
```

---

## 六、 下一步即刻执行准备 (Immediate Action Item)

方案已获得最终确定，**即将正式启动阶段一（基础设施底座与单机自闭环）**：
1. 本地安装并配置 K3s 单节点服务与 Docker Engine 权限；
2. 部署基础服务治理控制面（PolarisMesh）、存储套件（MySQL、Redis）与消息队列（RabbitMQ）；
3. 验证本地开发机对 K3s、北极星控制台（8090）及 RabbitMQ 管理后台（15672）的访问连通性。
