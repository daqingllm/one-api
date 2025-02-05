package common

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"io"
	"os"
)

// 定义密钥
var (
	secretKey = []byte(os.Getenv("TOKEN_SECRET_KEY")) // 32 字节密钥
)

// EncryptResult 加密结果结构体
type EncryptResult struct {
	IV            string `json:"iv"`
	EncryptedData string `json:"encryptedData"`
}

// Encrypt 加密函数
func Encrypt(text string) (*EncryptResult, error) {
	block, err := aes.NewCipher(secretKey)
	if err != nil {
		return nil, err
	}

	// 生成随机 IV
	iv := make([]byte, aes.BlockSize)
	if _, err := io.ReadFull(rand.Reader, iv); err != nil {
		return nil, err
	}

	plaintext := []byte(text)
	mode := cipher.NewCBCEncrypter(block, iv)

	// 确保明文长度是块大小的倍数
	if len(plaintext)%aes.BlockSize != 0 {
		padding := aes.BlockSize - len(plaintext)%aes.BlockSize
		padtext := bytes.Repeat([]byte{byte(padding)}, padding) // 使用 bytes 包填充
		plaintext = append(plaintext, padtext...)
	}

	ciphertext := make([]byte, len(plaintext))
	mode.CryptBlocks(ciphertext, plaintext)

	return &EncryptResult{
		IV:            hex.EncodeToString(iv),
		EncryptedData: hex.EncodeToString(ciphertext),
	}, nil
}

// Decrypt 解密函数
func Decrypt(encryptedData string, ivStr string) (string, error) {
	block, err := aes.NewCipher(secretKey)
	if err != nil {
		return "", err
	}

	ivBytes, err := hex.DecodeString(ivStr)
	if err != nil {
		return "", err
	}

	ciphertext, err := hex.DecodeString(encryptedData)
	if err != nil {
		return "", err
	}

	mode := cipher.NewCBCDecrypter(block, ivBytes)
	plaintext := make([]byte, len(ciphertext))
	mode.CryptBlocks(plaintext, ciphertext)

	// 去除填充
	length := len(plaintext)
	unpadding := int(plaintext[length-1])
	if unpadding > aes.BlockSize || unpadding > length {
		return "", errors.New("invalid padding size")
	}
	return string(plaintext[:(length - unpadding)]), nil
}
