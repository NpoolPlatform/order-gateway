package config

import (
	"context"
	"fmt"

	cruder "github.com/NpoolPlatform/libent-cruder/pkg/cruder"
	basetypes "github.com/NpoolPlatform/message/npool/basetypes/v1"
	configmwpb "github.com/NpoolPlatform/message/npool/order/mw/v1/simulate/config"
	configmwcli "github.com/NpoolPlatform/order-middleware/pkg/client/simulate/config"
)

func (h *Handler) DeleteSimulateConfig(ctx context.Context) (*configmwpb.SimulateConfig, error) {
	exist, err := configmwcli.ExistSimulateConfigConds(ctx, &configmwpb.Conds{
		ID:    &basetypes.Uint32Val{Op: cruder.EQ, Value: *h.ID},
		EntID: &basetypes.StringVal{Op: cruder.EQ, Value: *h.EntID},
		AppID: &basetypes.StringVal{Op: cruder.EQ, Value: *h.AppID},
	})
	if err != nil {
		return nil, err
	}
	if !exist {
		return nil, fmt.Errorf("invalid config")
	}

	info, err := configmwcli.DeleteSimulateConfig(ctx, *h.ID)
	if err != nil {
		return nil, err
	}

	return info, nil
}
