# 匠心版并行部署（不影响旧版）

适用场景：

- 服务器上已经运行旧版 `sub2api`
- 现在要把当前仓库里的“匠心版”额外部署上去
- 两套系统暂时并行存在，方便后续大版本切换
- 不迁移旧数据，不共用旧数据库/Redis/数据目录

## 设计原则

这套方案默认做了 4 层隔离：

- 端口隔离：匠心版默认绑定 `127.0.0.1:18081`
- 容器隔离：容器名默认是 `sub2api-jx / sub2api-jx-postgres / sub2api-jx-redis`
- 存储隔离：使用 `./jx-data / ./jx-postgres_data / ./jx-redis_data`
- 网络隔离：使用 `sub2api-jx-network`

这样旧版继续跑旧版，匠心版完全独立，不会抢端口，也不会共享数据。

## 文件

- Compose: [docker-compose.jx.yml](/home/heligong/Downloads/sub2api/deploy/docker-compose.jx.yml)
- 环境变量模板: [".env.jx.example"](/home/heligong/Downloads/sub2api/deploy/.env.jx.example)
- 反向代理示例: [Caddyfile.jx.example](/home/heligong/Downloads/sub2api/deploy/Caddyfile.jx.example)
- 初始化脚本: [docker-deploy-jx.sh](/home/heligong/Downloads/sub2api/deploy/docker-deploy-jx.sh)

## 快速部署

在新服务器上：

```bash
git clone <your-repo> sub2api-jx
cd sub2api-jx/deploy
chmod +x docker-deploy-jx.sh
./docker-deploy-jx.sh
docker compose --env-file .env.jx -f docker-compose.jx.yml up -d --build
```

## 生产环境约束

这个仓库的生产发布现在统一采用以下约束：

- 禁止在云服务器上执行 `docker compose ... --build`、`pnpm build`、`go build`
- 必须先在本地构建前端和 Linux 二进制
- 构建完成后，将二进制上传到服务器，再通过 override 挂载方式重启容器
- 服务器只负责备份、替换产物、重启，不负责源码编译

推荐流程：

```bash
# 本地
cd frontend
pnpm exec vite build

cd ../backend
CGO_ENABLED=0 GOOS=linux go build -tags embed -ldflags="-s -w -X main.BuildType=release" -trimpath -o ../deploy/sub2api-local ./cmd/server

scp ../deploy/sub2api-local root@<server>:/opt/sub2api-jx/deploy/sub2api-local.new

# 云服务器
cd /opt/sub2api-jx/deploy
mv -f sub2api-local.new sub2api-local
chmod 755 sub2api-local
cat > docker-compose.app-binary.override.yml <<'EOF'
services:
  sub2api-jx:
    volumes:
      - ./sub2api-local:/app/sub2api:ro
EOF
docker compose --env-file .env.jx -f docker-compose.jx.yml -f docker-compose.app-binary.override.yml up -d --force-recreate --no-deps --no-build sub2api-jx
```

## 验证

```bash
docker compose --env-file .env.jx -f docker-compose.jx.yml ps
docker compose --env-file .env.jx -f docker-compose.jx.yml logs -f sub2api-jx
curl http://127.0.0.1:18081/health
```

如果 `.env.jx` 里未设置 `ADMIN_PASSWORD`，首启后到日志里找自动生成的管理员密码。

## 推荐流量接入方式

推荐不要直接暴露匠心版公网端口，而是继续让反向代理接入：

- 旧版继续服务：`api.example.com -> 旧版 upstream`
- 匠心版单独域名：`jx-api.example.com -> 127.0.0.1:18081`

参考 [Caddyfile.jx.example](/home/heligong/Downloads/sub2api/deploy/Caddyfile.jx.example)。

## 大版本切换

等你确认匠心版可用后，切换只需要改反向代理目标：

- 切换前：`api.example.com -> 旧版`
- 切换后：`api.example.com -> 127.0.0.1:18081`

这样切流时不需要停旧版，也不需要立即删除旧版。

## 停止 / 回滚

停止匠心版：

```bash
docker compose --env-file .env.jx -f docker-compose.jx.yml down
```

旧版不受影响。

如果切流后要快速回滚，只需要把反向代理再指回旧版 upstream，然后 reload Caddy/Nginx。

## 注意事项

- 这套方案默认 **不迁移数据**
- 旧版和匠心版的管理员账号、数据库、Redis 都互不共享
- 如果你的服务器已经有 `5432/6379/8080` 被旧版占用，也没关系，因为这套并行方案默认不会把 Postgres/Redis 暴露到宿主机，只额外占用 `127.0.0.1:18081`
