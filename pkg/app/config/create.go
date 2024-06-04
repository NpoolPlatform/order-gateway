package appconfig

import (
	"context"

	appconfigmwpb "github.com/NpoolPlatform/message/npool/order/mw/v1/app/config"
	appconfigmwcli "github.com/NpoolPlatform/order-middleware/pkg/client/app/config"

	"github.com/google/uuid"
)

func (h *Handler) CreateAppConfig(ctx context.Context) (*appconfigmwpb.AppConfig, error) {
	if h.EntID == nil {
		h.EntID = func() *string { s := uuid.NewString(); return &s }()
	}
	if err := appconfigmwcli.CreateAppConfig(ctx, &appconfigmwpb.AppConfigReq{
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
