package modelscope

const (
	ChannelName = "ModelScope"

	// 任务状态
	TaskStatusSucceed    = "SUCCEED"
	TaskStatusFailed     = "FAILED"
	TaskStatusPending    = "PENDING"
	TaskStatusRunning    = "RUNNING"
	TaskStatusProcessing = "PROCESSING" // ModelScope 实际使用的状态

	// 轮询配置
	MaxPollAttempts     = 60 // 最大轮询次数
	PollInterval        = 5  // 轮询间隔（秒）
	InitialPollInterval = 2  // 初始轮询间隔（秒）
	MaxPollInterval     = 10 // 最大轮询间隔（秒）

	// LoRA 配置限制
	MaxLoRACount = 6 // 最大 LoRA 数量
)

var ModelList = []string{
	"Tongyi-MAI/Z-Image-Turbo",
	"modelscope/stable-diffusion-v1-5",
	"modelscope/stable-diffusion-xl-base-1.0",
	"AI-ModelScope/stable-diffusion-v1-4",
}
