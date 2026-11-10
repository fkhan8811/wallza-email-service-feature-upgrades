package mail_services

import (
	"billionmail-core/api/mail_services/v1"
	"billionmail-core/internal/service/abnormal_recipient"
	"billionmail-core/internal/service/public"
	"context"

	"github.com/gogf/gf/v2/errors/gerror"
)

func (c *ControllerV1) IngestProviderComplaint(ctx context.Context, req *v1.IngestProviderComplaintReq) (res *v1.IngestProviderComplaintRes, err error) {
	res = &v1.IngestProviderComplaintRes{}
	err = abnormal_recipient.RecordComplaint(ctx, abnormal_recipient.ComplaintInput{
		EventID:          req.ProviderEventID,
		Recipient:        req.Recipient,
		Provider:         req.Provider,
		PostfixMessageID: req.PostfixMessageID,
	})
	if err != nil {
		res.SetError(gerror.New(public.LangCtx(ctx, "Failed to ingest provider complaint: {}", err.Error())))
		return res, nil
	}

	res.SetSuccess(public.LangCtx(ctx, "Provider complaint recorded and recipient suppressed"))
	return res, nil
}
