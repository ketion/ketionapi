package modelscope

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/QuantumNous/new-api/common"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/service"
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
			if resp.Message != "" {
				return nil, fmt.Errorf("task failed: %s", resp.Message)
			}
			return nil, fmt.Errorf("task failed without message")
		case TaskStatusPending, TaskStatusRunning, TaskStatusProcessing:
			// 继续轮询，使用指数退避
			time.Sleep(time.Duration(interval) * time.Second)
			interval = min(interval*2, MaxPollInterval)
		default:
			return nil, fmt.Errorf("unknown task status: %s", resp.TaskStatus)
		}
	}

	return nil, fmt.Errorf("task polling timeout after %d attempts", MaxPollAttempts)
}

// queryTaskStatus 查询任务状态
func (a *Adaptor) queryTaskStatus(taskID string, info *relaycommon.RelayInfo) (*TaskQueryResponse, error) {
	baseURL := info.ChannelBaseUrl
	if baseURL == "" {
		baseURL = "https://api-inference.modelscope.cn"
	}

	// 构建查询 URL
	queryURL := fmt.Sprintf("%s/v1/tasks/%s", baseURL, taskID)

	// 创建请求
	req, err := http.NewRequest(http.MethodGet, queryURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create query request: %w", err)
	}

	// 设置请求头
	req.Header.Set("Authorization", "Bearer "+info.ApiKey)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-ModelScope-Task-Type", "image_generation")

	// 发送请求
	resp, err := service.GetHttpClient().Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to query task status: %w", err)
	}
	defer resp.Body.Close()

	// 读取响应
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read query response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("query task failed with status %d: %s", resp.StatusCode, string(respBody))
	}

	// 解析响应
	var taskResp TaskQueryResponse
	if err := common.Unmarshal(respBody, &taskResp); err != nil {
		return nil, fmt.Errorf("failed to decode query response: %w", err)
	}

	return &taskResp, nil
}

// min 返回两个整数中的较小值
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
