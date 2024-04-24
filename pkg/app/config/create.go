package appconfig

import (
	"context"
	"fmt"

	appconfigmwpb "github.com/NpoolPlatform/message/npool/order/mw/v1/app/config"
	appconfigmwcli "github.com/NpoolPlatform/order-middleware/pkg/client/app/config"

	cruder "github.com/NpoolPlatform/libent-cruder/pkg/cruder"
	basetypes "github.com/NpoolPlatform/message/npool/basetypes/v1"
)

type createHandler struct {
	*Handler
}

func (h *createHandler) checkRepeated(ctx context.Context) error {
	exist, err := appconfigmwcli.ExistSimulateConfigConds(ctx, &appconfigmwpb.Conds{
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

func (h *Handler) CreateSimulateConfig(ctx context.Context) (*appconfigmwpb.SimulateConfig, error) {
	handler := &createHandler{
		Handler: h,
	}

	if err := handler.checkRepeated(ctx); err != nil {
		return nil, err
	}

	info, err := appconfigmwcli.CreateSimulateConfig(ctx, &appconfigmwpb.SimulateConfigReq{
		AppID:                     h.AppID,
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
