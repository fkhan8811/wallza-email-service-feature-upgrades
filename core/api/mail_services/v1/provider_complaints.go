package v1

import (
	"billionmail-core/utility/types/api_v1"
	"github.com/gogf/gf/v2/frame/g"
)

type IngestProviderComplaintReq struct {
	g.Meta            `path:"/mail_services/complaint_ingest" method:"post" summary:"Ingest a verified email-provider complaint"`
	ProviderEventID   string `json:"provider_event_id" v:"required|min-length:1|max-length:255" dc:"Stable event ID from the upstream provider"`
	Provider          string `json:"provider" v:"required|min-length:1|max-length:64" dc:"Provider slug, e.g. ses, sendgrid, mailgun"`
	Recipient         string `json:"recipient" v:"required|email|max-length:320" dc:"Recipient address to suppress"`
	PostfixMessageID  string `json:"postfix_message_id" v:"max-length:255" dc:"Optional local Postfix queue ID, when available"`
}

type IngestProviderComplaintRes struct {
	api_v1.StandardRes
}
