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
