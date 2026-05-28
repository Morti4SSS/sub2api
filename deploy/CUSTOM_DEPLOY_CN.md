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
