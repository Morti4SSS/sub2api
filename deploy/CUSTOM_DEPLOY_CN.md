# 自定义版 VPS 部署说明

这份说明用于把当前仓库的 `VPS` 分支部署到服务器，同时保持“代码”和“配置/数据”分离，方便以后迁移 VPS 或继续跟进上游更新。

## 推荐部署方式

推荐在服务器上使用：

```bash
docker compose -f deploy/docker-compose.local.yml -f deploy/docker-compose.custom.override.yml up -d --build
```

含义如下：

- `deploy/docker-compose.local.yml`
  - 使用本地目录保存数据，便于整体迁移
- `deploy/docker-compose.custom.override.yml`
  - 不直接拉公共镜像，而是基于当前仓库源码构建你的自定义镜像

## 目录分离建议

部署目录下建议保留这些内容：

- `deploy/.env`
  - 环境变量配置，包含端口、管理员账号、数据库密码、JWT 密钥等
- `deploy/data/`
  - 应用运行数据
  - 如果你需要固定配置文件，放到 `deploy/data/config.yaml`
- `deploy/postgres_data/`
  - PostgreSQL 数据目录
- `deploy/redis_data/`
  - Redis 数据目录

这些目录和文件都不应该提交到 Git。

仓库里只保留：

- 代码
- Docker Compose 模板
- 部署说明文档
- `.env.example`

## 首次部署步骤

1. 拉取 `VPS` 分支代码

```bash
git clone -b VPS <你的仓库地址>
cd sub2api
```

2. 准备环境变量

```bash
cp deploy/.env.example deploy/.env
```

重点至少要改这些：

- `POSTGRES_PASSWORD`
- `JWT_SECRET`
- `TOTP_ENCRYPTION_KEY`
- `ADMIN_EMAIL`
- `ADMIN_PASSWORD`

3. 准备数据目录

```bash
mkdir -p deploy/data deploy/postgres_data deploy/redis_data
```

4. 启动

```bash
docker compose -f deploy/docker-compose.local.yml -f deploy/docker-compose.custom.override.yml up -d --build
```

5. 查看日志

```bash
docker compose -f deploy/docker-compose.local.yml -f deploy/docker-compose.custom.override.yml logs -f sub2api
```

## 配置文件放哪里

如果你后面需要把更多配置写进 `config.yaml`，推荐放这里：

```text
deploy/data/config.yaml
```

因为容器里已经把 `deploy/data/` 挂载到 `/app/data`，应用会优先从这些路径查找配置：

- `DATA_DIR`
- 当前目录
- `./backend`
- 向上查找后的 `backend`
- `./config`
- `/app/data`
- `/etc/sub2api`

这样做的好处是：

- 本地开发和 Docker 部署都更容易复用配置
- 迁移 VPS 时不需要把配置硬编码进镜像

## 账号 JSON 加密备份

这个备份是为了保护账号 JSON 里的 refresh token、access token 等敏感信息。推荐方式是：VPS 只保存加密快照和公钥，本机保存私钥，并定时把加密快照拉回本机解密成本地 JSON。

1. 在本机生成密钥

```bash
openssl genpkey -algorithm RSA -pkeyopt rsa_keygen_bits:4096 -out account_backup_private.pem
openssl rsa -pubout -in account_backup_private.pem -out account_backup_public.pem
```

2. 只把公钥上传到服务器

```bash
cp account_backup_public.pem deploy/data/account_backup_public.pem
```

私钥 `account_backup_private.pem` 不要上传 VPS，也不要提交到 Git。

3. 在 `deploy/.env` 启用服务端加密快照

```env
ACCOUNT_JSON_BACKUP_ENABLED=true
ACCOUNT_JSON_BACKUP_PUBLIC_KEY_FILE=/app/data/account_backup_public.pem
ACCOUNT_JSON_BACKUP_DIR=/app/data/account_backups
ACCOUNT_JSON_BACKUP_SERVER_RETAIN_COUNT=5
ACCOUNT_JSON_BACKUP_DEBOUNCE_SECONDS=120
```

启用后，账号凭证变更、手动令牌刷新、后台令牌刷新成功时会触发备份。批量刷新场景会做防抖，不会每个账号都生成一份快照。

4. 本机配置同步脚本

复制示例配置：

```powershell
Copy-Item deploy\scripts\account-backup-sync.example.env deploy\scripts\account-backup-sync.env
```

填写这些值：

```env
SUB2API_BASE_URL=https://你的域名
SUB2API_ADMIN_API_KEY=后台设置里的 Admin API Key
ACCOUNT_BACKUP_PRIVATE_KEY_FILE=C:\sub2api-backup\account_backup_private.pem
ACCOUNT_BACKUP_OUTPUT_DIR=C:\sub2api-backup\json
ACCOUNT_BACKUP_ENCRYPTED_DIR=C:\sub2api-backup\encrypted
ACCOUNT_BACKUP_RETAIN_COUNT=10
```

然后在本机运行：

```powershell
pwsh -NoProfile -ExecutionPolicy Bypass -File deploy\scripts\account-backup-sync.ps1 -Config deploy\scripts\account-backup-sync.env
```

Windows 任务计划程序里也用同一条命令即可。`ACCOUNT_BACKUP_RETAIN_COUNT=10` 表示本机只保留最近 10 份解密 JSON，超过后自动删除最旧的；加密缓存也会按同样数量保留。

## 后续迁移 VPS

迁移时，核心是带走“代码版本”和“数据目录”：

1. 在旧机器停服务

```bash
docker compose -f deploy/docker-compose.local.yml -f deploy/docker-compose.custom.override.yml down
```

2. 备份并迁移这些内容：

- 整个项目代码目录
- `deploy/.env`
- `deploy/data/`
- `deploy/postgres_data/`
- `deploy/redis_data/`

如果启用了账号 JSON 加密备份，`deploy/data/` 里会包含公钥和服务器端加密快照；本机私钥和已解密 JSON 不属于 VPS 迁移内容，只需要继续保存在你的本机备份目录。

3. 到新机器后重新执行：

```bash
docker compose -f deploy/docker-compose.local.yml -f deploy/docker-compose.custom.override.yml up -d --build
```

## 后续更新建议

建议长期保持下面这个思路：

- `main`
  - 跟随你 fork 的官方主线
- `VPS`
  - 专门存放服务器部署所需的少量定制

以后补功能时：

- 优先按新版结构增量修改
- 不要再整文件覆盖旧版代码
- 尽量把改动集中在少量后端文件、部署文件和独立配置项里

这样后续继续合并上游更新时，冲突会小很多。
