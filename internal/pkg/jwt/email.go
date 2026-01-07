package pkg

import (
	"crypto/rand"
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"sync"

	"math/big"

	"net/smtp"
	"strings"
	"time"

	"github.com/google/uuid"
)

func GenerateVerifyCode() (string, error) {
	const codeLen = 6     // 验证码长度
	const digitCount = 10 // 0-9共10个数字
	var code strings.Builder

	// 循环生成每位验证码（用crypto/rand保证随机性，比math/rand更安全）
	for i := 0; i < codeLen; i++ {
		// 生成0-9之间的随机数
		num, err := rand.Int(rand.Reader, big.NewInt(digitCount))
		if err != nil {
			return "", fmt.Errorf("生成验证码失败: %w", err)
		}
		// 拼接成字符串
		code.WriteString(fmt.Sprintf("%d", num.Int64()))
	}

	return code.String(), nil
}

// 2. 发送验证码邮件（参数：收件人邮箱、验证码）
func SendVerifyCodeMail(toMail, verifyCode string) error {
	saveCodeToMap(toMail, verifyCode)
	//if checkSendFrequency(toMail) {
	//	return fmt.Errorf("请求过于频繁")
	//}
	// -------------------------- 邮件配置（需替换为你的信息） --------------------------
	smtpServer := "smtp.qq.com"       // QQ邮箱SMTP服务器（163邮箱为smtp.163.com）
	smtpPort := 465                   // SSL加密端口（QQ/163均为465）
	senderMail := "3358138285@qq.com" // 发件人邮箱（如3358138285@qq.com）
	authCode := "zxtodducgkurdafj"    // 发件人邮箱SMTP授权码（非登录密码！）
	// ----------------------------------------------------------------------------------

	// 邮件主题（含中文需处理编码，避免乱码）
	subject := "=?UTF-8?B?" + base64Encode("您的邮箱验证码") + "?="
	// 邮件正文（明确告知验证码、有效期）
	expireMin := 5 // 验证码有效期（分钟）
	body := fmt.Sprintf(`
		<div style="font-family: Arial, sans-serif; max-width: 600px; margin: 0 auto;">
			<h3 style="color: #333;">邮箱验证通知</h3>
			<p>您好！您正在进行邮箱验证操作，您的验证码为：</p>
			<div style="font-size: 24px; font-weight: bold; color: #2E86AB; margin: 20px 0;">%s</div>
			<p>验证码有效期为 %d 分钟，请尽快使用，过期后需重新获取。</p>
			<p>若您未发起此操作，请忽略此邮件，感谢您的理解！</p>
		</div>`, verifyCode, expireMin)

	// 构建邮件头部（From/To/Subject/Content-Type等）
	mailHeader := fmt.Sprintf(`From: %s
To: %s
Subject: %s
MIME-Version: 1.0
Content-Type: text/html; charset=UTF-8

`, senderMail, toMail, subject)

	// 组合完整邮件内容（头部+正文）
	fullMail := mailHeader + body

	// -------------------------- 发送邮件（SSL加密） --------------------------
	// 1. 构建SMTP认证信息（QQ邮箱用授权码认证）
	auth := smtp.PlainAuth(
		"",         // 身份标识（留空即可）
		senderMail, // 发件人邮箱
		authCode,   // 邮箱SMTP授权码
		smtpServer, // SMTP服务器地址
	)

	// 2. 配置TLS（SSL加密，避免明文传输）
	tlsConfig := &tls.Config{
		ServerName:         smtpServer, // 服务器域名（需与SMTP服务器一致，避免证书校验失败）
		InsecureSkipVerify: false,      // 不跳过证书校验（安全模式）
	}

	// 3. 建立TLS连接（465端口为SSL专用端口）
	conn, err := tls.Dial("tcp", fmt.Sprintf("%s:%d", smtpServer, smtpPort), tlsConfig)
	if err != nil {
		return fmt.Errorf("连接邮件服务器失败: %w", err)
	}
	defer conn.Close() // 函数结束后关闭连接

	// 4. 创建SMTP客户端并登录
	client, err := smtp.NewClient(conn, smtpServer)
	if err != nil {
		return fmt.Errorf("创建SMTP客户端失败: %w", err)
	}
	if err := client.Auth(auth); err != nil {
		return fmt.Errorf("邮箱登录失败（请检查授权码）: %w", err)
	}

	// 5. 设置发件人、收件人
	if err := client.Mail(senderMail); err != nil {
		return fmt.Errorf("设置发件人失败: %w", err)
	}
	if err := client.Rcpt(toMail); err != nil {
		return fmt.Errorf("设置收件人失败（请检查邮箱格式）: %w", err)
	}

	// 6. 发送邮件内容
	dataWriter, err := client.Data()
	if err != nil {
		return fmt.Errorf("准备发送邮件内容失败: %w", err)
	}
	// 写入完整邮件内容
	_, err = dataWriter.Write([]byte(fullMail))
	if err != nil {
		return fmt.Errorf("写入邮件内容失败: %w", err)
	}
	// 关闭数据流（触发实际发送）
	if err := dataWriter.Close(); err != nil {
		return fmt.Errorf("发送邮件失败: %w", err)
	}

	// 7. 退出客户端
	client.Quit()
	return nil
}

// 辅助函数：Base64编码（处理中文主题乱码）
func base64Encode(str string) string {
	return fmt.Sprintf("%x", []byte(str)) // 简化版Base64编码（适配邮件主题格式）
}

//	var redisClient = redis.NewClient(&redis.Options{
//		Addr: "localhost:6379",
//	})
//
//	func saveCodeToRedis(email, code string) error {
//		ctx := context.Background()
//		return redisClient.Set(ctx, "verify:"+email, code, 5*time.Minute).Err()
//	}
//
// // 验证验证码（返回nil表示验证通过）
//
//	func VerifyCodeFromRedis(email, inputCode string) error {
//		ctx := context.Background()
//		// 1. 从Redis查询存储的验证码
//		storedCode, err := redisClient.Get(ctx, "verify:"+email).Result()
//		if err != nil {
//			if errors.Is(err, redis.Nil) {
//				// 未查询到记录：验证码无效或已过期
//				return errors.New("验证码无效")
//			}
//			// Redis查询失败：系统错误
//			return errors.New("验证服务异常，请重试")
//		}
//
//		// 2. 校验验证码正确性
//		if storedCode != inputCode {
//			return errors.New("验证码错误")
//		}
//
//		// 3. 验证通过后删除验证码（避免重复使用）
//		if err := redisClient.Del(ctx, "verify:"+email).Err(); err != nil {
//			return errors.New("验证后续处理失败，请重试")
//		}
//
//		return nil
//	}
//
// // 检查邮箱是否在1分钟内已发送过验证码
//
//	func checkSendFrequency(email string) bool {
//		ctx := context.Background()
//		key := "send_limit:" + email
//		// 尝试设置key，过期时间1分钟，仅当key不存在时设置成功
//		ok, _ := redisClient.SetNX(ctx, key, 1, 1*time.Minute).Result()
//		return !ok // 返回true表示触发限流
//	}
type codeItem struct {
	Code     string    // 验证码内容（如"123456"）
	ExpireAt time.Time // 过期时间（5分钟后失效）
}

// 2. 全局并发安全Map：key=邮箱，value=codeItem（保证多goroutine安全访问）
var codeStore = sync.Map{}

// 3. 保存验证码到内存Map（5分钟过期）
func saveCodeToMap(email, code string) {
	// 计算过期时间：当前时间 + 5分钟
	expireAt := time.Now().Add(5 * time.Minute)
	// 存入Map：key=邮箱，value=包含验证码和过期时间的结构体
	codeStore.Store(email, codeItem{
		Code:     code,
		ExpireAt: expireAt,
	})
	fmt.Println(codeStore.Load(email))
}

// 4. 从内存Map验证验证码（返回nil表示验证通过）
func VerifyCodeFromMap(email, inputCode string) error {
	// 从Map中读取对应邮箱的验证码信息
	val, exists := codeStore.Load(email)
	if !exists {
		// 邮箱不存在：验证码未发送或已被删除
		fmt.Println(codeStore.Load(email))
		return errors.New("验证码无效")
	}

	// 类型断言：将val转为codeItem（确保类型正确）
	item, ok := val.(codeItem)
	if !ok {
		return errors.New("验证数据异常")
	}

	// 检查验证码是否过期
	if time.Now().After(item.ExpireAt) {
		// 过期后删除，避免无效数据占用内存
		codeStore.Delete(email)
		return errors.New("验证码已过期")
	}

	// 检查验证码是否匹配
	if item.Code != inputCode {
		return errors.New("验证码错误")
	}

	// 验证通过后删除：避免重复使用
	codeStore.Delete(email)
	return nil
}

// SaveMdFile 保存MD文件（对齐SaveAvatar逻辑）
func SaveMdFile(file io.Reader, fileName string, userID uint64) (string, error) {
	// 1. 存储目录配置（和UpdateAvatar的头像存储目录风格一致）
	baseDir := "./uploads/md"
	userDir := filepath.Join(baseDir, strconv.FormatUint(userID, 10))
	if err := os.MkdirAll(userDir, 0755); err != nil {
		return "", fmt.Errorf("创建MD存储目录失败：%w", err)
	}

	// 2. 生成唯一文件名（对齐UpdateAvatar的命名逻辑）
	uniqueID := uuid.New().String()[:8]
	timestamp := strconv.FormatInt(time.Now().Unix(), 10)
	dotIndex := strings.LastIndex(fileName, ".")
	ext := fileName[dotIndex:]
	baseName := fileName[:dotIndex]
	uniqueFileName := fmt.Sprintf("%s_%s_%s%s", baseName, timestamp, uniqueID, ext)
	filePath := filepath.Join(userDir, uniqueFileName)

	// 3. 写入文件（对齐SaveAvatar的文件写入逻辑）
	outFile, err := os.Create(filePath)
	if err != nil {
		return "", fmt.Errorf("创建MD文件失败：%w", err)
	}
	defer outFile.Close()

	if _, err := io.Copy(outFile, file); err != nil {
		return "", fmt.Errorf("写入MD文件失败：%w", err)
	}

	// 4. 生成访问URL（对齐SaveAvatar的URL生成逻辑）
	accessURL := fmt.Sprintf("http://localhost:8080/uploads/md/%d/%s", userID, uniqueFileName)
	return accessURL, nil
}

// DeleteMdFile 删除MD文件（对齐DeleteAvatar逻辑）
func DeleteMdFile(fileURL string) error {
	// 解析URL为本地路径（对齐DeleteAvatar的URL解析逻辑）
	// 示例：http://localhost:8080/uploads/md/123/xxx.md → ./uploads/md/123/xxx.md
	pathPrefix := "http://localhost:8080/uploads/md/"
	if !strings.HasPrefix(fileURL, pathPrefix) {
		return errors.New("MD文件URL格式非法")
	}
	localPath := "./uploads/md/" + fileURL[len(pathPrefix):]

	// 删除文件（对齐DeleteAvatar的删除逻辑）
	if err := os.Remove(localPath); err != nil {
		if os.IsNotExist(err) {
			return nil // 文件不存在，无需报错
		}
		return fmt.Errorf("删除MD文件失败：%w", err)
	}
	return nil
}
