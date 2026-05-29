# VPS 镜像部署

这份说明用于服务器只拉镜像运行，不在 VPS 上构建源码。

## 镜像来源

`VPS` 分支推送后，GitHub Actions 会构建并推送镜像：

```text
ghcr.io/morti4sss/sub2api:vps
ghcr.io/morti4sss/sub2api:0.1.133-vps
```

如果镜像包保持私有，服务器需要先登录 GHCR：

```bash
echo "你的 GitHub PAT" | docker login ghcr.io -u Morti4SSS --password-stdin
```

PAT 至少需要 `read:packages` 权限。也可以在 GitHub Packages 页面把镜像设为 public，这样服务器无需登录即可拉取。

## 首次部署

```bash
git clone -b VPS https://github.com/Morti4SSS/sub2api.git
cd sub2api

cp deploy/.env.example deploy/.env
nano deploy/.env

mkdir -p deploy/data deploy/postgres_data deploy/redis_data

docker compose -f deploy/docker-compose.local.yml -f deploy/docker-compose.vps-image.override.yml pull
docker compose -f deploy/docker-compose.local.yml -f deploy/docker-compose.vps-image.override.yml up -d
```

`deploy/.env` 至少配置：

```env
POSTGRES_PASSWORD=换成强密码
JWT_SECRET=换成长随机字符串
TOTP_ENCRYPTION_KEY=换成长随机字符串
ADMIN_EMAIL=你的管理员邮箱
ADMIN_PASSWORD=你的管理员密码
SERVER_PORT=8080
```

## 查看状态

```bash
docker compose -f deploy/docker-compose.local.yml -f deploy/docker-compose.vps-image.override.yml ps
docker compose -f deploy/docker-compose.local.yml -f deploy/docker-compose.vps-image.override.yml logs -f sub2api
```

## 后续更新

本机合并并推送 `VPS` 后，等 GitHub Actions 的 `VPS Image` 成功，再到服务器执行：

```bash
cd sub2api
git pull origin VPS
docker compose -f deploy/docker-compose.local.yml -f deploy/docker-compose.vps-image.override.yml pull
docker compose -f deploy/docker-compose.local.yml -f deploy/docker-compose.vps-image.override.yml up -d
```

数据和配置仍然保存在：

```text
deploy/.env
deploy/data/
deploy/postgres_data/
deploy/redis_data/
```
