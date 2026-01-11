package modelscope

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/relay/channel"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/service"
	"github.com/QuantumNous/new-api/types"

	"github.com/gin-gonic/gin"
)

type Adaptor struct{}

func (a *Adaptor) Init(info *relaycommon.RelayInfo) {
	// 初始化适配器，当前无需特殊初始化
}

func (a *Adaptor) GetRequestURL(info *relaycommon.RelayInfo) (string, error) {
	if info == nil {
		return "", errors.New("modelscope adaptor: relay info is nil")
	}
	if info.ChannelBaseUrl == "" {
		info.ChannelBaseUrl = constant.ChannelBaseURLs[constant.ChannelTypeModelScope]
	}
	// 任务提交使用 /v1/images/generations
	return relaycommon.GetFullRequestURL(info.ChannelBaseUrl, "/v1/images/generations", info.ChannelType), nil
}

func (a *Adaptor) SetupRequestHeader(c *gin.Context, req *http.Header, info *relaycommon.RelayInfo) error {
	if info == nil {
		return errors.New("modelscope adaptor: relay info is nil")
	}
	if info.ApiKey == "" {
		return errors.New("modelscope adaptor: api key is required")
	}
	channel.SetupApiRequestHeader(info, c, req)
	req.Set("Authorization", "Bearer "+info.ApiKey)
	req.Set("Content-Type", "application/json")
	// 设置异步模式头
	req.Set("X-ModelScope-Async-Mode", "true")
	return nil
}

func (a *Adaptor) ConvertImageRequest(c *gin.Context, info *relaycommon.RelayInfo, request dto.ImageRequest) (any, error) {
	if info == nil {
		return nil, errors.New("modelscope adaptor: relay info is nil")
	}
	if strings.TrimSpace(request.Prompt) == "" {
		return nil, errors.New("modelscope adaptor: prompt is required")
	}

	modelName := strings.TrimSpace(info.UpstreamModelName)
	if modelName == "" {
		modelName = strings.TrimSpace(request.Model)
	}
	if modelName == "" {
		modelName = "Tongyi-MAI/Z-Image-Turbo" // 默认模型
	}
	info.UpstreamModelName = modelName

	msRequest := &TaskSubmitRequest{
		Model:  modelName,
		Prompt: request.Prompt,
	}

	// 传递 size 参数
	if request.Size != "" {
		msRequest.Size = request.Size
	}

	// 传递生成数量
	if request.N > 0 {
		msRequest.N = int(request.N)
	}

	// 处理 LoRA 配置
	if len(request.Extra) > 0 {
		if lorasRaw, ok := request.Extra["loras"]; ok {
			var loras interface{}
			if err := common.Unmarshal(lorasRaw, &loras); err == nil {
				// 验证 LoRA 配置
				if err := validateLoRAs(loras); err != nil {
					return nil, fmt.Errorf("modelscope adaptor: %w", err)
				}
				msRequest.Loras = loras
			}
		}
	}

	return msRequest, nil
}

// validateLoRAs 验证 LoRA 配置
func validateLoRAs(loras interface{}) error {
	switch v := loras.(type) {
	case string:
		// 单个 LoRA，格式正确
		return nil
	case map[string]interface{}:
		// 多个 LoRA，需要验证数量和权重
		if len(v) > MaxLoRACount {
			return fmt.Errorf("too many LoRAs: maximum %d allowed, got %d", MaxLoRACount, len(v))
		}
		// 验证权重总和
		totalWeight := 0.0
		for _, weight := range v {
			if w, ok := weight.(float64); ok {
				totalWeight += w
			} else {
				return fmt.Errorf("invalid LoRA weight type: expected float64")
			}
		}
		// 允许 ±0.001 的误差
		if totalWeight < 0.999 || totalWeight > 1.001 {
			return fmt.Errorf("LoRA weights must sum to 1.0, got %.3f", totalWeight)
		}
		return nil
	default:
		return fmt.Errorf("invalid LoRA configuration type")
	}
}

func (a *Adaptor) DoRequest(c *gin.Context, info *relaycommon.RelayInfo, requestBody io.Reader) (any, error) {
	return channel.DoApiRequest(a, c, info, requestBody)
}

func (a *Adaptor) DoResponse(c *gin.Context, resp *http.Response, info *relaycommon.RelayInfo) (any, *types.NewAPIError) {
	if resp == nil {
		return nil, types.NewError(errors.New("modelscope adaptor: empty response"), types.ErrorCodeBadResponse)
	}

	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, types.NewError(err, types.ErrorCodeReadResponseBodyFailed)
	}
	_ = resp.Body.Close()

	// 解析任务提交响应
	var submitResp TaskSubmitResponse
	if err := common.Unmarshal(responseBody, &submitResp); err != nil {
		return nil, types.NewError(fmt.Errorf("modelscope adaptor: failed to decode submit response: %w", err), types.ErrorCodeBadResponseBody)
	}

	if submitResp.TaskID == "" {
		return nil, types.NewError(errors.New("modelscope adaptor: empty task_id in response"), types.ErrorCodeBadResponseBody)
	}

	// 轮询任务状态
	taskResp, err := a.PollTaskStatus(c.Request.Context(), submitResp.TaskID, info)
	if err != nil {
		return nil, types.NewError(err, types.ErrorCodeBadResponse)
	}

	// 转换为 OpenAI 格式
	return a.convertToOpenAIResponse(c, taskResp, info)
}

func (a *Adaptor) convertToOpenAIResponse(c *gin.Context, taskResp *TaskQueryResponse, info *relaycommon.RelayInfo) (any, *types.NewAPIError) {
	if len(taskResp.OutputImages) == 0 {
		return nil, types.NewError(errors.New("modelscope adaptor: no output images"), types.ErrorCodeBadResponseBody)
	}

	var imageReq *dto.ImageRequest
	if info != nil {
		if req, ok := info.Request.(*dto.ImageRequest); ok {
			imageReq = req
		}
	}

	wantsBase64 := imageReq != nil && strings.EqualFold(imageReq.ResponseFormat, "b64_json")

	imageResponse := dto.ImageResponse{
		Created: common.GetTimestamp(),
		Data:    make([]dto.ImageData, 0),
	}

	if wantsBase64 {
		// 下载图片并转换为 base64
		for _, url := range taskResp.OutputImages {
			if strings.TrimSpace(url) == "" {
				continue
			}
			_, data, err := service.GetImageFromUrl(url)
			if err != nil {
				return nil, types.NewError(fmt.Errorf("modelscope adaptor: failed to download image from %s: %w", url, err), types.ErrorCodeBadResponse)
			}
			imageResponse.Data = append(imageResponse.Data, dto.ImageData{B64Json: data})
		}
	} else {
		// 直接返回 URL
		for _, url := range taskResp.OutputImages {
			if strings.TrimSpace(url) == "" {
				continue
			}
			imageResponse.Data = append(imageResponse.Data, dto.ImageData{Url: url})
		}
	}

	if len(imageResponse.Data) == 0 {
		return nil, types.NewError(errors.New("modelscope adaptor: no usable image data"), types.ErrorCodeBadResponse)
	}

	responseBytes, err := common.Marshal(imageResponse)
	if err != nil {
		return nil, types.NewError(fmt.Errorf("modelscope adaptor: encode response failed: %w", err), types.ErrorCodeBadResponseBody)
	}

	c.Writer.Header().Set("Content-Type", "application/json")
	c.Writer.WriteHeader(http.StatusOK)
	_, _ = c.Writer.Write(responseBytes)

	usage := &dto.Usage{}
	return usage, nil
}

func (a *Adaptor) GetModelList() []string {
	return ModelList
}

func (a *Adaptor) GetChannelName() string {
	return ChannelName
}

// 未使用的接口方法存根
func (a *Adaptor) ConvertOpenAIRequest(*gin.Context, *relaycommon.RelayInfo, *dto.GeneralOpenAIRequest) (any, error) {
	return nil, errors.New("modelscope adaptor: only supports image generation (/v1/images/generations), not chat completions")
}

func (a *Adaptor) ConvertRerankRequest(*gin.Context, int, dto.RerankRequest) (any, error) {
	return nil, errors.New("modelscope adaptor: ConvertRerankRequest is not implemented")
}

func (a *Adaptor) ConvertEmbeddingRequest(*gin.Context, *relaycommon.RelayInfo, dto.EmbeddingRequest) (any, error) {
	return nil, errors.New("modelscope adaptor: ConvertEmbeddingRequest is not implemented")
}

func (a *Adaptor) ConvertAudioRequest(*gin.Context, *relaycommon.RelayInfo, dto.AudioRequest) (io.Reader, error) {
	return nil, errors.New("modelscope adaptor: ConvertAudioRequest is not implemented")
}

func (a *Adaptor) ConvertOpenAIResponsesRequest(*gin.Context, *relaycommon.RelayInfo, dto.OpenAIResponsesRequest) (any, error) {
	return nil, errors.New("modelscope adaptor: ConvertOpenAIResponsesRequest is not implemented")
}

func (a *Adaptor) ConvertClaudeRequest(*gin.Context, *relaycommon.RelayInfo, *dto.ClaudeRequest) (any, error) {
	return nil, errors.New("modelscope adaptor: ConvertClaudeRequest is not implemented")
}

func (a *Adaptor) ConvertGeminiRequest(*gin.Context, *relaycommon.RelayInfo, *dto.GeminiChatRequest) (any, error) {
	return nil, errors.New("modelscope adaptor: ConvertGeminiRequest is not implemented")
}
