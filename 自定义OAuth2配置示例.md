# 自定义 OAuth2 配置示例

## 🔧 配置说明

本项目已经重构为支持任意标准 OAuth2 服务，并使用 GORM 进行数据库操作。

### 核心特性

1. **灵活的 OAuth2 支持**：支持任何标准 OAuth2 服务
2. **字段映射配置**：可自定义用户信息字段映射
3. **嵌套字段支持**：支持点号访问嵌套 JSON 字段
4. **GORM 数据库层**：使用 GORM 替代原始 SQL，更易维护

## 📝 配置步骤

### 1. 基础环境配置

复制并编辑环境变量文件：

```bash
cp .env.example .env
```

### 2. 自定义 OAuth2 配置

编辑 `.env` 文件：

```env
# OAuth2 基础配置
OAUTH2_PROVIDER=custom
OAUTH2_CLIENT_ID=your_client_id_here
OAUTH2_CLIENT_SECRET=your_client_secret_here
OAUTH2_REDIRECT_URL=http://localhost:6060/auth/callback

# OAuth2 端点配置
OAUTH2_AUTH_URL=https://your-oauth-server.com/oauth/authorize
OAUTH2_TOKEN_URL=https://your-oauth-server.com/oauth/token
OAUTH2_USERINFO_URL=https://your-oauth-server.com/api/user
OAUTH2_SCOPES=openid,profile,email

# 用户信息字段映射
OAUTH2_USER_ID_FIELD=id
OAUTH2_USER_EMAIL_FIELD=email
OAUTH2_USER_NAME_FIELD=name
OAUTH2_USER_AVATAR_FIELD=avatar
```

### 3. 常见 OAuth2 服务配置示例

#### 示例 1：标准格式
如果您的用户信息 API 返回：
```json
{
  "id": "12345",
  "email": "user@example.com",
  "name": "张三",
  "avatar": "https://example.com/avatar.jpg"
}
```

配置：
```env
OAUTH2_USER_ID_FIELD=id
OAUTH2_USER_EMAIL_FIELD=email
OAUTH2_USER_NAME_FIELD=name
OAUTH2_USER_AVATAR_FIELD=avatar
```

#### 示例 2：嵌套格式
如果您的用户信息 API 返回：
```json
{
  "user_id": "12345",
  "contact": {
    "email": "user@example.com"
  },
  "profile": {
    "display_name": "张三",
    "picture": {
      "url": "https://example.com/avatar.jpg"
    }
  }
}
```

配置：
```env
OAUTH2_USER_ID_FIELD=user_id
OAUTH2_USER_EMAIL_FIELD=contact.email
OAUTH2_USER_NAME_FIELD=profile.display_name
OAUTH2_USER_AVATAR_FIELD=profile.picture.url
```

#### 示例 3：企业内部系统
如果您的企业内部 OAuth2 返回：
```json
{
  "employee_id": "EMP001",
  "work_email": "zhang.san@company.com",
  "full_name": "张三",
  "photo_url": "https://hr.company.com/photos/emp001.jpg"
}
```

配置：
```env
OAUTH2_USER_ID_FIELD=employee_id
OAUTH2_USER_EMAIL_FIELD=work_email
OAUTH2_USER_NAME_FIELD=full_name
OAUTH2_USER_AVATAR_FIELD=photo_url
```

## 🚀 运行项目

### 方法 1：Docker 部署（推荐）

```bash
# 启动数据库服务
docker-compose -f docker/docker-compose.yml up -d postgres redis

# 构建并启动后端
docker-compose -f docker/docker-compose.yml up -d backend
```

### 方法 2：开发模式

```bash
# 启动数据库
docker-compose -f docker/docker-compose.yml up -d postgres redis

# 启动后端
cd backend
go run cmd/server/main.go
```

## 🔍 测试 OAuth2 配置

### 1. 检查配置
访问 `http://localhost:6060`，点击登录按钮，应该会重定向到您配置的 OAuth2 授权页面。

### 2. 调试用户信息映射
如果登录后出现用户信息错误，可以：

1. 查看后端日志，确认用户信息 API 的响应格式
2. 根据实际响应调整字段映射配置
3. 重启服务测试

### 3. 常见问题排查

**问题 1：重定向 URI 不匹配**
- 确保 OAuth2 服务中配置的重定向 URI 与 `OAUTH2_REDIRECT_URL` 一致

**问题 2：用户信息获取失败**
- 检查 `OAUTH2_USERINFO_URL` 是否正确
- 确认访问令牌有足够权限访问用户信息 API

**问题 3：字段映射错误**
- 使用浏览器开发者工具或 Postman 测试用户信息 API
- 根据实际响应格式调整字段映射

## 📊 数据库变更

项目已升级使用 GORM：

### 优势
- **自动迁移**：启动时自动创建/更新数据库表结构
- **类型安全**：编译时检查数据库操作
- **关系映射**：支持复杂的数据关系
- **查询构建器**：更直观的查询语法

### 迁移说明
- 原有的 SQL 迁移文件仍然保留作为参考
- GORM 会自动处理表结构的创建和更新
- 数据库索引和约束通过 GORM 标签定义

## 🛠️ 开发指南

### 添加新的 OAuth2 提供者

如果需要添加特定的 OAuth2 提供者（如钉钉、企业微信等），可以：

1. 在 `internal/auth/oauth2.go` 中添加新的提供者实现
2. 在配置中添加对应的 case 分支
3. 根据提供者特性实现特殊逻辑

### 自定义用户信息处理

在 `CustomProvider.GetUserInfo()` 方法中，可以添加：
- 用户信息验证逻辑
- 字段格式转换
- 默认值设置
- 权限检查

## 📞 技术支持

如果在配置过程中遇到问题：

1. 检查 `.env` 文件配置是否正确
2. 查看后端启动日志
3. 确认 OAuth2 服务端配置
4. 测试网络连接和端点可访问性

祝您使用愉快！🎮
