package mail_services

import (
	"billionmail-core/api/mail_services/v1"
	"billionmail-core/internal/service/public"
	"context"
	"time"

	"github.com/gogf/gf/v2/errors/gerror"
	"github.com/gogf/gf/v2/frame/g"
)

const wallzaOSWebhookDeliveryTable = "bm_wallzaos_webhook_deliveries"

func (c *ControllerV1) GetWallzaOSWebhookList(ctx context.Context, req *v1.GetWallzaOSWebhookListReq) (res *v1.GetWallzaOSWebhookListRes, err error) {
	res = &v1.GetWallzaOSWebhookListRes{}
	page := req.Page
	if page < 1 {
		page = 1
	}
	pageSize := req.PageSize
	if pageSize < 1 {
		pageSize = 20
	}
	if pageSize > 100 {
		pageSize = 100
	}

	model := g.DB().Model(wallzaOSWebhookDeliveryTable).Ctx(ctx)
	if req.Status != "" {
		model = model.Where("status", req.Status)
	}
	if req.Keyword != "" {
		keyword := "%" + req.Keyword + "%"
		model = model.Where("(event_id ILIKE ? OR event_type ILIKE ?)", keyword, keyword)
	}

	total, err := model.Count()
	if err != nil {
		res.SetError(gerror.New(public.LangCtx(ctx, "Failed to count WallzaOS webhook deliveries: {}", err.Error())))
		return res, nil
	}
	res.Data.Total = total
	res.Data.List = make([]v1.WallzaOSWebhookDelivery, 0, pageSize)
	if total > 0 {
		err = model.
			Fields("event_id, event_type, status, attempts, next_attempt_at, created_at, delivered_at, last_error").
			Page(page, pageSize).
			OrderDesc("created_at").
			Scan(&res.Data.List)
		if err != nil {
			res.SetError(gerror.New(public.LangCtx(ctx, "Failed to list WallzaOS webhook deliveries: {}", err.Error())))
			return res, nil
		}
	}

	res.SetSuccess(public.LangCtx(ctx, "Get WallzaOS webhook deliveries successfully"))
	return res, nil
}

func (c *ControllerV1) GetWallzaOSWebhookDetail(ctx context.Context, req *v1.GetWallzaOSWebhookDetailReq) (res *v1.GetWallzaOSWebhookDetailRes, err error) {
	res = &v1.GetWallzaOSWebhookDetailRes{}
	model := g.DB().Model(wallzaOSWebhookDeliveryTable).Ctx(ctx).Where("event_id", req.EventID)
	count, err := model.Count()
	if err != nil {
		res.SetError(gerror.New(public.LangCtx(ctx, "Failed to find WallzaOS webhook delivery: {}", err.Error())))
		return res, nil
	}
	if count == 0 {
		res.SetError(gerror.New(public.LangCtx(ctx, "WallzaOS webhook delivery not found")))
		return res, nil
	}

	var detail struct {
		EventID       string `json:"event_id"`
		EventType     string `json:"event_type"`
		Status        string `json:"status"`
		Attempts      int    `json:"attempts"`
		NextAttemptAt int64  `json:"next_attempt_at"`
		CreatedAt     int64  `json:"created_at"`
		DeliveredAt   int64  `json:"delivered_at"`
		LastError     string `json:"last_error"`
		Payload       string `json:"payload"`
	}
	err = model.Fields("event_id, event_type, status, attempts, next_attempt_at, created_at, delivered_at, last_error, payload").Scan(&detail)
	if err != nil {
		res.SetError(gerror.New(public.LangCtx(ctx, "Failed to read WallzaOS webhook delivery: {}", err.Error())))
		return res, nil
	}

	res.Data.Delivery = v1.WallzaOSWebhookDelivery{
		EventID:       detail.EventID,
		EventType:     detail.EventType,
		Status:        detail.Status,
		Attempts:      detail.Attempts,
		NextAttemptAt: detail.NextAttemptAt,
		CreatedAt:     detail.CreatedAt,
		DeliveredAt:   detail.DeliveredAt,
		LastError:     detail.LastError,
	}
	res.Data.Payload = detail.Payload
	res.SetSuccess(public.LangCtx(ctx, "Get WallzaOS webhook delivery successfully"))
	return res, nil
}

func (c *ControllerV1) RetryWallzaOSWebhook(ctx context.Context, req *v1.RetryWallzaOSWebhookReq) (res *v1.RetryWallzaOSWebhookRes, err error) {
	res = &v1.RetryWallzaOSWebhookRes{}
	result, err := g.DB().Model(wallzaOSWebhookDeliveryTable).Ctx(ctx).
		Where("event_id", req.EventID).
		Where("status", "failed").
		Update(g.Map{
			"status":                "pending",
			"attempts":              0,
			"next_attempt_at":       time.Now().Unix(),
			"processing_started_at": 0,
			"last_error":            "",
		})
	if err != nil {
		res.SetError(gerror.New(public.LangCtx(ctx, "Failed to retry WallzaOS webhook delivery: {}", err.Error())))
		return res, nil
	}
	affected, err := result.RowsAffected()
	if err != nil {
		res.SetError(gerror.New(public.LangCtx(ctx, "Failed to retry WallzaOS webhook delivery: {}", err.Error())))
		return res, nil
	}
	if affected == 0 {
		res.SetError(gerror.New(public.LangCtx(ctx, "Only failed WallzaOS webhook deliveries can be retried")))
		return res, nil
	}
	res.SetSuccess(public.LangCtx(ctx, "WallzaOS webhook delivery queued for retry"))
	return res, nil
}
