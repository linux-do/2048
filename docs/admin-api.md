# Admin API Documentation

## 缓存刷新接口

### 手动刷新排行榜缓存

**接口地址**: `GET /api/admin/refresh-cache`

**权限要求**: 只有用户ID为 "1" 的管理员用户才能调用此接口

**功能**: 手动清除排行榜缓存，强制下次请求时从数据库重新加载数据

#### 请求参数

| 参数名 | 类型 | 必填 | 说明 |
|--------|------|------|------|
| type | string | 否 | 指定要刷新的排行榜类型 |

**支持的排行榜类型**:
- `daily` - 日排行榜
- `weekly` - 周排行榜  
- `monthly` - 月排行榜
- `all` - 总排行榜

如果不提供 `type` 参数，将刷新所有类型的排行榜缓存。

#### 请求示例

```bash
# 刷新所有排行榜缓存
curl -X GET "http://localhost:8080/api/admin/refresh-cache" \
  -H "Cookie: auth_token=your_jwt_token"

# 刷新特定类型的排行榜缓存
curl -X GET "http://localhost:8080/api/admin/refresh-cache?type=daily" \
  -H "Cookie: auth_token=your_jwt_token"

# 使用Authorization头
curl -X GET "http://localhost:8080/api/admin/refresh-cache" \
  -H "Authorization: Bearer your_jwt_token"
```

#### 响应示例

**成功响应** (200 OK):
```json
{
  "message": "Cache refresh completed",
  "refreshed_types": ["daily", "weekly", "monthly", "all"]
}
```

**部分成功响应** (200 OK):
```json
{
  "message": "Cache refresh completed with some errors",
  "refreshed_types": ["daily", "weekly"],
  "errors": [
    "Failed to invalidate monthly cache: connection timeout",
    "Failed to invalidate all cache: connection timeout"
  ]
}
```

**权限不足** (403 Forbidden):
```json
{
  "error": "Access denied. Admin privileges required."
}
```

**未认证** (401 Unauthorized):
```json
{
  "error": "Authentication required"
}
```

**无效参数** (400 Bad Request):
```json
{
  "error": "Invalid leaderboard type. Must be 'daily', 'weekly', 'monthly', or 'all'"
}
```

**缓存服务不可用** (503 Service Unavailable):
```json
{
  "error": "Cache service not available"
}
```

**完全失败** (500 Internal Server Error):
```json
{
  "message": "Cache refresh completed with some errors",
  "refreshed_types": [],
  "errors": [
    "Failed to invalidate daily cache: connection failed",
    "Failed to invalidate weekly cache: connection failed",
    "Failed to invalidate monthly cache: connection failed",
    "Failed to invalidate all cache: connection failed"
  ]
}
```

#### 使用场景

1. **数据更新后**: 当直接在数据库中修改排行榜数据后，需要刷新缓存
2. **缓存异常**: 当发现排行榜显示的数据不正确时
3. **定期维护**: 作为定期维护任务的一部分
4. **测试环境**: 在测试环境中快速更新排行榜数据

#### 安全注意事项

- 此接口只能由ID为 "1" 的用户调用
- 需要有效的JWT认证token
- 建议在生产环境中限制此接口的访问频率
- 可以考虑添加IP白名单限制

#### 技术实现

- 使用Redis的缓存失效机制
- 支持单个类型或全部类型的缓存刷新
- 提供详细的错误信息和成功状态
- 线程安全的缓存操作

#### 监控建议

建议监控以下指标：
- 接口调用频率
- 缓存刷新成功率
- 缓存刷新耗时
- 错误日志

#### 扩展功能

未来可以考虑添加：
- 定时自动刷新缓存
- 更细粒度的缓存控制
- 缓存预热功能
- 缓存统计信息查询
