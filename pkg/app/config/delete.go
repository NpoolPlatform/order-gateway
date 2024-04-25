package appconfig

import (
	"context"
	"fmt"

	appconfigmwpb "github.com/NpoolPlatform/message/npool/order/mw/v1/app/config"
	appconfigmwcli "github.com/NpoolPlatform/order-middleware/pkg/client/app/config"
)

func (h *Handler) DeleteAppConfig(ctx context.Context) (*appconfigmwpb.AppConfig, error) {
	info, err := h.GetAppConfig(ctx)
	if err != nil {
		return nil, err
	}
	if info == nil {
		return nil, fmt.Errorf("invalid appconfig")
	}
	if err := appconfigmwcli.DeleteAppConfig(ctx, h.ID, h.EntID, h.AppID); err != nil {
		return nil, err
	}
	return info, nil
}
