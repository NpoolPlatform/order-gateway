package fee

import (
	"context"

	npool "github.com/NpoolPlatform/message/npool/order/gw/v1/fee"
)

type createHandler struct {
	*baseCreateHandler
}

func (h *Handler) CreateFeeOrder(ctx context.Context) (*npool.FeeOrder, error) {
	handler := &createHandler{
		baseCreateHandler: &baseCreateHandler{
			Handler: h,
		},
	}

	if err := handler.getParentOrder(ctx); err != nil {
		return nil, err
	}
	if err := handler.GetApp(ctx); err != nil {
		return nil, err
	}
	if err := handler.GetUser(ctx); err != nil {
		return nil, err
	}
	if err := handler.getAppGoods(ctx); err != nil {
		return nil, err
	}
	if err := handler.GetAllocatedCoupons(ctx); err != nil {
		return nil, err
	}
	if err := handler.ValidateCouponScope(ctx, &handler.parentOrder.AppGoodID); err != nil {
		return nil, err
	}
	if err := handler.ValidateCouponCount(); err != nil {
		return nil, err
	}
	if err := handler.ValidateMaxUnpaidOrders(ctx); err != nil {
		return nil, err
	}
	if err := handler.getParentGoodCoins(ctx); err != nil {
		return nil, err
	}
	if err := handler.GetAppCoins(ctx, func() (coinTypeIDs []string) {
		for _, goodCoin := range handler.parentGoodCoins {
			coinTypeIDs = append(coinTypeIDs, goodCoin.CoinTypeID)
		}
		return
	}()); err != nil {
		return nil, err
	}
	if err := handler.GetRequiredAppGoods(ctx); err != nil {
		return nil, err
	}
	if err := handler.validateRequiredAppGoods(); err != nil {
		return nil, err
	}

	return h.GetFeeOrder(ctx)
}
