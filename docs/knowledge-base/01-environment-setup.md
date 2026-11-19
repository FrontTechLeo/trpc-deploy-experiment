# 01. 本地开发环境配置与依赖避坑指南

> **记录时间**：2026-11-19  
> **适用系统**：Ubuntu 22.04 / 24.04 LTS (x86_64)  
> **关键工具**：Docker Engine, Docker Compose, Go 1.24+, trpc-cmd, buf  

---

## 一、 环境基准确认

本地宿主机基线要求：
- 内存：建议 $\ge$ 8GB（本环境实测 16GB，可用 12GB，完全满足）
- 磁盘：建议可用空间 $\ge$ 50GB（用于存放 Docker 镜像与数据卷）
- 内核：Linux 6.x / 7.x 64位系统

检查当前系统环境：
```bash
uname -a
free -h
df -h
go version
```

---

## 二、 Docker & Docker Compose 安装与避坑

### 1. 安装步骤
在 Ubuntu 24.04 上推荐使用 Docker 官方源进行安装：

```bash
# 1. 卸载旧版本（如有）
sudo apt-get remove docker docker-engine docker.io containerd runc -y

# 2. 安装依赖并配置官方 GPG Key
sudo apt-get update
sudo apt-get install ca-certificates curl gnupg -y
sudo install -m 0755 -d /etc/apt/keyrings
curl -fsSL https://download.docker.com/linux/ubuntu/gpg | sudo gpg --dearmor -o /etc/apt/keyrings/docker.gpg
sudo chmod a+r /etc/apt/keyrings/docker.gpg

# 3. 添加 apt 源
echo \
  "deb [arch=$(dpkg --print-architecture) signed-by=/etc/apt/keyrings/docker.gpg] https://download.docker.com/linux/ubuntu \
  $(. /etc/os-release && echo "$VERSION_CODENAME") stable" | \
  sudo tee /etc/apt/sources.list.d/docker.list > /dev/null

# 4. 安装 Docker 核心组件
sudo apt-get update
sudo apt-get install docker-ce docker-ce-cli containerd.io docker-buildx-plugin docker-compose-plugin -y
```

### 2. 免 sudo 权限配置（避坑点）
避免在运行 docker 命令时必须加 sudo：
```bash
sudo usermod -aG docker $USER
newgrp docker # 或重新登录当前终端
docker ps     # 验证是否无需 sudo 即可访问 docker socket
```

### 3. Docker 镜像加速与 DNS 配置（国内网络环境必备）
在 `/etc/docker/daemon.json` 中配置镜像源与日志限制：
```json
{
  "registry-mirrors": [
    "https://docker.m.daocloud.io",
    "https://dockerproxy.com"
  ],
  "log-driver": "json-file",
  "log-opts": {
    "max-size": "50m",
    "max-file": "3"
  }
}
```
配置完成后重启 Docker 守护进程：
```bash
sudo systemctl daemon-reload
sudo systemctl restart docker
```

---

## 三、 Go 语言与 tRPC 工具链配置

### 1. GOPROXY 配置
使用国内代理加速 Go 模块拉取：
```bash
go env -w GO111MODULE=on
go env -w GOPROXY=https://goproxy.cn,direct
```

### 2. tRPC-Go 脚手架工具安装
安装官方代码生成与初始化命令行工具 `trpc`：
```bash
# 安装 trpc-cmd
go install trpc.group/trpc-go/trpc-cmd/trpc@latest

# 安装 protobuf 编译器插件
go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
go install trpc.group/trpc-go/trpc-cmd/trpc/plugins/mock@latest

# 验证安装
trpc --version
```

### 3. Buf 契约工具安装
用于契约代码校验（lint）与兼容性检测（breaking check）：
```bash
# 快速安装二进制
BIN="/usr/local/bin" && \
VERSION="v1.34.0" && \
curl -sSL \
  "https://github.com/bufbuild/buf/releases/download/${VERSION}/buf-$(uname -s)-$(uname -m)" \
  -o "${BIN}/buf" && \
chmod +x "${BIN}/buf"
```

---

## 四、 常见问题与排错经验 (Troubleshooting)

1. **Ubuntu 24.04 软件源缺少对应 codename**：
   - 现象：`apt-get update` 提示 `noble` 源 404。
   - 解法：在 `/etc/apt/sources.list.d/docker.list` 中临时替换为 `jammy` (Ubuntu 22.04 LTS 源)，兼容性正常。
2. **Docker 日志把磁盘打满**：
   - 现象：长期运行微服务后磁盘空间不足。
   - 解法：必须在 `daemon.json` 配置 `log-opts` 中的 `max-size: 50m` 和 `max-file: 3`，防止日志无节制增长。
