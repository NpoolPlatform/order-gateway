package config

import (
	"context"
	"fmt"

	configmwpb "github.com/NpoolPlatform/message/npool/order/mw/v1/simulate/config"
	configmwcli "github.com/NpoolPlatform/order-middleware/pkg/client/simulate/config"

	cruder "github.com/NpoolPlatform/libent-cruder/pkg/cruder"
	basetypes "github.com/NpoolPlatform/message/npool/basetypes/v1"
)

type updateHandler struct {
	*Handler
}

func (h *updateHandler) checkExist(ctx context.Context) error {
	exist, err := configmwcli.ExistSimulateConfigConds(ctx, &configmwpb.Conds{
		AppID: &basetypes.StringVal{Op: cruder.EQ, Value: *h.AppID},
		ID:    &basetypes.Uint32Val{Op: cruder.EQ, Value: *h.ID},
		EntID: &basetypes.StringVal{Op: cruder.EQ, Value: *h.EntID},
	})
	if err != nil {
		return err
	}
	if !exist {
		return fmt.Errorf("invalid config")
	}
	return nil
}

func (h *updateHandler) checkEnabled(ctx context.Context) error {
	if h.Enabled == nil || !*h.Enabled {
		return nil
	}
	exist, err := configmwcli.ExistSimulateConfigConds(ctx, &configmwpb.Conds{
		AppID:   &basetypes.StringVal{Op: cruder.EQ, Value: *h.AppID},
		Enabled: &basetypes.BoolVal{Op: cruder.EQ, Value: *h.Enabled},
	})
	if err != nil {
		return err
	}
	if !exist {
		return fmt.Errorf("invalid config")
	}

	return nil
}

func (h *Handler) UpdateSimulateConfig(ctx context.Context) (*configmwpb.SimulateConfig, error) {
	handler := &updateHandler{
		Handler: h,
	}

	if err := handler.checkExist(ctx); err != nil {
		return nil, err
	}

	if err := handler.checkEnabled(ctx); err != nil {
		return nil, err
	}

	info, err := configmwcli.UpdateSimulateConfig(ctx, &configmwpb.SimulateConfigReq{
		ID:                    h.ID,
		Units:                 h.Units,
		SendCouponMode:        h.SendCouponMode,
		SendCouponProbability: h.SendCouponProbability,
		Enabled:               h.Enabled,
	})
	if err != nil {
		return nil, err
	}

	if h.Enabled != nil && *h.Enabled {
		if err := h.SetSimulateConfigRedis(ctx); err != nil {
			return nil, err
		}
	}

	return info, nil
}
