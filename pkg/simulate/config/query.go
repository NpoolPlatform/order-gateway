package config

import (
	"context"

	configmwpb "github.com/NpoolPlatform/message/npool/order/mw/v1/simulate/config"
	configmwcli "github.com/NpoolPlatform/order-middleware/pkg/client/simulate/config"

	cruder "github.com/NpoolPlatform/libent-cruder/pkg/cruder"
	basetypes "github.com/NpoolPlatform/message/npool/basetypes/v1"
)

func (h *Handler) GetSimulateConfigs(ctx context.Context) ([]*configmwpb.SimulateConfig, uint32, error) {
	infos, total, err := configmwcli.GetSimulateConfigs(ctx, &configmwpb.Conds{
		AppID: &basetypes.StringVal{Op: cruder.EQ, Value: *h.AppID},
	}, h.Offset, h.Limit)
	if err != nil {
		return nil, 0, err
	}

	return infos, total, nil
}

func (h *Handler) GetSimulateConfig(ctx context.Context) (*configmwpb.SimulateConfig, error) {
	info, err := configmwcli.GetSimulateConfig(ctx, *h.EntID)
	if err != nil {
		return nil, err
	}
	return info, nil
}
