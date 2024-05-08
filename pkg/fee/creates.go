package fee

import (
	"context"

	wlog "github.com/NpoolPlatform/go-service-framework/pkg/wlog"
	npool "github.com/NpoolPlatform/message/npool/order/gw/v1/fee"
	ordercommon "github.com/NpoolPlatform/order-gateway/pkg/order/common"
)

type createsHandler struct {
	*baseCreateHandler
}

func (h *Handler) CreateFeeOrders(ctx context.Context) ([]*npool.FeeOrder, error) {
	handler := &createsHandler{
		baseCreateHandler: &baseCreateHandler{
			Handler: h,
			OrderCreateHandler: &ordercommon.OrderCreateHandler{
				AppGoodCheckHandler:         h.AppGoodCheckHandler,
				CoinCheckHandler:            h.CoinCheckHandler,
				AllocatedCouponCheckHandler: h.AllocatedCouponCheckHandler,
				AppGoodIDs:                  h.AppGoodIDs,
				PaymentTransferCoinTypeID:   h.PaymentTransferCoinTypeID,
				PaymentBalanceReqs:          h.Balances,
			},
		},
	}

	if err := handler.getParentOrder(ctx); err != nil {
		return nil, wlog.WrapError(err)
	}
	handler.OrderCreateHandler.AppGoodIDs = append(
		handler.OrderCreateHandler.AppGoodIDs,
		handler.parentOrder.AppGoodID,
	)
	if err := handler.GetApp(ctx); err != nil {
		return nil, wlog.WrapError(err)
	}
	if err := handler.GetUser(ctx); err != nil {
		return nil, wlog.WrapError(err)
	}
	if err := handler.getAppGoods(ctx); err != nil {
		return nil, wlog.WrapError(err)
	}
	if err := handler.GetAllocatedCoupons(ctx); err != nil {
		return nil, wlog.WrapError(err)
	}
	if err := handler.ValidateCouponScope(ctx, &handler.parentOrder.AppGoodID); err != nil {
		return nil, wlog.WrapError(err)
	}
	if err := handler.ValidateCouponCount(); err != nil {
		return nil, wlog.WrapError(err)
	}
	if err := handler.ValidateMaxUnpaidOrders(ctx); err != nil {
		return nil, wlog.WrapError(err)
	}
	if err := handler.getParentGoodCoins(ctx); err != nil {
		return nil, wlog.WrapError(err)
	}
	if err := handler.GetAppCoins(ctx, func() (coinTypeIDs []string) {
		for _, goodCoin := range handler.parentGoodCoins {
			coinTypeIDs = append(coinTypeIDs, goodCoin.CoinTypeID)
		}
		return
	}()); err != nil {
		return nil, wlog.WrapError(err)
	}
	if err := handler.GetRequiredAppGoods(ctx); err != nil {
		return nil, wlog.WrapError(err)
	}
	if err := handler.validateRequiredAppGoods(); err != nil {
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
	handler.formalizePayment()
	handler.PrepareLedgerLockID()
	if err := handler.ValidateCouponConstraint(); err != nil {
		return nil, wlog.WrapError(err)
	}

	if err := handler.createFeeOrders(ctx); err != nil {
		return nil, wlog.WrapError(err)
	}

	infos, _, err := h.GetFeeOrders(ctx)
	if err != nil {
		return nil, wlog.WrapError(err)
	}
	return infos, nil
}
