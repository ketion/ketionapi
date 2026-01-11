package modelscope

// TaskSubmitRequest ModelScope 任务提交请求
type TaskSubmitRequest struct {
	Model  string      `json:"model"`
	Prompt string      `json:"prompt"`
	Size   string      `json:"size,omitempty"`   // 图片尺寸，如 "1024x1024"
	N      int         `json:"n,omitempty"`      // 生成图片数量
	Loras  interface{} `json:"loras,omitempty"`  // LoRA 配置
}

// TaskSubmitResponse ModelScope 任务提交响应
type TaskSubmitResponse struct {
	TaskID    string `json:"task_id"`
	RequestID string `json:"request_id"`
}

// TaskQueryResponse ModelScope 任务查询响应
type TaskQueryResponse struct {
	TaskID       string   `json:"task_id"`
	TaskStatus   string   `json:"task_status"` // SUCCEED, FAILED, PENDING, RUNNING
	OutputImages []string `json:"output_images,omitempty"`
	Message      string   `json:"message,omitempty"`
}
