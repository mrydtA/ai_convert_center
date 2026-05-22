# TokenHub - AI Token 中转站商业平台

> 项目代号：TokenHub  
> 版本：V1.0  
> 基于 New API (https://github.com/QuantumNous/new-api) 二次开发

## 项目简介

TokenHub 是一个商业级 AI Token 中转站平台，实现：

1. **统一 API 接入**：通过标准 OpenAI API 格式访问所有主流 AI 服务商
2. **隐藏真实 Key**：用户只能访问中转站 API Key，无需暴露上游真实 API Key
3. **多租户管理**：支持多用户、额度分配、权限控制
4. **商业化运营**：支持充值、套餐、计费体系，实现 API 转售盈利
5. **高可用架构**：支持高并发、多节点部署

## 支持的 AI 提供商

- ✅ OpenAI (GPT-4o, GPT-4 Turbo, GPT-3.5 Turbo)
- ✅ Anthropic (Claude 3.5 Sonnet, Claude 3 Opus)
- ✅ Google (Gemini Pro, Gemini Ultra)
- ✅ DeepSeek (DeepSeek Chat, DeepSeek Coder)
- ✅ MiniMax (MiniMax-ABAB 系列)
- ✅ 硅基流动 (多模型聚合)

## 快速开始

### 环境要求

- Docker 24.0+
- Docker Compose 2.20+
- Ubuntu 22.04 LTS (推荐)

### 一键部署

```bash
# 克隆项目
git clone <repository-url>
cd tokenhub/deploy

# 配置环境变量
cp .env.example .env
# 编辑 .env 文件，设置 MySQL/Redis 密码等

# 启动服务
docker-compose up -d

# 查看日志
docker-compose logs -f
```

### 默认访问地址

- 用户前台：http://localhost:3000
- 管理后台：http://localhost:3000/admin
- API 端点：http://localhost:3000/v1

### 初始管理员账号

首次部署后，请查看日志获取初始管理员账号：

```bash
docker-compose logs new-api | grep "admin"
```

## 项目结构

```
tokenhub/
├── deploy/                 # 部署配置文件
│   ├── docker-compose.yml  # Docker Compose 配置
│   ├── .env.example        # 环境变量模板
│   └── nginx/              # Nginx 配置
├── docs/                   # 项目文档
│   ├── 01_AI_Token_request.md      # 需求文档
│   ├── 02_AI-Token_designel.md     # 设计文档
│   ├── 03_AI-Token_task_plan.md    # 任务计划
│   └── 04_AI-Token_revie.md        # 验收清单
├── src/
│   ├── payment/            # 支付模块（新开发）
│   │   ├── alipay.go       # 支付宝集成
│   │   ├── wechat.go       # 微信支付集成
│   │   └── aggregate.go    # 聚合支付集成
│   └── frontend/           # 前端定制页面
│       ├── topup/          # 充值页面
│       └── plans/          # 套餐页面
└── README.md
```

## 功能特性

### 用户功能 ✅

- [x] 用户注册/登录
- [x] 邮箱验证
- [x] 密码重置
- [x] API Key 管理
- [x] 余额查看
- [ ] 在线充值（支付宝/微信）🆕
- [ ] 套餐购买 🆕
- [x] 用量明细
- [ ] 开发者文档 🆕

### 管理功能 ✅

- [x] 用户管理
- [x] 渠道管理
- [x] 模型管理
- [ ] 订单管理 🛠️
- [ ] 财务报表 🆕
- [x] 系统监控
- [x] 日志审计

### 核心业务 ✅

- [x] Token 计费（prompt + completion 分开计费）
- [x] 代理转发（兼容 OpenAI ChatGPT API 格式）
- [x] 流式返回（SSE 流式传输）
- [x] 多 Provider 接入
- [x] 智能路由
- [x] 限流策略
- [ ] 防滥用 🛠️

## 开发来源说明

| 标识 | 含义 | 说明 |
|------|------|------|
| ✅ | New API 现成 | 直接部署即可，无需开发 |
| 🛠️ | 二次开发 | 基于 New API 源码修改 |
| 🆕 | 全新开发 | 从零开发，新增功能 |

## 技术栈

| 组件 | 选型 | 说明 |
|------|------|------|
| 核心系统 | New API | 开源方案，二次开发 |
| 编程语言 | Go 1.21+ | New API 原生语言 |
| 数据库 | MySQL 8.0 | 主数据存储 |
| 缓存 | Redis 7.0 | 会话缓存、限流计数 |
| 部署方式 | Docker + Docker Compose | 容器化部署 |
| 反向代理 | Nginx 1.25+ | 负载均衡、SSL 终止 |

## API 使用示例

### 获取可用模型

```bash
curl http://localhost:3000/v1/models \
  -H "Authorization: Bearer YOUR_API_KEY"
```

### 聊天补全

```bash
curl http://localhost:3000/v1/chat/completions \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer YOUR_API_KEY" \
  -d '{
    "model": "gpt-4o",
    "messages": [
      {"role": "user", "content": "Hello!"}
    ]
  }'
```

### 流式调用

```bash
curl http://localhost:3000/v1/chat/completions \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer YOUR_API_KEY" \
  -d '{
    "model": "gpt-4o",
    "messages": [
      {"role": "user", "content": "Hello!"}
    ],
    "stream": true
  }'
```

## 性能指标

| 指标 | 目标值 |
|------|--------|
| API 延迟 | P99 < 500ms（不含上游 AI 服务延迟） |
| 并发支持 | 单节点 500+ 并发连接 |
| 可用性 | 99.5% 以上 |
| 扩展性 | 支持水平扩展，多节点部署 |

## 安全特性

- ✅ 数据加密：用户 API Key 加密存储（AES-256）
- ✅ 传输安全：全站 HTTPS，强制 TLS 1.2+
- ✅ 访问控制：API Key 与用户权限绑定
- ✅ 审计日志：完整操作日志，留存 180 天

## 待开发功能

### 🆕 全新开发（优先级高）

1. **支付模块**
   - 支付宝当面付对接
   - 微信支付对接
   - 聚合支付（Payspi/XPay）对接
   - 支付回调处理

2. **前端页面**
   - 充值页面 (/topup)
   - 套餐购买页面 (/plans)
   - 帮助文档中心 (/docs)
   - 财务报表页面 (/admin/finance)

3. **增强功能**
   - 邮件通知
   - 钉钉告警集成
   - IP 白名单
   - 用量告警

### 🛠️ 二次开发（优先级中）

1. **UI 定制**
   - Logo 更换
   - 主题色定制
   - 首页落地页定制

2. **安全增强**
   - Nginx 限流规则
   - 防爬机制
   - SSL/HTTPS 配置

## 开发计划

| 阶段 | 时间 | 主要内容 |
|------|------|---------|
| 阶段一 | 第 1-2 周 | 环境准备与 New API 部署 ✅ |
| 阶段二 | 第 3 周 | AI 渠道接入 ✅ |
| 阶段三 | 第 4-5 周 | 支付系统开发 🆕 |
| 阶段四 | 第 6-7 周 | UI 定制与增强 🛠️ |
| 阶段五 | 第 8-10 周 | 测试优化与上线 🛠️ |

## 参考文档

- [New API 官方文档](https://docs.newapi.pro/)
- [New API GitHub](https://github.com/QuantumNous/new-api)
- [One API（上游项目）](https://github.com/songquanpeng/one-api)

## License

本项目基于 New API 二次开发，遵循原项目开源协议。

## 联系方式

- 项目 Issues: [GitHub Issues](https://github.com/your-org/tokenhub/issues)
- 技术支持：support@tokenhub.example.com

---

**最后更新**: 2025 年 5 月
