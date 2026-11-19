# 前端容器化构建、自动化部署与版本管理方案指南 (Frontend DevOps & CD Guide)

> **文档版本**：v1.0.0  
> **制定日期**：2026-11-19  
> **适用框架**：Vue 3 (Vite), React, Next.js, Webpack/SPA 项目  
> **设计原则**：偏向生产环境同构、前后端契约打通、轻量快速构建、多版本回滚与动静分离。

---

## 一、 现成成熟的前端部署方案全景对比

| 方案模式 | 推荐方案 A：云原生标准 Nginx 多阶段容器化 + K3s 滚动更新【首选推荐】 | 方案 B：自建开源 PaaS 平台 (Coolify / Dokploy) | 方案 C：静态对象存储托管 (MinIO / S3) + CDN |
|---|---|---|---|
| **核心机制** | • 多阶段 Dockerfile 构建出极简 Nginx 镜像（< 25MB）。<br>• 通过 K3s Deployment 纳管与 Traefik Ingress 路由。 | • 开源版 Vercel / Netlify，提供 Web UI 一键 Git Push 自动化打包与发布。 | • 静态文件构建后上传私有 MinIO，Nginx 反代指向 MinIO 存储桶。 |
| **版本管理与回滚** | • 镜像版本打标准 Tag（`frontend-web:v1.2.0`）。<br>• 利用 K3s 原生 `kubectl rollout undo` 秒级回滚。 | • 平台 Web 界面查看构建历史，点击按钮秒级回滚到任意历史 commit。 | • 多版本目录并存（`/static/v1.0/`, `/static/v1.1/`），回滚只需切换 `index.html`。 |
| **与后端集成度** | **100% 统一**：与后端 tRPC-Go 服务同运行在 K3s 集群内，统一受 Ingress 调度，无跨域问题。 | 需单独维护一套 PaaS 控制面容器，与 K3s 集群略有割裂。 | 动静彻底解耦，但本地单机增加 MinIO 维护成本。 |
| **单机额外资源消耗** | **极低（< 50MB 内存）**，仅单 Nginx 容器运行。 | 中等（约 400 ~ 600MB 内存）。 | 中等（MinIO 约需 200MB 内存）。 |
| **实施难度与现成度** | **极高现成度**：Dockerfile 模板与 K3s YAML 现成可用，工业界标准落地。 | 高现成度：一键安装平台脚本，Web 点选。 | 中等：需编写上传脚本与路径重定向。 |

> **选型推荐**：首选 **方案 A（Nginx 多阶段容器化 + K3s 滚动更新）**，与当前项目已定选的 **方案 A（K3s + Polaris + trpc-gateway）** 在同一集群内原生协同，心智模型一致，开销极低。

---

## 二、 方案 A 落地实施规格规范

### 1. 标准多阶段 Docker 构建模板 (`Dockerfile`)
通过 Docker 多阶段构建，构建环境依赖与最终运行环境彻底隔离，产物镜像体积从几百兆缩小至 **20MB** 左右：

```dockerfile
# ==========================================
# 阶段 1: 构建编译阶段 (Build Stage)
# ==========================================
FROM node:20-alpine AS builder

WORKDIR /app

# 安装 pnpm (或使用 npm/yarn)
RUN corepack enable && corepack prepare pnpm@latest --activate

# 先复制依赖描述文件，充分利用 Docker 镜像缓存
COPY package.json pnpm-lock.yaml ./
RUN pnpm install --frozen-lockfile

# 复制前端源码并执行打包构建
COPY . .
RUN pnpm build

# ==========================================
# 阶段 2: 生产运行阶段 (Production Stage)
# ==========================================
FROM nginx:1.27-alpine

# 复制自定义 Nginx 配置文件
COPY nginx.conf /etc/nginx/conf.d/default.conf

# 从构建阶段复制打包好的静态资源
COPY --from=builder /app/dist /usr/share/nginx/html

EXPOSE 80

CMD ["nginx", "-g", "daemon off;"]
```

---

### 2. 前端 Nginx 生产配置模板 (`nginx.conf`)
核心要点：
1. **SPA 单页应用路由支持**：`try_files $uri $uri/ /index.html;`，防止前端路由（vue-router/react-router）刷新 404。
2. **防缓存策略 (No-Cache on HTML)**：`index.html` 严禁浏览器强缓存，保证发布新版后用户打开即拉取最新版本。
3. **长期静态缓存 (Long-term Cache on Assets)**：带 Hash 的 JS/CSS 文件开启 1 年强缓存，极大提升二次加载速度。
4. **反向代理后端网关**：`/api/` 路径自动转发至内部 `trpc-gateway`，彻底解决浏览器跨域（CORS）。

```nginx
server {
    listen 80;
    server_name localhost;

    # 启用 Gzip 高效压缩
    gzip on;
    gzip_min_length 1k;
    gzip_types text/plain application/javascript text/css application/json image/svg+xml;

    root /usr/share/nginx/html;

    # 1. 业务静态资源根路由
    location / {
        try_files $uri $uri/ /index.html;
    }

    # 2. index.html 绝不缓存 (保障版本即时生效)
    location = /index.html {
        add_header Cache-Control "no-cache, no-store, must-revalidate";
        add_header Pragma "no-cache";
        add_header Expires "0";
    }

    # 3. 静态构建产物 (js/css/img 包含 contenthash) 开启长期缓存
    location ~* \.(?:css|js|woff2?|svg|png|jpg|jpeg|gif|ico)$ {
        expires 1y;
        add_header Cache-Control "public, immutable";
        access_log off;
    }

    # 4. 后端 API 反向代理 (直通内网 trpc-gateway)
    location /api/ {
        proxy_pass http://trpc-gateway.default.svc.cluster.local:8080/;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }
}
```

---

### 3. K3s 自动化部署与版本管理 (`deploy/k8s-frontend.yaml`)

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: frontend-web
  namespace: default
  labels:
    app: frontend-web
spec:
  replicas: 2 # 生产多副本
  strategy:
    type: RollingUpdate
    rollingUpdate:
      maxSurge: 1
      maxUnavailable: 0 # 零停机滚动更新
  selector:
    matchLabels:
      app: frontend-web
  template:
    metadata:
      labels:
        app: frontend-web
    spec:
      containers:
        - name: web
          image: frontend-web:v1.0.0 # 镜像版本标签
          imagePullPolicy: IfNotPresent
          ports:
            - containerPort: 80
          readinessProbe:
            httpGet:
              path: /
              port: 80
            initialDelaySeconds: 2
            periodSeconds: 5
          resources:
            requests:
              cpu: 50m
              memory: 32Mi
            limits:
              cpu: 200m
              memory: 128Mi
---
apiVersion: v1
kind: Service
metadata:
  name: frontend-web
  namespace: default
spec:
  type: ClusterIP
  ports:
    - port: 80
      targetPort: 80
  selector:
    app: frontend-web
---
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: frontend-ingress
  namespace: default
spec:
  rules:
    - host: gametool.local # 本地自定义域名
      http:
        paths:
          - path: /
            pathType: Prefix
            backend:
              service:
                name: frontend-web
                port:
                  number: 80
```

---

## 三、 前端版本管理与秒级回滚机制

1. **镜像版本打标规范 (SemVer)**：
   - 每次发布新版本打唯一 Git Tag 与镜像 Tag：
     `docker build -t frontend-web:v1.0.1 .`
2. **零停机滚动更新命令**：
   ```bash
   kubectl set image deployment/frontend-web web=frontend-web:v1.0.1
   ```
   K3s 会先拉起新 Pod，等待其通过健康就绪检查后，再优雅销毁旧 Pod，整个切换过程对前端访问者完全无感。
3. **秒级紧急回滚 (Rollback)**：
   若新版本上线后发现重大前端 Bug：
   ```bash
   # 查看历史版本记录
   kubectl rollout history deployment/frontend-web

   # 一秒回滚到上一版本
   kubectl rollout undo deployment/frontend-web
   ```

---

## 四、 自动化流水线集成 (Makefile 一键指令)

在项目的 `Makefile` 中集成前端标准构建与发布流水线：

```makefile
FRONTEND_VERSION ?= v1.0.0

frontend-build:
	@echo ">>> 正在进行前端多阶段容器化构建: frontend-web:$(FRONTEND_VERSION)"
	docker build -t frontend-web:$(FRONTEND_VERSION) ./web

frontend-deploy:
	@echo ">>> 正在更新 K3s 前端部署镜像..."
	kubectl set image deployment/frontend-web web=frontend-web:$(FRONTEND_VERSION)
	kubectl rollout status deployment/frontend-web

frontend-rollback:
	@echo ">>> 正在回滚前端版本至上一稳定版本..."
	kubectl rollout undo deployment/frontend-web
```
