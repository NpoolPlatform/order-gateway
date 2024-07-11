package appconfig

import (
	"context"

	wlog "github.com/NpoolPlatform/go-service-framework/pkg/wlog"
	appconfigmwpb "github.com/NpoolPlatform/message/npool/order/mw/v1/app/config"
	appconfigmwcli "github.com/NpoolPlatform/order-middleware/pkg/client/app/config"
)

func (h *Handler) DeleteAppConfig(ctx context.Context) (*appconfigmwpb.AppConfig, error) {
	info, err := h.GetAppConfig(ctx)
	if err != nil {
		return nil, wlog.WrapError(err)
	}
	if info == nil {
		return nil, wlog.Errorf("invalid appconfig")
	}
	if err := appconfigmwcli.DeleteAppConfig(ctx, h.ID, h.EntID, h.AppID); err != nil {
		return nil, wlog.WrapError(err)
	}
	return info, nil
}
