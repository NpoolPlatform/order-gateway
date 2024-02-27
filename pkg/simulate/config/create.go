package config

import (
	"context"
	"fmt"

	configmwpb "github.com/NpoolPlatform/message/npool/order/mw/v1/simulate/config"
	configmwcli "github.com/NpoolPlatform/order-middleware/pkg/client/simulate/config"

	cruder "github.com/NpoolPlatform/libent-cruder/pkg/cruder"
	basetypes "github.com/NpoolPlatform/message/npool/basetypes/v1"
)

type createHandler struct {
	*Handler
}

func (h *createHandler) checkRepeated(ctx context.Context) error {
	exist, err := configmwcli.ExistSimulateConfigConds(ctx, &configmwpb.Conds{
		AppID: &basetypes.StringVal{Op: cruder.EQ, Value: *h.AppID},
	})
	if err != nil {
		return err
	}
	if exist {
		return fmt.Errorf("repeated config")
	}

	return nil
}

func (h *Handler) CreateSimulateConfig(ctx context.Context) (*configmwpb.SimulateConfig, error) {
	handler := &createHandler{
		Handler: h,
	}

	if err := handler.checkRepeated(ctx); err != nil {
		return nil, err
	}

	info, err := configmwcli.CreateSimulateConfig(ctx, &configmwpb.SimulateConfigReq{
		AppID:                     h.AppID,
		EnabledCashableProfit:     h.EnabledCashableProfit,
		CashableProfitProbability: h.CashableProfitProbability,
		SendCouponMode:            h.SendCouponMode,
		SendCouponProbability:     h.SendCouponProbability,
		Enabled:                   h.Enabled,
	})
	if err != nil {
		return nil, err
	}

	return info, nil
}
