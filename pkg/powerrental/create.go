package powerrental

import (
	"context"

	wlog "github.com/NpoolPlatform/go-service-framework/pkg/wlog"
	npool "github.com/NpoolPlatform/message/npool/order/gw/v1/powerrental"
	ordercommon "github.com/NpoolPlatform/order-gateway/pkg/order/common"
)

type createHandler struct {
	*baseCreateHandler
}

func (h *Handler) CreatePowerRentalOrder(ctx context.Context) (*npool.PowerRentalOrder, error) {
	handler := &createHandler{
		baseCreateHandler: &baseCreateHandler{
			Handler: h,
			OrderOpHandler: &ordercommon.OrderOpHandler{
				AppGoodCheckHandler:         h.AppGoodCheckHandler,
				CoinCheckHandler:            h.CoinCheckHandler,
				AppGoodIDs:                  append(h.FeeAppGoodIDs, *h.AppGoodID),
				AllocatedCouponCheckHandler: h.AllocatedCouponCheckHandler,
				PaymentTransferCoinTypeID:   h.PaymentTransferCoinTypeID,
				PaymentBalanceReqs:          h.Balances,
			},
		},
	}

	if err := handler.GetApp(ctx); err != nil {
		return nil, wlog.WrapError(err)
	}
	if err := handler.GetUser(ctx); err != nil {
		return nil, wlog.WrapError(err)
	}
	if err := handler.GetRequiredAppGoods(ctx, *h.AppGoodID); err != nil {
		return nil, wlog.WrapError(err)
	}
	if err := handler.getAppGoods(ctx); err != nil {
		return nil, wlog.WrapError(err)
	}
	if err := handler.validateRequiredAppGoods(); err != nil {
		return nil, wlog.WrapError(err)
	}
	if err := handler.GetAllocatedCoupons(ctx); err != nil {
		return nil, wlog.WrapError(err)
	}
	if err := handler.ValidateCouponScope(ctx, h.AppGoodID); err != nil {
		return nil, wlog.WrapError(err)
	}
	if err := handler.ValidateCouponCount(); err != nil {
		return nil, wlog.WrapError(err)
	}
	if err := handler.ValidateMaxUnpaidOrders(ctx); err != nil {
		return nil, wlog.WrapError(err)
	}
	if err := handler.getAppPowerRental(ctx); err != nil {
		return nil, wlog.WrapError(err)
	}
	if err := handler.getGoodCoins(ctx); err != nil {
		return nil, wlog.WrapError(err)
	}
	if err := handler.GetAppCoins(ctx, func() (coinTypeIDs []string) {
		for _, goodCoin := range handler.goodCoins {
			coinTypeIDs = append(coinTypeIDs, goodCoin.CoinTypeID)
		}
		return
	}()); err != nil {
		return nil, wlog.WrapError(err)
	}
	if err := handler.GetTopMostAppGoods(ctx); err != nil {
		return nil, wlog.WrapError(err)
	}
	if err := handler.GetCoinUSDCurrencies(ctx); err != nil {
		return nil, wlog.WrapError(err)
	}
	if err := handler.getAppFees(ctx); err != nil {
		return nil, wlog.WrapError(err)
	}
	if err := handler.AcquirePaymentTransferAccount(ctx); err != nil {
		return nil, wlog.WrapError(err)
	}
	defer handler.ReleasePaymentTransferAccount()
	if err := handler.GetPaymentTransferStartAmount(ctx); err != nil {
		return nil, wlog.WrapError(err)
	}
	if err := handler.constructPowerRentalOrderReq(); err != nil {
		return nil, wlog.WrapError(err)
	}
	if err := handler.constructFeeOrderReqs(); err != nil {
		return nil, wlog.WrapError(err)
	}
	if err := handler.calculateTotalGoodValueUSD(); err != nil {
		return nil, wlog.WrapError(err)
	}
	if err := handler.CalculateDeductAmountUSD(); err != nil {
		return nil, wlog.WrapError(err)
	}
	handler.CalculatePaymentAmountUSD()
	if err := handler.ConstructOrderPayment(); err != nil {
		return nil, wlog.WrapError(err)
	}
	if err := handler.ResolvePaymentType(); err != nil {
		return nil, wlog.WrapError(err)
	}
	handler.PrepareLedgerLockID()
	handler.formalizePayment()
	if err := handler.ValidateCouponConstraint(); err != nil {
		return nil, wlog.WrapError(err)
	}

	if err := handler.createPowerRentalOrder(ctx); err != nil {
		return nil, wlog.WrapError(err)
	}

	return h.GetPowerRentalOrder(ctx)
}
