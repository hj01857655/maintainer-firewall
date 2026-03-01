# Dashboard Metrics 时区Bug报告

## 🐛 Bug描述
Dashboard页面图表无法显示24小时内数据，metrics API始终返回0

## 🔍 问题分析

### 发现过程
1. **初始检查**：通过API检查数据库状态，发现metrics返回0
2. **直接查询**：使用MySQL命令直接查询数据库，发现有大量数据
3. **对比分析**：API查询24小时内数据为0，但数据库有100条24小时内记录
4. **根因定位**：时区不匹配导致查询条件错误

### 具体问题
- **API查询**：`parseWindowStart`函数使用`time.Now().UTC()`计算时间窗口
- **数据库存储**：时间戳存储为本地时间（非UTC）
- **结果**：时间不匹配，查询不到数据

## 📊 证据数据

### 数据库实际数据
```sql
-- 24小时内数据统计
SELECT COUNT(*) FROM webhook_events WHERE received_at >= DATE_SUB(NOW(), INTERVAL 24 HOUR);
-- 结果：100条

-- 数据时间范围
SELECT MIN(received_at), MAX(received_at) FROM webhook_events;
-- 结果：2026-03-01 06:02:58 ~ 2026-03-01 08:23:45
```

### API响应
```json
{
  "ok": true,
  "window": "24h",
  "since": "2026-03-01T10:23:45.123Z",
  "overview": {
    "events_24h": 0,
    "alerts_24h": 0,
    "failures_24h": 0,
    "success_rate_24h": 0,
    "p95_latency_ms_24h": 0
  }
}
```

## 🛠️ 修复方案

### 修改文件
`apps/api-go/internal/http/handlers/observability.go`

### 当前代码 (342-354行)
```go
func parseWindowStart(v string) (time.Time, error) {
	now := time.Now().UTC()  // ❌ 使用UTC时间
	switch strings.ToLower(strings.TrimSpace(v)) {
	case "24h", "1d", "day":
		return now.Add(-24 * time.Hour), nil
	case "12h":
		return now.Add(-12 * time.Hour), nil
	case "6h":
		return now.Add(-6 * time.Hour), nil
	default:
		return time.Time{}, fmt.Errorf("window must be one of: 6h, 12h, 24h")
	}
}
```

### 修复后的代码
```go
func parseWindowStart(v string) (time.Time, error) {
	now := time.Now()  // ✅ 使用本地时区
	switch strings.ToLower(strings.TrimSpace(v)) {
	case "24h", "1d", "day":
		return now.Add(-24 * time.Hour), nil
	case "12h":
		return now.Add(-12 * time.Hour), nil
	case "6h":
		return now.Add(-6 * time.Hour), nil
	default:
		return time.Time{}, fmt.Errorf("window must be one of: 6h, 12h, 24h")
	}
}
```

### 关键修改
```diff
- now := time.Now().UTC()  // UTC时间
+ now := time.Now()         // 本地时区
```

## ✅ 修复验证
修复后API应该返回正确的数据：
```json
{
  "overview": {
    "events_24h": 100,
    "alerts_24h": 2,
    "failures_24h": 2
  }
}
```

## 📅 报告日期
2026-03-01

## 🔄 状态
- [ ] 待修复
- [ ] 已修复
- [ ] 已验证
