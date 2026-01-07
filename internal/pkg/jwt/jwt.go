package pkg

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	jwtv5 "github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

func CleanToken(tokenStr string) string {
	// 移除Bearer前缀（兼容Authorization头的Bearer Token格式）
	tokenStr = strings.TrimSpace(tokenStr)
	if len(tokenStr) > 7 && strings.EqualFold(tokenStr[:7], "bearer ") {
		tokenStr = tokenStr[7:]
	}
	return tokenStr
}

// JWTConfig JWT配置项
type JWTConfig struct {
	Secret []byte        // 签名密钥
	Expire time.Duration // Token有效期
	Issuer string        // 签发者
}

var abc = JWTConfig{
	Secret: []byte("aiuegfiuewgfiuwfeiuwheqowhfoiqfiifenfeqnfeq"),
	Expire: 30 * time.Minute,
	Issuer: "https://jwt.io",
}

// UserClaims JWT自定义声明
type UserClaims struct {
	UserID   string `json:"uuid"`
	Username string `json:"username"`
	Role     string `json:"role"`
	jwtv5.RegisteredClaims
}

// GenerateToken 生成JWT Token
func GenerateToken(uuid string, username, role string) (string, error) {
	claims := UserClaims{
		UserID:   uuid,
		Username: username,
		Role:     role,

		RegisteredClaims: jwtv5.RegisteredClaims{
			ExpiresAt: jwtv5.NewNumericDate(time.Now().Add(abc.Expire)),
			IssuedAt:  jwtv5.NewNumericDate(time.Now()),
			Issuer:    abc.Issuer,
		},
	}

	token := jwtv5.NewWithClaims(jwtv5.SigningMethodHS256, claims)
	return token.SignedString(abc.Secret)
}

// ParseToken 解析并验证JWT Token
func ParseToken(cfg JWTConfig, tokenStr string) (*UserClaims, error) {
	tokenStr = CleanToken(tokenStr)
	if tokenStr == "" {
		return nil, errors.New("token为空")
	}

	token, err := jwtv5.ParseWithClaims(tokenStr, &UserClaims{}, func(token *jwtv5.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwtv5.SigningMethodHMAC); !ok {
			return nil, errors.New("不支持的签名算法")
		}
		return cfg.Secret, nil
	})
	if err != nil {
		return nil, err
	}

	claims, ok := token.Claims.(*UserClaims)
	if !ok || !token.Valid {
		return nil, errors.New("token无效或已过期")
	}
	return claims, nil
}

// HashPassword 密码哈希（bcrypt）
func HashPassword(password string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(hash), nil
}

// CheckPassword 验证密码哈希
func CheckPassword(password, hash string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)) == nil
}

// GenerateUUID 生成UUIDv4
func GenerateUUID() string {
	return uuid.NewString()
}

const (
	AvatarSaveRoot  = "./static/avatars" // 本地保存根目录
	AvatarBaseURL   = "/static/avatars/" // 前端访问基础URL
	MaxAvatarSize   = 2 * 1024 * 1024    // 最大2MB
	AllowAvatarExts = ".jpg,.jpeg,.png"  // 允许的格式
)

// SaveAvatar 保存头像文件并生成访问URL
func SaveAvatar(file io.Reader, fileName string) (string, error) {
	// 1. 空文件校验
	if file == nil {
		return "", errors.New("头像文件为空")
	}

	// 2. 解析后缀并校验格式
	ext := strings.ToLower(filepath.Ext(fileName))
	if ext == "" {
		return "", errors.New("头像文件无后缀，无法识别格式")
	}
	if !strings.Contains(AllowAvatarExts, ext) {
		return "", fmt.Errorf("不支持的头像格式：%s，仅允许%s", ext, AllowAvatarExts)
	}

	// 3. 生成唯一文件名（避免覆盖）
	uniqueID := uuid.New().String()
	saveFileName := uniqueID + ext
	savePath := filepath.Join(AvatarSaveRoot, saveFileName)

	// 4. 自动创建目录
	if err := os.MkdirAll(AvatarSaveRoot, 0755); err != nil {
		return "", fmt.Errorf("创建头像目录失败：%w", err)
	}

	// 5. 创建文件并写入（带大小限制）
	dstFile, err := os.Create(savePath)
	if err != nil {
		return "", fmt.Errorf("创建头像文件失败：%w", err)
	}
	defer dstFile.Close()

	// 6. 限制文件大小
	sizeCounter := &sizeLimitWriter{writer: dstFile, maxSize: MaxAvatarSize}
	written, err := io.Copy(sizeCounter, file)
	if err != nil {
		os.Remove(savePath) // 写入失败删除临时文件
		if errors.Is(err, errSizeExceed) {
			return "", fmt.Errorf("头像文件超过%dMB限制", MaxAvatarSize/1024/1024)
		}
		return "", fmt.Errorf("写入头像文件失败：%w", err)
	}
	if written == 0 {
		os.Remove(savePath)
		return "", errors.New("头像文件内容为空")
	}

	// 7. 生成访问URL
	return AvatarBaseURL + saveFileName, nil
}

// 辅助：大小限制Writer
var errSizeExceed = errors.New("文件大小超限")

type sizeLimitWriter struct {
	writer  io.Writer
	maxSize int64
	written int64
}

func (w *sizeLimitWriter) Write(p []byte) (int, error) {
	if w.written+int64(len(p)) > w.maxSize {
		return 0, errSizeExceed
	}
	n, err := w.writer.Write(p)
	w.written += int64(n)
	return n, err
}
func DeleteAvatar(avatarURL string) error {
	const AvatarSaveRoot = "D:\\朱子墨\\Go\\CMS\\static\\avatars" // 本地保存根目录
	fileName := filepath.Base(avatarURL)
	fullPath := filepath.Join(AvatarSaveRoot, fileName)

	// 删除文件
	if err := os.Remove(fullPath); err != nil {
		// 处理文件不存在的情况（无需报错）
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("删除本地头像文件失败：%w", err)
	}
	return nil
}
