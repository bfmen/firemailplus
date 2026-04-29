# 🔥 FireMail Plus - 现代化邮件客户端

<div align="center">

![FireMail Plus Logo](https://img.shields.io/badge/FireMail-Plus-orange?style=for-the-badge&logo=mail&logoColor=white)

**基于 Next.js 15 + React 19 + Go 的现代化邮件管理平台**

[![GitHub stars](https://img.shields.io/github/stars/fengyuanluo/firemailplus?style=social)](https://github.com/fengyuanluo/firemailplus/stargazers)
[![GitHub forks](https://img.shields.io/github/forks/fengyuanluo/firemailplus?style=social)](https://github.com/fengyuanluo/firemailplus/network/members)
[![GitHub issues](https://img.shields.io/github/issues/fengyuanluo/firemailplus)](https://github.com/fengyuanluo/firemailplus/issues)
[![GitHub license](https://img.shields.io/github/license/fengyuanluo/firemailplus)](https://github.com/fengyuanluo/firemailplus/blob/main/LICENSE)

[![Next.js](https://img.shields.io/badge/Next.js-15-black?logo=next.js)](https://nextjs.org/)
[![React](https://img.shields.io/badge/React-19-blue?logo=react)](https://reactjs.org/)
[![Go](https://img.shields.io/badge/Go-1.24-00ADD8?logo=go)](https://golang.org/)
[![TypeScript](https://img.shields.io/badge/TypeScript-5-blue?logo=typescript)](https://www.typescriptlang.org/)
[![Docker](https://img.shields.io/badge/Docker-Ready-2496ED?logo=docker)](https://www.docker.com/)

</div>

## 📖 项目介绍

FireMail Plus 是一个现代化的邮件客户端应用，采用最新的技术栈构建，为用户提供优雅、高效的邮件管理体验。项目采用前后端分离架构，支持多种邮件提供商，具备完整的邮件收发、管理和搜索功能。

### ✨ 核心特性

- 🚀 **现代化技术栈** - Next.js 15 + React 19 + Go 1.24
- 📱 **响应式设计** - 完美适配桌面端和移动端
- 🔐 **多重认证** - 支持 OAuth2（Gmail、Outlook）和自定义 IMAP/SMTP
- ⚡ **实时同步** - 基于 SSE 的实时邮件同步
- 🎨 **优雅界面** - 基于 shadcn/ui 的现代化 UI 设计
- 🔍 **智能搜索** - 全文搜索和高级过滤功能
- 📎 **附件支持** - 完整的附件上传、下载和预览
- 🌙 **主题切换** - 支持明暗主题自动切换

## 🖼️ 项目截图

![image](https://git.adust.f5.si/gh/fengyuanluo/tuchuang@main/20250628083833.png)

![image](https://git.adust.f5.si/gh/fengyuanluo/tuchuang@main/20250628083648.png)

![image](https://git.adust.f5.si/gh/fengyuanluo/tuchuang@main/20250628083306.png)

## 🏆 项目优势

### 🎯 技术先进性
- **前端技术栈**：采用最新的 Next.js 15 App Router 和 React 19，享受最新的性能优化和开发体验
- **后端架构**：使用 Go 语言构建高性能后端，支持并发处理和快速响应
- **类型安全**：全栈 TypeScript 支持，确保代码质量和开发效率
- **组件化设计**：基于 shadcn/ui 的模块化组件系统，易于维护和扩展

### 🛡️ 安全可靠
- **OAuth2 认证**：支持 Gmail、Outlook 等主流邮件服务的安全认证
- **JWT 令牌**：安全的用户会话管理
- **数据加密**：敏感信息加密存储
- **权限控制**：细粒度的用户权限管理

### 📱 多平台适配
- **响应式布局**：自适应桌面、平板和手机屏幕
- **移动端优化**：专门优化的移动端交互体验
- **跨浏览器兼容**：支持主流现代浏览器

### ⚡ 性能卓越
- **服务端渲染**：Next.js SSR 提供更快的首屏加载
- **智能缓存**：多层缓存策略优化性能
- **并发处理**：Go 协程支持高并发邮件同步
- **增量同步**：只同步新邮件，减少网络开销

### 🔧 易于部署
- **Docker 容器化**：一键部署，环境隔离
- **配置简单**：环境变量配置，无需复杂设置
- **自动备份**：数据库自动备份机制
- **监控友好**：内置健康检查和日志系统

## 📧 已支持邮箱类型
- Gmail (暂时只支持应用密码登录)
- Outlook
- QQ邮箱
- 163邮箱
- 自定义IMAP/SMTP

## 🚀 部署方式

### 方式一：Docker Compose 部署（推荐）

这是最简单的部署方式，适合生产环境使用。

```bash
# 1. 克隆项目
git clone https://github.com/fengyuanluo/firemailplus.git
cd firemailplus

# 2. 配置环境变量（可选）
cp .env.example .env
# 编辑 .env 文件，修改管理员密码和 JWT 密钥
# 警告：请长期保存 JWT_SECRET；如配置 ENCRYPTION_KEY 也必须保存，否则历史邮箱凭据无法解密。

# 3. 启动服务
docker-compose up -d

# 4. 查看服务状态
docker-compose ps

# 5. 查看日志
docker-compose logs -f
```

服务启动后，访问 `http://localhost:3000` 即可使用。

默认管理员账户：
- 用户名：`admin`
- 密码：`admin123`（建议修改）

### 方式二：Docker CLI 部署

适合快速体验和测试环境。

```bash
# 1. 拉取镜像
docker pull luofengyuan/firemailplus:latest

# 2. 创建数据卷
docker volume create firemail_data
docker volume create firemail_logs

# 3. 运行容器
docker run -d \
  --name firemail-app \
  -p 3000:3000 \
  -v firemail_data:/app/data \
  -v firemail_logs:/app/logs \
  -e ADMIN_USERNAME=admin \
  -e ADMIN_PASSWORD=your_secure_password \
  -e JWT_SECRET=your_jwt_secret_key \
  -e EXTERNAL_OAUTH_SERVER_URL=https://oauth.windyl.de \
  luofengyuan/firemailplus:latest

# 4. 查看容器状态
docker ps
docker logs firemail-app
```

### 方式三：开发环境部署

适合开发者进行二次开发和调试。

```bash
# 1. 克隆项目
git clone https://github.com/fengyuanluo/firemailplus.git
cd firemailplus

# 2. 启动后端服务
cd backend
cp .env.example .env
# 编辑 .env 文件配置
go mod download
go run cmd/firemail/main.go

# 3. 启动前端服务（新终端）
cd frontend
pnpm install
pnpm dev
```

前端服务：`http://localhost:3000`  
后端服务：`http://localhost:8080`

## 🛠️ 已知BUG
- Gmail授权登录无法使用（谷歌得先经过认证...看我啥时候有时间写隐私说明和用户说明什么的吧）
- 部分复杂邮件解析仍然存在问题（这方面真的尽力了...例子是163某些真的不知道什么鬼）

## 📅 开发计划
- 支持PWA
- 支持谷歌授权登录
- 支持更多邮箱（尤其IMAP/SMTP基本认证支持的欢迎提Issue，虽然期末周了可能会比较拖）

## 📝 结语

FireMail Plus 致力于为用户提供现代化、高效的邮件管理体验。我们采用最新的技术栈，遵循最佳实践，确保项目的可维护性和可扩展性。

### 🤝 贡献指南

我们欢迎社区贡献！如果您想为项目做出贡献，请：

1. Fork 本仓库
2. 创建您的特性分支 (`git checkout -b feature/AmazingFeature`)
3. 提交您的更改 (`git commit -m 'Add some AmazingFeature'`)
4. 推送到分支 (`git push origin feature/AmazingFeature`)
5. 打开一个 Pull Request

## ⚠️ 免责声明

1. 本工具仅用于方便用户管理自己的邮箱账户，请勿用于非法用途。
2. 使用本工具过程中产生的任何数据安全问题、账户安全问题或违反相关服务条款的行为，均由用户自行承担责任。
3. 开发者不对使用本工具过程中可能出现的任何损失或风险负责。
4. 本工具与Microsoft、Google等邮箱服务提供商没有任何官方关联，使用时请遵守相关服务条款。
5. 邮箱账号和密码等敏感信息仅存储在本地SQLite数据库中，请确保服务器安全，防止数据泄露。
6. 使用本工具可能会受到邮箱服务提供商的API访问限制或策略变更的影响，如遇访问受限，请遵循相关提供商的政策调整使用方式。
7. 本工具不保证100%的兼容性和可用性，可能因第三方服务变更而需要更新。
8. 用户在使用过程中应遵守当地法律法规，不得用于侵犯他人隐私或其他非法活动。
9. 本软件按"原样"提供，不提供任何形式的保证，无论是明示的还是暗示的。

---

<div align="center">

**如果这个项目对您有帮助，请给我们一个 ⭐ Star！**

Made with ❤️ by [fengyuanluo](https://github.com/fengyuanluo)

</div>
