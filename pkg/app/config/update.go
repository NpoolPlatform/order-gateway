package appconfig

import (
	"context"

	appconfigmwpb "github.com/NpoolPlatform/message/npool/order/mw/v1/app/config"
	appconfigmwcli "github.com/NpoolPlatform/order-middleware/pkg/client/app/config"
)

type updateHandler struct {
	*checkHandler
}

func (h *Handler) UpdateAppConfig(ctx context.Context) (*appconfigmwpb.AppConfig, error) {
	handler := &updateHandler{
		checkHandler: &checkHandler{
			Handler: h,
		},
	}
	if err := handler.checkAppConfig(ctx); err != nil {
		return nil, err
	}
	if err := appconfigmwcli.UpdateAppConfig(ctx, &appconfigmwpb.AppConfigReq{
		ID:                                     h.ID,
		EntID:                                  h.EntID,
		AppID:                                  h.AppID,
		EnableSimulateOrder:                    h.EnableSimulateOrder,
		SimulateOrderCouponMode:                h.SimulateOrderCouponMode,
		SimulateOrderCouponProbability:         h.SimulateOrderCouponProbability,
		SimulateOrderCashableProfitProbability: h.SimulateOrderCashableProfitProbability,
		MaxUnpaidOrders:                        h.MaxUnpaidOrders,
		MaxTypedCouponsPerOrder:                h.MaxTypedCouponsPerOrder,
	}); err != nil {
		return nil, err
	}
	return h.GetAppConfig(ctx)
}
