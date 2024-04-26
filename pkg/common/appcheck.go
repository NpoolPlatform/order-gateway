package common

import (
	"context"
	"fmt"

	appmwcli "github.com/NpoolPlatform/appuser-middleware/pkg/client/app"
)

type AppCheckHandler struct {
	AppID *string
}

func (h *AppCheckHandler) CheckAppWithAppID(ctx context.Context, appID string) error {
	exist, err := appmwcli.ExistApp(ctx, appID)
	if err != nil {
		return err
	}
	if !exist {
		return fmt.Errorf("invalid app")
	}
	return nil
}

func (h *AppCheckHandler) CheckApp(ctx context.Context) error {
	return h.CheckAppWithAppID(ctx, *h.AppID)
}
