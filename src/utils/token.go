package utils

import (
	"strings"
	"unicode"
)

func EstimateTokenCount(text string) int {
	if text == "" {
		return 0
	}

	// 计算中日韩字符数量
	cjkCount := 0
	for _, r := range text {
		if isCJKCharacter(r) {
			cjkCount++
		}
	}

	// 剩余非CJK字符数量
	nonCJKCount := len([]rune(text)) - cjkCount

	// 估算token数: CJK字符每个算1个token，其他字符每4个算1个token
	return cjkCount + (nonCJKCount+3)/4
}

// TruncateToTokenCount 截取字符串到指定的token长度
func TruncateToTokenCount(text string, maxTokens int) string {
	// 先估算整体token数
	totalTokens := EstimateTokenCount(text)

	// 如果未超过限制，直接返回原字符串
	if totalTokens <= maxTokens {
		return text
	}

	// 需要截断，逐个字符处理
	runes := []rune(text)
	currentTokens := float64(0)
	lastSafePos := 0 // 最后一个安全截断位置（例如单词末尾）

	for i := 0; i < len(runes); i++ {
		// 当前字符是否为CJK字符
		isCJK := isCJKCharacter(runes[i])

		// 计算添加当前字符后的token数
		if isCJK {
			// CJK字符直接增加1个token
			currentTokens++
		} else {
			// 非CJK字符累计到4个增加1个token
			// 这里简化处理，每遇到非CJK字符增加0.25个token
			currentTokens += 1.0 / 4.0
		}

		// 检查是否超过或达到token限制
		if currentTokens > float64(maxTokens) {
			// 如果有安全截断位置，使用该位置
			if lastSafePos > 0 {
				return string(runes[:lastSafePos])
			}
			// 否则在当前位置截断
			return string(runes[:i])
		}

		// 更新最后一个安全截断位置
		// 这里定义安全位置为空格或标点之后
		if unicode.IsSpace(runes[i]) || isPunctuation(runes[i]) {
			lastSafePos = i + 1
		}
	}

	// 正常情况下不会执行到这里
	return text
}

func isCJKCharacter(r rune) bool {
	// CJK统一表意文字范围
	return (r >= 0x4E00 && r <= 0x9FFF) ||
		// CJK部首补充
		(r >= 0x2E80 && r <= 0x2EFF) ||
		// CJK笔画
		(r >= 0x31C0 && r <= 0x31EF) ||
		// CJK兼容表意文字
		(r >= 0xF900 && r <= 0xFAFF) ||
		// CJK统一表意文字扩展A
		(r >= 0x3400 && r <= 0x4DBF) ||
		// 日文平假名
		(r >= 0x3040 && r <= 0x309F) ||
		// 日文片假名
		(r >= 0x30A0 && r <= 0x30FF)
}

// 判断是否为标点符号
func isPunctuation(r rune) bool {
	// 简化版标点符号判断，实际应用中可以扩展
	return strings.ContainsRune(".,;:!?'\"()[]{}<>", r)
}
