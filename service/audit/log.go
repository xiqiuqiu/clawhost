package audit

import (
	"encoding/json"

	"github.com/clawhost/clawhost/model"
)

type RequestContext struct {
	ActorAdminID string
	ActorEmail   string
	SourceIP     string
	UserAgent    string
}

type Entry struct {
	AppID       string
	Action      string
	TargetType  string
	TargetID    string
	TargetLabel string
	Result      string
	Metadata    map[string]interface{}
}

func Write(ctx RequestContext, entry Entry) (*model.AuditLog, error) {
	var metadata json.RawMessage
	if len(entry.Metadata) > 0 {
		raw, err := json.Marshal(entry.Metadata)
		if err != nil {
			return nil, err
		}
		metadata = raw
	}

	logEntry := &model.AuditLog{
		ActorAdminID: ctx.ActorAdminID,
		ActorEmail:   ctx.ActorEmail,
		AppID:        entry.AppID,
		Action:       entry.Action,
		TargetType:   entry.TargetType,
		TargetID:     entry.TargetID,
		TargetLabel:  entry.TargetLabel,
		Result:       entry.Result,
		SourceIP:     ctx.SourceIP,
		UserAgent:    ctx.UserAgent,
		Metadata:     metadata,
	}
	if logEntry.Result == "" {
		logEntry.Result = model.AuditResultSuccess
	}

	if err := model.CreateAuditLog(logEntry); err != nil {
		return nil, err
	}
	return logEntry, nil
}
