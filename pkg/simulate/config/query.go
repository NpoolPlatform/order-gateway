package config

import (
	"context"
	"fmt"

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
	info, err := configmwcli.GetSimulateConfigOnly(ctx, &configmwpb.Conds{
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
