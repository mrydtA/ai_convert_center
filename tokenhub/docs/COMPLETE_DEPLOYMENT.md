# TokenHub 完整部署指南

> 文档版本：V1.0  
> 更新日期：2025 年 5 月 22 日  
> 适用环境：Linux (Ubuntu 20.04+/CentOS 7+)

---

## 📋 目录

1. [项目概述](#1-项目概述)
2. [系统架构](#2-系统架构)
3. [前置准备](#3-前置准备)
4. [快速部署（Docker Compose）](#4-快速部署 docker-compose)
5. [手动部署（生产环境）](#5-手动部署生产环境)
6. [New API 配置](#6-new-api-配置)
7. [支付系统集成](#7-支付系统集成)
8. [前端页面部署](#8-前端页面部署)
9. [验证与测试](#9-验证与测试)
10. [运维管理](#10-运维管理)
11. [常见问题](#11-常见问题)

---

## 1. 项目概述

### 1.1 项目组成

TokenHub 项目由以下核心组件构成：

| 组件 | 说明 | 来源 |
|------|------|------|
| **New API** | 核心 API 中转系统 | ✅ 开源项目直接部署 |
| **MySQL** | 数据库 | ✅ 标准组件 |
| **Redis** | 缓存与会话管理 | ✅ 标准组件 |
| **Nginx** | 反向代理与负载均衡 | 🛠️ 定制配置 |
| **支付模块** | 支付宝/微信支付对接 | 🆕 全新开发 |
| **充值页面** | 在线充值前端 | 🆕 全新开发 |
| **套餐页面** | 套餐购买前端 | 🆕 全新开发 |

### 1.2 已交付内容

```
/workspace/tokenhub/
├── deploy/                          # 部署配置
│   ├── docker-compose.yml           # Docker Compose 配置
│   ├── .env.example                 # 环境变量模板
│   ├── nginx/
│   │   ├── nginx.conf               # Nginx 主配置
│   │   └── conf.d/default.conf      # 站点配置
│   ├── init-scripts/
│   │   └── 01_init_tables.sql       # 数据库初始化脚本
│   ├── ssl/                         # SSL 证书目录
│   ├── data/                        # 数据持久化目录
│   └── logs/                        # 日志目录
├── src/                             # 源代码
│   ├── payment/
│   │   └── gateway.go               # 支付网关（Go）
│   └── frontend/
│       ├── topup/index.html         # 充值页面
│       └── plans/index.html         # 套餐购买页面
└── docs/                            # 文档
    ├── DEPLOYMENT.md                # 部署文档
    └── API_GUIDE.md                 # API 文档
```

---

## 2. 系统架构

```
                                    ┌─────────────────┐
                                    │     用户端       │
                                    └────────┬────────┘
                                             │
                                    ┌────────▼────────┐
                                    │    Nginx        │
                                    │  (反向代理)      │
                                    └────────┬────────┘
                                             │
                    ┌────────────────────────┼────────────────────────┐
                    │                        │                        │
          ┌─────────▼──────────┐  ┌──────────▼───────────┐  ┌────────▼────────┐
          │   New API          │  │   支付网关服务        │  │  静态资源服务器  │
          │   (Go)             │  │   (Go)               │  │  (Nginx)        │
          │   - API 转发        │  │   - 支付宝对接        │  │  - 充值页面     │
          │   - Token 计费      │  │   - 微信支付对接      │  │  - 套餐页面     │
          │   - 用户管理        │  │   - 订单管理         │  │                 │
          └─────────┬──────────┘  └──────────┬───────────┘  └─────────────────┘
                    │                        │
          ┌─────────▼──────────┐  ┌──────────▼───────────┐
          │      MySQL          │  │       Redis          │
          │   - 用户数据         │  │   - 会话缓存         │
          │   - 订单记录         │  │   - 限流计数         │
          │   - 用量统计         │  │   - 支付状态         │
          └─────────────────────┘  └──────────────────────┘
```

---

## 3. 前置准备

### 3.1 服务器要求

| 配置项 | 最低要求 | 推荐配置 |
|--------|---------|---------|
| CPU | 2 核 | 4 核+ |
| 内存 | 4GB | 8GB+ |
| 磁盘 | 40GB SSD | 100GB+ SSD |
| 带宽 | 5Mbps | 10Mbps+ |
| 系统 | Ubuntu 20.04 / CentOS 7+ | Ubuntu 22.04 LTS |

### 3.2 域名与备案

1. **域名注册**：准备一个已备案的域名（如：`example.com`）
2. **ICP 备案**：国内服务器必须完成 ICP 备案
3. **DNS 解析**：将域名解析到服务器 IP

```bash
# DNS 解析示例（在域名服务商处配置）
A 记录：example.com → 你的服务器 IP
A 记录：api.example.com → 你的服务器 IP
A 记录：www.example.com → 你的服务器 IP
```

### 3.3 SSL 证书

使用 Let's Encrypt 免费证书：

```bash
# 安装 Certbot
sudo apt update
sudo apt install certbot python3-certbot-nginx -y

# 获取证书
sudo certbot --nginx -d example.com -d www.example.com -d api.example.com
```

### 3.4 软件依赖

```bash
# 更新系统
sudo apt update && sudo apt upgrade -y

# 安装基础工具
sudo apt install -y curl wget git vim unzip

# 安装 Docker
curl -fsSL https://get.docker.com -o get-docker.sh
sudo sh get-docker.sh
sudo systemctl enable docker
sudo systemctl start docker

# 安装 Docker Compose
sudo curl -L "https://github.com/docker/compose/releases/latest/download/docker-compose-$(uname -s)-$(uname -m)" -o /usr/local/bin/docker-compose
sudo chmod +x /usr/local/bin/docker-compose

# 验证安装
docker --version
docker-compose --version
```

---

## 4. 快速部署（Docker Compose）

### 4.1 克隆项目

```bash
cd /opt
# 如果是 Git 仓库
git clone <your-repo-url> tokenhub
cd tokenhub/deploy

# 或者复制已有文件
# cp -r /workspace/tokenhub/deploy/* /opt/tokenhub/deploy/
```

### 4.2 配置环境变量

```bash
# 复制环境变量模板
cp .env.example .env

# 编辑配置文件
vim .env
```

**.env 文件配置说明**：

```ini
# ========== New API 配置 ==========
NEW_API_VERSION=latest
NEW_API_PORT=3000

# 数据库配置
DB_TYPE=mysql
DB_HOST=mysql
DB_PORT=3306
DB_NAME=newapi
DB_USER=newapi
DB_PASSWORD=YourStrongPassword123!

# Redis 配置
REDIS_HOST=redis
REDIS_PORT=6379
REDIS_PASSWORD=YourRedisPassword123!

# New API 管理员初始密码
NEW_API_ADMIN_PASSWORD=AdminSecurePass123!

# JWT 密钥（随机生成）
JWT_SECRET=$(openssl rand -hex 32)

# 会话密钥
SESSION_SECRET=$(openssl rand -hex 32)

# ========== 支付配置 ==========
PAYMENT_ALIPAY_APP_ID=你的支付宝 APP_ID
PAYMENT_ALIPAY_PRIVATE_KEY=你的支付宝私钥
PAYMENT_ALIPAY_PUBLIC_KEY=支付宝公钥

PAYMENT_WECHAT_APP_ID=你的微信 APP_ID
PAYMENT_WECHAT_MCH_ID=你的微信商户号
PAYMENT_WECHAT_API_KEY=你的微信 API 密钥

# 支付回调地址
PAYMENT_NOTIFY_URL=https://example.com/api/payment/notify

# ========== 邮件配置（可选）==========
MAIL_SERVER=smtp.qq.com
MAIL_PORT=587
MAIL_USER=noreply@example.com
MAIL_PASSWORD=你的 SMTP 密码
MAIL_FROM=TokenHub <noreply@example.com>

# ========== 系统配置 ==========
SYSTEM_NAME=TokenHub
SYSTEM_URL=https://example.com
FRONTEND_URL=https://example.com
API_URL=https://api.example.com
```

### 4.3 初始化数据库

```bash
# 确保 init-scripts 目录有正确的权限
chmod 755 ./init-scripts

# 启动 MySQL 并执行初始化脚本
docker-compose up -d mysql
sleep 30

# 执行初始化 SQL
docker exec -i tokenhub-mysql mysql -u root -proot < ./init-scripts/01_init_tables.sql
```

### 4.4 启动所有服务

```bash
# 构建并启动所有服务
docker-compose up -d

# 查看服务状态
docker-compose ps

# 查看日志
docker-compose logs -f
```

### 4.5 验证部署

```bash
# 检查服务是否正常运行
curl http://localhost:3000/api/status

# 访问前端页面
# 浏览器打开：http://你的服务器 IP/topup/index.html
# 浏览器打开：http://你的服务器 IP/plans/index.html
```

---

## 5. 手动部署（生产环境）

### 5.1 部署 New API

```bash
# 拉取 New API 镜像
docker pull justsong/new-api:latest

# 创建运行目录
mkdir -p /opt/new-api/{data,logs}

# 运行 New API
docker run -d \
  --name new-api \
  -p 3000:3000 \
  -v /opt/new-api/data:/app/data \
  -v /opt/new-api/logs:/app/logs \
  -e TZ=Asia/Shanghai \
  -e MYSQL_DSN="root:password@tcp(mysql:3306)/newapi?charset=utf8mb4&parseTime=True&loc=Local" \
  -e REDIS_CONN_STRING="redis://:password@redis:6379/0" \
  -e SESSION_SECRET=$(openssl rand -hex 32) \
  -e JWT_SECRET=$(openssl rand -hex 32) \
  --restart always \
  justsong/new-api:latest
```

### 5.2 部署支付网关服务

```bash
# 进入支付模块目录
cd /workspace/tokenhub/src/payment

# 编译 Go 程序
go build -o payment-gateway main.go

# 创建 systemd 服务
sudo vim /etc/systemd/system/payment-gateway.service
```

**systemd 服务配置**：

```ini
[Unit]
Description=TokenHub Payment Gateway
After=network.target

[Service]
Type=simple
User=www-data
WorkingDirectory=/opt/tokenhub/payment
ExecStart=/opt/tokenhub/payment/payment-gateway
Restart=on-failure
RestartSec=5
Environment="GIN_MODE=release"
Environment="ALIPAY_APP_ID=xxx"
Environment="ALIPAY_PRIVATE_KEY=xxx"

[Install]
WantedBy=multi-user.target
```

```bash
# 启用服务
sudo systemctl daemon-reload
sudo systemctl enable payment-gateway
sudo systemctl start payment-gateway
sudo systemctl status payment-gateway
```

### 5.3 配置 Nginx

```bash
# 备份默认配置
sudo cp /etc/nginx/nginx.conf /etc/nginx/nginx.conf.bak

# 复制定制配置
sudo cp /workspace/tokenhub/deploy/nginx/nginx.conf /etc/nginx/nginx.conf
sudo cp /workspace/tokenhub/deploy/nginx/conf.d/default.conf /etc/nginx/sites-available/tokenhub

# 启用站点
sudo ln -s /etc/nginx/sites-available/tokenhub /etc/nginx/sites-enabled/

# 测试配置
sudo nginx -t

# 重载 Nginx
sudo systemctl reload nginx
```

### 5.4 部署前端页面

```bash
# 创建前端目录
sudo mkdir -p /var/www/tokenhub/{topup,plans}

# 复制前端文件
sudo cp /workspace/tokenhub/src/frontend/topup/index.html /var/www/tokenhub/topup/
sudo cp /workspace/tokenhub/src/frontend/plans/index.html /var/www/tokenhub/plans/

# 设置权限
sudo chown -R www-data:www-data /var/www/tokenhub
sudo chmod -R 755 /var/www/tokenhub
```

---

## 6. New API 配置

### 6.1 初始化管理员账号

首次访问 New API 时，会提示设置管理员账号：

1. 访问 `http://你的域名/admin`
2. 使用 `.env` 中设置的 `NEW_API_ADMIN_PASSWORD` 登录
3. 修改默认密码

### 6.2 配置 AI 渠道

登录管理后台后，依次配置：

#### 添加 OpenAI 渠道

```
渠道类型：OpenAI
渠道名称：OpenAI 官方
API Key：sk-xxxxxxxxxxxxx
模型列表：gpt-3.5-turbo, gpt-4, gpt-4o
```

#### 添加 Claude 渠道

```
渠道类型：Anthropic
渠道名称：Claude 官方
API Key：sk-ant-xxxxxxxxxxxxx
模型列表：claude-3-sonnet-20240229, claude-3-opus-20240229
```

#### 添加其他渠道

按照相同方式添加 Gemini、DeepSeek 等渠道。

### 6.3 配置模型价格

在管理后台 → 模型管理 中设置价格：

| 模型 | 输入价格（$/M tokens） | 输出价格（$/M tokens） |
|------|---------------------|---------------------|
| gpt-3.5-turbo | 0.5 | 1.5 |
| gpt-4o | 5.0 | 15.0 |
| claude-3-sonnet | 3.0 | 15.0 |

### 6.4 配置用户注册

```
管理后台 → 系统设置 → 用户设置
- 启用用户注册：✓
- 需要邮箱验证：✓
- 新用户赠送额度：10000 tokens
- 默认提现比例：1.0
```

---

## 7. 支付系统集成

### 7.1 支付宝当面付接入

#### 申请流程

1. 登录 [支付宝开放平台](https://open.alipay.com/)
2. 创建应用 → 选择"网页/移动应用"
3. 添加功能 → 当面付
4. 提交资质审核

#### 配置参数

获取以下参数后填入 `.env`：

```ini
PAYMENT_ALIPAY_APP_ID=2021xxxxxxxxxxxxx
PAYMENT_ALIPAY_PRIVATE_KEY=MIIEvQIBADANBgkqhkiG9w0BAQEFAASCBKcwggSjAgEAAoIBAQC...
PAYMENT_ALIPAY_PUBLIC_KEY=MIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIBCgKCAQEArU5Pq...
```

#### 测试支付

```bash
# 调用支付接口测试
curl -X POST https://example.com/api/payment/alipay/create \
  -H "Content-Type: application/json" \
  -d '{
    "amount": 0.01,
    "subject": "测试订单",
    "order_id": "TEST_'$(date +%s)'"
  }'
```

### 7.2 微信支付接入

#### 申请流程

1. 登录 [微信支付商户平台](https://pay.weixin.qq.com/)
2. 注册商户账号
3. 完成资质审核
4. 配置 Native 支付

#### 配置参数

```ini
PAYMENT_WECHAT_APP_ID=wxaaaaaaaaaaaaaaaa
PAYMENT_WECHAT_MCH_ID=1234567890
PAYMENT_WECHAT_API_KEY=your32characterapikey123456789
PAYMENT_WECHAT_CERT_PATH=/opt/tokenhub/ssl/apiclient_cert.pem
PAYMENT_WECHAT_KEY_PATH=/opt/tokenhub/ssl/apiclient_key.pem
```

### 7.3 聚合支付（备选方案）

如果无法申请官方支付，可使用第三方聚合支付：

#### Payspi 接入

```ini
PAYMENT_PROVIDER=payspi
PAYMENT_PAYSPI_MERCHANT_ID=your_merchant_id
PAYMENT_PAYSPI_API_KEY=your_api_key
```

---

## 8. 前端页面部署

### 8.1 充值页面

**访问地址**：`https://example.com/topup/index.html`

功能特性：
- ✅ 自定义充值金额
- ✅ 支付宝/微信支付选择
- ✅ 二维码扫码支付
- ✅ 订单状态轮询
- ✅ 支付倒计时
- ✅ 响应式设计

### 8.2 套餐购买页面

**访问地址**：`https://example.com/plans/index.html`

功能特性：
- ✅ 四档套餐展示（体验版/标准版/专业版/企业版）
- ✅ 套餐详情对比
- ✅ 一键购买
- ✅ 支付弹窗
- ✅ 订单管理

### 8.3 与 New API 集成

前端页面需要与 New API 后端交互，修改 API 地址：

```javascript
// 在 index.html 中找到 API 配置
const API_BASE_URL = 'https://api.example.com'; // 改为你的 API 地址

// 用户认证 Token（从 New API 登录后获取）
const USER_TOKEN = localStorage.getItem('token');
```

---

## 9. 验证与测试

### 9.1 功能测试清单

```bash
# 1. 用户注册登录
- [ ] 用户可以注册新账号
- [ ] 邮箱验证正常
- [ ] 密码登录成功
- [ ] 密码重置功能正常

# 2. API Key 管理
- [ ] 创建 API Key
- [ ] 删除 API Key
- [ ] 查看 API Key 列表

# 3. API 调用测试
curl https://api.example.com/v1/chat/completions \
  -H "Authorization: Bearer YOUR_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "gpt-3.5-turbo",
    "messages": [{"role": "user", "content": "Hello"}]
  }'

# 4. 充值功能
- [ ] 访问充值页面
- [ ] 输入充值金额
- [ ] 生成支付订单
- [ ] 扫码支付（可用 0.01 元测试）
- [ ] 支付成功后余额更新

# 5. 套餐购买
- [ ] 访问套餐页面
- [ ] 选择套餐
- [ ] 完成支付
- [ ] Token 额度到账
```

### 9.2 性能测试

```bash
# 安装 Apache Bench
sudo apt install apache2-utils -y

# 并发测试（100 并发，总共 1000 请求）
ab -n 1000 -c 100 https://api.example.com/v1/models

# 查看结果
# Requests per second: 应该 > 500
# Time per request: 应该 < 500ms
```

### 9.3 安全测试

```bash
# HTTPS 强制跳转
curl -I http://example.com
# 应该返回 301 重定向到 https

# SSL 检查
openssl s_client -connect example.com:443

# 检查敏感信息泄露
curl https://api.example.com/.env
# 应该返回 404 或 403
```

---

## 10. 运维管理

### 10.1 日志查看

```bash
# New API 日志
docker logs -f tokenhub-new-api

# Nginx 日志
sudo tail -f /var/log/nginx/access.log
sudo tail -f /var/log/nginx/error.log

# 支付服务日志
sudo journalctl -u payment-gateway -f
```

### 10.2 数据备份

```bash
#!/bin/bash
# backup.sh - 数据库备份脚本

BACKUP_DIR="/opt/backups"
DATE=$(date +%Y%m%d_%H%M%S)
DB_NAME="newapi"
DB_USER="newapi"
DB_PASS="YourStrongPassword123!"

mkdir -p $BACKUP_DIR

# 备份 MySQL
docker exec tokenhub-mysql mysqldump -u$DB_USER -p$DB_PASS $DB_NAME > $BACKUP_DIR/mysql_$DATE.sql

# 压缩备份
gzip $BACKUP_DIR/mysql_$DATE.sql

# 删除 7 天前的备份
find $BACKUP_DIR -name "mysql_*.sql.gz" -mtime +7 -delete

echo "Backup completed: $BACKUP_DIR/mysql_$DATE.sql.gz"
```

```bash
# 添加到 crontab（每天凌晨 2 点备份）
crontab -e
0 2 * * * /opt/tokenhub/backup.sh
```

### 10.3 监控告警

#### Prometheus + Grafana 监控

```bash
# 启动监控栈
docker-compose -f monitor-docker-compose.yml up -d

# 访问 Grafana
# http://你的域名:3001
# 默认账号：admin / admin
```

#### 关键监控指标

- API 响应时间（P99 < 500ms）
- 错误率（< 1%）
- 并发连接数
- 数据库连接池使用率
- 磁盘使用率（< 80%）
- 内存使用率（< 80%）

### 10.4 扩容方案

#### 水平扩展 New API

```bash
# 启动多个 New API 实例
docker run -d --name new-api-1 ...
docker run -d --name new-api-2 ...
docker run -d --name new-api-3 ...

# Nginx 配置上游负载均衡
upstream new_api {
    server new-api-1:3000;
    server new-api-2:3000;
    server new-api-3:3000;
}
```

---

## 11. 常见问题

### Q1: 无法访问 New API 管理后台

**解决方案**：
```bash
# 检查容器状态
docker ps | grep new-api

# 查看日志
docker logs tokenhub-new-api

# 检查数据库连接
docker exec tokenhub-new-api ping mysql

# 重启服务
docker-compose restart new-api
```

### Q2: 支付回调失败

**排查步骤**：
1. 检查 `PAYMENT_NOTIFY_URL` 是否可公网访问
2. 确认 Nginx 配置了正确的回调路由
3. 查看支付服务日志
4. 在支付平台检查回调地址配置

### Q3: API 调用返回 402 余额不足

**解决方案**：
1. 检查用户账户余额
2. 确认渠道配置正确
3. 验证模型价格设置
4. 查看计费日志

### Q4: Docker 容器频繁重启

**排查步骤**：
```bash
# 查看重启原因
docker inspect --format='{{.State.Health}}' tokenhub-new-api

# 检查资源限制
docker stats

# 增加内存限制
# 在 docker-compose.yml 中调整：
services:
  new-api:
    deploy:
      resources:
        limits:
          memory: 2G
```

### Q5: SSL 证书过期

**续期方法**：
```bash
# Certbot 自动续期
sudo certbot renew

# 验证续期
sudo certbot certificates

# 重载 Nginx
sudo systemctl reload nginx
```

---

## 附录 A：端口清单

| 服务 | 端口 | 说明 |
|------|------|------|
| Nginx HTTP | 80 | Web 访问 |
| Nginx HTTPS | 443 | 加密 Web 访问 |
| New API | 3000 | API 服务 |
| MySQL | 3306 | 数据库（仅内网） |
| Redis | 6379 | 缓存（仅内网） |
| Grafana | 3001 | 监控面板 |

## 附录 B：目录结构

```
/opt/tokenhub/
├── deploy/                  # 部署配置
├── src/                     # 源代码
├── ssl/                     # SSL 证书
├── backups/                 # 数据备份
└── logs/                    # 日志文件
    ├── nginx/
    ├── new-api/
    └── payment/
```

## 附录 C：紧急联系

- New API 官方文档：https://docs.newapi.pro/
- New API GitHub：https://github.com/QuantumNous/new-api
- 本项目 Issues：[你的仓库地址]

---

**文档结束**

如需技术支持，请提交 Issue 或联系运维团队。
