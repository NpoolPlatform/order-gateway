package appconfig

import (
	"context"
	"fmt"

	cruder "github.com/NpoolPlatform/libent-cruder/pkg/cruder"
	basetypes "github.com/NpoolPlatform/message/npool/basetypes/v1"
	appconfigmwpb "github.com/NpoolPlatform/message/npool/order/mw/v1/app/config"
	appconfigmwcli "github.com/NpoolPlatform/order-middleware/pkg/client/app/config"
)

func (h *Handler) GetAppConfigs(ctx context.Context) ([]*appconfigmwpb.AppConfig, uint32, error) {
	conds := &appconfigmwpb.Conds{}
	if h.AppID != nil {
		conds.AppID = &basetypes.StringVal{Op: cruder.EQ, Value: *h.AppID}
	}
	return appconfigmwcli.GetAppConfigs(ctx, conds, h.Offset, h.Limit)
}

func (h *Handler) GetAppConfig(ctx context.Context) (*appconfigmwpb.AppConfig, error) {
	conds := &appconfigmwpb.Conds{
		AppID: &basetypes.StringVal{Op: cruder.EQ, Value: *h.AppID},
	}
	if h.EntID != nil {
		conds.EntID = &basetypes.StringVal{Op: cruder.EQ, Value: *h.EntID}
	}
	info, err := appconfigmwcli.GetAppConfigOnly(ctx, conds)
	if err != nil {
		return nil, err
	}
	if info == nil {
		return nil, fmt.Errorf("invalid appconfig")
	}
	return info, nil
}
