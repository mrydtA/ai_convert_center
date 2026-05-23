# TokenHub - AI Token 中转站商业平台

> 项目版本：V1.0  
> 最后更新：2025 年 5 月 22 日  
> 基于 New API 二次开发

---

## 📖 项目简介

TokenHub 是一个**商业级 AI Token 中转站平台**，基于开源 New API 项目进行二次开发，提供统一的 AI API 接入、用户管理、计费系统和支付功能。

### 核心价值

- 🔒 **安全**：隐藏真实 API Key，防止泄露
- 💰 **商业化**：完整的充值、套餐、计费体系
- 🚀 **高效**：兼容 OpenAI API 格式，开箱即用
- 📊 **可观测**：用量统计、财务报表、监控告警

---

## 🏗️ 项目结构

```
/workspace/tokenhub/
├── deploy/                          # 部署配置目录
│   ├── docker-compose.yml           # Docker Compose 编排文件
│   ├── .env.example                 # 环境变量模板
│   ├── nginx/
│   │   ├── nginx.conf               # Nginx 主配置（限流、Gzip）
│   │   └── conf.d/default.conf      # 站点配置（HTTPS、WebSocket）
│   ├── init-scripts/
│   │   └── 01_init_tables.sql       # MySQL 初始化脚本（含扩展表）
│   ├── ssl/                         # SSL 证书目录
│   ├── data/                        # 数据持久化目录
│   └── logs/                        # 日志目录
│
├── src/                             # 源代码目录（🆕 全新开发）
│   ├── payment/
│   │   └── gateway.go               # 支付网关（支付宝/微信）
│   └── frontend/
│       ├── topup/index.html         # 在线充值页面
│       └── plans/index.html         # 套餐购买页面
│
└── docs/                            # 文档目录
    ├── README.md                    # 本文件
    ├── COMPLETE_DEPLOYMENT.md       # 📘 完整部署指南（821 行）
    ├── DEPLOYMENT.md                # 基础部署文档
    └── API_GUIDE.md                 # API 使用文档
```

---

## ✅ 已交付内容

### 1. 部署配置（deploy/）

| 文件 | 状态 | 说明 |
|------|------|------|
| `docker-compose.yml` | ✅ | New API + MySQL + Redis + Nginx 完整编排 |
| `.env.example` | ✅ | 包含所有必要环境变量的模板 |
| `nginx/nginx.conf` | 🛠️ | 定制配置（限流、Gzip 压缩、安全头） |
| `nginx/conf.d/default.conf` | 🛠️ | 站点配置（HTTP/HTTPS、WebSocket、反向代理） |
| `init-scripts/01_init_tables.sql` | 🆕 | 扩展数据库表（充值记录、套餐、支付流水） |

### 2. 新开发代码（src/）

| 文件 | 状态 | 说明 |
|------|------|------|
| `payment/gateway.go` | 🆕 | 支付网关 Go 代码框架 |
| `frontend/topup/index.html` | 🆕 | 响应式充值页面（扫码支付、订单轮询） |
| `frontend/plans/index.html` | 🆕 | 套餐购买页面（4 档套餐、支付弹窗） |
| `frontend/home/index.html` | 🆕 | 品牌首页/落地页（功能展示、价格方案） |
| `frontend/docs/index.html` | 🆕 | API 文档中心（侧边栏导航、代码示例） |
| `frontend/reports/index.html` | 🆕 | 财务报表页面（统计卡片、数据表格） |
| `frontend/help/index.html` | 🆕 | 帮助中心页面（FAQ、分类指南） |
| `frontend/assets/css/style.css` | 🆕 | 全局样式系统（品牌主题色） |
| `frontend/assets/js/main.js` | 🆕 | 公共 JavaScript 功能 |

### 3. 文档（docs/）

| 文件 | 页数 | 说明 |
|------|------|------|
| `COMPLETE_DEPLOYMENT.md` | 821 行 | 📘 **完整部署指南**（含生产环境配置） |
| `DEPLOYMENT.md` | 327 行 | 基础部署文档 |
| `API_GUIDE.md` | 429 行 | API 使用文档（含代码示例） |

---

## 🎯 核心功能

### 用户侧功能

| 功能 | 状态 | 说明 |
|------|------|------|
| 用户注册/登录 | ✅ | New API 原生支持 |
| 邮箱验证 | ✅ | New API 原生支持 |
| API Key 管理 | ✅ | New API 原生支持 |
| 额度查看 | ✅ | New API 原生支持 |
| **在线充值** | 🆕 | 支付宝/微信支付 |
| **套餐购买** | 🆕 | 4 档套餐可选 |
| 用量明细 | ✅ | New API 原生支持 |
| 开发者文档 | ✅ | 见 `API_GUIDE.md` |

### 管理侧功能

| 功能 | 状态 | 说明 |
|------|------|------|
| 用户管理 | ✅ | New API 原生支持 |
| 渠道管理 | ✅ | New API 原生支持 |
| 模型管理 | ✅ | New API 原生支持 |
| **订单管理** | 🆕 | 充值/套餐订单 |
| 财务管理 | 🆕 | 报表功能（待完善） |
| 系统监控 | 🛠️ | Prometheus+Grafana |

---

## 🚀 快速开始

### 方式一：Docker Compose 部署（推荐）

```bash
# 1. 进入部署目录
cd /workspace/tokenhub/deploy

# 2. 配置环境变量
cp .env.example .env
vim .env  # 编辑配置

# 3. 启动所有服务
docker-compose up -d

# 4. 查看服务状态
docker-compose ps

# 5. 访问前端页面
# 浏览器打开：http://你的服务器 IP/topup/index.html
# 浏览器打开：http://你的服务器 IP/plans/index.html
```

### 方式二：手动部署（生产环境）

详细步骤请参考：**[完整部署指南](docs/COMPLETE_DEPLOYMENT.md)**

---

## 📦 技术栈

| 组件 | 技术选型 | 说明 |
|------|---------|------|
| 核心系统 | New API (Go) | 开源 API 中转系统 |
| 数据库 | MySQL 8.0 | 数据存储 |
| 缓存 | Redis 7.0 | 会话/限流 |
| Web 服务器 | Nginx | 反向代理/负载均衡 |
| 前端 | HTML5 + CSS3 + JS | 响应式设计 |
| 支付 | 支付宝/微信 | 扫码支付 |
| 部署 | Docker + Compose | 容器化部署 |

---

## 📋 下一步建议

根据任务计划书，后续还需完成：

### 阶段三：支付系统开发（第 4-5 周）🆕

- [ ] 完善支付后端 API（`payment/gateway.go`）
- [ ] 对接支付宝官方 API
- [ ] 对接微信支付官方 API
- [ ] 实现支付回调处理
- [ ] 订单管理功能

### 阶段四：UI 定制与增强（第 6-7 周）✅

- [x] Logo 更换与品牌定制
- [x] 主题色定制
- [x] 首页落地页开发
- [x] 帮助文档页面
- [x] 财务报表页面
- [ ] 邮件通知功能（后端功能）

### 阶段五：测试优化与上线（第 8-10 周）

- [ ] 功能测试
- [ ] 压力测试
- [ ] 性能优化
- [ ] 监控告警配置
- [ ] 正式上线

---

## 📚 文档导航

| 文档 | 用途 | 链接 |
|------|------|------|
| 📘 完整部署指南 | 生产环境部署 | [查看](docs/COMPLETE_DEPLOYMENT.md) |
| 📗 基础部署文档 | 快速测试部署 | [查看](docs/DEPLOYMENT.md) |
| 📙 API 使用文档 | 开发者参考 | [查看](docs/API_GUIDE.md) |
| 📕 需求说明书 | 功能需求详情 | [查看](/workspace/01_AI_Token_request.md) |
| 📔 设计方案 | 技术设计细节 | [查看](/workspace/02_AI-Token_designel.md) |
| 📓 任务计划 | 开发进度规划 | [查看](/workspace/03_AI-Token_task_plan.md) |

---

## 🔧 常用命令

```bash
# 查看服务状态
docker-compose ps

# 查看日志
docker-compose logs -f

# 重启服务
docker-compose restart

# 停止服务
docker-compose down

# 重新构建
docker-compose build --no-cache

# 进入容器
docker exec -it tokenhub-new-api sh

# 备份数据库
docker exec tokenhub-mysql mysqldump -u newapi -p newapi > backup.sql
```

---

## ⚠️ 注意事项

1. **支付资质**：支付宝/微信支付需要商户资质，可使用第三方聚合支付替代
2. **ICP 备案**：国内服务器必须完成 ICP 备案
3. **SSL 证书**：生产环境必须启用 HTTPS
4. **数据安全**：定期备份数据库，妥善保管密钥
5. **合规经营**：确保 AI 服务内容符合法律法规

---

## 📞 技术支持

- New API 官方文档：https://docs.newapi.pro/
- New API GitHub：https://github.com/QuantumNous/new-api
- 本项目 Issues：[待添加]

---

## 📄 许可证

本项目基于 New API 二次开发，遵循原项目许可证。

---

**TokenHub 项目组**  
2025 年 5 月
