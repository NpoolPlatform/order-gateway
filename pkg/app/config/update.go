package appconfig

import (
	"context"
	"fmt"

	appconfigmwpb "github.com/NpoolPlatform/message/npool/order/mw/v1/app/config"
	appconfigmwcli "github.com/NpoolPlatform/order-middleware/pkg/client/app/config"

	cruder "github.com/NpoolPlatform/libent-cruder/pkg/cruder"
	basetypes "github.com/NpoolPlatform/message/npool/basetypes/v1"
)

type updateHandler struct {
	*Handler
}

func (h *updateHandler) checkExist(ctx context.Context) error {
	exist, err := appconfigmwcli.ExistSimulateConfigConds(ctx, &appconfigmwpb.Conds{
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

func (h *Handler) UpdateSimulateConfig(ctx context.Context) (*appconfigmwpb.SimulateConfig, error) {
	handler := &updateHandler{
		Handler: h,
	}

	if err := handler.checkExist(ctx); err != nil {
		return nil, err
	}

	info, err := appconfigmwcli.UpdateSimulateConfig(ctx, &appconfigmwpb.SimulateConfigReq{
		ID:                        h.ID,
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
