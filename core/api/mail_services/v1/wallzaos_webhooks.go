package v1

import (
	"billionmail-core/utility/types/api_v1"
	"github.com/gogf/gf/v2/frame/g"
)

type WallzaOSWebhookDelivery struct {
	EventID       string `json:"event_id" dc:"Delivery ID"`
	EventType     string `json:"event_type" dc:"Event type"`
	Status        string `json:"status" dc:"Delivery status"`
	Attempts      int    `json:"attempts" dc:"Delivery attempts"`
	NextAttemptAt int64  `json:"next_attempt_at" dc:"Next retry time"`
	CreatedAt     int64  `json:"created_at" dc:"Creation time"`
	DeliveredAt   int64  `json:"delivered_at" dc:"Delivery time"`
	LastError     string `json:"last_error" dc:"Most recent delivery error"`
}

type GetWallzaOSWebhookListReq struct {
	g.Meta   `path:"/wallzaos_webhooks/list" method:"get" summary:"List WallzaOS webhook deliveries"`
	Page     int    `json:"page" v:"min:1" d:"1"`
	PageSize int    `json:"page_size" v:"min:1|max:100" d:"20"`
	Status   string `json:"status" v:"in:pending,processing,delivered,failed" dc:"Optional status filter"`
	Keyword  string `json:"keyword" dc:"Event ID or event type search"`
}

type GetWallzaOSWebhookListRes struct {
	api_v1.StandardRes
	Data struct {
		Total int                       `json:"total"`
		List  []WallzaOSWebhookDelivery `json:"list"`
	} `json:"data"`
}

type GetWallzaOSWebhookDetailReq struct {
	g.Meta  `path:"/wallzaos_webhooks/detail" method:"get" summary:"Get WallzaOS webhook delivery payload"`
	EventID string `json:"event_id" v:"required" dc:"Delivery ID"`
}

type GetWallzaOSWebhookDetailRes struct {
	api_v1.StandardRes
	Data struct {
		Delivery WallzaOSWebhookDelivery `json:"delivery"`
		Payload  string                  `json:"payload"`
	} `json:"data"`
}

type RetryWallzaOSWebhookReq struct {
	g.Meta  `path:"/wallzaos_webhooks/retry" method:"post" summary:"Retry a failed WallzaOS webhook delivery"`
	EventID string `json:"event_id" v:"required" dc:"Delivery ID"`
}

type RetryWallzaOSWebhookRes struct {
	api_v1.StandardRes
}
