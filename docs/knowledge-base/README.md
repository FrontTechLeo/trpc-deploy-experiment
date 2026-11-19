# 实施知识经验沉淀库 (Knowledge Base)

本目录用于沉淀 **tRPC-Go 微服务与生产级治理体系在本地落地过程中** 的所有知识经验、最佳实践、踩坑记录及排错手册。

---

## 沉淀规范与准则

为了保证团队知识沉淀的系统性与可持续维护性，后续所有技术实践建议遵循以下沉淀规范：

1. **命名规范**：
   - 编号 + 业务/主题短语，例如 `01-environment-setup.md`、`02-polaris-standalone-setup.md`。
   - 专项故障分析与排障记录统一命名为 `troubleshooting-<date>-<issue>.md`。
2. **文档结构模板**：
   - **背景与目标**：解决什么问题，预期达成什么效果。
   - **技术方案与关键配置**：具体配置文件、代码片段或命令行步骤。
   - **常见踩坑与解决方案 (Pitfalls & Troubleshooting)**：记录遇到的具体错误日志、原因分析与修复手段。
   - **关键经验总结 (Lessons Learned)**：架构决策或开发调试的最佳实践提炼。
3. **版本同构与可重现**：
   - 记录具体的软件版本号（如 `Go 1.24.11`、`Polaris v1.16`、`Docker 27.x`）。
   - 提供可直接复制粘贴并在本地干净环境复现的命令与配置。

---

## 知识库索引

| 序号 | 文档名称 | 核心内容 | 状态 |
|---|---|---|---|
| 01 | [本地开发环境配置与依赖避坑指南](01-environment-setup.md) | Ubuntu 环境下 Docker、Docker Compose、Go 1.24 配置与国内镜像加速 | 已完成 |
| 02 | [北极星单机版部署与 tRPC 接入实践](02-polaris-standalone-setup.md) | PolarisMesh standalone 容器部署、配置中心热更新与路由规则验证 | 已完成 |
| 03 | [轻量化可观测体系搭建经验](03-observability-quickstart.md) | Loki + Vector + Prometheus + Jaeger 资源控制与 Trace-to-Log 联动 | 已完成 |
