# 🎁 Prometheus-CPA (凭证捐赠站)

[![Docker Image](https://img.shields.io/docker/v/123nhh/prometheus-cpa?label=Docker%20Hub&logo=docker)](https://hub.docker.com/r/123nhh/prometheus-cpa)
[![GitHub](https://img.shields.io/badge/GitHub-123nhh%2Fprometheus--CPA-blue?logo=github)](https://github.com/123nhh/prometheus-CPA)

一个独立部署的凭证捐赠站系统，通过调用 CLIProxyAPI (CPA) 的管理接口实现 OAuth 授权流程，用户完成授权后自动生成 CDK。

## ✨ 功能特点

- 🔗 **CPA 集成** - 通过调用 CPA 管理 API 获取 OAuth 授权链接
- 🎫 **CDK 生成** - 凭证授权成功后自动生成唯一 CDK
- � **CDK 分组** - 支持创建分组管理不同用途的 CDK- 🔄 **CDK 自动补充** - CDK 用完时自动创建"待兑换"分组并生成新 CDK
- 🍪 **iFlow Cookie 登录** - 支持通过 Cookie 直接登录 iFlow- 🔔 **回调通知** - 支持配置回调接口，CDK 生成后自动通知目标系统
- 🚫 **去重检查** - 基于授权状态哈希，自动检测重复领取
- 🎨 **多主题支持** - 4 款精美主题可切换（晨曦微光、深海极光、虚空暗金、赛博霓虹）
- 🖼️ **自定义背景** - 支持自定义网页背景图片
- 🔒 **安全设计** - 凭证由 CPA 处理，捐赠站不接触敏感信息
- 📊 **管理后台** - 现代化仪表盘，卡片式分组管理，完整的数据统计
- 📱 **响应式设计** - 完美适配桌面端和移动端

## 🖼️ 界面预览

### 首页
- 四大渠道选择卡片（Antigravity、Gemini CLI、Codex、iFlow）
- CDK 分组选择（可选择不同奖励类型）
- 实时显示已收录凭证数量
- 主题切换悬浮球

### 管理后台
- **仪表盘**: 6 格数据概览（凭证总数/已验证/待处理/CDK总数/可用/已用）
- **CDK 管理**: 卡片式工具栏（单个添加、批量导入、实时统计）
- **CDK 分组**: 卡片式分组列表，直观展示每组的 CDK 使用情况
- **渠道管理**: 一键开关各捐赠渠道
- **站点配置**: 自定义站点名称、副标题、背景图

## 🔄 工作流程

**OAuth 授权流程** (Antigravity / Gemini CLI / Codex):
```
用户 → 捐赠站 → CPA (获取OAuth链接) → Google授权 → CPA (处理凭证) → 捐赠站 (生成CDK)
```

**iFlow Cookie 流程**:
```
用户 → 捐赠站 (提交Cookie) → CPA (验证Cookie) → 捐赠站 (生成CDK)
```

1. 用户在捐赠站选择凭证类型和 CDK 分组（可选）
2. **OAuth 渠道**: 捐赠站调用 CPA 获取授权链接，用户完成 Google 授权
3. **iFlow 渠道**: 用户直接提交 Cookie，CPA 验证并保存凭证
4. 捐赠站检测授权完成，从指定分组获取/生成 CDK 并展示给用户
5. 如果 CDK 用完，自动创建"待兑换"分组并生成新 CDK

## 🏗️ 技术栈

- **后端**: Go + Gin
- **数据库**: PostgreSQL
- **前端**: HTML5 + CSS3 + Vanilla JS
- **部署**: Docker + Docker Compose

## 📁 项目结构

```
prometheus-CPA/
├── cmd/
│   └── server/
│       └── main.go           # 应用入口
├── internal/
│   ├── api/
│   │   └── server.go         # HTTP API 服务
│   ├── cpa/
│   │   └── client.go         # CPA API 客户端
│   ├── callback/
│   │   └── notifier.go       # 回调通知器
│   ├── cdk/
│   │   └── generator.go      # CDK 生成器
│   ├── config/
│   │   └── config.go         # 配置管理
│   └── database/
│       └── database.go       # 数据库操作
├── static/
│   ├── index.html            # 捐赠页面
│   ├── waiting.html          # 等待授权页面
│   ├── success.html          # 成功页面
│   ├── error.html            # 错误页面
│   └── admin.html            # 管理后台
├── docker-compose.yml        # Docker 编排
├── Dockerfile                # Docker 构建
├── go.mod                    # Go 模块
├── .env.example              # 环境变量示例
└── README.md
```

## 🚀 快速开始

### 前置要求

在开始之前，请确保你已经：

1. ✅ **已部署并运行 [CLIProxyAPI (CPA)](https://github.com/router-for-me/CLIProxyAPI)** - 这是核心依赖
2. ✅ **CPA 已启用远程管理功能** - 在 CPA 配置中设置 `allow-remote-management: true`
3. ✅ **已获取 CPA 管理密钥** - 对应 CPA 配置中的 `management-secret-key`
4. ✅ **已安装 Docker 和 Docker Compose** (推荐) 或 Go 1.21+

---

### 方法一：使用 Docker 镜像 (最简单)

直接从 Docker Hub 拉取预构建镜像：

```bash
# 拉取镜像
docker pull 123nhh/prometheus-cpa:latest
```

然后下载 docker-compose.yml 并启动：

```bash
# 下载配置文件
curl -O https://raw.githubusercontent.com/123nhh/prometheus-CPA/main/docker-compose.yml
curl -O https://raw.githubusercontent.com/123nhh/prometheus-CPA/main/.env.example
cp .env.example .env

# 编辑配置
nano .env

# 启动服务
docker compose up -d
```

---

### 方法二：使用 Docker Compose 构建 (推荐开发)

从源码构建，适合需要自定义修改的用户。

#### 第一步：获取项目

```bash
# 克隆项目到本地
git clone https://github.com/123nhh/prometheus-CPA.git
cd prometheus-CPA
```

#### 第二步：创建配置文件

```bash
# 复制环境变量模板
cp .env.example .env
```

#### 第三步：编辑配置文件

用你喜欢的编辑器打开 `.env` 文件：

```bash
nano .env  # 或 vim .env
```

**必须修改的配置项：**

```bash
# ========== CPA 连接配置 (必填) ==========
# 你的 CPA 服务地址，不要带末尾斜杠
CPA_BASE_URL=https://your-cpa-server.com

# CPA 管理密钥，对应 CPA 配置文件中的 management-secret-key
CPA_MANAGEMENT_KEY=your-cpa-management-key

# ========== 安全配置 (必须修改) ==========
# JWT 签名密钥，用于生成 token，请使用随机字符串
JWT_SECRET=your-random-jwt-secret-key-change-this

# ========== 管理员账号 (建议修改) ==========
ADMIN_USERNAME=admin
ADMIN_PASSWORD=your-strong-password
```

**可选配置项：**

```bash
# 服务端口，默认 8080
PORT=8080

# CDK 前缀，默认 DS，生成的 CDK 格式如：DS-XXXX-XXXX-XXXX-XXXX-1234
CDK_PREFIX=DS

# 站点名称，显示在页面标题
SITE_NAME=凭证捐赠站

# 自定义背景图片 URL (可选)
BACKGROUND_IMAGE=https://example.com/your-background.jpg

# 回调通知地址 (可选，用于通知第三方系统)
CALLBACK_URL=
CALLBACK_SECRET=
```

#### 第四步：启动服务

```bash
# 构建并启动所有服务 (后台运行)
docker-compose up -d

# 查看启动日志
docker-compose logs -f
```

等待几秒钟让服务初始化完成，你会看到类似这样的日志：
```
donation-station  | [GIN] Listening and serving HTTP on :8080
```

#### 第五步：访问服务

- 🎁 **捐赠页面** (用户访问): http://localhost:8080
- 🔧 **管理后台** (管理员): http://localhost:8080/admin

首次访问管理后台时，使用你在 `.env` 中配置的 `ADMIN_USERNAME` 和 `ADMIN_PASSWORD` 登录。

---

### 方法三：本地开发

适合需要修改代码或调试的开发者。

#### 第一步：启动 PostgreSQL 数据库

```bash
# 使用 Docker 快速启动 PostgreSQL
docker run -d \
  --name donation-postgres \
  -e POSTGRES_USER=donation \
  -e POSTGRES_PASSWORD=donation123 \
  -e POSTGRES_DB=donation_station \
  -p 5432:5432 \
  postgres:16-alpine

# 等待数据库就绪
sleep 5
```

#### 第二步：设置环境变量

```bash
# 数据库连接
export DATABASE_URL="postgres://donation:donation123@localhost:5432/donation_station?sslmode=disable"

# CPA 配置 (必填)
export CPA_BASE_URL="https://your-cpa-server.com"
export CPA_MANAGEMENT_KEY="your-cpa-management-key"

# 安全配置
export JWT_SECRET="dev-jwt-secret"
export ADMIN_USERNAME="admin"
export ADMIN_PASSWORD="admin123"
```

#### 第三步：安装依赖并运行

```bash
# 下载 Go 依赖
go mod download

# 运行应用
go run cmd/server/main.go
```

---

### 常用命令

```bash
# 查看运行状态
docker compose ps

# 查看日志
docker compose logs -f prometheus-cpa

# 重启服务
docker compose restart prometheus-cpa

# 停止所有服务
docker compose down

# 停止并删除数据 (谨慎使用!)
docker compose down -v

# 更新镜像到最新版本
docker compose pull
docker compose up -d

# 从源码更新
git pull
docker compose up -d --build
```

### 数据备份

```bash
# 备份数据库
docker compose exec postgres pg_dump -U donation donation_station > backup.sql

# 恢复数据库
cat backup.sql | docker compose exec -T postgres psql -U donation donation_station
```

## ⚙️ 配置说明

### 环境变量

| 变量名 | 说明 | 默认值 |
|--------|------|--------|
| `DATABASE_URL` | PostgreSQL 连接字符串 | 必填 |
| `PORT` | 服务端口 | `8080` |
| `CPA_BASE_URL` | CPA 服务地址 | **必填** |
| `CPA_MANAGEMENT_KEY` | CPA 管理密钥 | **必填** |
| `CALLBACK_URL` | 回调通知地址 | 空 (不通知) |
| `CALLBACK_SECRET` | 回调签名密钥 | 必须修改 |
| `JWT_SECRET` | JWT 签名密钥 | 必须修改 |
| `CDK_PREFIX` | CDK 前缀 | `DS` |
| `SITE_NAME` | 站点名称 | `凭证捐赠站` |
| `BACKGROUND_IMAGE` | 背景图片URL | 空 |
| `ADMIN_USERNAME` | 管理员用户名 | `admin` |
| `ADMIN_PASSWORD` | 管理员密码 | `admin123` |

### CPA 配置

捐赠站需要连接到你的 CLIProxyAPI 服务：

```bash
# CPA 服务地址
CPA_BASE_URL=https://your-cpa-server.com

# CPA 管理密钥 (对应 CPA 配置中的 management-secret-key 或 MANAGEMENT_PASSWORD)
CPA_MANAGEMENT_KEY=your-management-key
```

确保你的 CPA 服务：
1. 已启用远程管理 (`allow-remote-management: true`)
2. 已配置管理密钥
3. 捐赠站服务器的 IP 可以访问 CPA

### 回调接口说明

当凭证验证成功并生成 CDK 后，系统会向配置的 `CALLBACK_URL` 发送 POST 请求：

**请求头:**
```
Content-Type: application/json
X-Signature: <HMAC-SHA256 签名>
X-Timestamp: <Unix 时间戳>
```

**请求体:**
```json
{
  "credential_id": 1,
  "credential_type": "antigravity",
  "email": "user@example.com",
  "project_id": "project-123",
  "cdk_code": "DS-XXXX-XXXX-XXXX-XXXX-1234",
  "timestamp": 1706745600,
  "signature": "abc123..."
}
```

**签名验证:**
签名使用 HMAC-SHA256 算法生成，签名字符串格式：
```
{credential_id}:{email}:{project_id}:{cdk_code}:{timestamp}
```

## 📡 API 接口

### 公开接口

#### 开始授权
```
POST /api/auth/start
Content-Type: application/json

{
  "type": "antigravity",  // 或 "gemini_cli", "gemini-cli", "codex"
  "group_id": 1           // 可选，指定 CDK 分组
}
```

响应:
```json
{
  "success": true,
  "auth_url": "https://accounts.google.com/o/oauth2/v2/auth?...",
  "state": "xxx"
}
```

前端获取到 `auth_url` 后打开新窗口跳转到 Google 授权，然后跳转到等待页面。

#### iFlow Cookie 登录
```
POST /api/auth/iflow
Content-Type: application/json

{
  "cookie": "BXAuth=...",  // iFlow Cookie，必须以 BXAuth= 开头
  "group_id": 1            // 可选，指定 CDK 分组
}
```

响应:
```json
{
  "success": true,
  "cdk": "DS-XXXX-XXXX-XXXX-XXXX-1234",
  "email": "user@example.com"
}
```

#### 查询授权状态
```
GET /api/auth/status?state=xxx
```

响应:
```json
{
  "success": true,
  "status": "pending",  // "pending", "completed", "error"
  "message": "正在等待授权完成..."
}
```

#### 确认领取 CDK
```
POST /api/auth/complete
Content-Type: application/json

{
  "state": "xxx"
}
```

响应:
```json
{
  "success": true,
  "status": "success",
  "cdk": "DS-XXXX-XXXX-XXXX-XXXX-1234"
}
```

#### 获取站点配置
```
GET /api/site-config
```

### 管理接口 (需要 Basic Auth)

#### 获取统计数据
```
GET /api/admin/stats
```

#### 获取凭证列表
```
GET /api/admin/credentials?limit=20&offset=0
```

#### 获取 CDK 列表
```
GET /api/admin/cdks?limit=20&offset=0
```

#### 设置站点配置
```
POST /api/admin/site-config
Content-Type: application/json

{
  "key": "site_name",
  "value": "我的捐赠站"
}
```

## 🔒 安全特性

1. **凭证安全** - 凭证由 CPA 处理，捐赠站不接触敏感信息
2. **状态哈希** - 使用 SHA-256 哈希防止重复领取
3. **回调签名** - 回调请求使用 HMAC-SHA256 签名，防止伪造
4. **管理员认证** - 管理后台使用 HTTP Basic Auth 保护
5. **HTTPS 支持** - 生产环境建议配置 HTTPS

## 🎨 自定义外观

### 修改背景图片

方法一：通过环境变量
```bash
BACKGROUND_IMAGE=https://example.com/your-image.jpg
```

方法二：通过管理后台
1. 访问 `/admin`
2. 输入管理员凭据
3. 在"站点配置"标签页中设置背景图片URL

### 修改站点名称

同上，可通过环境变量 `SITE_NAME` 或管理后台修改。

## 🛠️ 与 CPA 集成说明

### 工作原理

捐赠站通过调用 CPA 的管理 API 实现 OAuth 授权：

1. **获取授权链接**: `GET /management/antigravity-auth-url` 或 `GET /management/gemini-cli-auth-url`
2. **查询授权状态**: `GET /management/auth-status?state=xxx`
3. 当 CPA 返回 `unknown or expired state` 时，表示授权已完成

### CPA 要求

- CLIProxyAPI v6.x 或更高版本
- 已启用远程管理功能
- 已配置管理密钥

### 数据库表结构

- `credentials` - 凭证记录表
- `cdks` - CDK 记录表
- `cdk_groups` - CDK 分组表
- `callback_logs` - 回调日志表
- `site_config` - 站点配置表

## 📄 许可证

GNU GPLv3

## 🤝 贡献

欢迎提交 Issue 和 Pull Request！

---

**⚠️ 免责声明**: 本项目仅供学习和研究使用，请遵守相关服务条款和法律法规。
