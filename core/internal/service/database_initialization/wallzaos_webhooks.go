package database_initialization

import (
	"context"

	"github.com/gogf/gf/v2/frame/g"
)

func init() {
	registerHandler(func() {
		sqlList := []string{
			`CREATE TABLE IF NOT EXISTS bm_wallzaos_webhook_deliveries (
				event_id VARCHAR(64) PRIMARY KEY,
				event_type VARCHAR(120) NOT NULL,
				payload TEXT NOT NULL,
				status VARCHAR(20) NOT NULL DEFAULT 'pending',
				attempts INT NOT NULL DEFAULT 0,
				next_attempt_at BIGINT NOT NULL DEFAULT 0,
				created_at BIGINT NOT NULL DEFAULT 0,
				delivered_at BIGINT NOT NULL DEFAULT 0,
				last_error TEXT NOT NULL DEFAULT ''
			)`,
			`CREATE INDEX IF NOT EXISTS idx_wallzaos_webhook_pending
				ON bm_wallzaos_webhook_deliveries(status, next_attempt_at, created_at)`,
			`CREATE INDEX IF NOT EXISTS idx_wallzaos_webhook_created
				ON bm_wallzaos_webhook_deliveries(created_at)`,
		}

		for _, sql := range sqlList {
			if _, err := g.DB().Exec(context.Background(), sql); err != nil {
				g.Log().Error(context.Background(), "Failed to initialize WallzaOS webhook delivery table: ", err)
				return
			}
		}
	})
}
