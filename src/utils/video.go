package utils

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// VideoStandard 视频标准配置
type VideoStandard struct {
	VideoCodec      string  // H.264
	PixelFormat     string  // yuv420p
	MaxWidth        int     // 1920
	MaxHeight       int     // 1080
	AudioCodec      string  // AAC
	MaxChannels     int     // 2
	MaxSampleRate   int     // 44100
	MaxVideoBitrate int     // 1500 kbps
	MaxFrameRate    float64 // 25 fps
}

// DefaultVideoStandard 默认视频标准
var DefaultVideoStandard = VideoStandard{
	VideoCodec:      "h264",
	PixelFormat:     "yuv420p",
	MaxWidth:        1920,
	MaxHeight:       1080,
	AudioCodec:      "aac",
	MaxChannels:     2,
	MaxSampleRate:   44100,
	MaxVideoBitrate: 1500000, // 1500 kbps in bps
	MaxFrameRate:    25.0,
}

// VideoInfo 视频信息
type VideoInfo struct {
	VideoCodec   string
	PixelFormat  string
	Width        int
	Height       int
	AudioCodec   string
	Channels     int
	SampleRate   int
	VideoBitrate int
	FrameRate    float64
}

// VideoValidationResult 视频校验结果
type VideoValidationResult struct {
	Valid   bool       `json:"valid"`
	Message string     `json:"message"`
	Errors  []string   `json:"errors,omitempty"`
	Info    *VideoInfo `json:"info,omitempty"`
}

// FFProbeFormat ffprobe format 输出结构
type FFProbeFormat struct {
	BitRate string `json:"bit_rate"`
}

// FFProbeStream ffprobe stream 输出结构
type FFProbeStream struct {
	CodecName    string `json:"codec_name"`
	CodecType    string `json:"codec_type"`
	PixFmt       string `json:"pix_fmt"`
	Width        int    `json:"width"`
	Height       int    `json:"height"`
	Channels     int    `json:"channels"`
	SampleRate   string `json:"sample_rate"`
	BitRate      string `json:"bit_rate"`
	RFrameRate   string `json:"r_frame_rate"`
	AvgFrameRate string `json:"avg_frame_rate"`
}

// FFProbeOutput ffprobe 完整输出结构
type FFProbeOutput struct {
	Streams []FFProbeStream `json:"streams"`
	Format  FFProbeFormat   `json:"format"`
}

// ValidateVideoURL 校验视频链接是否符合标准
func ValidateVideoURL(videoURL string) *VideoValidationResult {
	return ValidateVideoWithStandard(videoURL, DefaultVideoStandard)
}

// ValidateVideoWithStandard 使用自定义标准校验视频
func ValidateVideoWithStandard(videoURL string, standard VideoStandard) *VideoValidationResult {
	result := &VideoValidationResult{
		Valid:  true,
		Errors: []string{},
	}

	// 获取视频信息
	videoInfo, err := GetVideoInfo(videoURL)
	if err != nil {
		result.Valid = false
		result.Message = fmt.Sprintf("获取视频信息失败: %v", err)
		return result
	}

	result.Info = videoInfo

	// 校验视频编码
	if !strings.EqualFold(videoInfo.VideoCodec, standard.VideoCodec) {
		result.Valid = false
		result.Errors = append(result.Errors, fmt.Sprintf("视频编码不符合要求: 当前为 %s, 要求为 %s", videoInfo.VideoCodec, standard.VideoCodec))
	}

	// 校验像素格式
	if videoInfo.PixelFormat != "" && videoInfo.PixelFormat != standard.PixelFormat {
		result.Valid = false
		result.Errors = append(result.Errors, fmt.Sprintf("像素格式不符合要求: 当前为 %s, 要求为 %s", videoInfo.PixelFormat, standard.PixelFormat))
	}

	// 校验分辨率
	if videoInfo.Width > standard.MaxWidth || videoInfo.Height > standard.MaxHeight {
		result.Valid = false
		result.Errors = append(result.Errors, fmt.Sprintf("分辨率超过限制: 当前为 %dx%d, 最大为 %dx%d", videoInfo.Width, videoInfo.Height, standard.MaxWidth, standard.MaxHeight))
	}

	// 校验音频编码
	if videoInfo.AudioCodec != "" && !strings.EqualFold(videoInfo.AudioCodec, standard.AudioCodec) {
		result.Valid = false
		result.Errors = append(result.Errors, fmt.Sprintf("音频编码不符合要求: 当前为 %s, 要求为 %s", videoInfo.AudioCodec, standard.AudioCodec))
	}

	// 校验音频声道
	if videoInfo.Channels > standard.MaxChannels {
		result.Valid = false
		result.Errors = append(result.Errors, fmt.Sprintf("音频声道超过限制: 当前为 %d 声道, 最大为 %d 声道", videoInfo.Channels, standard.MaxChannels))
	}

	// 校验采样率
	if videoInfo.SampleRate > standard.MaxSampleRate {
		result.Valid = false
		result.Errors = append(result.Errors, fmt.Sprintf("采样率超过限制: 当前为 %d Hz, 最大为 %d Hz", videoInfo.SampleRate, standard.MaxSampleRate))
	}

	// 校验视频码率
	if videoInfo.VideoBitrate > standard.MaxVideoBitrate {
		result.Valid = false
		result.Errors = append(result.Errors, fmt.Sprintf("视频码率超过限制: 当前为 %d kbps, 最大为 %d kbps", videoInfo.VideoBitrate/1000, standard.MaxVideoBitrate/1000))
	}

	// 校验帧率
	if videoInfo.FrameRate > standard.MaxFrameRate {
		result.Valid = false
		result.Errors = append(result.Errors, fmt.Sprintf("帧率超过限制: 当前为 %.2f fps, 最大为 %.2f fps", videoInfo.FrameRate, standard.MaxFrameRate))
	}

	if result.Valid {
		result.Message = "视频符合所有标准"
	} else {
		result.Message = fmt.Sprintf("视频不符合标准，共有 %d 项不合格", len(result.Errors))
	}

	return result
}

// GetVideoInfo 获取视频信息
func GetVideoInfo(videoURL string) (*VideoInfo, error) {
	// 使用 ffprobe 获取视频信息
	cmd := exec.Command("ffprobe",
		"-v", "quiet",
		"-print_format", "json",
		"-show_streams",
		"-show_format",
		videoURL,
	)

	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("执行 ffprobe 失败: %v", err)
	}

	var probeOutput FFProbeOutput
	if err := json.Unmarshal(output, &probeOutput); err != nil {
		return nil, fmt.Errorf("解析 ffprobe 输出失败: %v", err)
	}

	videoInfo := &VideoInfo{}

	// 解析视频流和音频流信息
	for _, stream := range probeOutput.Streams {
		switch stream.CodecType {
		case "video":
			videoInfo.VideoCodec = stream.CodecName
			videoInfo.PixelFormat = stream.PixFmt
			videoInfo.Width = stream.Width
			videoInfo.Height = stream.Height

			// 解析帧率
			if stream.RFrameRate != "" {
				videoInfo.FrameRate = parseFrameRate(stream.RFrameRate)
			} else if stream.AvgFrameRate != "" {
				videoInfo.FrameRate = parseFrameRate(stream.AvgFrameRate)
			}

			// 解析视频码率
			if stream.BitRate != "" {
				if bitrate, err := strconv.Atoi(stream.BitRate); err == nil {
					videoInfo.VideoBitrate = bitrate
				}
			}

		case "audio":
			videoInfo.AudioCodec = stream.CodecName
			videoInfo.Channels = stream.Channels

			// 解析采样率
			if stream.SampleRate != "" {
				if sampleRate, err := strconv.Atoi(stream.SampleRate); err == nil {
					videoInfo.SampleRate = sampleRate
				}
			}
		}
	}

	// 如果视频流中没有码率信息，尝试从 format 中获取
	if videoInfo.VideoBitrate == 0 && probeOutput.Format.BitRate != "" {
		if bitrate, err := strconv.Atoi(probeOutput.Format.BitRate); err == nil {
			videoInfo.VideoBitrate = bitrate
		}
	}

	return videoInfo, nil
}

// parseFrameRate 解析帧率字符串 (例如 "25/1" -> 25.0)
func parseFrameRate(rateStr string) float64 {
	parts := strings.Split(rateStr, "/")
	if len(parts) != 2 {
		return 0
	}

	numerator, err1 := strconv.ParseFloat(parts[0], 64)
	denominator, err2 := strconv.ParseFloat(parts[1], 64)

	if err1 != nil || err2 != nil || denominator == 0 {
		return 0
	}

	return numerator / denominator
}

// ProcessVideoResult 视频处理结果
type ProcessVideoResult struct {
	Success      bool     `json:"success"`
	Message      string   `json:"message"`
	OutputPath   string   `json:"outputPath,omitempty"`
	NeedProcess  bool     `json:"needProcess"`            // 是否需要处理
	ProcessedOps []string `json:"processedOps,omitempty"` // 执行的处理操作
}

// ProcessVideoToStandard 处理视频使其符合标准
// 返回处理后的临时文件路径，调用者需要负责清理临时文件
func ProcessVideoToStandard(videoURL string) (*ProcessVideoResult, error) {
	return ProcessVideoWithStandard(videoURL, DefaultVideoStandard)
}

// ProcessVideoWithStandard 使用自定义标准处理视频
func ProcessVideoWithStandard(videoURL string, standard VideoStandard) (*ProcessVideoResult, error) {
	result := &ProcessVideoResult{
		Success:      false,
		ProcessedOps: []string{},
	}

	// 先校验视频
	validation := ValidateVideoWithStandard(videoURL, standard)
	if validation.Valid {
		result.Success = true
		result.NeedProcess = false
		result.Message = "视频已符合标准，无需处理"
		return result, nil
	}

	result.NeedProcess = true
	videoInfo := validation.Info
	if videoInfo == nil {
		return nil, fmt.Errorf("无法获取视频信息")
	}

	// 创建临时输出文件
	tmpDir := os.TempDir()
	timestamp := time.Now().Unix()
	outputPath := filepath.Join(tmpDir, fmt.Sprintf("processed_video_%d.mp4", timestamp))

	// 构建 ffmpeg 参数
	args := []string{
		"-i", videoURL,
		"-y", // 覆盖输出文件
	}

	// 视频和音频处理标记
	needVideoProcess := false

	// 检查视频编码
	if !strings.EqualFold(videoInfo.VideoCodec, standard.VideoCodec) {
		args = append(args, "-c:v", "libx264")
		result.ProcessedOps = append(result.ProcessedOps, fmt.Sprintf("转换视频编码: %s -> H.264", videoInfo.VideoCodec))
	} else {
		// 如果不需要重新编码，默认复制视频流（后续可能需要重新编码）
		args = append(args, "-c:v", "libx264")
	}

	// 检查像素格式
	if videoInfo.PixelFormat != "" && videoInfo.PixelFormat != standard.PixelFormat {
		args = append(args, "-pix_fmt", standard.PixelFormat)
		result.ProcessedOps = append(result.ProcessedOps, fmt.Sprintf("转换像素格式: %s -> %s", videoInfo.PixelFormat, standard.PixelFormat))
	} else if videoInfo.PixelFormat == "" {
		// 强制设置像素格式
		args = append(args, "-pix_fmt", standard.PixelFormat)
		result.ProcessedOps = append(result.ProcessedOps, "设置像素格式: yuv420p")
	}

	// 检查分辨率
	if videoInfo.Width > standard.MaxWidth || videoInfo.Height > standard.MaxHeight {
		// 等比例缩放
		scale := fmt.Sprintf("scale='min(%d,iw)':'min(%d,ih)':force_original_aspect_ratio=decrease", standard.MaxWidth, standard.MaxHeight)
		args = append(args, "-vf", scale)
		needVideoProcess = true
		result.ProcessedOps = append(result.ProcessedOps, fmt.Sprintf("缩放分辨率: %dx%d -> ≤%dx%d", videoInfo.Width, videoInfo.Height, standard.MaxWidth, standard.MaxHeight))
	}

	// 检查帧率
	if videoInfo.FrameRate > standard.MaxFrameRate {
		if videoInfo.FrameRate > 25 {
			// 高帧率限制为 25fps
			args = append(args, "-r", "25")
			result.ProcessedOps = append(result.ProcessedOps, fmt.Sprintf("限制帧率: %.2f fps -> 25 fps", videoInfo.FrameRate))
		} else {
			// 使用可变帧率
			args = append(args, "-vsync", "vfr")
			result.ProcessedOps = append(result.ProcessedOps, "使用可变帧率 (VFR)")
		}
		needVideoProcess = true
	}

	// 检查视频码率
	if videoInfo.VideoBitrate > standard.MaxVideoBitrate {
		args = append(args, "-b:v", fmt.Sprintf("%dk", standard.MaxVideoBitrate/1000))
		args = append(args, "-maxrate", fmt.Sprintf("%dk", standard.MaxVideoBitrate/1000))
		args = append(args, "-bufsize", fmt.Sprintf("%dk", standard.MaxVideoBitrate/1000*2))
		needVideoProcess = true
		result.ProcessedOps = append(result.ProcessedOps, fmt.Sprintf("限制视频码率: %d kbps -> %d kbps", videoInfo.VideoBitrate/1000, standard.MaxVideoBitrate/1000))
	}

	// 检查音频编码
	if videoInfo.AudioCodec != "" && !strings.EqualFold(videoInfo.AudioCodec, standard.AudioCodec) {
		args = append(args, "-c:a", "aac")
		result.ProcessedOps = append(result.ProcessedOps, fmt.Sprintf("转换音频编码: %s -> AAC", videoInfo.AudioCodec))
	} else if videoInfo.AudioCodec != "" {
		args = append(args, "-c:a", "aac")
	}

	// 检查音频声道
	if videoInfo.Channels > standard.MaxChannels {
		args = append(args, "-ac", fmt.Sprintf("%d", standard.MaxChannels))
		result.ProcessedOps = append(result.ProcessedOps, fmt.Sprintf("限制音频声道: %d -> %d", videoInfo.Channels, standard.MaxChannels))
	}

	// 检查采样率
	if videoInfo.SampleRate > standard.MaxSampleRate {
		args = append(args, "-ar", fmt.Sprintf("%d", standard.MaxSampleRate))
		result.ProcessedOps = append(result.ProcessedOps, fmt.Sprintf("限制采样率: %d Hz -> %d Hz", videoInfo.SampleRate, standard.MaxSampleRate))
	}

	// H.264 优化参数
	if needVideoProcess {
		args = append(args,
			"-preset", "medium", // 编码速度与压缩率的平衡
			"-crf", "23", // 恒定质量模式，值越小质量越好（范围 0-51）
		)
	}

	// 输出文件
	args = append(args, outputPath)

	// 执行 ffmpeg
	cmd := exec.Command("ffmpeg", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		// 清理可能创建的文件（忽略清理错误）
		_ = os.Remove(outputPath)
		return nil, fmt.Errorf("ffmpeg 处理失败: %v, 输出: %s", err, string(output))
	}

	// 验证输出文件
	if _, err := os.Stat(outputPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("处理后的视频文件不存在")
	}

	result.Success = true
	result.OutputPath = outputPath
	result.Message = fmt.Sprintf("视频处理成功，共执行 %d 项优化", len(result.ProcessedOps))

	return result, nil
}
