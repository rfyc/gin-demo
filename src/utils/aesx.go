package utils

import (
	"bytes"
	"compress/zlib"
	"crypto/aes"
	"crypto/cipher"
	"crypto/md5"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"io"
)

func encrypt(plaintext []byte, key []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err = io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, err
	}

	return gcm.Seal(nonce, nonce, plaintext, nil), nil
}

func decrypt(ciphertext []byte, key []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	if len(ciphertext) < gcm.NonceSize() {
		return nil, fmt.Errorf("ciphertext too short")
	}

	nonce, ciphertext := ciphertext[:gcm.NonceSize()], ciphertext[gcm.NonceSize():]
	return gcm.Open(nil, nonce, ciphertext, nil)
}

func compress(data []byte) ([]byte, error) {
	var b bytes.Buffer
	w := zlib.NewWriter(&b)
	_, err := w.Write(data)
	if err != nil {
		return nil, err
	}
	err = w.Close()
	if err != nil {
		return nil, err
	}
	return b.Bytes(), nil
}

func decompress(data []byte) ([]byte, error) {
	r, err := zlib.NewReader(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	defer r.Close()
	return io.ReadAll(r)
}

func EncryptAndCompress(plaintext string, key []byte) (string, error) {
	encrypted, err := encrypt([]byte(plaintext), key)
	if err != nil {
		return "", err
	}

	compressed, err := compress(encrypted)
	if err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(compressed), nil
}

func DecompressAndDecrypt(ciphertext string, key []byte) (string, error) {
	decoded, err := base64.URLEncoding.DecodeString(ciphertext)
	if err != nil {
		return "", err
	}

	decompressed, err := decompress(decoded)
	if err != nil {
		return "", err
	}

	decrypted, err := decrypt(decompressed, key)
	if err != nil {
		return "", err
	}

	return string(decrypted), nil
}
func MD5File(data []byte) string {
	hash := md5.New()
	hash.Write(data)
	return hex.EncodeToString(hash.Sum(nil))
}

func MD5(str string) string {
	h := md5.New()
	h.Write([]byte(str))
	return hex.EncodeToString(h.Sum(nil))
}

// PKCS7Padding 填充数据至 blockSize 的倍数
func PKCS7Padding(data []byte, blockSize int) []byte {
	padding := blockSize - len(data)%blockSize
	padtext := bytes.Repeat([]byte{byte(padding)}, padding)
	return append(data, padtext...)
}

// PKCS7Unpadding 去除 PKCS7 填充
func PKCS7Unpadding(data []byte) ([]byte, error) {
	length := len(data)
	if length == 0 {
		return nil, fmt.Errorf("data is empty")
	}
	unpadding := int(data[length-1])
	if unpadding > length {
		return nil, fmt.Errorf("invalid padding")
	}
	return data[:(length - unpadding)], nil
}

// AESCBCEncrypt AES-128-CBC 加密
// plaintext: 明文
// key: 密钥，必须是 16 字节（AES-128）
// iv: 初始化向量，必须是 16 字节
// 返回 base64 编码的密文
func AESCBCEncrypt(plaintext string, key []byte, iv []byte) (string, error) {
	if len(key) != 16 {
		return "", fmt.Errorf("key must be 16 bytes for AES-128")
	}
	if len(iv) != 16 {
		return "", fmt.Errorf("iv must be 16 bytes")
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}

	// PKCS7 填充
	paddedData := PKCS7Padding([]byte(plaintext), aes.BlockSize)

	// CBC 加密
	ciphertext := make([]byte, len(paddedData))
	mode := cipher.NewCBCEncrypter(block, iv)
	mode.CryptBlocks(ciphertext, paddedData)

	// base64 编码
	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

// AESCBCDecrypt AES-128-CBC 解密
// ciphertext: base64 编码的密文
// key: 密钥，必须是 16 字节（AES-128）
// iv: 初始化向量，必须是 16 字节
// 返回明文
func AESCBCDecrypt(ciphertext string, key []byte, iv []byte) (string, error) {
	if len(key) != 16 {
		return "", fmt.Errorf("key must be 16 bytes for AES-128")
	}
	if len(iv) != 16 {
		return "", fmt.Errorf("iv must be 16 bytes")
	}

	// base64 解码
	data, err := base64.StdEncoding.DecodeString(ciphertext)
	if err != nil {
		return "", fmt.Errorf("base64 decode error: %w", err)
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}

	if len(data)%aes.BlockSize != 0 {
		return "", fmt.Errorf("ciphertext is not a multiple of the block size")
	}

	// CBC 解密
	plaintext := make([]byte, len(data))
	mode := cipher.NewCBCDecrypter(block, iv)
	mode.CryptBlocks(plaintext, data)

	// 去除 PKCS7 填充
	unpaddedData, err := PKCS7Unpadding(plaintext)
	if err != nil {
		return "", fmt.Errorf("unpadding error: %w", err)
	}

	return string(unpaddedData), nil
}
