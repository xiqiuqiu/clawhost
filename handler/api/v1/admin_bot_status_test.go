package v1

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/clawhost/clawhost/model"
	"github.com/clawhost/clawhost/service/k8s"
	"github.com/clawhost/clawhost/util"
	"github.com/labstack/echo/v4"
)

func TestAdminListBotsPrefersRuntimeStartingStatus(t *testing.T) {
	db := setupAdminAuthTestDB(t)
	if err := db.AutoMigrate(&model.Bot{}); err != nil {
		t.Fatalf("migrate bot model: %v", err)
	}

	bot := &model.Bot{
		AppID:    "app-123",
		UserID:   "admin",
		Name:     "Pending Bot",
		Status:   model.BotStatusError,
		Endpoint: "oc-pending-svc.clawhost.svc.cluster.local:18789",
	}
	if err := model.CreateBot(bot); err != nil {
		t.Fatalf("create bot: %v", err)
	}

	restore := getAdminDeploymentStatusInfoFn
	defer func() {
		getAdminDeploymentStatusInfoFn = restore
	}()
	getAdminDeploymentStatusInfoFn = func(_ string) (*k8s.DeploymentStatusInfo, error) {
		return &k8s.DeploymentStatusInfo{
			Status:          "starting",
			ReadyReplicas:   0,
			DesiredReplicas: 1,
			UpdatedReplicas: 1,
		}, nil
	}

	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/bot/api/v1/admin/bots", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	if err := AdminListBots(c); err != nil {
		t.Fatalf("AdminListBots returned error: %v", err)
	}
	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rec.Code)
	}

	var resp util.Response
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode list response: %v", err)
	}

	data, ok := resp.Data.([]interface{})
	if !ok || len(data) != 1 {
		t.Fatalf("expected one bot item, got %#v", resp.Data)
	}
	item, ok := data[0].(map[string]interface{})
	if !ok {
		t.Fatalf("expected bot object, got %#v", data[0])
	}
	if item["status"] != string(model.BotStatusStarting) {
		t.Fatalf("expected starting status, got %#v", item["status"])
	}

	storedBot, err := model.GetBotByID(bot.ID)
	if err != nil {
		t.Fatalf("reload bot: %v", err)
	}
	if storedBot.Status != model.BotStatusStarting {
		t.Fatalf("expected database status to sync to starting, got %s", storedBot.Status)
	}
}
