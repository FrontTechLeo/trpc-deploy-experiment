#!/usr/bin/env bash
set -euo pipefail

echo "=========================================================="
echo ">>> tRPC-Go 微服务治理体系 - 宿主机底座 (Docker + K3s) 初始化"
echo "=========================================================="

if [ "$EUID" -ne 0 ]; then
  SUDO='sudo'
else
  SUDO=''
fi

echo ">>> [1/4] 安装 Docker 与 Docker Compose 插件..."
if ! command -v docker &>/dev/null; then
  $SUDO apt-get update
  $SUDO apt-get install -y docker.io docker-compose-v2
  $SUDO usermod -aG docker "$USER" || true
  $SUDO systemctl enable --now docker
  echo ">>> Docker 安装完成。"
else
  echo ">>> Docker 已存在: $(docker --version)"
fi

echo ">>> [2/4] 验证 Docker 服务状态..."
$SUDO systemctl is-active --quiet docker && echo ">>> Docker 服务运行正常。"

echo ">>> [3/4] 安装 K3s 单节点轻量级集群..."
if ! command -v k3s &>/dev/null; then
  curl -sfL https://get.k3s.io | INSTALL_K3S_EXEC="--write-kubeconfig-mode 644" sh -
  echo ">>> K3s 单节点集群安装完成: $(k3s --version)"
else
  echo ">>> K3s 已存在: $(k3s --version)"
fi

echo ">>> [4/4] 配置当前用户免密 sudo 权限 (支持 Agent 后续自主编排拉起容器)..."
echo "$USER ALL=(ALL) NOPASSWD:ALL" | $SUDO tee /etc/sudoers.d/99-agent-nopasswd >/dev/null
$SUDO chmod 440 /etc/sudoers.d/99-agent-nopasswd
echo ">>> 免密 sudo 配置完成！"

echo "=========================================================="
echo ">>> 宿主机环境底座初始化全部完成！"
echo "=========================================================="
