# TokenHub 部署指南

> 版本：V1.0  
> 最后更新：2025 年 5 月

本文档提供 TokenHub 平台的完整部署步骤。

## 前置要求

### 硬件要求

- **最低配置**: 2 核 CPU / 4GB 内存 / 40GB 磁盘
- **推荐配置**: 4 核 CPU / 8GB 内存 / 100GB 磁盘
- **操作系统**: Ubuntu 22.04 LTS (推荐) 或其他 Linux 发行版

### 软件要求

- Docker 24.0+
- Docker Compose 2.20+
- Git

## 快速部署（开发环境）

### 步骤 1: 安装 Docker

```bash
# Ubuntu/Debian
curl -fsSL https://get.docker.com | sh

# 启动 Docker
sudo systemctl enable docker
sudo systemctl start docker
```

### 步骤 2: 克隆项目

```bash
git clone <repository-url> tokenhub
cd tokenhub/deploy
```

### 步骤 3: 配置环境变量

```bash
# 复制环境变量模板
cp .env.example .env

# 编辑配置文件
nano .env
```

**必须修改的配置项**:

```bash
# MySQL 密码（请修改为强密码）
MYSQL_ROOT_PASSWORD=your_secure_mysql_root_password
MYSQL_PASSWORD=your_secure_mysql_password

# Redis 密码（请修改为强密码）
REDIS_PASSWORD=your_secure_redis_password

# New API 密钥（使用 openssl rand -base64 32 生成）
SESSION_SECRET=$(openssl rand -base64 32)
CRYPTO_SECRET=$(openssl rand -base64 32)
```

### 步骤 4: 启动服务

```bash
# 启动所有服务
docker-compose up -d

# 查看启动日志
docker-compose logs -f
```

### 步骤 5: 访问平台

- **用户前台**: http://localhost:3000
- **管理后台**: http://localhost:3000/admin
- **API 端点**: http://localhost:3000/v1

首次登录后，请在管理后台配置 AI 渠道和模型价格。

## 生产环境部署

### 1. 域名与 SSL 证书

#### 1.1 域名备案（中国大陆服务器必需）

在阿里云/腾讯云等平台完成 ICP 备案。

#### 1.2 申请 SSL 证书

使用 Let's Encrypt 免费证书：

```bash
# 安装 Certbot
sudo apt install certbot python3-certbot-nginx

# 申请证书
sudo certbot certonly --standalone -d your-domain.com
```

证书文件位置：
- `/etc/letsencrypt/live/your-domain.com/fullchain.pem`
- `/etc/letsencrypt/live/your-domain.com/privkey.pem`

#### 1.3 配置 Nginx HTTPS

编辑 `deploy/nginx/conf.d/default.conf`，取消 HTTPS 配置的注释，并修改域名和证书路径。

### 2. 安全加固

#### 2.1 防火墙配置

```bash
# 只开放必要端口
sudo ufw allow 22/tcp    # SSH
sudo ufw allow 80/tcp    # HTTP
sudo ufw allow 443/tcp   # HTTPS
sudo ufw enable
```

#### 2.2 修改默认端口（可选）

编辑 `.env` 文件，修改暴露的端口：

```bash
MYSQL_PORT=13306
REDIS_PORT=16379
```

### 3. 数据备份

#### 3.1 MySQL 备份脚本

创建备份脚本 `/opt/tokenhub/backup.sh`:

```bash
#!/bin/bash
BACKUP_DIR="/opt/backups/mysql"
DATE=$(date +%Y%m%d_%H%M%S)
DB_NAME="tokenhub"
DB_USER="tokenhub_user"
DB_PASS="your_mysql_password"

mkdir -p $BACKUP_DIR
mysqldump -u$DB_USER -p$DB_PASS $DB_NAME > $BACKUP_DIR/backup_$DATE.sql

# 保留最近 7 天的备份
find $BACKUP_DIR -name "backup_*.sql" -mtime +7 -delete
```

设置定时任务：

```bash
crontab -e
# 每天凌晨 2 点备份
0 2 * * * /opt/tokenhub/backup.sh
```

#### 3.2 Redis 备份

Redis 已配置 RDB 持久化，数据保存在 `deploy/data/redis` 目录。

### 4. 监控告警

#### 4.1 服务健康检查

```bash
# 检查容器状态
docker-compose ps

# 检查 New API 健康状态
curl http://localhost:3000/api/status
```

#### 4.2 日志查看

```bash
# 查看实时日志
docker-compose logs -f new-api

# 查看错误日志
docker-compose logs --tail=100 new-api | grep ERROR
```

### 5. 性能优化

#### 5.1 调整 Docker 资源限制

编辑 `docker-compose.yml`，添加资源限制：

```yaml
services:
  new-api:
    deploy:
      resources:
        limits:
          cpus: '2'
          memory: 2G
        reservations:
          cpus: '1'
          memory: 1G
```

#### 5.2 MySQL 优化

编辑 `deploy/init-scripts/mysql.cnf`，添加优化配置。

## AI 渠道配置

登录管理后台后，按以下步骤配置 AI 渠道：

### 1. OpenAI 渠道

1. 进入 **管理后台 > 渠道管理**
2. 点击 **添加渠道**
3. 选择 **OpenAI** 类型
4. 填写 API Key（从 OpenAI 官网获取）
5. 配置模型和价格
6. 保存并测试

### 2. Claude 渠道

1. 进入 **管理后台 > 渠道管理**
2. 点击 **添加渠道**
3. 选择 **Anthropic** 类型
4. 填写 API Key（从 Anthropic 官网获取）
5. 配置模型和价格
6. 保存并测试

### 3. 其他渠道

类似步骤配置 Gemini、DeepSeek、MiniMax 等渠道。

## 支付配置（新开发功能）

### 1. 支付宝当面付

1. 申请企业支付宝账号
2. 开通当面付功能
3. 获取 APP_ID 和应用私钥/公钥
4. 编辑 `.env` 文件，添加配置：

```bash
ALIPAY_APP_ID=your_app_id
ALIPAY_PRIVATE_KEY=your_private_key
ALIPAY_PUBLIC_KEY=alipay_public_key
```

### 2. 微信支付

1. 申请微信商户账号
2. 获取 APP_ID、MCH_ID 和 API Key
3. 配置回调地址
4. 编辑 `.env` 文件，添加配置

### 3. 聚合支付（推荐个人使用）

1. 注册 Payspi/XPay 账号
2. 获取 API Key
3. 配置 Webhook 地址
4. 编辑 `.env` 文件，添加配置

## 常见问题

### Q1: 容器启动失败

检查 Docker 日志：

```bash
docker-compose logs new-api
docker-compose logs mysql
docker-compose logs redis
```

常见原因：
- 端口被占用
- 环境变量配置错误
- 磁盘空间不足

### Q2: 无法连接数据库

1. 检查 MySQL 容器是否正常运行
2. 验证 `.env` 中的数据库配置
3. 检查网络连接

### Q3: API 调用返回 402

余额不足，请充值或联系管理员增加额度。

### Q4: 渠道显示离线

1. 检查渠道 API Key 是否有效
2. 检查服务器网络连接
3. 查看错误日志

## 升级指南

### 升级到新版本

```bash
# 备份数据
docker-compose down

# 拉取最新镜像
docker-compose pull

# 重新启动
docker-compose up -d

# 查看日志确认启动成功
docker-compose logs -f
```

## 技术支持

- 项目文档：`/workspace/docs/`
- GitHub Issues: [提交问题](https://github.com/your-org/tokenhub/issues)
- 技术支持邮箱：support@tokenhub.example.com

---

**祝您部署顺利！**
