# Implementation Plan: ModelScope 异步文生图支持

## Overview

本实施计划将 ModelScope 平台的异步文生图功能集成到 NewAPI 项目中。采用插件式架构，确保完全向后兼容，不影响现有功能。实施分为渠道类型注册、适配器实现、集成测试和 Docker 打包四个主要阶段。

## Tasks

- [x] 1. 添加 ModelScope 渠道类型常量和配置
  - 在 `constant/channel.go` 中添加 `ChannelTypeModelScope` 常量
  - 在 `ChannelBaseURLs` 数组中添加 ModelScope 默认 URL
  - 在 `ChannelTypeNames` 映射中添加 ModelScope 名称
  - 在 `constant/api_type.go` 中添加 `APITypeModelScope` 常量
  - _Requirements: 1.1, 1.2, 1.3_

- [x] 2. 添加渠道类型映射
  - 在 `common/channel.go` 的 `ChannelType2APIType` 函数中添加 ModelScope 映射
  - _Requirements: 1.4_

- [x] 3. 创建 ModelScope 适配器包结构
  - 创建目录 `relay/channel/modelscope/`
  - 创建文件 `constants.go` 定义常量和模型列表
  - 创建文件 `dto.go` 定义数据结构
  - _Requirements: 2.1_

- [x] 4. 实现 ModelScope 数据结构
  - [x] 4.1 实现任务提交请求结构 `TaskSubmitRequest`
    - 定义 Model、Prompt、Loras 字段
    - 支持额外参数的 Extra 字段
    - _Requirements: 2.6, 5.1, 5.2_

  - [x] 4.2 实现任务提交响应结构 `TaskSubmitResponse`
    - 定义 TaskID、RequestID 字段
    - _Requirements: 2.4_

  - [x] 4.3 实现任务查询响应结构 `TaskQueryResponse`
    - 定义 TaskID、TaskStatus、OutputImages、Message 字段
    - _Requirements: 3.4_

- [x] 5. 实现 ModelScope 适配器核心功能
  - [x] 5.1 实现 `Init` 方法
    - 初始化适配器状态
    - _Requirements: 2.1_

  - [x] 5.2 实现 `GetRequestURL` 方法
    - 根据操作类型返回正确的 URL
    - 任务提交: `/v1/images/generations`
    - 任务查询: `/v1/tasks/{task_id}`
    - _Requirements: 2.2_

  - [x] 5.3 实现 `SetupRequestHeader` 方法
    - 设置 Authorization 头（Bearer Token）
    - 设置 Content-Type 为 application/json
    - 任务提交时设置 X-ModelScope-Async-Mode: true
    - 任务查询时设置 X-ModelScope-Task-Type: image_generation
    - _Requirements: 2.3, 3.2_

  - [x] 5.4 实现 `ConvertImageRequest` 方法
    - 将 OpenAI ImageRequest 转换为 ModelScope TaskSubmitRequest
    - 提取 model 和 prompt 字段
    - 处理 LoRA 配置（单个和多个）
    - 验证 LoRA 权重总和为 1.0
    - 验证 LoRA 数量不超过 6 个
    - 传递额外参数
    - _Requirements: 2.2, 2.6, 5.1, 5.2, 5.3, 5.4, 5.5, 5.6_

  - [ ]* 5.5 编写 ConvertImageRequest 的属性测试
    - **Property 1: 请求格式转换保持语义**
    - **Property 4: LoRA 权重验证**
    - **Property 5: LoRA 数量限制**
    - **Validates: Requirements 2.2, 2.6, 5.4, 5.5**

- [ ] 6. 实现任务提交和轮询逻辑
  - [x] 6.1 创建 `polling.go` 文件
    - _Requirements: 3.1_

  - [x] 6.2 实现 `submitTask` 方法
    - 发送 POST 请求到 `/v1/images/generations`
    - 解析响应获取 task_id
    - 处理提交错误
    - _Requirements: 2.4, 6.2_

  - [ ]* 6.3 编写 submitTask 的单元测试
    - 测试成功提交场景
    - 测试错误处理
    - _Requirements: 2.4_

  - [x] 6.4 实现 `queryTaskStatus` 方法
    - 发送 GET 请求到 `/v1/tasks/{task_id}`
    - 设置正确的请求头
    - 解析任务状态响应
    - _Requirements: 3.2, 3.4_

  - [x] 6.5 实现 `PollTaskStatus` 方法
    - 实现轮询循环逻辑
    - 初始间隔 2 秒，指数退避，最大 10 秒
    - 最大轮询次数 60 次
    - 处理 SUCCEED、FAILED、PENDING、RUNNING 状态
    - 支持 context 取消
    - _Requirements: 3.3, 3.5, 3.6, 3.7, 6.5_

  - [ ]* 6.6 编写 PollTaskStatus 的属性测试
    - **Property 2: 异步任务提交返回有效 task_id**
    - **Property 3: 任务轮询最终收敛**
    - **Validates: Requirements 2.4, 3.3, 3.7**

- [ ] 7. 实现请求和响应处理
  - [x] 7.1 实现 `DoRequest` 方法
    - 调用 submitTask 提交任务
    - 调用 PollTaskStatus 轮询状态
    - 返回最终任务结果
    - _Requirements: 2.4, 3.1_

  - [x] 7.2 实现 `DoResponse` 方法
    - 将 ModelScope 响应转换为 OpenAI ImageResponse 格式
    - 提取 output_images 中的 URL
    - 支持 b64_json 格式（下载图片并编码）
    - 设置 created 时间戳
    - 处理多张图片
    - _Requirements: 10.1, 10.2, 10.3, 10.4, 10.5_

  - [ ]* 7.3 编写 DoResponse 的属性测试
    - **Property 6: 响应格式转换保持图片 URL**
    - **Validates: Requirements 10.1**

- [ ] 8. 实现错误处理
  - [ ] 8.1 实现 `handleSubmitError` 函数
    - 识别认证错误
    - 识别速率限制错误
    - 识别网络错误
    - 返回适当的错误码和消息
    - _Requirements: 6.1, 6.2, 6.3_

  - [ ] 8.2 实现 `handlePollError` 函数
    - 处理轮询超时
    - 处理任务失败
    - 处理网络错误
    - _Requirements: 6.4, 6.5_

  - [ ]* 8.3 编写错误处理的单元测试
    - 测试各种错误场景
    - 验证错误消息正确性
    - _Requirements: 6.1, 6.2, 6.3, 6.4_

  - [ ]* 8.4 编写错误处理的属性测试
    - **Property 7: 错误响应正确传播**
    - **Validates: Requirements 6.1**

- [ ] 9. 实现适配器辅助方法
  - [x] 9.1 实现 `GetModelList` 方法
    - 返回支持的模型列表
    - _Requirements: 5.1_

  - [x] 9.2 实现 `GetChannelName` 方法
    - 返回 "ModelScope"
    - _Requirements: 1.2_

  - [x] 9.3 实现未使用接口的存根方法
    - ConvertOpenAIRequest: 返回 not implemented 错误
    - ConvertRerankRequest: 返回 not implemented 错误
    - ConvertEmbeddingRequest: 返回 not implemented 错误
    - ConvertAudioRequest: 返回 not implemented 错误
    - ConvertOpenAIResponsesRequest: 返回 not implemented 错误
    - ConvertClaudeRequest: 返回 not implemented 错误
    - ConvertGeminiRequest: 返回 not implemented 错误
    - _Requirements: 2.1_

- [ ] 10. 注册 ModelScope 适配器
  - 在 `relay/relay_adaptor.go` 的 `GetAdaptor` 函数中添加 ModelScope case
  - 导入 modelscope 包
  - _Requirements: 4.1, 4.2_

- [ ] 11. Checkpoint - 基础功能验证
  - 确保所有代码编译通过
  - 运行单元测试验证基础功能
  - 询问用户是否有问题

- [ ] 12. 集成测试和验证
  - [ ]* 12.1 编写端到端集成测试
    - 测试完整的图片生成流程
    - 测试 LoRA 配置
    - 测试错误场景
    - _Requirements: 4.3, 4.4, 4.5_

  - [ ]* 12.2 编写向后兼容性测试
    - **Property 8: 现有渠道不受影响**
    - 验证所有现有渠道仍然正常工作
    - 验证现有测试仍然通过
    - **Validates: Requirements 11.1, 11.2, 11.10**

  - [ ]* 12.3 编写配额计算测试
    - **Property 9: 配额计算一致性**
    - 验证 ModelScope 使用现有配额逻辑
    - **Validates: Requirements 4.4**

  - [ ]* 12.4 编写数据库记录测试
    - **Property 10: 数据库记录完整性**
    - 验证任务记录正确保存
    - 验证状态更新正确
    - **Validates: Requirements 7.1, 7.2, 7.3, 7.4, 7.5, 7.6**

- [ ] 13. 文档更新
  - [ ] 13.1 更新渠道配置文档
    - 在 `docs/channel/` 目录添加 ModelScope 配置说明
    - 说明如何获取 API Key
    - 说明支持的模型列表
    - 提供配置示例
    - _Requirements: 8.1, 8.2, 8.3, 8.4_

  - [ ] 13.2 更新 README
    - 在支持的渠道列表中添加 ModelScope
    - _Requirements: 8.1_

- [ ] 14. Docker 构建和部署准备
  - [ ] 14.1 验证 Dockerfile 无需修改
    - 确认新代码会自动包含在构建中
    - _Requirements: 11.13_

  - [ ] 14.2 构建 Docker 镜像
    - 运行 `docker build -t new-api:modelscope .`
    - 验证镜像构建成功
    - _Requirements: 11.13_

  - [ ] 14.3 测试 Docker 镜像
    - 运行容器
    - 验证 ModelScope 渠道可用
    - 测试图片生成功能
    - _Requirements: 11.13_

  - [ ] 14.4 创建 docker-compose 配置示例
    - 提供完整的部署配置
    - 包含环境变量说明
    - _Requirements: 11.13_

- [ ] 15. 最终验证和清理
  - [ ] 15.1 运行完整测试套件
    - 运行所有单元测试
    - 运行所有属性测试
    - 运行所有集成测试
    - 确保所有测试通过
    - _Requirements: 11.9_

  - [ ] 15.2 代码审查检查清单
    - 确认没有修改现有渠道代码
    - 确认没有修改核心处理函数
    - 确认没有修改数据库结构
    - 确认所有新代码在独立包中
    - _Requirements: 11.1, 11.2, 11.3, 11.4, 11.11_

  - [ ] 15.3 性能测试
    - 测试并发请求处理
    - 验证现有渠道性能未降低
    - _Requirements: 11.10_

  - [ ] 15.4 准备发布说明
    - 列出新增功能
    - 说明配置方法
    - 提供使用示例
    - _Requirements: 8.1_

- [ ] 16. Final Checkpoint - 完成验证
  - 确保所有任务完成
  - 确保所有测试通过
  - 确保 Docker 镜像可用
  - 询问用户是否准备发布

## Notes

- 任务标记 `*` 的为可选测试任务，可以根据需要跳过以加快 MVP 开发
- 每个任务都引用了具体的需求编号，确保可追溯性
- 实施过程中严格遵循"不修改现有代码"的原则
- 所有 ModelScope 相关代码都在 `relay/channel/modelscope/` 包中
- 使用现有的接口和扩展点进行集成
- Docker 构建无需特殊配置，新代码会自动包含
