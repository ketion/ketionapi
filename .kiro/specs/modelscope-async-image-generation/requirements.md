# Requirements Document

## Introduction

本需求文档描述为 NewAPI 项目添加魔塔（ModelScope）平台异步文生图模型支持的功能。魔塔平台的文生图 API 采用异步模式：首先提交任务获取 task_id，然后通过轮询方式获取生成结果。

**核心原则：完全向后兼容，零影响现有功能**

该功能作为新增渠道类型，采用插件式架构集成到现有系统中，不修改任何现有渠道的代码逻辑，确保：
- 现有渠道（OpenAI、Replicate 等）的功能和行为完全不受影响
- 现有 API 端点和路由逻辑保持不变
- 现有数据库结构和模型无需修改
- 现有配置和管理界面保持兼容

## Glossary

- **ModelScope**: 魔塔平台，阿里云提供的 AI 模型服务平台
- **NewAPI**: 本项目，一个 AI API 聚合网关系统
- **Adaptor**: 适配器，用于适配不同 AI 服务提供商的接口
- **Channel**: 渠道，代表一个 AI 服务提供商的配置实例
- **Task**: 任务，异步操作的执行单元
- **Relay**: 中继，请求转发和处理的核心模块
- **ChannelType**: 渠道类型，标识不同的 AI 服务提供商
- **RelayMode**: 中继模式，标识不同的 API 操作类型

## Requirements

### Requirement 1: 添加 ModelScope 渠道类型（零影响扩展）

**User Story:** 作为系统管理员，我希望在渠道类型列表中看到 ModelScope 选项，以便我可以配置魔塔平台的接入，同时不影响现有渠道的使用。

#### Acceptance Criteria

1. THE System SHALL define a new channel type constant for ModelScope without modifying existing channel type constants
2. THE System SHALL register ModelScope in the channel type names mapping as a new entry
3. THE System SHALL provide the default base URL for ModelScope API (https://api-inference.modelscope.cn/) in the ChannelBaseURLs array
4. WHEN the system initializes THEN the ModelScope channel type SHALL be available in the channel type list
5. THE System SHALL NOT modify any existing channel type definitions or behaviors
6. WHEN existing channels are used THEN they SHALL function exactly as before without any changes

### Requirement 2: 实现 ModelScope 独立适配器（隔离实现）

**User Story:** 作为开发者，我希望实现 ModelScope 的独立适配器，以便系统能够正确处理魔塔平台的异步文生图请求，同时不影响其他适配器的实现。

#### Acceptance Criteria

1. THE ModelScope_Adaptor SHALL be implemented in a separate package (relay/channel/modelscope)
2. THE ModelScope_Adaptor SHALL implement the standard adaptor interface without modifying the interface definition
3. WHEN converting an image request THEN the adaptor SHALL transform OpenAI format to ModelScope format
4. THE ModelScope_Adaptor SHALL support the X-ModelScope-Async-Mode header with value "true"
5. WHEN submitting a request THEN the adaptor SHALL extract and return the task_id from the response
6. THE ModelScope_Adaptor SHALL support model parameter mapping from OpenAI format to ModelScope model IDs
7. THE ModelScope_Adaptor SHALL support prompt parameter conversion
8. WHERE LoRA models are specified THEN the adaptor SHALL include them in the request payload
9. THE System SHALL NOT modify any existing adaptor implementations (OpenAI, Replicate, etc.)
10. THE ModelScope_Adaptor SHALL handle all ModelScope-specific logic internally without affecting shared code

### Requirement 3: 实现任务状态轮询机制

**User Story:** 作为系统，我需要轮询魔塔平台获取任务执行结果，以便在任务完成后返回生成的图片。

#### Acceptance Criteria

1. THE System SHALL implement a task polling mechanism for ModelScope
2. WHEN polling a task THEN the system SHALL use the X-ModelScope-Task-Type header with value "image_generation"
3. THE System SHALL check task status and handle SUCCEED, FAILED, and in-progress states
4. WHEN task status is SUCCEED THEN the system SHALL extract image URLs from output_images field
5. WHEN task status is FAILED THEN the system SHALL return an appropriate error message
6. WHILE task is in progress THEN the system SHALL continue polling with appropriate intervals
7. THE System SHALL implement a maximum polling timeout to prevent infinite loops

### Requirement 4: 集成到现有图片生成处理流程（无侵入式集成）

**User Story:** 作为系统架构师，我希望 ModelScope 适配器能够无缝集成到现有的图片生成处理流程中，同时保持现有代码的完整性和稳定性。

#### Acceptance Criteria

1. THE System SHALL register the ModelScope adaptor in the adaptor factory using existing registration mechanism
2. WHEN the relay mode is image generation AND channel type is ModelScope THEN the system SHALL route to ModelScope adaptor
3. WHEN the relay mode is image generation AND channel type is NOT ModelScope THEN the system SHALL route to the original adaptor without any changes
4. THE System SHALL handle ModelScope requests through the standard /v1/images/generations endpoint
5. THE System SHALL apply quota calculation and consumption for ModelScope requests using existing quota logic
6. THE System SHALL log ModelScope operations with appropriate metadata using existing logging mechanism
7. THE System SHALL support parameter override functionality for ModelScope channels using existing override mechanism
8. THE System SHALL NOT modify the ImageHelper function or any existing image processing logic
9. THE System SHALL NOT modify routing logic for existing channels
10. WHEN existing channels process requests THEN they SHALL use their original code paths without any ModelScope-related checks

### Requirement 5: 支持 ModelScope 特定参数

**User Story:** 作为 API 用户，我希望能够使用 ModelScope 特定的参数，以便充分利用魔塔平台的功能。

#### Acceptance Criteria

1. THE System SHALL support the model parameter for specifying ModelScope model IDs
2. THE System SHALL support the prompt parameter for image generation
3. WHERE LoRA models are specified THEN the system SHALL support single and multiple LoRA configurations
4. THE System SHALL validate that multiple LoRA weight coefficients sum to 1.0
5. THE System SHALL support up to 6 LoRA models per request
6. THE System SHALL pass through additional ModelScope-specific parameters in the request body

### Requirement 6: 实现错误处理和重试机制

**User Story:** 作为系统运维人员，我希望系统能够妥善处理 ModelScope API 的各种错误情况，以便提高系统的稳定性和可靠性。

#### Acceptance Criteria

1. WHEN the ModelScope API returns an error THEN the system SHALL parse and return a meaningful error message
2. WHEN network errors occur THEN the system SHALL handle them gracefully
3. WHEN task submission fails THEN the system SHALL return an appropriate error response
4. WHEN polling timeout occurs THEN the system SHALL return a timeout error
5. THE System SHALL implement exponential backoff for polling retries
6. THE System SHALL respect rate limits from ModelScope API

### Requirement 7: 数据库任务记录（复用现有结构）

**User Story:** 作为系统管理员，我希望系统能够记录 ModelScope 的任务信息，以便追踪任务状态和进行问题排查，同时不需要修改现有数据库结构。

#### Acceptance Criteria

1. WHEN a ModelScope task is submitted THEN the system SHALL create a task record using the existing Task model
2. THE Task_Record SHALL include task_id, user_id, channel_id, model_name, and status using existing fields
3. THE Task_Record SHALL store the original request data using existing data field
4. WHEN task status changes THEN the system SHALL update the task record using existing update methods
5. THE System SHALL support querying task status by task_id using existing query methods
6. THE System SHALL store the final image URLs when task succeeds using existing fields
7. THE System SHALL NOT add new database tables or modify existing table schemas
8. THE System SHALL NOT modify existing task-related database operations for other channels

### Requirement 8: 管理后台配置界面支持

**User Story:** 作为系统管理员，我希望在管理后台能够配置 ModelScope 渠道，以便轻松管理魔塔平台的接入。

#### Acceptance Criteria

1. WHEN creating a new channel THEN the system SHALL display ModelScope as a channel type option
2. THE Channel_Configuration_Form SHALL include fields for API key (ModelScope Token)
3. THE Channel_Configuration_Form SHALL include fields for base URL with default value
4. THE Channel_Configuration_Form SHALL support model list configuration
5. THE System SHALL validate the ModelScope API key format
6. THE System SHALL allow testing the ModelScope channel connection

### Requirement 9: API 端点路由（保持现有路由逻辑）

**User Story:** 作为 API 用户，我希望通过标准的 /v1/images/generations 端点访问 ModelScope 文生图功能，同时不影响现有端点的行为。

#### Acceptance Criteria

1. THE System SHALL route /v1/images/generations requests to the appropriate channel adaptor based on channel type
2. WHEN the selected channel is ModelScope THEN the system SHALL use the ModelScope adaptor
3. WHEN the selected channel is NOT ModelScope THEN the system SHALL use the original adaptor without any changes
4. THE System SHALL maintain compatibility with OpenAI image generation API format
5. THE System SHALL return responses in OpenAI-compatible format
6. THE System SHALL support both synchronous-style API calls with internal async handling for ModelScope
7. THE System SHALL NOT modify existing routing logic or middleware
8. THE System SHALL NOT add new API endpoints
9. WHEN existing channels handle requests THEN they SHALL process them exactly as before

### Requirement 10: 图片格式和响应转换

**User Story:** 作为 API 用户，我希望系统能够将 ModelScope 的响应转换为标准格式，以便我的客户端代码无需修改。

#### Acceptance Criteria

1. WHEN ModelScope returns image URLs THEN the system SHALL convert them to OpenAI response format
2. THE System SHALL support both URL and b64_json response formats
3. WHEN response_format is b64_json THEN the system SHALL download images and encode them
4. THE System SHALL include created timestamp in the response
5. THE System SHALL handle multiple images in the response
6. THE System SHALL preserve image metadata where applicable


### Requirement 11: 向后兼容性保证

**User Story:** 作为系统维护者，我希望确保添加 ModelScope 支持后，所有现有功能都能正常工作，不会引入任何回归问题。

#### Acceptance Criteria

1. THE System SHALL NOT modify any existing channel adaptor code (OpenAI, Replicate, Azure, etc.)
2. THE System SHALL NOT modify the core ImageHelper function logic
3. THE System SHALL NOT modify the RelayTaskSubmit function logic
4. THE System SHALL NOT modify existing database models or schemas
5. THE System SHALL NOT modify existing API routing or middleware
6. THE System SHALL NOT modify existing error handling logic for other channels
7. THE System SHALL NOT modify existing quota calculation logic
8. THE System SHALL NOT modify existing logging mechanisms
9. WHEN running existing test suites THEN all tests SHALL pass without modifications
10. WHEN existing channels are used THEN their performance SHALL not be degraded
11. THE ModelScope implementation SHALL be completely isolated in its own package
12. THE System SHALL use existing interfaces and extension points only
13. WHEN ModelScope channel is disabled or removed THEN the system SHALL function exactly as before
