package common

import (
	"context"

	appmwcli "github.com/NpoolPlatform/appuser-middleware/pkg/client/app"
	wlog "github.com/NpoolPlatform/go-service-framework/pkg/wlog"
)

type AppCheckHandler struct {
	AppID *string
}

func (h *AppCheckHandler) CheckAppWithAppID(ctx context.Context, appID string) error {
	exist, err := appmwcli.ExistApp(ctx, appID)
	if err != nil {
		return wlog.WrapError(err)
	}
	if !exist {
		return wlog.Errorf("invalid app")
	}
	return nil
}

func (h *AppCheckHandler) CheckApp(ctx context.Context) error {
	return h.CheckAppWithAppID(ctx, *h.AppID)
}
