package payment

import (
	"context"
	"crypto"
	"crypto/hmac"
	"crypto/md5"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
	_ "github.com/go-sql-driver/mysql"
	"github.com/jmoiron/sqlx"
)

// PaymentMethod 支付方式
type PaymentMethod string

const (
	PaymentMethodAlipay  PaymentMethod = "alipay"
	PaymentMethodWechat  PaymentMethod = "wechat"
	PaymentMethodPayspi  PaymentMethod = "payspi"
	PaymentMethodManual  PaymentMethod = "manual"
)

// OrderStatus 订单状态
type OrderStatus string

const (
	OrderStatusPending   OrderStatus = "pending"
	OrderStatusCompleted OrderStatus = "completed"
	OrderStatusFailed    OrderStatus = "failed"
	OrderStatusRefunded  OrderStatus = "refunded"
)

// TopupOrder 充值订单
type TopupOrder struct {
	ID            int64           `json:"id"`
	UserID        int             `json:"user_id"`
	OrderNo       string          `json:"order_no"`
	Amount        float64         `json:"amount"`
	PaymentMethod PaymentMethod   `json:"payment_method"`
	Status        OrderStatus     `json:"status"`
	TransactionID string          `json:"transaction_id,omitempty"`
	CreatedAt     time.Time       `json:"created_at"`
	UpdatedAt     time.Time       `json:"updated_at"`
}

// CreateTopupRequest 创建充值请求
type CreateTopupRequest struct {
	Amount        float64       `json:"amount" binding:"required,min=1"`
	PaymentMethod PaymentMethod `json:"payment_method" binding:"required,oneof=alipay wechat payspi"`
}

// PaymentResponse 支付响应
type PaymentResponse struct {
	OrderNo      string                 `json:"order_no"`
	PayURL       string                 `json:"pay_url,omitempty"`
	QRCode       string                 `json:"qr_code,omitempty"`
	FormData     map[string]string      `json:"form_data,omitempty"`
	ExtraData    map[string]interface{} `json:"extra_data,omitempty"`
	ExpireTime   int64                  `json:"expire_time"`
}

// PaymentService 支付服务接口
type PaymentService interface {
	// CreateOrder 创建订单
	CreateOrder(ctx context.Context, userID int, amount float64) (*TopupOrder, error)
	
	// Pay 发起支付
	Pay(ctx context.Context, orderNo string, method PaymentMethod) (*PaymentResponse, error)
	
	// HandleCallback 处理支付回调
	HandleCallback(ctx context.Context, method PaymentMethod, data interface{}) error
	
	// QueryOrder 查询订单状态
	QueryOrder(ctx context.Context, orderNo string) (*TopupOrder, error)
	
	// Refund 退款
	Refund(ctx context.Context, orderNo string, reason string) error
}

// AlipayConfig 支付宝配置
type AlipayConfig struct {
	AppID         string `json:"app_id"`
	PrivateKey    string `json:"private_key"`
	PublicKey     string `json:"public_key"`
	GatewayURL    string `json:"gateway_url"`
	NotifyURL     string `json:"notify_url"`
	ReturnURL     string `json:"return_url"`
}

// WechatConfig 微信支付配置
type WechatConfig struct {
	AppID     string `json:"app_id"`
	MchID     string `json:"mch_id"`
	APIKey    string `json:"api_key"`
	NotifyURL string `json:"notify_url"`
	CertPath  string `json:"cert_path"`
	KeyPath   string `json:"key_path"`
}

// PayspiConfig 聚合支付配置
type PayspiConfig struct {
	APIKey        string `json:"api_key"`
	WebhookSecret string `json:"webhook_secret"`
	GatewayURL    string `json:"gateway_url"`
}

// PaymentGateway 支付网关
type PaymentGateway struct {
	alipayConfig  *AlipayConfig
	wechatConfig  *WechatConfig
	payspiConfig  *PayspiConfig
	db            *sqlx.DB
	redis         *redis.Client
}

// NewPaymentGateway 创建支付网关
func NewPaymentGateway(
	alipayCfg *AlipayConfig,
	wechatCfg *WechatConfig,
	payspiCfg *PayspiConfig,
	db *sqlx.DB,
	redisClient *redis.Client,
) *PaymentGateway {
	return &PaymentGateway{
		alipayConfig:  alipayCfg,
		wechatConfig:  wechatCfg,
		payspiConfig:  payspiCfg,
		db:            db,
		redis:         redisClient,
	}
}

// CreateOrder 创建充值订单
func (pg *PaymentGateway) CreateOrder(ctx context.Context, userID int, amount float64) (*TopupOrder, error) {
	orderNo := generateOrderNo()
	
	// 开启事务
	tx, err := pg.db.Beginx()
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()
	
	// 插入订单
	query := `
		INSERT INTO topup_orders 
		(user_id, order_no, amount, payment_method, status, created_at, updated_at)
		VALUES (?, ?, ?, '', 'pending', NOW(), NOW())
	`
	result, err := tx.ExecContext(ctx, query, userID, orderNo, amount)
	if err != nil {
		return nil, fmt.Errorf("failed to insert order: %w", err)
	}
	
	orderID, err := result.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("failed to get order id: %w", err)
	}
	
	// 查询刚创建的订单
	order := &TopupOrder{}
	err = tx.GetContext(ctx, order, `
		SELECT id, user_id, order_no, amount, payment_method, status, 
		       transaction_id, created_at, updated_at
		FROM topup_orders
		WHERE id = ?
	`, orderID)
	if err != nil {
		return nil, fmt.Errorf("failed to query order: %w", err)
	}
	
	// 提交事务
	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}
	
	// 缓存订单信息到 Redis，30 分钟过期
	cacheKey := fmt.Sprintf("order:%s", orderNo)
	orderJSON, _ := json.Marshal(order)
	pg.redis.SetEX(ctx, cacheKey, string(orderJSON), 30*time.Minute)
	
	return order, nil
}

// Pay 发起支付
func (pg *PaymentGateway) Pay(ctx context.Context, orderNo string, method PaymentMethod) (*PaymentResponse, error) {
	switch method {
	case PaymentMethodAlipay:
		return pg.alipayPay(ctx, orderNo)
	case PaymentMethodWechat:
		return pg.wechatPay(ctx, orderNo)
	case PaymentMethodPayspi:
		return pg.payspiPay(ctx, orderNo)
	default:
		return nil, errors.New("unsupported payment method")
	}
}

// HandleCallback 处理支付回调
func (pg *PaymentGateway) HandleCallback(ctx context.Context, method PaymentMethod, data interface{}) error {
	switch method {
	case PaymentMethodAlipay:
		return pg.handleAlipayCallback(ctx, data)
	case PaymentMethodWechat:
		return pg.handleWechatCallback(ctx, data)
	case PaymentMethodPayspi:
		return pg.handlePayspiCallback(ctx, data)
	default:
		return errors.New("unsupported payment method")
	}
}

// QueryOrder 查询订单状态
func (pg *PaymentGateway) QueryOrder(ctx context.Context, orderNo string) (*TopupOrder, error) {
	// 先从 Redis 缓存中查找
	cacheKey := fmt.Sprintf("order:%s", orderNo)
	cachedData, err := pg.redis.Get(ctx, cacheKey).Result()
	if err == nil && cachedData != "" {
		order := &TopupOrder{}
		if json.Unmarshal([]byte(cachedData), order) == nil {
			return order, nil
		}
	}
	
	// 从数据库查询
	order := &TopupOrder{}
	err = pg.db.GetContext(ctx, order, `
		SELECT id, user_id, order_no, amount, payment_method, status, 
		       transaction_id, created_at, updated_at
		FROM topup_orders
		WHERE order_no = ?
	`, orderNo)
	if err != nil {
		return nil, fmt.Errorf("order not found: %w", err)
	}
	
	// 更新缓存
	orderJSON, _ := json.Marshal(order)
	pg.redis.SetEX(ctx, cacheKey, string(orderJSON), 30*time.Minute)
	
	return order, nil
}

// Refund 退款
func (pg *PaymentGateway) Refund(ctx context.Context, orderNo string, reason string) error {
	// 查询订单
	order, err := pg.QueryOrder(ctx, orderNo)
	if err != nil {
		return err
	}
	
	if order.Status != OrderStatusCompleted {
		return errors.New("only completed orders can be refunded")
	}
	
	// 开启事务
	tx, err := pg.db.Beginx()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()
	
	// 根据支付方式调用相应的退款 API
	switch order.PaymentMethod {
	case PaymentMethodAlipay:
		if err := pg.alipayRefund(ctx, orderNo, order.Amount, reason); err != nil {
			return err
		}
	case PaymentMethodWechat:
		if err := pg.wechatRefund(ctx, orderNo, order.Amount, reason); err != nil {
			return err
		}
	case PaymentMethodPayspi:
		if err := pg.payspiRefund(ctx, orderNo, order.Amount, reason); err != nil {
			return err
		}
	default:
		return errors.New("unsupported payment method for refund")
	}
	
	// 更新订单状态为已退款
	_, err = tx.ExecContext(ctx, `
		UPDATE topup_orders 
		SET status = 'refunded', updated_at = NOW()
		WHERE order_no = ?
	`, orderNo)
	if err != nil {
		return fmt.Errorf("failed to update order status: %w", err)
	}
	
	// 扣除用户余额
	_, err = tx.ExecContext(ctx, `
		UPDATE users 
		SET balance = balance - ?
		WHERE id = ?
	`, order.Amount, order.UserID)
	if err != nil {
		return fmt.Errorf("failed to deduct user balance: %w", err)
	}
	
	// 记录退款流水
	_, err = tx.ExecContext(ctx, `
		INSERT INTO payment_flows 
		(order_no, flow_type, amount, status, remark, created_at)
		VALUES (?, 'refund', ?, 'completed', ?, NOW())
	`, orderNo, order.Amount, reason)
	if err != nil {
		return fmt.Errorf("failed to record refund flow: %w", err)
	}
	
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}
	
	// 删除缓存
	cacheKey := fmt.Sprintf("order:%s", orderNo)
	pg.redis.Del(ctx, cacheKey)
	
	return nil
}

// 支付宝支付实现
func (pg *PaymentGateway) alipayPay(ctx context.Context, orderNo string) (*PaymentResponse, error) {
	if pg.alipayConfig == nil {
		return nil, errors.New("alipay not configured")
	}
	
	// 查询订单
	order, err := pg.QueryOrder(ctx, orderNo)
	if err != nil {
		return nil, err
	}
	
	// 构建支付宝当面付请求参数
	params := url.Values{}
	params.Set("app_id", pg.alipayConfig.AppID)
	params.Set("method", "alipay.trade.precreate")
	params.Set("format", "JSON")
	params.Set("charset", "utf-8")
	params.Set("sign_type", "RSA2")
	params.Set("timestamp", time.Now().Format("2006-01-02 15:04:05"))
	params.Set("version", "1.0")
	params.Set("notify_url", pg.alipayConfig.NotifyURL)
	
	// 业务参数
	bizContent := map[string]interface{}{
		"out_trade_no": orderNo,
		"total_amount": fmt.Sprintf("%.2f", order.Amount),
		"subject":      "TokenHub 充值",
		"product_code": "FACE_TO_FACE_PAYMENT",
	}
	bizContentJSON, _ := json.Marshal(bizContent)
	params.Set("biz_content", string(bizContentJSON))
	
	// 生成签名
	sign := pg.generateAlipaySign(params)
	params.Set("sign", sign)
	
	// 调用支付宝 API
	alipayURL := pg.alipayConfig.GatewayURL
	if alipayURL == "" {
		alipayURL = "https://openapi.alipay.com/gateway.do"
	}
	
	reqURL := alipayURL + "?" + params.Encode()
	resp, err := http.Get(reqURL)
	if err != nil {
		return nil, fmt.Errorf("failed to call alipay API: %w", err)
	}
	defer resp.Body.Close()
	
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read alipay response: %w", err)
	}
	
	// 解析响应
	var result struct {
		AlipayTradePrecreateResponse struct {
			Code    string `json:"code"`
			QRCode  string `json:"qr_code"`
			Msg     string `json:"msg"`
			SubMsg  string `json:"sub_msg"`
		} `json:"alipay_trade_precreate_response"`
		Sign string `json:"sign"`
	}
	
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("failed to parse alipay response: %w", err)
	}
	
	if result.AlipayTradePrecreateResponse.Code != "10000" {
		return nil, fmt.Errorf("alipay error: %s - %s", 
			result.AlipayTradePrecreateResponse.Msg,
			result.AlipayTradePrecreateResponse.SubMsg)
	}
	
	// 更新订单支付方式
	pg.db.ExecContext(ctx, `
		UPDATE topup_orders 
		SET payment_method = 'alipay', updated_at = NOW()
		WHERE order_no = ?
	`, orderNo)
	
	return &PaymentResponse{
		OrderNo:    orderNo,
		QRCode:     result.AlipayTradePrecreateResponse.QRCode,
		ExpireTime: time.Now().Add(30 * time.Minute).Unix(),
	}, nil
}

// 微信支付实现
func (pg *PaymentGateway) wechatPay(ctx context.Context, orderNo string) (*PaymentResponse, error) {
	if pg.wechatConfig == nil {
		return nil, errors.New("wechat not configured")
	}
	
	// 查询订单
	order, err := pg.QueryOrder(ctx, orderNo)
	if err != nil {
		return nil, err
	}
	
	// 构建微信支付 Native 下单参数
	unifiedOrderURL := "https://api.mch.weixin.qq.com/v3/pay/transactions/native"
	
	// 请求体
	requestBody := map[string]interface{}{
		"appid":       pg.wechatConfig.AppID,
		"mchid":       pg.wechatConfig.MchID,
		"description": "TokenHub 充值",
		"out_trade_no": orderNo,
		"notify_url":   pg.wechatConfig.NotifyURL,
		"amount": map[string]interface{}{
			"total":    int(order.Amount * 100), // 单位：分
			"currency": "CNY",
		},
	}
	
	bodyJSON, _ := json.Marshal(requestBody)
	
	// 创建 HTTP 请求
	req, err := http.NewRequestWithContext(ctx, "POST", unifiedOrderURL, strings.NewReader(string(bodyJSON)))
	if err != nil {
		return nil, fmt.Errorf("failed to create wechat request: %w", err)
	}
	
	// 设置请求头
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	
	// TODO: 添加微信支付 v3 的签名和证书认证
	// 需要加载商户私钥和证书
	
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to call wechat API: %w", err)
	}
	defer resp.Body.Close()
	
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read wechat response: %w", err)
	}
	
	// 解析响应
	var result struct {
		CodeURL    string `json:"code_url"`
		PrepayID   string `json:"prepay_id"`
		ErrCode    string `json:"error_code"`
		ErrMsg     string `json:"error_message"`
	}
	
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("failed to parse wechat response: %w", err)
	}
	
	if result.ErrCode != "" {
		return nil, fmt.Errorf("wechat error: %s - %s", result.ErrCode, result.ErrMsg)
	}
	
	// 更新订单支付方式
	pg.db.ExecContext(ctx, `
		UPDATE topup_orders 
		SET payment_method = 'wechat', updated_at = NOW()
		WHERE order_no = ?
	`, orderNo)
	
	return &PaymentResponse{
		OrderNo:    orderNo,
		QRCode:     result.CodeURL,
		ExtraData:  map[string]interface{}{"prepay_id": result.PrepayID},
		ExpireTime: time.Now().Add(30 * time.Minute).Unix(),
	}, nil
}

// 聚合支付实现
func (pg *PaymentGateway) payspiPay(ctx context.Context, orderNo string) (*PaymentResponse, error) {
	if pg.payspiConfig == nil {
		return nil, errors.New("payspi not configured")
	}
	
	// 查询订单
	order, err := pg.QueryOrder(ctx, orderNo)
	if err != nil {
		return nil, err
	}
	
	// 构建 Payspi 支付请求
	gatewayURL := pg.payspiConfig.GatewayURL
	if gatewayURL == "" {
		gatewayURL = "https://api.payspi.com/v1/payment"
	}
	
	requestBody := map[string]interface{}{
		"order_id":   orderNo,
		"amount":     order.Amount,
		"currency":   "CNY",
		"notify_url": pg.payspiConfig.WebhookSecret,
		"return_url": pg.alipayConfig.ReturnURL,
	}
	
	bodyJSON, _ := json.Marshal(requestBody)
	
	// 生成签名
	signature := pg.generatePayspiSign(requestBody)
	
	req, err := http.NewRequestWithContext(ctx, "POST", gatewayURL, strings.NewReader(string(bodyJSON)))
	if err != nil {
		return nil, fmt.Errorf("failed to create payspi request: %w", err)
	}
	
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-API-Key", pg.payspiConfig.APIKey)
	req.Header.Set("X-Signature", signature)
	
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to call payspi API: %w", err)
	}
	defer resp.Body.Close()
	
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read payspi response: %w", err)
	}
	
	// 解析响应
	var result struct {
		PaymentURL string `json:"payment_url"`
		QRCode     string `json:"qr_code"`
		Status     string `json:"status"`
		Error      string `json:"error"`
	}
	
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("failed to parse payspi response: %w", err)
	}
	
	if result.Error != "" {
		return nil, fmt.Errorf("payspi error: %s", result.Error)
	}
	
	return &PaymentResponse{
		OrderNo:    orderNo,
		PayURL:     result.PaymentURL,
		QRCode:     result.QRCode,
		ExpireTime: time.Now().Add(30 * time.Minute).Unix(),
	}, nil
}

// 处理支付宝回调
func (pg *PaymentGateway) handleAlipayCallback(ctx context.Context, data interface{}) error {
	formData, ok := data.(map[string]string)
	if !ok {
		return errors.New("invalid alipay callback data")
	}
	
	// 验证签名
	if !pg.verifyAlipaySign(formData) {
		return errors.New("invalid alipay signature")
	}
	
	// 获取订单信息
	orderNo := formData["out_trade_no"]
	tradeNo := formData["trade_no"]
	tradeStatus := formData["trade_status"]
	totalAmount := formData["total_amount"]
	
	// 只处理交易成功的回调
	if tradeStatus != "TRADE_SUCCESS" && tradeStatus != "TRADE_FINISHED" {
		return nil
	}
	
	// 开启事务
	tx, err := pg.db.Beginx()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()
	
	// 查询订单
	order := &TopupOrder{}
	err = tx.GetContext(ctx, order, `
		SELECT id, user_id, order_no, amount, payment_method, status, 
		       transaction_id, created_at, updated_at
		FROM topup_orders
		WHERE order_no = ?
	`, orderNo)
	if err != nil {
		return fmt.Errorf("order not found: %w", err)
	}
	
	// 检查订单是否已处理
	if order.Status == OrderStatusCompleted {
		return nil // 幂等处理
	}
	
	// 验证金额
	parsedAmount, _ := strconv.ParseFloat(totalAmount, 64)
	if parsedAmount != order.Amount {
		return fmt.Errorf("amount mismatch: expected %.2f, got %s", order.Amount, totalAmount)
	}
	
	// 更新订单状态
	_, err = tx.ExecContext(ctx, `
		UPDATE topup_orders 
		SET status = 'completed', transaction_id = ?, updated_at = NOW()
		WHERE order_no = ?
	`, tradeNo, orderNo)
	if err != nil {
		return fmt.Errorf("failed to update order status: %w", err)
	}
	
	// 增加用户余额
	_, err = tx.ExecContext(ctx, `
		UPDATE users 
		SET balance = balance + ?
		WHERE id = ?
	`, order.Amount, order.UserID)
	if err != nil {
		return fmt.Errorf("failed to update user balance: %w", err)
	}
	
	// 记录支付流水
	_, err = tx.ExecContext(ctx, `
		INSERT INTO payment_flows 
		(order_no, flow_type, amount, status, transaction_id, remark, created_at)
		VALUES (?, 'payment', ?, 'completed', ?, 'alipay', NOW())
	`, orderNo, order.Amount, tradeNo)
	if err != nil {
		return fmt.Errorf("failed to record payment flow: %w", err)
	}
	
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}
	
	// 删除缓存
	cacheKey := fmt.Sprintf("order:%s", orderNo)
	pg.redis.Del(ctx, cacheKey)
	
	return nil
}

// 处理微信支付回调
func (pg *PaymentGateway) handleWechatCallback(ctx context.Context, data interface{}) error {
	callbackData, ok := data.(map[string]interface{})
	if !ok {
		return errors.New("invalid wechat callback data")
	}
	
	// TODO: 验证微信支付签名
	
	// 获取资源数据
	resource, ok := callbackData["resource"].(map[string]interface{})
	if !ok {
		return errors.New("invalid wechat resource data")
	}
	
	// 解密数据 (微信支付 v3 使用 AES-256-GCM 加密)
	// TODO: 实现解密逻辑
	
	// 获取订单信息
	outTradeNo := ""
	if v, ok := resource["out_trade_no"].(string); ok {
		outTradeNo = v
	}
	
	tradeState := ""
	if v, ok := resource["trade_state"].(string); ok {
		tradeState = v
	}
	
	transactionID := ""
	if v, ok := resource["transaction_id"].(string); ok {
		transactionID = v
	}
	
	// 只处理成功状态
	if tradeState != "SUCCESS" {
		return nil
	}
	
	// 开启事务
	tx, err := pg.db.Beginx()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()
	
	// 查询订单
	order := &TopupOrder{}
	err = tx.GetContext(ctx, order, `
		SELECT id, user_id, order_no, amount, payment_method, status, 
		       transaction_id, created_at, updated_at
		FROM topup_orders
		WHERE order_no = ?
	`, outTradeNo)
	if err != nil {
		return fmt.Errorf("order not found: %w", err)
	}
	
	// 检查订单是否已处理
	if order.Status == OrderStatusCompleted {
		return nil
	}
	
	// 更新订单状态
	_, err = tx.ExecContext(ctx, `
		UPDATE topup_orders 
		SET status = 'completed', transaction_id = ?, updated_at = NOW()
		WHERE order_no = ?
	`, transactionID, outTradeNo)
	if err != nil {
		return fmt.Errorf("failed to update order status: %w", err)
	}
	
	// 增加用户余额
	_, err = tx.ExecContext(ctx, `
		UPDATE users 
		SET balance = balance + ?
		WHERE id = ?
	`, order.Amount, order.UserID)
	if err != nil {
		return fmt.Errorf("failed to update user balance: %w", err)
	}
	
	// 记录支付流水
	_, err = tx.ExecContext(ctx, `
		INSERT INTO payment_flows 
		(order_no, flow_type, amount, status, transaction_id, remark, created_at)
		VALUES (?, 'payment', ?, 'completed', ?, 'wechat', NOW())
	`, outTradeNo, order.Amount, transactionID)
	if err != nil {
		return fmt.Errorf("failed to record payment flow: %w", err)
	}
	
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}
	
	// 删除缓存
	cacheKey := fmt.Sprintf("order:%s", outTradeNo)
	pg.redis.Del(ctx, cacheKey)
	
	return nil
}

// 处理聚合支付回调
func (pg *PaymentGateway) handlePayspiCallback(ctx context.Context, data interface{}) error {
	callbackData, ok := data.(map[string]interface{})
	if !ok {
		return errors.New("invalid payspi callback data")
	}
	
	// 验证签名
	if !pg.verifyPayspiSign(callbackData) {
		return errors.New("invalid payspi signature")
	}
	
	// 获取订单信息
	orderNo := ""
	if v, ok := callbackData["order_id"].(string); ok {
		orderNo = v
	}
	
	status := ""
	if v, ok := callbackData["status"].(string); ok {
		status = v
	}
	
	transactionID := ""
	if v, ok := callbackData["transaction_id"].(string); ok {
		transactionID = v
	}
	
	// 只处理成功状态
	if status != "success" && status != "completed" {
		return nil
	}
	
	// 开启事务
	tx, err := pg.db.Beginx()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()
	
	// 查询订单
	order := &TopupOrder{}
	err = tx.GetContext(ctx, order, `
		SELECT id, user_id, order_no, amount, payment_method, status, 
		       transaction_id, created_at, updated_at
		FROM topup_orders
		WHERE order_no = ?
	`, orderNo)
	if err != nil {
		return fmt.Errorf("order not found: %w", err)
	}
	
	// 检查订单是否已处理
	if order.Status == OrderStatusCompleted {
		return nil
	}
	
	// 更新订单状态
	_, err = tx.ExecContext(ctx, `
		UPDATE topup_orders 
		SET status = 'completed', transaction_id = ?, updated_at = NOW()
		WHERE order_no = ?
	`, transactionID, orderNo)
	if err != nil {
		return fmt.Errorf("failed to update order status: %w", err)
	}
	
	// 增加用户余额
	_, err = tx.ExecContext(ctx, `
		UPDATE users 
		SET balance = balance + ?
		WHERE id = ?
	`, order.Amount, order.UserID)
	if err != nil {
		return fmt.Errorf("failed to update user balance: %w", err)
	}
	
	// 记录支付流水
	_, err = tx.ExecContext(ctx, `
		INSERT INTO payment_flows 
		(order_no, flow_type, amount, status, transaction_id, remark, created_at)
		VALUES (?, 'payment', ?, 'completed', ?, 'payspi', NOW())
	`, orderNo, order.Amount, transactionID)
	if err != nil {
		return fmt.Errorf("failed to record payment flow: %w", err)
	}
	
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}
	
	// 删除缓存
	cacheKey := fmt.Sprintf("order:%s", orderNo)
	pg.redis.Del(ctx, cacheKey)
	
	return nil
}

// 支付宝退款实现
func (pg *PaymentGateway) alipayRefund(ctx context.Context, orderNo string, amount float64, reason string) error {
	if pg.alipayConfig == nil {
		return errors.New("alipay not configured")
	}
	
	// 构建支付宝退款请求参数
	params := url.Values{}
	params.Set("app_id", pg.alipayConfig.AppID)
	params.Set("method", "alipay.trade.refund")
	params.Set("format", "JSON")
	params.Set("charset", "utf-8")
	params.Set("sign_type", "RSA2")
	params.Set("timestamp", time.Now().Format("2006-01-02 15:04:05"))
	params.Set("version", "1.0")
	
	// 业务参数
	bizContent := map[string]interface{}{
		"out_trade_no": orderNo,
		"refund_amount": fmt.Sprintf("%.2f", amount),
		"refund_reason": reason,
	}
	bizContentJSON, _ := json.Marshal(bizContent)
	params.Set("biz_content", string(bizContentJSON))
	
	// 生成签名
	sign := pg.generateAlipaySign(params)
	params.Set("sign", sign)
	
	// 调用支付宝退款 API
	alipayURL := pg.alipayConfig.GatewayURL
	if alipayURL == "" {
		alipayURL = "https://openapi.alipay.com/gateway.do"
	}
	
	reqURL := alipayURL + "?" + params.Encode()
	resp, err := http.Get(reqURL)
	if err != nil {
		return fmt.Errorf("failed to call alipay refund API: %w", err)
	}
	defer resp.Body.Close()
	
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read alipay response: %w", err)
	}
	
	// 解析响应
	var result struct {
		AlipayTradeRefundResponse struct {
			Code   string `json:"code"`
			Msg    string `json:"msg"`
			SubMsg string `json:"sub_msg"`
		} `json:"alipay_trade_refund_response"`
	}
	
	if err := json.Unmarshal(body, &result); err != nil {
		return fmt.Errorf("failed to parse alipay refund response: %w", err)
	}
	
	if result.AlipayTradeRefundResponse.Code != "10000" {
		return fmt.Errorf("alipay refund error: %s - %s", 
			result.AlipayTradeRefundResponse.Msg,
			result.AlipayTradeRefundResponse.SubMsg)
	}
	
	return nil
}

// 微信支付退款实现
func (pg *PaymentGateway) wechatRefund(ctx context.Context, orderNo string, amount float64, reason string) error {
	if pg.wechatConfig == nil {
		return errors.New("wechat not configured")
	}
	
	// 构建微信支付退款请求
	refundURL := "https://api.mch.weixin.qq.com/v3/refund/domestic/refunds"
	
	requestBody := map[string]interface{}{
		"out_trade_no": orderNo,
		"out_refund_no": "R" + orderNo,
		"reason": reason,
		"amount": map[string]interface{}{
			"refund": int(amount * 100), // 单位：分
			"total":  int(amount * 100),
			"currency": "CNY",
		},
		"notify_url": pg.wechatConfig.NotifyURL,
	}
	
	bodyJSON, _ := json.Marshal(requestBody)
	
	req, err := http.NewRequestWithContext(ctx, "POST", refundURL, strings.NewReader(string(bodyJSON)))
	if err != nil {
		return fmt.Errorf("failed to create wechat refund request: %w", err)
	}
	
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	
	// TODO: 添加微信支付 v3 的签名和证书认证
	
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to call wechat refund API: %w", err)
	}
	defer resp.Body.Close()
	
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read wechat refund response: %w", err)
	}
	
	// 解析响应
	var result struct {
		ErrCode string `json:"error_code"`
		ErrMsg  string `json:"error_message"`
	}
	
	if err := json.Unmarshal(respBody, &result); err != nil {
		return fmt.Errorf("failed to parse wechat refund response: %w", err)
	}
	
	if result.ErrCode != "" {
		return fmt.Errorf("wechat refund error: %s - %s", result.ErrCode, result.ErrMsg)
	}
	
	return nil
}

// 聚合支付退款实现
func (pg *PaymentGateway) payspiRefund(ctx context.Context, orderNo string, amount float64, reason string) error {
	if pg.payspiConfig == nil {
		return errors.New("payspi not configured")
	}
	
	gatewayURL := pg.payspiConfig.GatewayURL
	if gatewayURL == "" {
		gatewayURL = "https://api.payspi.com/v1/refund"
	}
	
	requestBody := map[string]interface{}{
		"order_id": orderNo,
		"refund_id": "R" + orderNo,
		"amount": amount,
		"reason": reason,
	}
	
	bodyJSON, _ := json.Marshal(requestBody)
	signature := pg.generatePayspiSign(requestBody)
	
	req, err := http.NewRequestWithContext(ctx, "POST", gatewayURL, strings.NewReader(string(bodyJSON)))
	if err != nil {
		return fmt.Errorf("failed to create payspi refund request: %w", err)
	}
	
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-API-Key", pg.payspiConfig.APIKey)
	req.Header.Set("X-Signature", signature)
	
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to call payspi refund API: %w", err)
	}
	defer resp.Body.Close()
	
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read payspi refund response: %w", err)
	}
	
	var result struct {
		Status string `json:"status"`
		Error  string `json:"error"`
	}
	
	if err := json.Unmarshal(respBody, &result); err != nil {
		return fmt.Errorf("failed to parse payspi refund response: %w", err)
	}
	
	if result.Error != "" {
		return fmt.Errorf("payspi refund error: %s", result.Error)
	}
	
	return nil
}

// 生成支付宝签名
func (pg *PaymentGateway) generateAlipaySign(params url.Values) string {
	// 对参数排序
	keys := make([]string, 0, len(params))
	for k := range params {
		if k != "sign" && k != "sign_type" {
			keys = append(keys, k)
		}
	}
	sort.Strings(keys)
	
	// 拼接签名字符串
	var signStr strings.Builder
	for i, k := range keys {
		if i > 0 {
			signStr.WriteString("&")
		}
		signStr.WriteString(k)
		signStr.WriteString("=")
		signStr.WriteString(params.Get(k))
	}
	
	// RSA2 签名
	privateKey, err := parsePrivateKey(pg.alipayConfig.PrivateKey)
	if err != nil {
		return ""
	}
	
	h := crypto.Hash.New(crypto.SHA256)
	h.Write([]byte(signStr.String()))
	hashed := h.Sum(nil)
	
	signature, err := rsa.SignPKCS1v15(rand.Reader, privateKey, crypto.SHA256, hashed)
	if err != nil {
		return ""
	}
	
	return base64.StdEncoding.EncodeToString(signature)
}

// 验证支付宝签名
func (pg *PaymentGateway) verifyAlipaySign(params map[string]string) bool {
	sign, ok := params["sign"]
	if !ok {
		return false
	}
	
	// 对参数排序
	keys := make([]string, 0, len(params))
	for k := range params {
		if k != "sign" && k != "sign_type" {
			keys = append(keys, k)
		}
	}
	sort.Strings(keys)
	
	// 拼接签名字符串
	var signStr strings.Builder
	for i, k := range keys {
		if i > 0 {
			signStr.WriteString("&")
		}
		signStr.WriteString(k)
		signStr.WriteString("=")
		signStr.WriteString(params[k])
	}
	
	// 验证签名
	publicKey, err := parsePublicKey(pg.alipayConfig.PublicKey)
	if err != nil {
		return false
	}
	
	signature, err := base64.StdEncoding.DecodeString(sign)
	if err != nil {
		return false
	}
	
	h := crypto.Hash.New(crypto.SHA256)
	h.Write([]byte(signStr.String()))
	hashed := h.Sum(nil)
	
	err = rsa.VerifyPKCS1v15(publicKey, crypto.SHA256, hashed, signature)
	return err == nil
}

// 生成 Payspi 签名
func (pg *PaymentGateway) generatePayspiSign(data map[string]interface{}) string {
	// 对参数排序
	keys := make([]string, 0, len(data))
	for k := range data {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	
	// 拼接签名字符串
	var signStr strings.Builder
	for _, k := range keys {
		signStr.WriteString(fmt.Sprintf("%v", data[k]))
	}
	signStr.WriteString(pg.payspiConfig.WebhookSecret)
	
	// MD5 签名
	h := md5.New()
	h.Write([]byte(signStr.String()))
	return hex.EncodeToString(h.Sum(nil))
}

// 验证 Payspi 签名
func (pg *PaymentGateway) verifyPayspiSign(data map[string]interface{}) bool {
	sign, ok := data["signature"].(string)
	if !ok {
		return false
	}
	
	// 复制数据并移除签名
	dataCopy := make(map[string]interface{})
	for k, v := range data {
		if k != "signature" {
			dataCopy[k] = v
		}
	}
	
	expectedSign := pg.generatePayspiSign(dataCopy)
	return hmac.Equal([]byte(sign), []byte(expectedSign))
}

// 解析私钥
func parsePrivateKey(keyStr string) (*rsa.PrivateKey, error) {
	block, _ := pem.Decode([]byte(keyStr))
	if block == nil {
		return nil, errors.New("failed to decode private key")
	}
	
	key, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		return nil, err
	}
	
	privKey, ok := key.(*rsa.PrivateKey)
	if !ok {
		return nil, errors.New("not an RSA private key")
	}
	
	return privKey, nil
}

// 解析公钥
func parsePublicKey(keyStr string) (*rsa.PublicKey, error) {
	block, _ := pem.Decode([]byte(keyStr))
	if block == nil {
		return nil, errors.New("failed to decode public key")
	}
	
	key, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		return nil, err
	}
	
	pubKey, ok := key.(*rsa.PublicKey)
	if !ok {
		return nil, errors.New("not an RSA public key")
	}
	
	return pubKey, nil
}

// HTTP Handler 函数

// CreateTopupHandler 创建充值订单 Handler
func CreateTopupHandler(gateway *PaymentGateway) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req CreateTopupRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(400, gin.H{"error": err.Error()})
			return
		}
		
		// TODO: 获取当前用户 ID
		userID := 1 // 示例
		
		order, err := gateway.CreateOrder(c.Request.Context(), userID, req.Amount)
		if err != nil {
			c.JSON(500, gin.H{"error": "failed to create order"})
			return
		}
		
		c.JSON(200, gin.H{
			"order_no": order.OrderNo,
			"amount":   order.Amount,
		})
	}
}

// PayHandler 发起支付 Handler
func PayHandler(gateway *PaymentGateway) gin.HandlerFunc {
	return func(c *gin.Context) {
		orderNo := c.Param("order_no")
		method := PaymentMethod(c.Query("method"))
		
		resp, err := gateway.Pay(c.Request.Context(), orderNo, method)
		if err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
			return
		}
		
		c.JSON(200, resp)
	}
}

// AlipayCallbackHandler 支付宝回调 Handler
func AlipayCallbackHandler(gateway *PaymentGateway) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 获取支付宝回调参数
		var formData map[string]string
		if err := c.ShouldBind(&formData); err != nil {
			c.String(400, "fail")
			return
		}
		
		err := gateway.HandleCallback(c.Request.Context(), PaymentMethodAlipay, formData)
		if err != nil {
			c.String(500, "fail")
			return
		}
		
		c.String(200, "success")
	}
}

// WechatCallbackHandler 微信支付回调 Handler
func WechatCallbackHandler(gateway *PaymentGateway) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 获取微信支付回调数据
		var body []byte
		body = c.GetRawData()
		
		var callbackData map[string]interface{}
		if err := json.Unmarshal(body, &callbackData); err != nil {
			c.JSON(400, gin.H{"code": "FAIL", "message": "invalid data"})
			return
		}
		
		err := gateway.HandleCallback(c.Request.Context(), PaymentMethodWechat, callbackData)
		if err != nil {
			c.JSON(500, gin.H{"code": "FAIL", "message": "process failed"})
			return
		}
		
		c.JSON(200, gin.H{"code": "SUCCESS", "message": "ok"})
	}
}

// PayspiCallbackHandler 聚合支付回调 Handler
func PayspiCallbackHandler(gateway *PaymentGateway) gin.HandlerFunc {
	return func(c *gin.Context) {
		var callbackData map[string]interface{}
		if err := c.ShouldBindJSON(&callbackData); err != nil {
			c.JSON(400, gin.H{"error": "invalid data"})
			return
		}
		
		err := gateway.HandleCallback(c.Request.Context(), PaymentMethodPayspi, callbackData)
		if err != nil {
			c.JSON(500, gin.H{"error": "process failed"})
			return
		}
		
		c.JSON(200, gin.H{"status": "success"})
	}
}

// 生成订单号
func generateOrderNo() string {
return fmt.Sprintf("TP%d", time.Now().UnixNano())
}

