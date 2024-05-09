package fee

import (
	"context"

	wlog "github.com/NpoolPlatform/go-service-framework/pkg/wlog"
	npool "github.com/NpoolPlatform/message/npool/order/gw/v1/fee"
	ordercommon "github.com/NpoolPlatform/order-gateway/pkg/order/common"

	"github.com/shopspring/decimal"
)

type updateHandler struct {
	*baseUpdateHandler
}

func (h *Handler) UpdateFeeOrder(ctx context.Context) (*npool.FeeOrder, error) {
	if err := h.CheckOrder(ctx); err != nil {
		return nil, wlog.WrapError(err)
	}

	handler := &updateHandler{
		baseUpdateHandler: &baseUpdateHandler{
			Handler: h,
			OrderOpHandler: &ordercommon.OrderOpHandler{
				AppGoodCheckHandler:         h.AppGoodCheckHandler,
				CoinCheckHandler:            h.CoinCheckHandler,
				AllocatedCouponCheckHandler: h.AllocatedCouponCheckHandler,
				PaymentTransferCoinTypeID:   h.PaymentTransferCoinTypeID,
				PaymentBalanceReqs:          h.Balances,
			},
		},
	}

	if err := handler.getFeeOrder(ctx); err != nil {
		return nil, wlog.WrapError(err)
	}
	if err := handler.GetAppCoins(ctx, nil); err != nil {
		return nil, wlog.WrapError(err)
	}
	if err := handler.GetCoinUSDCurrencies(ctx); err != nil {
		return nil, wlog.WrapError(err)
	}
	handler.OrderOpHandler.PaymentAmountUSD, _ = decimal.NewFromString(handler.feeOrder.PaymentAmountUSD)
	if err := handler.GetCoinUSDCurrencies(ctx); err != nil {
		return nil, wlog.WrapError(err)
	}
	if err := handler.AcquirePaymentTransferAccount(ctx); err != nil {
		return nil, wlog.WrapError(err)
	}
	defer handler.ReleasePaymentTransferAccount()
	if err := handler.GetPaymentTransferStartAmount(ctx); err != nil {
		return nil, wlog.WrapError(err)
	}
	handler.constructFeeOrderReq()
	if err := handler.ConstructOrderPayment(); err != nil {
		return nil, wlog.WrapError(err)
	}
	if err := handler.ResolvePaymentType(); err != nil {
		return nil, wlog.WrapError(err)
	}
	handler.PrepareLedgerLockID()
	handler.formalizePayment()

	if err := handler.updateFeeOrder(ctx); err != nil {
		return nil, wlog.WrapError(err)
	}

	return h.GetFeeOrder(ctx)
}
