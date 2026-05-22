package payment

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/gin-gonic/gin"
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
}

// NewPaymentGateway 创建支付网关
func NewPaymentGateway(
	alipayCfg *AlipayConfig,
	wechatCfg *WechatConfig,
	payspiCfg *PayspiConfig,
) *PaymentGateway {
	return &PaymentGateway{
		alipayConfig:  alipayCfg,
		wechatConfig:  wechatCfg,
		payspiConfig:  payspiCfg,
	}
}

// CreateOrder 创建充值订单
func (pg *PaymentGateway) CreateOrder(ctx context.Context, userID int, amount float64) (*TopupOrder, error) {
	// TODO: 实现订单创建逻辑
	// 1. 生成订单号
	// 2. 保存订单到数据库
	// 3. 返回订单信息
	
	orderNo := generateOrderNo()
	order := &TopupOrder{
		UserID:        userID,
		OrderNo:       orderNo,
		Amount:        amount,
		Status:        OrderStatusPending,
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}
	
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
	// TODO: 从数据库查询订单状态
	return nil, errors.New("not implemented")
}

// Refund 退款
func (pg *PaymentGateway) Refund(ctx context.Context, orderNo string, reason string) error {
	// TODO: 实现退款逻辑
	return errors.New("not implemented")
}

// 支付宝支付实现
func (pg *PaymentGateway) alipayPay(ctx context.Context, orderNo string) (*PaymentResponse, error) {
	if pg.alipayConfig == nil {
		return nil, errors.New("alipay not configured")
	}
	
	// TODO: 调用支付宝 API 生成支付二维码或表单
	// 使用支付宝当面付 API
	
	return &PaymentResponse{
		OrderNo:    orderNo,
		PayURL:     "",
		QRCode:     "",
		ExpireTime: time.Now().Add(30 * time.Minute).Unix(),
	}, nil
}

// 微信支付实现
func (pg *PaymentGateway) wechatPay(ctx context.Context, orderNo string) (*PaymentResponse, error) {
	if pg.wechatConfig == nil {
		return nil, errors.New("wechat not configured")
	}
	
	// TODO: 调用微信支付 API 生成支付参数
	
	return &PaymentResponse{
		OrderNo:    orderNo,
		ExtraData:  map[string]interface{}{},
		ExpireTime: time.Now().Add(30 * time.Minute).Unix(),
	}, nil
}

// 聚合支付实现
func (pg *PaymentGateway) payspiPay(ctx context.Context, orderNo string) (*PaymentResponse, error) {
	if pg.payspiConfig == nil {
		return nil, errors.New("payspi not configured")
	}
	
	// TODO: 调用聚合支付 API
	
	return &PaymentResponse{
		OrderNo:    orderNo,
		PayURL:     "",
		ExpireTime: time.Now().Add(30 * time.Minute).Unix(),
	}, nil
}

// 处理支付宝回调
func (pg *PaymentGateway) handleAlipayCallback(ctx context.Context, data interface{}) error {
	// TODO: 验证签名
	// TODO: 更新订单状态
	// TODO: 增加用户余额
	
	return nil
}

// 处理微信支付回调
func (pg *PaymentGateway) handleWechatCallback(ctx context.Context, data interface{}) error {
	// TODO: 验证签名
	// TODO: 更新订单状态
	// TODO: 增加用户余额
	
	return nil
}

// 处理聚合支付回调
func (pg *PaymentGateway) handlePayspiCallback(ctx context.Context, data interface{}) error {
	// TODO: 验证签名
	// TODO: 更新订单状态
	// TODO: 增加用户余额
	
	return nil
}

// 生成订单号
func generateOrderNo() string {
	return fmt.Sprintf("TP%d", time.Now().UnixNano())
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
