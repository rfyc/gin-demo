package utils

// CommonMimeTypes 常用文件扩展名与 MIME 类型映射
var CommonMimeTypes = map[string]string{
	// 图片类型
	".jpg":  "image/jpeg",
	".jpeg": "image/jpeg",
	".png":  "image/png",
	".gif":  "image/gif",
	".bmp":  "image/bmp",
	".webp": "image/webp",
	".ico":  "image/x-icon",
	".svg":  "image/svg+xml",
	".tiff": "image/tiff",
	".heic": "image/heic",

	// 文档 & 文本
	".txt":  "text/plain",
	".md":   "text/markdown",
	".pdf":  "application/pdf",
	".rtf":  "application/rtf",
	".json": "application/json",
	".xml":  "application/xml",
	".html": "text/html",
	".htm":  "text/html",
	".css":  "text/css",
	".js":   "application/javascript",
	".csv":  "text/csv",

	// 办公文档
	".doc":  "application/msword",
	".docx": "application/vnd.openxmlformats-officedocument.wordprocessingml.document",
	".xls":  "application/vnd.ms-excel",
	".xlsx": "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet",
	".ppt":  "application/vnd.ms-powerpoint",
	".pptx": "application/vnd.openxmlformats-officedocument.presentationml.presentation",

	// 视频
	".mp4":  "video/mp4",
	".avi":  "video/x-msvideo",
	".mov":  "video/quicktime",
	".wmv":  "video/x-ms-wmv",
	".flv":  "video/x-flv",
	".mkv":  "video/x-matroska",
	".webm": "video/webm",
	".mpeg": "video/mpeg",
	".mpg":  "video/mpeg",

	// 音频
	".mp3":  "audio/mpeg",
	".wav":  "audio/wav",
	".flac": "audio/flac",
	".aac":  "audio/aac",
	".ogg":  "audio/ogg",
	".m4a":  "audio/mp4",

	// 压缩包
	".zip": "application/zip",
	".rar": "application/vnd.rar",
	".7z":  "application/x-7z-compressed",
	".tar": "application/x-tar",
	".gz":  "application/gzip",
	".bz2": "application/x-bzip2",

	// 安装包 & 可执行
	".exe": "application/vnd.microsoft.portable-executable",
	".apk": "application/vnd.android.package-archive",
	".ipa": "application/octet-stream",
	".dmg": "application/x-apple-diskimage",
	".deb": "application/x-debian-package",
	".rpm": "application/x-rpm",

	// 其他常用
	".bin":  "application/octet-stream",
	".log":  "text/plain",
	".ini":  "text/plain",
	".yml":  "text/yaml",
	".yaml": "text/yaml",
	".sql":  "application/sql",
}
