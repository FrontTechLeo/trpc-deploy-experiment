# 02. PolarisMesh 北极星单机版部署与 tRPC-Go 接入实践

> **记录时间**：2026-11-19  
> **组件定位**：注册中心、配置中心、灰度路由与熔断降级控制面  
> **镜像版本**：`polarismesh/polaris-server:latest`  

---

## 一、 北极星单机版 Docker Compose 部署

北极星官方提供了一体化单机镜像（内置 embedded 本地存储），在本地开发和单机测试场景中无需额外部署 MySQL 即可跑通全部功能。

### 1. 编排配置 (`deploy/docker-compose.infra.yml`)
```yaml
version: '3.8'

services:
  polaris:
    image: polarismesh/polaris-server:latest
    container_name: polaris-standalone
    restart: always
    ports:
      - "8090:8090" # Web 控制台
      - "8091:8091" # gRPC 注册发现端口
      - "8093:8093" # HTTP 配置与服务接口
    environment:
      - POLARIS_MODE=standalone
    volumes:
      - polaris_data:/root/polaris/data

volumes:
  polaris_data:
```

### 2. 启动与验证
```bash
docker compose -f deploy/docker-compose.infra.yml up -d polaris

# 验证健康状态
curl http://localhost:8090
```
浏览器打开 `http://localhost:8090`，默认账号密码为 `polaris / polaris`。

---

## 二、 tRPC-Go 接入北极星治理

tRPC-Go 官方提供了北极星命名服务与配置中心插件包：`trpc.group/trpc-go/trpc-naming-polaris` 与 `trpc.group/trpc-go/trpc-config-polaris`。

### 1. `trpc_go.yaml` 服务配置样例
```yaml
global:
  namespace: Development
  env_name: local

server:
  app: gametool
  server: auth-service
  bin_path: ./bin/auth-service
  conf_path: ./conf
  registry: polaris # 声明注册中心使用 polaris
  service:
    - name: trpc.gametool.auth.AuthService
      ip: 127.0.0.1
      port: 10001
      protocol: trpc
      network: tcp

plugins:
  registry:
    polaris:
      address_list: ["127.0.0.1:8091"]
      heartbeat_interval: 5000 # 5秒心跳

  config:
    polaris:
      address_list: ["127.0.0.1:8091"]
      namespace: Development
      group: gametool
```

### 2. 代码中配置热更新监听实践
```go
package main

import (
	"fmt"
	"trpc.group/trpc-go/trpc-go"
	"trpc.group/trpc-go/trpc-go/config"
	"trpc.group/trpc-go/trpc-go/log"
	_ "trpc.group/trpc-go/trpc-naming-polaris"
)

func main() {
	s := trpc.NewServer()

	// 监听北极星配置中心变更
	confWatcher, err := config.Watch("app_dynamic.json")
	if err == nil {
		go func() {
			for event := range confWatcher {
				log.Infof("收到北极星配置热更新: %s", event.Value)
			}
		}()
	}

	if err := s.Serve(); err != nil {
		log.Fatal(err)
	}
}
```

---

## 三、 灰度路由规则实操配置

在北极星 Web 控制台中配置路由规则：
1. **服务路由** $\rightarrow$ **创建路由规则**。
2. **规则匹配条件**：请求标签 `traffic_env` == `canary`。
3. **目标服务子集**：路由至实例标签 `version` == `v2.0.0` 的灰度实例。
4. **降级策略**：若灰度实例健康实例数为 0，自动降级路由至 `v1.0.0` 稳定版。
