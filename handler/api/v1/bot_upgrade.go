package v1

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"

	"github.com/clawhost/clawhost/model"
	auditservice "github.com/clawhost/clawhost/service/audit"
	"github.com/clawhost/clawhost/service/k8s"
	"github.com/clawhost/clawhost/util"
	"github.com/labstack/echo/v4"
	"github.com/spf13/viper"
)

var (
	getDeploymentImageFn    = k8s.GetDeploymentImage
	updateDeploymentImageFn = k8s.UpdateDeploymentImage
	syncConfigToPodFn       = k8s.SyncConfigToPod
)

type UpgradeBotsRequest struct {
	Image string `json:"image"` // Target image, defaults to config value if empty
}

type UpgradeResult struct {
	BotID   string `json:"bot_id"`
	Status  string `json:"status"` // "upgraded", "failed", "skipped"
	Message string `json:"message,omitempty"`
}

type UpgradeBotsResponse struct {
	Image    string          `json:"image"`
	Total    int             `json:"total"`
	Upgraded int64           `json:"upgraded"`
	Failed   int64           `json:"failed"`
	Skipped  int64           `json:"skipped"`
	Results  []UpgradeResult `json:"results"`
}

// UpgradeBot upgrades a single bot's openclaw image
// POST /bot/api/v1/admin/bots/:id/upgrade
func UpgradeBot(c echo.Context) error {
	botID := c.Param("id")
	if botID == "" {
		return util.BadRequest(c, "bot id is required")
	}

	bot, err := model.GetBotByID(botID)
	if err != nil {
		return util.NotFound(c, "bot not found")
	}

	if bot.Status != model.BotStatusRunning {
		writeAuditEntry(c, auditservice.Entry{
			AppID:       bot.AppID,
			Action:      "bot.upgrade",
			TargetType:  "bot",
			TargetID:    bot.ID,
			TargetLabel: bot.Name,
			Result:      model.AuditResultFailure,
			Metadata: map[string]interface{}{
				"reason": "bot is not running",
			},
		})
		return util.BadRequest(c, "bot is not running")
	}

	var req UpgradeBotsRequest
	if err := c.Bind(&req); err != nil {
		return util.BadRequest(c, "invalid request body")
	}

	image := req.Image
	if image == "" {
		image = viper.GetString("openclaw.image")
	}
	if image == "" {
		return util.BadRequest(c, "no image specified")
	}

	ctx := context.Background()

	// Check current image
	currentImage, _ := getDeploymentImageFn(ctx, bot.ID)
	if currentImage == image {
		writeAuditEntry(c, auditservice.Entry{
			AppID:       bot.AppID,
			Action:      "bot.upgrade",
			TargetType:  "bot",
			TargetID:    bot.ID,
			TargetLabel: bot.Name,
			Result:      model.AuditResultSuccess,
			Metadata: map[string]interface{}{
				"image":          image,
				"previous_image": currentImage,
				"status":         "skipped",
			},
		})
		return util.Success(c, map[string]string{
			"status":  "skipped",
			"message": "already running target image",
			"image":   image,
		})
	}

	if err := updateDeploymentImageFn(ctx, bot.ID, image); err != nil {
		writeAuditEntry(c, auditservice.Entry{
			AppID:       bot.AppID,
			Action:      "bot.upgrade",
			TargetType:  "bot",
			TargetID:    bot.ID,
			TargetLabel: bot.Name,
			Result:      model.AuditResultFailure,
			Metadata: map[string]interface{}{
				"image":          image,
				"previous_image": currentImage,
				"reason":         err.Error(),
			},
		})
		return util.InternalError(c, "failed to upgrade: "+err.Error())
	}

	// Sync config to new pod after image upgrade (applies latest gateway settings)
	go func() {
		if err := syncConfigToPodFn(context.Background(), bot.ID); err != nil {
			fmt.Printf("[Upgrade] failed to sync config for bot %s: %v\n", bot.ID, err)
		}
	}()

	writeAuditEntry(c, auditservice.Entry{
		AppID:       bot.AppID,
		Action:      "bot.upgrade",
		TargetType:  "bot",
		TargetID:    bot.ID,
		TargetLabel: bot.Name,
		Result:      model.AuditResultSuccess,
		Metadata: map[string]interface{}{
			"image":          image,
			"previous_image": currentImage,
			"status":         "upgraded",
		},
	})
	return util.Success(c, map[string]string{
		"status":         "upgraded",
		"image":          image,
		"previous_image": currentImage,
	})
}

// UpgradeAllBots upgrades all running bots to a new openclaw image
// POST /bot/api/v1/admin/bots/upgrade
func UpgradeAllBots(c echo.Context) error {
	var req UpgradeBotsRequest
	if err := c.Bind(&req); err != nil {
		return util.BadRequest(c, "invalid request body")
	}

	image := req.Image
	if image == "" {
		image = viper.GetString("openclaw.image")
	}
	if image == "" {
		return util.BadRequest(c, "no image specified")
	}

	bots, err := model.ListBotsByStatus(model.BotStatusRunning)
	if err != nil {
		return util.InternalError(c, "failed to list running bots")
	}

	if len(bots) == 0 {
		writeAuditEntry(c, auditservice.Entry{
			Action:     "bot.upgrade_all",
			TargetType: "bot",
			TargetID:   "*",
			Result:     model.AuditResultSuccess,
			Metadata: map[string]interface{}{
				"image":   image,
				"total":   0,
				"status":  "skipped",
				"message": "no running bots to upgrade",
			},
		})
		return util.Success(c, &UpgradeBotsResponse{
			Image: image,
			Total: 0,
		})
	}

	ctx := context.Background()
	var upgraded, failed, skipped atomic.Int64
	results := make([]UpgradeResult, len(bots))

	// Upgrade concurrently with limited parallelism
	var wg sync.WaitGroup
	sem := make(chan struct{}, 10) // max 10 concurrent upgrades

	for i, bot := range bots {
		wg.Add(1)
		go func(idx int, b *model.Bot) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			result := UpgradeResult{BotID: b.ID}

			// Check current image
			currentImage, err := getDeploymentImageFn(ctx, b.ID)
			if err != nil {
				// Deployment might not exist, skip
				skipped.Add(1)
				result.Status = "skipped"
				result.Message = fmt.Sprintf("deployment not found: %v", err)
				results[idx] = result
				return
			}

			if currentImage == image {
				skipped.Add(1)
				result.Status = "skipped"
				result.Message = "already running target image"
				results[idx] = result
				return
			}

			if err := updateDeploymentImageFn(ctx, b.ID, image); err != nil {
				failed.Add(1)
				result.Status = "failed"
				result.Message = err.Error()
			} else {
				upgraded.Add(1)
				result.Status = "upgraded"
				// Sync config to new pod after image upgrade
				go func(botID string) {
					if err := syncConfigToPodFn(context.Background(), botID); err != nil {
						fmt.Printf("[Upgrade] failed to sync config for bot %s: %v\n", botID, err)
					}
				}(b.ID)
			}
			results[idx] = result
		}(i, bot)
	}

	wg.Wait()

	auditResult := model.AuditResultSuccess
	if failed.Load() > 0 && upgraded.Load() == 0 {
		auditResult = model.AuditResultFailure
	}
	writeAuditEntry(c, auditservice.Entry{
		Action:     "bot.upgrade_all",
		TargetType: "bot",
		TargetID:   "*",
		Result:     auditResult,
		Metadata: map[string]interface{}{
			"image":    image,
			"total":    len(bots),
			"upgraded": upgraded.Load(),
			"failed":   failed.Load(),
			"skipped":  skipped.Load(),
		},
	})
	return util.Success(c, &UpgradeBotsResponse{
		Image:    image,
		Total:    len(bots),
		Upgraded: upgraded.Load(),
		Failed:   failed.Load(),
		Skipped:  skipped.Load(),
		Results:  results,
	})
}
