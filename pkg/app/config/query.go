package appconfig

import (
	"context"
	"fmt"

	appconfigmwpb "github.com/NpoolPlatform/message/npool/order/mw/v1/app/config"
	appconfigmwcli "github.com/NpoolPlatform/order-middleware/pkg/client/app/config"

	cruder "github.com/NpoolPlatform/libent-cruder/pkg/cruder"
	basetypes "github.com/NpoolPlatform/message/npool/basetypes/v1"
)

func (h *Handler) GetSimulateConfigs(ctx context.Context) ([]*appconfigmwpb.SimulateConfig, uint32, error) {
	infos, total, err := appconfigmwcli.GetSimulateConfigs(ctx, &appconfigmwpb.Conds{
		AppID: &basetypes.StringVal{Op: cruder.EQ, Value: *h.AppID},
	}, h.Offset, h.Limit)
	if err != nil {
		return nil, 0, err
	}

	return infos, total, nil
}

func (h *Handler) GetSimulateConfig(ctx context.Context) (*appconfigmwpb.SimulateConfig, error) {
	info, err := appconfigmwcli.GetSimulateConfigOnly(ctx, &appconfigmwpb.Conds{
		AppID: &basetypes.StringVal{Op: cruder.EQ, Value: *h.AppID},
		EntID: &basetypes.StringVal{Op: cruder.EQ, Value: *h.EntID},
	})
	if err != nil {
		return nil, err
	}
	if info == nil {
		return nil, fmt.Errorf("invalid config")
	}

	return info, nil
}
