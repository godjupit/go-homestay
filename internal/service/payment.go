package service

import (
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"time"

	"gin-looklook/internal/config"
	"gin-looklook/internal/model"
	"gin-looklook/internal/platform"
	"gin-looklook/internal/repository"

	"github.com/wechatpay-apiv3/wechatpay-go/core"
	"github.com/wechatpay-apiv3/wechatpay-go/core/auth/verifiers"
	"github.com/wechatpay-apiv3/wechatpay-go/core/downloader"
	"github.com/wechatpay-apiv3/wechatpay-go/core/notify"
	"github.com/wechatpay-apiv3/wechatpay-go/core/option"
	"github.com/wechatpay-apiv3/wechatpay-go/services/payments"
	"github.com/wechatpay-apiv3/wechatpay-go/services/payments/jsapi"
	"github.com/wechatpay-apiv3/wechatpay-go/utils"
)

type PrepayResult struct{ AppID, NonceStr, PaySign, Package, Timestamp, SignType string }
type PaymentService struct {
	repo   *repository.Repository
	users  *UserService
	orders *OrderService
	cfg    config.Config
}

func NewPaymentService(repo *repository.Repository, users *UserService, orders *OrderService, cfg config.Config) *PaymentService {
	return &PaymentService{repo: repo, users: users, orders: orders, cfg: cfg}
}
func (s *PaymentService) ByOrder(ctx context.Context, orderSN string) (*model.ThirdPayment, error) {
	v, err := s.repo.PaymentByOrder(ctx, orderSN)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, platform.E(platform.CodeDB, "get payment record fail", err)
	}
	return v, nil
}

func (s *PaymentService) wxClient(ctx context.Context) (*core.Client, error) {
	if s.cfg.WxMchID == "" || s.cfg.WxMchPrivateKey == "" || s.cfg.WxMchCertSerial == "" || s.cfg.WxAPIv3Key == "" {
		return nil, platform.E(platform.CodeCommon, "wechat pay is not configured", nil)
	}
	key, err := utils.LoadPrivateKey(s.cfg.WxMchPrivateKey)
	if err != nil {
		return nil, platform.E(platform.CodeCommon, "wechat pay init fail", err)
	}
	client, err := core.NewClient(ctx, option.WithWechatPayAutoAuthCipher(s.cfg.WxMchID, s.cfg.WxMchCertSerial, key, s.cfg.WxAPIv3Key))
	if err != nil {
		return nil, platform.E(platform.CodeCommon, "wechat pay init fail", err)
	}
	return client, nil
}

func (s *PaymentService) Prepay(ctx context.Context, userID int64, orderSN, serviceType string) (PrepayResult, error) {
	if serviceType != model.PaymentServiceHomestay {
		return PrepayResult{}, platform.E(platform.CodeCommon, "Payment for this business type is not supported", nil)
	}
	order, err := s.orders.Detail(ctx, userID, orderSN)
	if err != nil {
		return PrepayResult{}, err
	}
	auth, err := s.users.AuthByUser(ctx, userID, model.UserAuthTypeSmallWX)
	if err == sql.ErrNoRows {
		return PrepayResult{}, platform.E(platform.CodeCommon, "Please authorize by WeChat before payment", nil)
	}
	if err != nil {
		return PrepayResult{}, platform.E(platform.CodeDB, "数据库繁忙,请稍后再试", err)
	}
	flow := &model.ThirdPayment{SN: platform.GenSN("PMT"), UserID: userID, PayMode: model.PaymentModeWechat, PayTotal: order.OrderTotalPrice, OrderSN: orderSN, ServiceType: serviceType, PayStatus: model.PaymentStatusWait, PayTime: sql.NullTime{}}
	if err = s.repo.CreatePayment(ctx, flow); err != nil {
		return PrepayResult{}, platform.E(platform.CodeDB, "create local third payment record fail", err)
	}
	client, err := s.wxClient(ctx)
	if err != nil {
		return PrepayResult{}, err
	}
	svc := jsapi.JsapiApiService{Client: client}
	resp, _, err := svc.PrepayWithRequestPayment(ctx, jsapi.PrepayRequest{Appid: core.String(s.cfg.WxAppID), Mchid: core.String(s.cfg.WxMchID), Description: core.String("homestay pay"), OutTradeNo: core.String(flow.SN), Attach: core.String("homestay pay"), NotifyUrl: core.String(s.cfg.WxNotifyURL), Amount: &jsapi.Amount{Total: core.Int64(flow.PayTotal)}, Payer: &jsapi.Payer{Openid: core.String(auth.AuthKey)}})
	if err != nil {
		return PrepayResult{}, platform.E(platform.CodeCommon, "wechat pay fail", err)
	}
	return PrepayResult{AppID: s.cfg.WxAppID, NonceStr: *resp.NonceStr, PaySign: *resp.PaySign, Package: *resp.Package, Timestamp: *resp.TimeStamp, SignType: *resp.SignType}, nil
}

func payStatus(state string) int64 {
	switch state {
	case "SUCCESS":
		return model.PaymentStatusSuccess
	case "USERPAYING":
		return model.PaymentStatusWait
	case "REFUND":
		return model.PaymentStatusWait
	default:
		return model.PaymentStatusFail
	}
}

func (s *PaymentService) HandleNotify(ctx context.Context, req *http.Request) error {
	if _, err := s.wxClient(ctx); err != nil {
		return err
	}
	visitor := downloader.MgrInstance().GetCertificateVisitor(s.cfg.WxMchID)
	handler := notify.NewNotifyHandler(s.cfg.WxAPIv3Key, verifiers.NewSHA256WithRSAVerifier(visitor))
	transaction := new(payments.Transaction)
	if _, err := handler.ParseNotifyRequest(ctx, req, transaction); err != nil {
		return platform.E(platform.CodeCommon, "Failed to parse payment notification", err)
	}
	if transaction.OutTradeNo == nil || transaction.Amount == nil || transaction.Amount.PayerTotal == nil || transaction.TradeState == nil {
		return platform.E(platform.CodeCommon, "invalid payment notification", nil)
	}
	flow, err := s.repo.PaymentBySN(ctx, *transaction.OutTradeNo)
	if err != nil {
		return platform.E(platform.CodeCommon, "payment record no exists", err)
	}
	if flow.PayTotal != *transaction.Amount.PayerTotal {
		return platform.E(platform.CodeCommon, "Order amount exception", nil)
	}
	status := payStatus(*transaction.TradeState)
	if status != model.PaymentStatusSuccess {
		return nil
	}
	if flow.PayStatus != model.PaymentStatusWait {
		return nil
	}
	flow.PayStatus = status
	flow.TradeState = *transaction.TradeState
	if transaction.TransactionId != nil {
		flow.TransactionID = *transaction.TransactionId
	}
	if transaction.TradeType != nil {
		flow.TradeType = *transaction.TradeType
	}
	if transaction.TradeStateDesc != nil {
		flow.TradeStateDesc = *transaction.TradeStateDesc
	}
	flow.PayTime = sql.NullTime{Time: time.Now(), Valid: status == model.PaymentStatusSuccess}
	event := model.PaymentStatusEvent{PaymentSN: flow.SN, OrderSN: flow.OrderSN, PayStatus: status}
	body, _ := json.Marshal(event)
	outbox := model.OutboxEvent{EventKey: flow.SN + ":" + *transaction.TradeState, Topic: s.cfg.PaymentTopic, MessageKey: flow.OrderSN, Payload: body}
	if err = s.repo.UpdatePaymentWithOutbox(ctx, flow, outbox); err != nil {
		return platform.E(platform.CodeDB, "update payment state fail", err)
	}
	return nil
}
