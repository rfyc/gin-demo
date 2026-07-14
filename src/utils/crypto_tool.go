package utils

import (
	"fmt"
)

// InteractiveCryptoExample 互动提示词加密解密示例
func InteractiveCryptoExample() {
	fmt.Println("=== 互动提示词加密解密示例 ===")
	fmt.Println()

	// 测试用例
	testCases := []string{
		"根据内容生成互动，互动类型：选择-结束判断",
		`{"type": "interactive", "content": "测试内容"}`,
		"这是一段包含中文、English和数字123的混合文本",
	}

	for i, plaintext := range testCases {
		fmt.Printf("测试 #%d\n", i+1)
		fmt.Printf("原文: %s\n", plaintext)

		// 加密
		encrypted, err := EncryptInteractivePrompt(plaintext)
		if err != nil {
			fmt.Printf("加密失败: %v\n\n", err)
			continue
		}
		fmt.Printf("密文: %s\n", encrypted)

		// 解密
		decrypted, err := DecryptInteractivePrompt(encrypted)
		if err != nil {
			fmt.Printf("解密失败: %v\n\n", err)
			continue
		}
		fmt.Printf("解密: %s\n", decrypted)

		// 验证
		if plaintext == decrypted {
			fmt.Println("✓ 验证成功")
		} else {
			fmt.Println("✗ 验证失败")
		}
		fmt.Println()
	}

	fmt.Println("=== 加密参数 ===")
	fmt.Printf("密钥: %q (长度: %d 字节)\n", InteractivePromptKey, len(InteractivePromptKey))
	fmt.Println("IV: 随机生成（每次加密不同）")
}
