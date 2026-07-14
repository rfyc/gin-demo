package utils

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"strings"
)

const (
	// InteractivePromptKey 互动提示词加密密钥（16 字节 AES-128）
	// 注意：此密钥存在于源码中，应通过环境变量或配置中心管理
	InteractivePromptKey = "wangxiaohudong\x00\x00"

	ivPrefix = "v2:"
)

// EncryptInteractivePrompt 加密 interactivePrompt。
// 每次加密使用随机 IV，格式："v2:" + base64(IV[16] + ciphertext)
func EncryptInteractivePrompt(plaintext string) (string, error) {
	if strings.TrimSpace(plaintext) == "" {
		return "", nil
	}
	iv := make([]byte, 16)
	if _, err := rand.Read(iv); err != nil {
		return "", fmt.Errorf("生成随机 IV 失败: %w", err)
	}
	ciphertext, err := AESCBCEncrypt(plaintext, []byte(InteractivePromptKey), iv)
	if err != nil {
		return "", err
	}
	cipherbytes, err := base64.StdEncoding.DecodeString(ciphertext)
	if err != nil {
		return "", err
	}
	payload := append(iv, cipherbytes...)
	return ivPrefix + base64.StdEncoding.EncodeToString(payload), nil
}

// DecryptInteractivePrompt 解密 interactivePrompt。
// 支持新格式（"v2:" 前缀，IV 随机）和旧格式（兼容存量数据，固定零 IV）。
func DecryptInteractivePrompt(ciphertext string) (string, error) {
	if strings.TrimSpace(ciphertext) == "" {
		return "", nil
	}
	if strings.HasPrefix(ciphertext, ivPrefix) {
		payload, err := base64.StdEncoding.DecodeString(ciphertext[len(ivPrefix):])
		if err != nil {
			return "", fmt.Errorf("base64 解码失败: %w", err)
		}
		if len(payload) < 16 {
			return "", fmt.Errorf("密文长度不足")
		}
		iv := payload[:16]
		encoded := base64.StdEncoding.EncodeToString(payload[16:])
		return AESCBCDecrypt(encoded, []byte(InteractivePromptKey), iv)
	}
	// 兼容旧格式（固定零 IV）
	return AESCBCDecrypt(ciphertext, []byte(InteractivePromptKey), make([]byte, 16))
}
