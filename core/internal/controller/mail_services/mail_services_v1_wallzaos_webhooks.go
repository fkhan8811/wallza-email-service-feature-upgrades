package mail_services

import (
	"billionmail-core/api/mail_services/v1"
	"billionmail-core/internal/service/public"
	"context"

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
	var delivery v1.WallzaOSWebhookDelivery
	var payload string

	err = g.DB().Model(wallzaOSWebhookDeliveryTable).Ctx(ctx).
		Fields("event_id, event_type, status, attempts, next_attempt_at, created_at, delivered_at, last_error, payload").
		Where("event_id", req.EventID).
		Scan(&struct {
			v1.WallzaOSWebhookDelivery
			Payload *string `json:"payload"`
		}{
			WallzaOSWebhookDelivery: delivery,
		})
	if err != nil {
		res.SetError(gerror.New(public.LangCtx(ctx, "Failed to get WallzaOS webhook delivery: {}", err.Error())))
		return res, nil
	}

	var detail struct {
		v1.WallzaOSWebhookDelivery
		Payload *string `json:"payload"`
	}
	err = g.DB().Model(wallzaOSWebhookDeliveryTable).Ctx(ctx).
		Fields("event_id, event_type, status, attempts, next_attempt_at, created_at, delivered_at, last_error, payload").
		Where("event_id", req.EventID).
		Scan(&detail)
	if err != nil {
		res.SetError(gerror.New(public.LangCtx(ctx, "Failed to get WallzaOS webhook delivery: {}", err.Error())))
		return res, nil
	}
	if detail.Payload == nil {
		res.SetError(gerror.New(public.LangCtx(ctx, "WallzaOS webhook delivery not found")))
		return res, nil
	}
	res.Data.Delivery = detail.WallzaOSWebhookDelivery
	payload = *detail.Payload
	res.Data.Payload = payload
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
			"next_attempt_at":       0,
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
