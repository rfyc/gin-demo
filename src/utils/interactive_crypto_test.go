package utils

import (
	"testing"
)

func TestEncryptDecryptInteractivePrompt(t *testing.T) {
	tests := []struct {
		name      string
		plaintext string
	}{
		{
			name:      "简单文本",
			plaintext: "根据内容生成互动，互动类型：选择-结束判断",
		},
		{
			name:      "包含JSON",
			plaintext: `{"type": "interactive", "content": "测试内容"}`,
		},
		{
			name:      "空字符串",
			plaintext: "",
		},
		{
			name:      "长文本",
			plaintext: "这是一段很长的文本，用于测试加密解密功能是否能正确处理较长的内容。这段文本包含了中文、英文和数字123456，以及一些特殊字符：!@#$%^&*()",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 加密
			encrypted, err := EncryptInteractivePrompt(tt.plaintext)
			if err != nil {
				t.Fatalf("加密失败: %v", err)
			}

			// 空字符串应该返回空
			if tt.plaintext == "" {
				if encrypted != "" {
					t.Errorf("空字符串加密应返回空字符串，实际返回: %s", encrypted)
				}
				return
			}

			// 加密后的内容不应该等于原文
			if encrypted == tt.plaintext {
				t.Errorf("加密后的内容不应该等于原文")
			}

			t.Logf("原文: %s", tt.plaintext)
			t.Logf("密文: %s", encrypted)

			// 解密
			decrypted, err := DecryptInteractivePrompt(encrypted)
			if err != nil {
				t.Fatalf("解密失败: %v", err)
			}

			// 解密后应该等于原文
			if decrypted != tt.plaintext {
				t.Errorf("解密结果不匹配\n原文: %s\n解密: %s", tt.plaintext, decrypted)
			}

			t.Logf("解密: %s", decrypted)
		})
	}
}

func TestAESCBCWithSpecificKeyAndIV(t *testing.T) {
	// 测试特定的密钥和 IV
	key := []byte("wangxiaohudong\x00\x00") // 16 字节
	iv := make([]byte, 16)                    // 全零 IV

	plaintext := "测试文本"

	// 加密
	encrypted, err := AESCBCEncrypt(plaintext, key, iv)
	if err != nil {
		t.Fatalf("加密失败: %v", err)
	}

	t.Logf("加密结果: %s", encrypted)

	// 解密
	decrypted, err := AESCBCDecrypt(encrypted, key, iv)
	if err != nil {
		t.Fatalf("解密失败: %v", err)
	}

	if decrypted != plaintext {
		t.Errorf("解密结果不匹配\n原文: %s\n解密: %s", plaintext, decrypted)
	}

	t.Logf("解密结果: %s", decrypted)
}

func TestAESCBCInvalidInput(t *testing.T) {
	tests := []struct {
		name        string
		key         []byte
		iv          []byte
		shouldError bool
	}{
		{
			name:        "密钥长度错误",
			key:         []byte("shortkey"),
			iv:          make([]byte, 16),
			shouldError: true,
		},
		{
			name:        "IV长度错误",
			key:         []byte("wangxiaohudong\x00\x00"),
			iv:          []byte("shortiv"),
			shouldError: true,
		},
		{
			name:        "正确的密钥和IV",
			key:         []byte("wangxiaohudong\x00\x00"),
			iv:          make([]byte, 16),
			shouldError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := AESCBCEncrypt("test", tt.key, tt.iv)
			if tt.shouldError && err == nil {
				t.Errorf("期望加密失败，但成功了")
			}
			if !tt.shouldError && err != nil {
				t.Errorf("期望加密成功，但失败了: %v", err)
			}
		})
	}
}
