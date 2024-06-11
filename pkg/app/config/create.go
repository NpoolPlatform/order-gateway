package appconfig

import (
	"context"

	wlog "github.com/NpoolPlatform/go-service-framework/pkg/wlog"
	appconfigmwpb "github.com/NpoolPlatform/message/npool/order/mw/v1/app/config"
	appconfigmwcli "github.com/NpoolPlatform/order-middleware/pkg/client/app/config"
)

func (h *Handler) CreateAppConfig(ctx context.Context) (*appconfigmwpb.AppConfig, error) {
	if err := appconfigmwcli.CreateAppConfig(ctx, &appconfigmwpb.AppConfigReq{
		AppID:                                  h.AppID,
		EnableSimulateOrder:                    h.EnableSimulateOrder,
		SimulateOrderCouponMode:                h.SimulateOrderCouponMode,
		SimulateOrderCouponProbability:         h.SimulateOrderCouponProbability,
		SimulateOrderCashableProfitProbability: h.SimulateOrderCashableProfitProbability,
		MaxUnpaidOrders:                        h.MaxUnpaidOrders,
		MaxTypedCouponsPerOrder:                h.MaxTypedCouponsPerOrder,
	}); err != nil {
		return nil, wlog.WrapError(err)
	}
	return h.GetAppConfig(ctx)
}
