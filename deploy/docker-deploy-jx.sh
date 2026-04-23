#!/bin/bash
# =============================================================================
# Sub2API "匠心版" Side-by-Side Deployment Preparation Script
# =============================================================================
# Prepares an isolated deployment next to an existing old Sub2API instance.
#
# Outputs:
#   - .env.jx
#   - jx-data/
#   - jx-postgres_data/
#   - jx-redis_data/
#
# Start with:
#   docker compose --env-file .env.jx -f docker-compose.jx.yml up -d --build
# =============================================================================

set -euo pipefail

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

print_info() {
  echo -e "${BLUE}[INFO]${NC} $1"
}

print_success() {
  echo -e "${GREEN}[SUCCESS]${NC} $1"
}

print_warning() {
  echo -e "${YELLOW}[WARNING]${NC} $1"
}

print_error() {
  echo -e "${RED}[ERROR]${NC} $1"
}

command_exists() {
  command -v "$1" >/dev/null 2>&1
}

generate_secret() {
  openssl rand -hex 32
}

main() {
  echo ""
  echo "=============================================="
  echo "  Sub2API 匠心版并行部署准备"
  echo "=============================================="
  echo ""

  if ! command_exists openssl; then
    print_error "openssl 未安装，请先安装 openssl。"
    exit 1
  fi

  if [ ! -f ".env.jx.example" ]; then
    print_error "请在 deploy 目录内执行该脚本。"
    exit 1
  fi

  if [ -f ".env.jx" ]; then
    print_warning ".env.jx 已存在。"
    read -p "覆盖并重新生成密钥？(y/N): " -r
    echo
    if [[ ! ${REPLY:-} =~ ^[Yy]$ ]]; then
      print_info "已取消。"
      exit 0
    fi
  fi

  JWT_SECRET=$(generate_secret)
  TOTP_ENCRYPTION_KEY=$(generate_secret)
  POSTGRES_PASSWORD=$(generate_secret)

  cp .env.jx.example .env.jx

  if sed --version >/dev/null 2>&1; then
    sed -i "s/^JWT_SECRET=.*/JWT_SECRET=${JWT_SECRET}/" .env.jx
    sed -i "s/^TOTP_ENCRYPTION_KEY=.*/TOTP_ENCRYPTION_KEY=${TOTP_ENCRYPTION_KEY}/" .env.jx
    sed -i "s/^POSTGRES_PASSWORD=.*/POSTGRES_PASSWORD=${POSTGRES_PASSWORD}/" .env.jx
  else
    sed -i '' "s/^JWT_SECRET=.*/JWT_SECRET=${JWT_SECRET}/" .env.jx
    sed -i '' "s/^TOTP_ENCRYPTION_KEY=.*/TOTP_ENCRYPTION_KEY=${TOTP_ENCRYPTION_KEY}/" .env.jx
    sed -i '' "s/^POSTGRES_PASSWORD=.*/POSTGRES_PASSWORD=${POSTGRES_PASSWORD}/" .env.jx
  fi

  mkdir -p jx-data jx-postgres_data jx-redis_data
  chmod 600 .env.jx

  echo ""
  print_success "匠心版并行部署文件已准备完成。"
  echo ""
  echo "已生成："
  echo "  .env.jx"
  echo "  jx-data/"
  echo "  jx-postgres_data/"
  echo "  jx-redis_data/"
  echo ""
  echo "关键参数："
  echo "  SERVER_PORT=${SERVER_PORT:-18081} (默认写在 .env.jx 中)"
  echo "  POSTGRES_PASSWORD=${POSTGRES_PASSWORD}"
  echo "  JWT_SECRET=${JWT_SECRET}"
  echo "  TOTP_ENCRYPTION_KEY=${TOTP_ENCRYPTION_KEY}"
  echo ""
  echo "下一步："
  echo "  1. 按需编辑 .env.jx（域名、管理员邮箱、端口）"
  echo "  2. 启动匠心版："
  echo "     docker compose --env-file .env.jx -f docker-compose.jx.yml up -d --build"
  echo ""
  echo "  3. 查看状态："
  echo "     docker compose --env-file .env.jx -f docker-compose.jx.yml ps"
  echo ""
  echo "  4. 查看日志："
  echo "     docker compose --env-file .env.jx -f docker-compose.jx.yml logs -f sub2api-jx"
  echo ""
  echo "  5. 健康检查："
  echo "     curl http://127.0.0.1:18081/health"
  echo ""
  print_info "默认仅绑定 127.0.0.1:18081，不会直接影响旧版流量。"
}

main "$@"
