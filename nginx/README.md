# Nginx Configuration for 2048 Game

## 配置选项

### 1. 直接在服务器上使用 Nginx

如果你在服务器上直接运行应用（不使用Docker），使用 `game2048.conf`：

```bash
# 复制配置文件
sudo cp nginx/game2048.conf /etc/nginx/sites-available/game2048

# 创建软链接启用站点
sudo ln -s /etc/nginx/sites-available/game2048 /etc/nginx/sites-enabled/

# 修改配置文件中的域名
sudo nano /etc/nginx/sites-available/game2048
# 将 your-domain.com 替换为你的实际域名

# 测试配置
sudo nginx -t

# 重载 Nginx
sudo systemctl reload nginx
```

**重要配置项：**
- `server_name`: 替换为你的域名
- `upstream game2048_backend`: 确保后端地址正确（默认 127.0.0.1:8080）

### 2. 使用 Docker Compose

如果使用Docker Compose部署，配置已经包含在 `docker-compose.yml` 中：

```bash
# 启动所有服务（包括 Nginx）
docker-compose -f docker/docker-compose.yml up -d

# 查看服务状态
docker-compose -f docker/docker-compose.yml ps

# 查看 Nginx 日志
docker-compose -f docker/docker-compose.yml logs nginx
```

## 主要特性

### 🚀 性能优化
- **Gzip 压缩**: 自动压缩 CSS、JS、JSON 等文件
- **静态文件缓存**: CSS/JS 文件缓存 1 年（版本化URL处理缓存失效）
- **Keep-alive 连接**: 减少连接开销
- **缓冲优化**: 合理的代理缓冲设置

### 🔒 安全特性
- **速率限制**: 
  - API 请求: 10 req/s
  - 认证请求: 5 req/s  
  - WebSocket: 20 req/s
- **安全头**: X-Frame-Options, X-Content-Type-Options, X-XSS-Protection
- **敏感文件保护**: 阻止访问 .env、.git 等文件

### 🌐 WebSocket 支持
- **完整的 WebSocket 代理**: 支持游戏实时通信
- **长连接**: 24小时超时设置
- **升级头处理**: 正确的 HTTP 升级到 WebSocket

### 📊 监控和日志
- **健康检查**: 自动检查后端服务状态
- **访问日志**: 记录所有请求（健康检查除外）
- **错误页面**: 自定义 50x 错误页面

## SSL/HTTPS 配置

### 使用 Let's Encrypt

```bash
# 安装 Certbot
sudo apt install certbot python3-certbot-nginx

# 获取证书
sudo certbot --nginx -d your-domain.com

# 自动续期
sudo crontab -e
# 添加: 0 12 * * * /usr/bin/certbot renew --quiet
```

### 手动 SSL 证书

取消注释 `game2048.conf` 中的 HTTPS 部分，并更新证书路径：

```nginx
ssl_certificate /path/to/your/certificate.crt;
ssl_certificate_key /path/to/your/private.key;
```

## 负载均衡

如果有多个后端实例，更新 upstream 配置：

```nginx
upstream game2048_backend {
    server 127.0.0.1:8080;
    server 127.0.0.1:8081;
    server 127.0.0.1:8082;
    
    # 负载均衡方法
    # least_conn;  # 最少连接
    # ip_hash;     # IP 哈希
    
    keepalive 32;
}
```

## 故障排除

### 常见问题

1. **502 Bad Gateway**
   ```bash
   # 检查后端服务是否运行
   curl http://localhost:8080/health
   
   # 检查 Nginx 错误日志
   sudo tail -f /var/log/nginx/error.log
   ```

2. **WebSocket 连接失败**
   ```bash
   # 检查 WebSocket 升级头
   curl -i -N -H "Connection: Upgrade" -H "Upgrade: websocket" http://localhost/ws
   ```

3. **静态文件 404**
   ```bash
   # 检查文件路径和权限
   ls -la /path/to/static/files
   ```

### 性能调优

```nginx
# 在 http 块中添加
worker_processes auto;
worker_connections 1024;

# 调整缓冲区大小
proxy_buffer_size 8k;
proxy_buffers 16 8k;

# 启用 HTTP/2
listen 443 ssl http2;
```

## 监控建议

### 日志分析
```bash
# 实时查看访问日志
sudo tail -f /var/log/nginx/access.log

# 分析错误日志
sudo grep "error" /var/log/nginx/error.log

# 统计请求状态码
awk '{print $9}' /var/log/nginx/access.log | sort | uniq -c
```

### 性能监控
- 使用 `nginx-module-vts` 模块获取详细统计
- 配置 Prometheus + Grafana 监控
- 设置告警规则监控 5xx 错误率

## 备份和恢复

```bash
# 备份配置
sudo cp /etc/nginx/sites-available/game2048 /backup/nginx-game2048-$(date +%Y%m%d).conf

# 恢复配置
sudo cp /backup/nginx-game2048-20231201.conf /etc/nginx/sites-available/game2048
sudo nginx -t && sudo systemctl reload nginx
```
