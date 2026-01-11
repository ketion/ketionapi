# Design Document

## Overview

本设计文档描述如何为 NewAPI 项目添加魔塔（ModelScope）平台异步文生图模型支持。设计遵循**完全向后兼容、零影响现有功能**的核心原则，采用插件式架构，将 ModelScope 作为新的渠道类型独立实现。

### 设计原则

1. **隔离实现**：所有 ModelScope 相关代码在独立包中实现
2. **零侵入集成**：使用现有接口和扩展点，不修改核心代码
3. **条件路由**：仅当渠道类型为 ModelScope 时才使用新适配器
4. **可插拔设计**：可以随时禁用或移除而不影响系统

## Architecture

### 系统架构图

```
┌─────────────────────────────────────────────────────────────┐
│                     API Gateway Layer                        │
│                  /v1/images/generations                      │
└────────────────────────┬────────────────────────────────────┘
                         │
                         ▼
┌─────────────────────────────────────────────────────────────┐
│                   Relay Layer (不修改)                       │
│                   ImageHelper Function                       │
└────────────────────────┬────────────────────────────────────┘
                         │
                         ▼
┌─────────────────────────────────────────────────────────────┐
│              Adaptor Factory (扩展注册)                      │
│                   GetAdaptor(apiType)                        │
└─────┬───────────────────────────────────────────────────────┘
      │
      ├─ 现有渠道 (不修改)
      │  ├─ OpenAI Adaptor
      │  ├─ Replicate Adaptor
      │  └─ ...
      │
      └─ 新增渠道 (独立实现)
         └─ ModelScope Adaptor ← 新增
            ├─ 异步任务提交
            ├─ 任务状态轮询
            └─ 响应格式转换
```

### 数据流程

```
客户端请求 (OpenAI 格式)
    │
    ▼
ImageHelper (现有代码，不修改)
    │
    ├─ 解析请求
    ├─ 选择适配器 (基于 channel_type)
    │
    ▼
ModelScope Adaptor (新增)
    │
    ├─ ConvertImageRequest: OpenAI → ModelScope 格式
    ├─ DoRequest: 提交异步任务 → 获取 task_id
    ├─ PollTaskStatus: 轮询任务状态
    │   ├─ SUCCEED → 提取图片 URL
    │   ├─ FAILED → 返回错误
    │   └─ PROCESSING → 继续轮询
    │
    ▼
DoResponse: ModelScope → OpenAI 格式
    │
    ▼
返回给客户端 (OpenAI 格式)
```

## Components and Interfaces

### 1. 渠道类型常量 (constant/channel.go)

**修改方式**：仅添加新常量，不修改现有代码

```go
// 在现有常量列表末尾添加
const (
    // ... 现有常量 ...
    ChannelTypeReplicate      = 56
    ChannelTypeModelScope     = 57  // 新增
    ChannelTypeDummy          // this one is only for count
)

// 在 ChannelBaseURLs 数组末尾添加
var ChannelBaseURLs = []string{
    // ... 现有 URL ...
    "https://api.replicate.com",                 // 56
    "https://api-inference.modelscope.cn",       // 57 新增
}

// 在 ChannelTypeNames 映射中添加
var ChannelTypeNames = map[int]string{
    // ... 现有映射 ...
    ChannelTypeReplicate:      "Replicate",
    ChannelTypeModelScope:     "ModelScope",     // 新增
}
```

### 2. API 类型常量 (constant/api_type.go)

**修改方式**：仅添加新常量

```go
const (
    // ... 现有常量 ...
    APITypeReplicate      = 56
    APITypeModelScope     = 57  // 新增
)
```

### 3. ModelScope 适配器包结构

**新增目录**：`relay/channel/modelscope/`

```
relay/channel/modelscope/
├── adaptor.go      # 适配器实现
├── dto.go          # 数据结构定义
├── constants.go    # 常量定义
└── polling.go      # 轮询逻辑
```

### 4. ModelScope 适配器接口实现

#### adaptor.go

```go
package modelscope

import (
    "github.com/QuantumNous/new-api/relay/channel"
    relaycommon "github.com/QuantumNous/new-api/relay/common"
    "github.com/gin-gonic/gin"
)

type Adaptor struct{}

// 实现 channel.Adaptor 接口的所有方法
func (a *Adaptor) Init(info *relaycommon.RelayInfo)
func (a *Adaptor) GetRequestURL(info *relaycommon.RelayInfo) (string, error)
func (a *Adaptor) SetupRequestHeader(c *gin.Context, req *http.Header, info *relaycommon.RelayInfo) error
func (a *Adaptor) ConvertImageRequest(c *gin.Context, info *relaycommon.RelayInfo, request dto.ImageRequest) (any, error)
func (a *Adaptor) DoRequest(c *gin.Context, info *relaycommon.RelayInfo, requestBody io.Reader) (any, error)
func (a *Adaptor) DoResponse(c *gin.Context, resp *http.Response, info *relaycommon.RelayInfo) (any, *types.NewAPIError)
func (a *Adaptor) GetModelList() []string
func (a *Adaptor) GetChannelName() string
```

#### dto.go

```go
package modelscope

// ModelScope 任务提交请求
type TaskSubmitRequest struct {
    Model  string                 `json:"model"`
    Prompt string                 `json:"prompt"`
    Loras  interface{}            `json:"loras,omitempty"`
    Extra  map[string]interface{} `json:"-"`
}

// ModelScope 任务提交响应
type TaskSubmitResponse struct {
    TaskID    string `json:"task_id"`
    RequestID string `json:"request_id"`
}

// ModelScope 任务查询响应
type TaskQueryResponse struct {
    TaskID       string   `json:"task_id"`
    TaskStatus   string   `json:"task_status"` // SUCCEED, FAILED, PENDING, RUNNING
    OutputImages []string `json:"output_images,omitempty"`
    Message      string   `json:"message,omitempty"`
}
```

#### constants.go

```go
package modelscope

const (
    ChannelName = "ModelScope"
    
    // 任务状态
    TaskStatusSucceed    = "SUCCEED"
    TaskStatusFailed     = "FAILED"
    TaskStatusPending    = "PENDING"
    TaskStatusRunning    = "RUNNING"
    
    // 轮询配置
    MaxPollAttempts      = 60        // 最大轮询次数
    PollInterval         = 5         // 轮询间隔（秒）
    InitialPollInterval  = 2         // 初始轮询间隔（秒）
    MaxPollInterval      = 10        // 最大轮询间隔（秒）
)

var ModelList = []string{
    "Tongyi-MAI/Z-Image-Turbo",
    "modelscope/stable-diffusion-v1-5",
    // 可以添加更多支持的模型
}
```

#### polling.go

```go
package modelscope

import (
    "context"
    "time"
)

// PollTaskStatus 轮询任务状态直到完成或超时
func (a *Adaptor) PollTaskStatus(ctx context.Context, taskID string, info *relaycommon.RelayInfo) (*TaskQueryResponse, error) {
    interval := InitialPollInterval
    
    for attempt := 0; attempt < MaxPollAttempts; attempt++ {
        // 检查上下文是否已取消
        select {
        case <-ctx.Done():
            return nil, ctx.Err()
        default:
        }
        
        // 查询任务状态
        resp, err := a.queryTaskStatus(taskID, info)
        if err != nil {
            return nil, err
        }
        
        // 根据状态决定是否继续轮询
        switch resp.TaskStatus {
        case TaskStatusSucceed:
            return resp, nil
        case TaskStatusFailed:
            return resp, fmt.Errorf("task failed: %s", resp.Message)
        case TaskStatusPending, TaskStatusRunning:
            // 继续轮询，使用指数退避
            time.Sleep(time.Duration(interval) * time.Second)
            interval = min(interval*2, MaxPollInterval)
        default:
            return nil, fmt.Errorf("unknown task status: %s", resp.TaskStatus)
        }
    }
    
    return nil, fmt.Errorf("task polling timeout after %d attempts", MaxPollAttempts)
}

func (a *Adaptor) queryTaskStatus(taskID string, info *relaycommon.RelayInfo) (*TaskQueryResponse, error) {
    // 实现 HTTP GET 请求到 /v1/tasks/{task_id}
    // 设置 X-ModelScope-Task-Type: image_generation 头
}
```

### 5. 适配器注册 (relay/relay_adaptor.go)

**修改方式**：仅在 switch 语句中添加新 case

```go
func GetAdaptor(apiType int) channel.Adaptor {
    switch apiType {
    // ... 现有 case ...
    case constant.APITypeReplicate:
        return &replicate.Adaptor{}
    case constant.APITypeModelScope:  // 新增
        return &modelscope.Adaptor{}
    }
    return nil
}
```

### 6. 渠道类型到 API 类型映射 (common/channel.go)

**修改方式**：在映射函数中添加新 case

```go
func ChannelType2APIType(channelType int) (int, error) {
    apiType := channelType
    switch channelType {
    // ... 现有映射 ...
    case constant.ChannelTypeModelScope:  // 新增
        apiType = constant.APITypeModelScope
    }
    return apiType, nil
}
```

## Data Models

### 请求转换

**OpenAI 格式 → ModelScope 格式**

```json
// 输入 (OpenAI 格式)
{
  "model": "Tongyi-MAI/Z-Image-Turbo",
  "prompt": "A golden cat",
  "n": 1,
  "size": "1024x1024",
  "response_format": "url"
}

// 输出 (ModelScope 格式)
{
  "model": "Tongyi-MAI/Z-Image-Turbo",
  "prompt": "A golden cat"
}
```

### 响应转换

**ModelScope 格式 → OpenAI 格式**

```json
// ModelScope 任务查询响应
{
  "task_id": "abc123",
  "task_status": "SUCCEED",
  "output_images": [
    "https://modelscope.cn/api/v1/models/xxx/output/image1.png"
  ]
}

// 转换为 OpenAI 格式
{
  "created": 1704067200,
  "data": [
    {
      "url": "https://modelscope.cn/api/v1/models/xxx/output/image1.png"
    }
  ]
}
```

### LoRA 配置支持

```json
// 单个 LoRA
{
  "loras": "lora-repo-id"
}

// 多个 LoRA (权重总和必须为 1.0)
{
  "loras": {
    "lora-repo-id1": 0.6,
    "lora-repo-id2": 0.4
  }
}
```

## Correctness Properties

*属性基于测试（Property-Based Testing）是一种强大的软件正确性验证工具。该过程从开发者决定代码应满足的形式化规范开始，并将该规范编码为可执行的属性。*

### Property 1: 请求格式转换保持语义

*对于任意* 有效的 OpenAI 图片生成请求，转换为 ModelScope 格式后再转换回来，应保持核心语义不变（model 和 prompt 字段）

**Validates: Requirements 2.2, 2.6**

### Property 2: 异步任务提交返回有效 task_id

*对于任意* 有效的 ModelScope 请求，提交任务后应返回非空的 task_id 字符串

**Validates: Requirements 2.4**

### Property 3: 任务轮询最终收敛

*对于任意* 已提交的任务，轮询操作应在有限时间内收敛到 SUCCEED、FAILED 或 TIMEOUT 状态之一

**Validates: Requirements 3.3, 3.7**

### Property 4: LoRA 权重验证

*对于任意* 包含多个 LoRA 的请求，所有 LoRA 权重系数之和应等于 1.0（误差范围 ±0.001）

**Validates: Requirements 5.4**

### Property 5: LoRA 数量限制

*对于任意* 包含 LoRA 的请求，LoRA 模型数量应不超过 6 个

**Validates: Requirements 5.5**

### Property 6: 响应格式转换保持图片 URL

*对于任意* ModelScope 成功响应，转换为 OpenAI 格式后应保留所有图片 URL

**Validates: Requirements 10.1**

### Property 7: 错误响应正确传播

*对于任意* ModelScope 错误响应，应转换为有意义的错误消息并正确传播给客户端

**Validates: Requirements 6.1**

### Property 8: 现有渠道不受影响

*对于任意* 非 ModelScope 渠道的请求，处理流程应与添加 ModelScope 支持前完全一致

**Validates: Requirements 11.1, 11.2, 11.10**

### Property 9: 配额计算一致性

*对于任意* ModelScope 请求，配额计算应使用与其他渠道相同的逻辑和公式

**Validates: Requirements 4.4**

### Property 10: 数据库记录完整性

*对于任意* ModelScope 任务，数据库记录应包含所有必需字段（task_id, user_id, channel_id, status）

**Validates: Requirements 7.1, 7.2**

## Error Handling

### 错误分类

1. **请求验证错误**
   - 缺少必需参数（prompt）
   - 无效的 LoRA 配置
   - 模型不支持

2. **网络错误**
   - 连接超时
   - DNS 解析失败
   - 网络不可达

3. **API 错误**
   - 认证失败（无效的 API Key）
   - 配额不足
   - 速率限制

4. **任务执行错误**
   - 任务提交失败
   - 任务执行失败
   - 轮询超时

### 错误处理策略

```go
// 错误处理示例
func (a *Adaptor) DoRequest(c *gin.Context, info *relaycommon.RelayInfo, requestBody io.Reader) (any, error) {
    // 1. 提交任务
    resp, err := a.submitTask(c, info, requestBody)
    if err != nil {
        return nil, handleSubmitError(err)
    }
    
    // 2. 轮询任务状态
    taskResp, err := a.PollTaskStatus(c.Request.Context(), resp.TaskID, info)
    if err != nil {
        return nil, handlePollError(err)
    }
    
    return taskResp, nil
}

func handleSubmitError(err error) error {
    // 根据错误类型返回适当的错误码和消息
    if isAuthError(err) {
        return types.NewError(err, types.ErrorCodeUnauthorized)
    }
    if isRateLimitError(err) {
        return types.NewError(err, types.ErrorCodeRateLimitExceeded)
    }
    return types.NewError(err, types.ErrorCodeDoRequestFailed)
}
```

### 重试策略

- **指数退避**：轮询间隔从 2 秒开始，每次失败后翻倍，最大 10 秒
- **最大重试次数**：60 次（总计约 5 分钟）
- **不重试的错误**：认证失败、参数验证失败

## Testing Strategy

### 单元测试

**测试范围**：
- 请求格式转换函数
- 响应格式转换函数
- LoRA 配置验证
- 错误处理逻辑

**测试示例**：
```go
func TestConvertImageRequest(t *testing.T) {
    adaptor := &Adaptor{}
    request := dto.ImageRequest{
        Model:  "Tongyi-MAI/Z-Image-Turbo",
        Prompt: "A golden cat",
    }
    
    result, err := adaptor.ConvertImageRequest(nil, nil, request)
    assert.NoError(t, err)
    assert.Equal(t, "Tongyi-MAI/Z-Image-Turbo", result.Model)
    assert.Equal(t, "A golden cat", result.Prompt)
}
```

### 属性测试

**测试配置**：
- 每个属性测试运行 100 次迭代
- 使用随机生成的输入数据
- 标签格式：`Feature: modelscope-async-image-generation, Property {number}: {property_text}`

**测试示例**：
```go
// Feature: modelscope-async-image-generation, Property 4: LoRA 权重验证
func TestProperty_LoRAWeightValidation(t *testing.T) {
    rapid.Check(t, func(t *rapid.T) {
        // 生成随机 LoRA 配置
        loraCount := rapid.IntRange(2, 6).Draw(t, "lora_count")
        loras := generateRandomLoras(t, loraCount)
        
        // 验证权重总和
        totalWeight := calculateTotalWeight(loras)
        assert.InDelta(t, 1.0, totalWeight, 0.001)
    })
}
```

### 集成测试

**测试场景**：
1. 端到端图片生成流程
2. 与现有渠道的隔离性
3. 错误场景处理
4. 并发请求处理

### 向后兼容性测试

**测试目标**：确保现有功能不受影响

```go
func TestBackwardCompatibility(t *testing.T) {
    // 测试现有渠道（OpenAI, Replicate 等）
    existingChannels := []int{
        constant.ChannelTypeOpenAI,
        constant.ChannelTypeReplicate,
        // ... 其他渠道
    }
    
    for _, channelType := range existingChannels {
        t.Run(fmt.Sprintf("Channel_%d", channelType), func(t *testing.T) {
            // 验证适配器仍然正常工作
            apiType, _ := common.ChannelType2APIType(channelType)
            adaptor := relay.GetAdaptor(apiType)
            assert.NotNil(t, adaptor)
            
            // 验证适配器类型未改变
            // 验证接口方法仍然可用
        })
    }
}
```

## Implementation Notes

### 关键实现细节

1. **异步转同步处理**
   - 对外提供同步式 API
   - 内部处理异步轮询逻辑
   - 使用 context 支持请求取消

2. **轮询优化**
   - 初始间隔短（2秒），快速响应
   - 指数退避，减少服务器压力
   - 最大间隔限制（10秒），避免过长等待

3. **资源管理**
   - 及时关闭 HTTP 连接
   - 使用 context 控制超时
   - 避免 goroutine 泄漏

4. **日志记录**
   - 记录任务提交和状态变化
   - 记录轮询次数和耗时
   - 记录错误详情便于排查

### 性能考虑

1. **并发控制**
   - 每个请求独立轮询
   - 不共享状态，避免锁竞争
   - 使用 HTTP 连接池

2. **内存使用**
   - 及时释放响应体
   - 避免缓存大量图片数据
   - 使用流式处理

3. **网络优化**
   - 复用 HTTP 连接
   - 设置合理的超时时间
   - 支持代理配置

## Deployment Considerations

### Docker 构建

**Dockerfile 修改**：无需修改，新代码会自动包含在构建中

**构建命令**：
```bash
docker build -t new-api:modelscope .
```

### 环境变量

无需新增环境变量，使用现有配置系统

### 配置示例

```yaml
# 在管理后台添加 ModelScope 渠道
channel:
  type: ModelScope
  name: "魔塔文生图"
  base_url: "https://api-inference.modelscope.cn"
  api_key: "ms-xxxxxxxxxxxx"
  models:
    - "Tongyi-MAI/Z-Image-Turbo"
```

### 监控指标

建议监控的指标：
- ModelScope 请求成功率
- 平均轮询次数
- 平均任务完成时间
- 错误率分布

### 回滚计划

如果出现问题，可以：
1. 在管理后台禁用 ModelScope 渠道
2. 系统自动回退到其他可用渠道
3. 不影响现有渠道的使用

## Migration Path

### 阶段 1：开发和测试
1. 实现 ModelScope 适配器
2. 编写单元测试和属性测试
3. 本地测试验证

### 阶段 2：灰度发布
1. 在测试环境部署
2. 创建测试渠道
3. 小范围用户测试

### 阶段 3：全量发布
1. 在生产环境部署
2. 在管理后台启用 ModelScope 渠道类型
3. 监控系统运行状态

### 阶段 4：优化迭代
1. 根据用户反馈优化
2. 调整轮询策略
3. 添加更多支持的模型
