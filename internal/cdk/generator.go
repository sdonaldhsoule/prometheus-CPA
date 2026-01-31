package cdk

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"strings"
	"time"
)

// Generator CDK生成器
type Generator struct {
	prefix string
}

// NewGenerator 创建CDK生成器
func NewGenerator(prefix string) *Generator {
	if prefix == "" {
		prefix = "DS"
	}
	return &Generator{prefix: strings.ToUpper(prefix)}
}

// Generate 生成CDK
func (g *Generator) Generate() (string, error) {
	// 生成16字节随机数
	randomBytes := make([]byte, 16)
	if _, err := rand.Read(randomBytes); err != nil {
		return "", fmt.Errorf("generate random bytes: %w", err)
	}

	// 转换为十六进制并取前16个字符
	randomHex := strings.ToUpper(hex.EncodeToString(randomBytes))[:16]

	// 添加时间戳后缀 (4位)
	timestamp := fmt.Sprintf("%04d", time.Now().Unix()%10000)

	// 格式: PREFIX-XXXX-XXXX-XXXX-XXXX-TTTT
	code := fmt.Sprintf("%s-%s-%s-%s-%s-%s",
		g.prefix,
		randomHex[0:4],
		randomHex[4:8],
		randomHex[8:12],
		randomHex[12:16],
		timestamp,
	)

	return code, nil
}

// ValidateFormat 验证CDK格式
func (g *Generator) ValidateFormat(code string) bool {
	// 基本格式检查: PREFIX-XXXX-XXXX-XXXX-XXXX-TTTT
	parts := strings.Split(code, "-")
	if len(parts) != 6 {
		return false
	}

	// 检查前缀
	if parts[0] != g.prefix {
		return false
	}

	// 检查每个部分长度
	for i := 1; i < 6; i++ {
		if len(parts[i]) != 4 {
			return false
		}
	}

	return true
}
