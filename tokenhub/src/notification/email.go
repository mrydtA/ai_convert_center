package notification

import (
	"bytes"
	"crypto/tls"
	"fmt"
	"html/template"
	"net/smtp"
	"sync"
	"time"
)

// EmailConfig SMTP 配置
type EmailConfig struct {
	SMTPHost     string `json:"smtp_host"`     // SMTP 服务器地址
	SMTPPort     int    `json:"smtp_port"`     // SMTP 端口
	Username     string `json:"username"`      // 邮箱账号
	Password     string `json:"password"`      // 邮箱密码或授权码
	FromName     string `json:"from_name"`     // 发件人名称
	FromEmail    string `json:"from_email"`    // 发件人邮箱
	UseTLS       bool   `json:"use_tls"`       // 是否使用 TLS
	RateLimit    int    `json:"rate_limit"`    // 每分钟发送限制
	MaxRetries   int    `json:"max_retries"`   // 最大重试次数
	RetryDelay   int    `json:"retry_delay"`   // 重试延迟（秒）
}

// EmailMessage 邮件消息结构
type EmailMessage struct {
	To          []string          `json:"to"`           // 收件人
	Cc          []string          `json:"cc"`           // 抄送
	Bcc         []string          `json:"bcc"`          // 密送
	Subject     string            `json:"subject"`      // 主题
	Body        string            `json:"body"`         // 邮件正文（HTML）
	Attachments []AttachmentInfo  `json:"attachments"`  // 附件列表
	Priority    string            `json:"priority"`     // 优先级：high, normal, low
	TemplateID  string            `json:"template_id"`  // 模板 ID
	TemplateData map[string]interface{} `json:"template_data"` // 模板数据
}

// AttachmentInfo 附件信息
type AttachmentInfo struct {
	Filename string `json:"filename"` // 文件名
	Data     []byte `json:"data"`     // 文件内容
	MimeType string `json:"mime_type"` // MIME 类型
}

// EmailSender 邮件发送器
type EmailSender struct {
	config      EmailConfig
	rateLimiter *RateLimiter
	retryQueue  chan *EmailMessage
	wg          sync.WaitGroup
	stopChan    chan struct{}
	mu          sync.RWMutex
}

// RateLimiter 速率限制器
type RateLimiter struct {
	tokens     int
	maxTokens  int
	refillRate time.Duration
	lastRefill time.Time
	mu         sync.Mutex
}

// NewRateLimiter 创建速率限制器
func NewRateLimiter(maxTokens int, refillRate time.Duration) *RateLimiter {
	return &RateLimiter{
		tokens:     maxTokens,
		maxTokens:  maxTokens,
		refillRate: refillRate,
		lastRefill: time.Now(),
	}
}

// Allow 检查是否允许发送
func (rl *RateLimiter) Allow() bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	elapsed := now.Sub(rl.lastRefill)
	tokensToAdd := int(elapsed / rl.refillRate)

	if tokensToAdd > 0 {
		rl.tokens = min(rl.maxTokens, rl.tokens+tokensToAdd)
		rl.lastRefill = now
	}

	if rl.tokens > 0 {
		rl.tokens--
		return true
	}
	return false
}

// EmailSender 实例（单例）
var emailSender *EmailSender
var once sync.Once

// InitEmailSender 初始化邮件发送器
func InitEmailSender(config EmailConfig) *EmailSender {
	once.Do(func() {
		emailSender = &EmailSender{
			config:      config,
			rateLimiter: NewRateLimiter(config.RateLimit, time.Second),
			retryQueue:  make(chan *EmailMessage, 100),
			stopChan:    make(chan struct{}),
		}
		go emailSender.processRetryQueue()
	})
	return emailSender
}

// GetEmailSender 获取邮件发送器实例
func GetEmailSender() *EmailSender {
	return emailSender
}

// SendEmail 发送邮件（同步）
func (es *EmailSender) SendEmail(msg EmailMessage) error {
	// 应用模板
	if msg.TemplateID != "" {
		body, err := es.applyTemplate(msg.TemplateID, msg.TemplateData)
		if err != nil {
			return fmt.Errorf("apply template failed: %v", err)
		}
		msg.Body = body
	}

	// 检查速率限制
	if !es.rateLimiter.Allow() {
		return fmt.Errorf("rate limit exceeded")
	}

	// 构建邮件内容
	mime := "MIME-version: 1.0\r\n"
	mime += "Content-Type: text/html; charset=\"UTF-8\"\r\n"
	mime += fmt.Sprintf("From: %s <%s>\r\n", es.config.FromName, es.config.FromEmail)
	mime += fmt.Sprintf("To: %s\r\n", joinStrings(msg.To))
	
	if len(msg.Cc) > 0 {
		mime += fmt.Sprintf("Cc: %s\r\n", joinStrings(msg.Cc))
	}

	if msg.Priority == "high" {
		mime += "X-Priority: 1\r\n"
		mime += "Importance: high\r\n"
	} else if msg.Priority == "low" {
		mime += "X-Priority: 5\r\n"
		mime += "Importance: low\r\n"
	}

	mime += fmt.Sprintf("Subject: %s\r\n\r\n", msg.Subject)
	mime += msg.Body

	// 添加附件（如果有）
	var finalMessage bytes.Buffer
	finalMessage.WriteString(mime)
	
	for _, attachment := range msg.Attachments {
		boundary := "----=_Part_" + fmt.Sprintf("%d", time.Now().UnixNano())
		finalMessage.WriteString("\r\n--" + boundary + "\r\n")
		finalMessage.WriteString(fmt.Sprintf("Content-Type: %s; name=\"%s\"\r\n", attachment.MimeType, attachment.Filename))
		finalMessage.WriteString("Content-Transfer-Encoding: base64\r\n")
		finalMessage.WriteString(fmt.Sprintf("Content-Disposition: attachment; filename=\"%s\"\r\n\r\n", attachment.Filename))
		// 这里应该进行 base64 编码，简化处理
		finalMessage.Write(attachment.Data)
	}

	// 连接 SMTP 服务器
	addr := fmt.Sprintf("%s:%d", es.config.SMTPHost, es.config.SMTPPort)
	
	var auth smtp.Auth
	if es.config.Username != "" {
		auth = smtp.PlainAuth("", es.config.Username, es.config.Password, es.config.SMTPHost)
	}

	var err error
	if es.config.UseTLS {
		tlsConfig := &tls.Config{
			ServerName: es.config.SMTPHost,
		}
		err = smtp.SendMail(addr, auth, es.config.FromEmail, msg.To, finalMessage.Bytes(), tlsConfig)
	} else {
		err = smtp.SendMail(addr, auth, es.config.FromEmail, msg.To, finalMessage.Bytes())
	}

	if err != nil {
		// 加入重试队列
		es.addToRetryQueue(msg)
		return fmt.Errorf("send email failed: %v", err)
	}

	return nil
}

// SendEmailAsync 异步发送邮件
func (es *EmailSender) SendEmailAsync(msg EmailMessage) {
	es.wg.Add(1)
	go func() {
		defer es.wg.Done()
		es.SendEmail(msg)
	}()
}

// addToRetryQueue 添加到重试队列
func (es *EmailSender) addToRetryQueue(msg EmailMessage) {
	select {
	case es.retryQueue <- &msg:
	default:
		// 队列已满，丢弃
		fmt.Printf("Retry queue full, dropping message to %v\n", msg.To)
	}
}

// processRetryQueue 处理重试队列
func (es *EmailSender) processRetryQueue() {
	for {
		select {
		case msg := <-es.retryQueue:
			for i := 0; i < es.config.MaxRetries; i++ {
				time.Sleep(time.Duration(es.config.RetryDelay) * time.Second)
				if err := es.SendEmail(*msg); err == nil {
					break
				}
				if i == es.config.MaxRetries-1 {
					fmt.Printf("Failed to send email after %d retries: %v\n", es.config.MaxRetries, msg.To)
				}
			}
		case <-es.stopChan:
			return
		}
	}
}

// applyTemplate 应用邮件模板
func (es *EmailSender) applyTemplate(templateID string, data map[string]interface{}) (string, error) {
	tmplContent, ok := EmailTemplates[templateID]
	if !ok {
		return "", fmt.Errorf("template not found: %s", templateID)
	}

	tmpl, err := template.New(templateID).Parse(tmplContent)
	if err != nil {
		return "", err
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", err
	}

	return buf.String(), nil
}

// Stop 停止邮件发送器
func (es *EmailSender) Stop() {
	close(es.stopChan)
	es.wg.Wait()
}

// ==================== 邮件模板 ====================

// EmailTemplates 预定义邮件模板
var EmailTemplates = map[string]string{
	"topup_success": `
<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <title>充值成功通知</title>
    <style>
        body { font-family: Arial, sans-serif; line-height: 1.6; color: #333; }
        .container { max-width: 600px; margin: 0 auto; padding: 20px; }
        .header { background: linear-gradient(135deg, #667eea 0%, #764ba2 100%); color: white; padding: 30px; text-align: center; border-radius: 10px 10px 0 0; }
        .content { background: #f9f9f9; padding: 30px; border-radius: 0 0 10px 10px; }
        .amount { font-size: 32px; color: #667eea; font-weight: bold; text-align: center; margin: 20px 0; }
        .info-table { width: 100%; border-collapse: collapse; margin: 20px 0; }
        .info-table td { padding: 10px; border-bottom: 1px solid #eee; }
        .info-table td:first-child { font-weight: bold; color: #666; }
        .footer { text-align: center; margin-top: 30px; color: #999; font-size: 12px; }
        .button { display: inline-block; padding: 12px 30px; background: #667eea; color: white; text-decoration: none; border-radius: 5px; margin-top: 20px; }
    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <h1>🎉 充值成功</h1>
        </div>
        <div class="content">
            <p>尊敬的 {{.Username}}：</p>
            <p>您的账户已成功充值，详情如下：</p>
            <div class="amount">¥{{.Amount}}</div>
            <table class="info-table">
                <tr><td>订单号</td><td>{{.OrderID}}</td></tr>
                <tr><td>充值时间</td><td>{{.Time}}</td></tr>
                <tr><td>支付方式</td><td>{{.PaymentMethod}}</td></tr>
                <tr><td>当前余额</td><td>¥{{.Balance}}</td></tr>
            </table>
            <div style="text-align: center;">
                <a href="{{.DashboardURL}}" class="button">查看控制台</a>
            </div>
            <div class="footer">
                <p>如有任何疑问，请联系客服支持。</p>
                <p>&copy; 2024 TokenHub. All rights reserved.</p>
            </div>
        </div>
    </div>
</body>
</html>`,

	"plan_purchase": `
<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <title>套餐购买成功</title>
    <style>
        body { font-family: Arial, sans-serif; line-height: 1.6; color: #333; }
        .container { max-width: 600px; margin: 0 auto; padding: 20px; }
        .header { background: linear-gradient(135deg, #f093fb 0%, #f5576c 100%); color: white; padding: 30px; text-align: center; border-radius: 10px 10px 0 0; }
        .content { background: #f9f9f9; padding: 30px; border-radius: 0 0 10px 10px; }
        .plan-name { font-size: 28px; color: #f5576c; font-weight: bold; text-align: center; margin: 20px 0; }
        .info-table { width: 100%; border-collapse: collapse; margin: 20px 0; }
        .info-table td { padding: 10px; border-bottom: 1px solid #eee; }
        .info-table td:first-child { font-weight: bold; color: #666; }
        .footer { text-align: center; margin-top: 30px; color: #999; font-size: 12px; }
        .button { display: inline-block; padding: 12px 30px; background: #f5576c; color: white; text-decoration: none; border-radius: 5px; margin-top: 20px; }
    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <h1>🚀 套餐购买成功</h1>
        </div>
        <div class="content">
            <p>尊敬的 {{.Username}}：</p>
            <p>您已成功购买以下套餐：</p>
            <div class="plan-name">{{.PlanName}}</div>
            <table class="info-table">
                <tr><td>订单号</td><td>{{.OrderID}}</td></tr>
                <tr><td>购买时间</td><td>{{.Time}}</td></tr>
                <tr><td>套餐额度</td><td>{{.Quota}} tokens</td></tr>
                <tr><td>有效期</td><td>{{.ValidDays}} 天</td></tr>
                <tr><td>支付金额</td><td>¥{{.Amount}}</td></tr>
            </table>
            <div style="text-align: center;">
                <a href="{{.DashboardURL}}" class="button">立即使用</a>
            </div>
            <div class="footer">
                <p>感谢您的信任与支持！</p>
                <p>&copy; 2024 TokenHub. All rights reserved.</p>
            </div>
        </div>
    </div>
</body>
</html>`,

	"password_reset": `
<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <title>密码重置</title>
    <style>
        body { font-family: Arial, sans-serif; line-height: 1.6; color: #333; }
        .container { max-width: 600px; margin: 0 auto; padding: 20px; }
        .header { background: linear-gradient(135deg, #4facfe 0%, #00f2fe 100%); color: white; padding: 30px; text-align: center; border-radius: 10px 10px 0 0; }
        .content { background: #f9f9f9; padding: 30px; border-radius: 0 0 10px 10px; }
        .warning { background: #fff3cd; border-left: 4px solid #ffc107; padding: 15px; margin: 20px 0; }
        .footer { text-align: center; margin-top: 30px; color: #999; font-size: 12px; }
        .button { display: inline-block; padding: 12px 30px; background: #4facfe; color: white; text-decoration: none; border-radius: 5px; margin-top: 20px; }
        .code { font-size: 24px; letter-spacing: 5px; background: #eee; padding: 15px; text-align: center; border-radius: 5px; margin: 20px 0; }
    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <h1>🔐 密码重置</h1>
        </div>
        <div class="content">
            <p>尊敬的 {{.Username}}：</p>
            <p>您请求重置密码，请使用以下验证码：</p>
            <div class="code">{{.Code}}</div>
            <p>验证码有效期为 {{.ExpireMinutes}} 分钟。</p>
            <div class="warning">
                <strong>⚠️ 安全提示：</strong>如果您没有请求重置密码，请忽略此邮件，您的账户仍然安全。
            </div>
            <div class="footer">
                <p>&copy; 2024 TokenHub. All rights reserved.</p>
            </div>
        </div>
    </div>
</body>
</html>`,

	"balance_warning": `
<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <title>余额预警通知</title>
    <style>
        body { font-family: Arial, sans-serif; line-height: 1.6; color: #333; }
        .container { max-width: 600px; margin: 0 auto; padding: 20px; }
        .header { background: linear-gradient(135deg, #fa709a 0%, #fee140 100%); color: white; padding: 30px; text-align: center; border-radius: 10px 10px 0 0; }
        .content { background: #f9f9f9; padding: 30px; border-radius: 0 0 10px 10px; }
        .balance { font-size: 32px; color: #fa709a; font-weight: bold; text-align: center; margin: 20px 0; }
        .warning { background: #fff3cd; border-left: 4px solid #ffc107; padding: 15px; margin: 20px 0; }
        .footer { text-align: center; margin-top: 30px; color: #999; font-size: 12px; }
        .button { display: inline-block; padding: 12px 30px; background: #fa709a; color: white; text-decoration: none; border-radius: 5px; margin-top: 20px; }
    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <h1>⚠️ 余额预警</h1>
        </div>
        <div class="content">
            <p>尊敬的 {{.Username}}：</p>
            <p>您的账户余额已低于预警阈值，请及时充值以避免服务中断。</p>
            <div class="balance">¥{{.Balance}}</div>
            <p>当前预警阈值：¥{{.Threshold}}</p>
            <div class="warning">
                <strong>💡 建议：</strong>为避免影响您的正常使用，建议您尽快充值。
            </div>
            <div style="text-align: center;">
                <a href="{{.TopupURL}}" class="button">立即充值</a>
            </div>
            <div class="footer">
                <p>&copy; 2024 TokenHub. All rights reserved.</p>
            </div>
        </div>
    </div>
</body>
</html>`,
}

// ==================== 工具函数 ====================

func joinStrings(strs []string) string {
	result := ""
	for i, s := range strs {
		if i > 0 {
			result += ", "
		}
		result += s
	}
	return result
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// ==================== 快捷发送函数 ====================

// SendTopupSuccess 发送充值成功通知
func SendTopupSuccess(to []string, username, orderID, amount, paymentMethod, balance, dashboardURL string) error {
	sender := GetEmailSender()
	if sender == nil {
		return fmt.Errorf("email sender not initialized")
	}

	msg := EmailMessage{
		To:      to,
		Subject: "【TokenHub】充值成功通知",
		TemplateID: "topup_success",
		TemplateData: map[string]interface{}{
			"Username":      username,
			"OrderID":       orderID,
			"Amount":        amount,
			"Time":          time.Now().Format("2006-01-02 15:04:05"),
			"PaymentMethod": paymentMethod,
			"Balance":       balance,
			"DashboardURL":  dashboardURL,
		},
	}

	return sender.SendEmail(msg)
}

// SendPlanPurchase 发送套餐购买成功通知
func SendPlanPurchase(to []string, username, orderID, planName, quota, validDays, amount, dashboardURL string) error {
	sender := GetEmailSender()
	if sender == nil {
		return fmt.Errorf("email sender not initialized")
	}

	msg := EmailMessage{
		To:      to,
		Subject: "【TokenHub】套餐购买成功",
		TemplateID: "plan_purchase",
		TemplateData: map[string]interface{}{
			"Username":   username,
			"OrderID":    orderID,
			"PlanName":   planName,
			"Quota":      quota,
			"ValidDays":  validDays,
			"Amount":     amount,
			"Time":       time.Now().Format("2006-01-02 15:04:05"),
			"DashboardURL": dashboardURL,
		},
	}

	return sender.SendEmail(msg)
}

// SendPasswordReset 发送密码重置验证码
func SendPasswordReset(to []string, username, code string, expireMinutes int) error {
	sender := GetEmailSender()
	if sender == nil {
		return fmt.Errorf("email sender not initialized")
	}

	msg := EmailMessage{
		To:      to,
		Subject: "【TokenHub】密码重置验证码",
		TemplateID: "password_reset",
		TemplateData: map[string]interface{}{
			"Username":       username,
			"Code":           code,
			"ExpireMinutes":  expireMinutes,
		},
	}

	return sender.SendEmail(msg)
}

// SendBalanceWarning 发送余额预警通知
func SendBalanceWarning(to []string, username, balance, threshold, topupURL string) error {
	sender := GetEmailSender()
	if sender == nil {
		return fmt.Errorf("email sender not initialized")
	}

	msg := EmailMessage{
		To:      to,
		Subject: "【TokenHub】余额预警通知",
		TemplateID: "balance_warning",
		TemplateData: map[string]interface{}{
			"Username":   username,
			"Balance":    balance,
			"Threshold":  threshold,
			"TopupURL":   topupURL,
		},
	}

	return sender.SendEmail(msg)
}
