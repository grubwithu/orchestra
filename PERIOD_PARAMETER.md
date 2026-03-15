# Period 参数说明

## 概述

在 `reportCorpus` API 中添加了 `period` 参数，用于标记 Fuzzer 测试的不同阶段。

## 参数详情

### 请求体结构

```go
type CorpusReportReqBody struct {
	Fuzzer   string   `json:"fuzzer"`
	Identity string   `json:"identity"`
	Corpus   []string `json:"corpus"`
	Period   string   `json:"period,omitempty"` // "begin" or "end", optional
}
```

### 参数说明

- **`period`** (可选)
  - **类型**: `string`
  - **有效值**: 
    - `"begin"` - 标记新一轮测试的开始
    - `"end"` - 标记当前轮测试的结束
    - 空字符串 - 普通报告（向后兼容）
  - **默认**: 空（omitempty，可选参数）
  - **验证**: 服务器会验证值必须是 `"begin"`、`"end"` 或空

## 使用场景

### 1. 开始新一轮测试 (`period: "begin"`)

**场景**: Fuzzer 开始新的一轮测试周期

**服务器行为**:
- 重置或重新初始化该 Fuzzer 的覆盖率跟踪
- 设置新的基线覆盖率
- 开始新的统计周期

**示例请求**:
```json
{
  "fuzzer": "afl",
  "identity": "round-1",
  "corpus": ["/path/to/initial/corpus"],
  "period": "begin"
}
```

### 2. 结束当前轮测试 (`period: "end"`)

**场景**: Fuzzer 完成当前轮测试

**服务器行为**:
- 进行最终的统计计算
- 标记该轮测试完成
- 准备下一轮测试的数据

**示例请求**:
```json
{
  "fuzzer": "afl",
  "identity": "round-1",
  "corpus": ["/path/to/final/corpus"],
  "period": "end"
}
```

### 3. 普通报告 (无 `period` 或 `period: ""`)

**场景**: 常规的 corpus 报告，向后兼容

**服务器行为**:
- 使用现有的覆盖率数据进行对比
- 追加到当前统计中
- 保持原有逻辑不变

**示例请求**:
```json
{
  "fuzzer": "afl",
  "identity": "intermediate",
  "corpus": ["/path/to/corpus"]
}
```

## 实现细节

### 服务器端处理 (`processCorpus` 函数)

#### 对于 `period: "begin"`:
1. **Fuzzer 覆盖率**: 重置为该 Fuzzer 的开始数组
   ```go
   s.FuzzerCovs[fuzzer] = []int{cov}
   ```
2. **全局覆盖率**: 记录为新的基线
3. **行覆盖率**: 使用重置的基线进行对比

#### 对于 `period: "end"` 或未指定:
1. **Fuzzer 覆盖率**: 追加到现有数组
2. **全局覆盖率**: 累加到现有数据
3. **行覆盖率**: 使用之前的数据进行对比

### 验证逻辑 (`handleReportCorpus` 函数)

```go
// Validate period parameter if provided
if report.Period != "" && report.Period != "begin" && report.Period != "end" {
    c.JSON(http.StatusBadRequest, APIResponse{
        Success: false,
        Message: "Invalid period value. Must be 'begin' or 'end' if provided",
    })
    return
}
```

## 向后兼容性

1. **可选参数**: 使用 `omitempty` 标签，确保现有客户端无需修改
2. **默认行为**: 当 `period` 为空时，保持原有处理逻辑
3. **错误处理**: 无效的 `period` 值会返回明确的错误信息

## 示例工作流程

### 完整测试周期

```bash
# 1. 开始新一轮测试
curl -X POST http://localhost:8080/reportCorpus \
  -H "Content-Type: application/json" \
  -d '{
    "fuzzer": "afl",
    "identity": "round-1-start",
    "corpus": ["/corpus/initial"],
    "period": "begin"
  }'

# 2. 中间报告（多次）
curl -X POST http://localhost:8080/reportCorpus \
  -H "Content-Type: application/json" \
  -d '{
    "fuzzer": "afl",
    "identity": "round-1-mid",
    "corpus": ["/corpus/mid1"]
  }'

# 3. 结束当前轮测试
curl -X POST http://localhost:8080/reportCorpus \
  -H "Content-Type: application/json" \
  -d '{
    "fuzzer": "afl",
    "identity": "round-1-end",
    "corpus": ["/corpus/final"],
    "period": "end"
  }'

# 4. 开始下一轮测试
curl -X POST http://localhost:8080/reportCorpus \
  -H "Content-Type: application/json" \
  -d '{
    "fuzzer": "afl",
    "identity": "round-2-start",
    "corpus": ["/corpus/new-initial"],
    "period": "begin"
  }'
```

## 错误处理

### 1. 无效的 period 值
```json
{
  "success": false,
  "message": "Invalid period value. Must be 'begin' or 'end' if provided"
}
```

### 2. 空的 corpus 数组
```json
{
  "success": false,
  "message": "Corpus is empty"
}
```

## 日志输出

服务器会记录 `period` 参数的使用情况：

```
Processing corpus for task [task-id], fuzzer: [fuzzer-name], period: [period-value]
Period 'begin' for fuzzer: [fuzzer-name]. Initializing coverage tracking.
Period 'begin': Starting fresh line coverage comparison
```

---

**状态**: ✅ 已实现
**修改者**: 白坂小梅 (Shirasaka Koume)
**日期**: 2026-03-15
**对应需求**: 在 reportCorpus API 中添加 period 参数，支持 begin/end 两种值