package powerrental

import (
	"context"

	wlog "github.com/NpoolPlatform/go-service-framework/pkg/wlog"
	npool "github.com/NpoolPlatform/message/npool/order/gw/v1/powerrental"
	ordercommon "github.com/NpoolPlatform/order-gateway/pkg/order/common"

	"github.com/shopspring/decimal"
)

type updateHandler struct {
	*baseUpdateHandler
}

//nolint:gocyclo
func (h *Handler) UpdatePowerRentalOrder(ctx context.Context) (*npool.PowerRentalOrder, error) {
	if err := h.CheckOrder(ctx); err != nil {
		return nil, wlog.WrapError(err)
	}

	handler := &updateHandler{
		baseUpdateHandler: &baseUpdateHandler{
			dtmHandler: &dtmHandler{
				checkHandler: &checkHandler{
					Handler: h,
				},
			},
			OrderOpHandler: &ordercommon.OrderOpHandler{
				AppGoodCheckHandler:         h.AppGoodCheckHandler,
				CoinCheckHandler:            h.CoinCheckHandler,
				AllocatedCouponCheckHandler: h.AllocatedCouponCheckHandler,
				PaymentTransferCoinTypeID:   h.PaymentTransferCoinTypeID,
				PaymentBalanceReqs:          h.Balances,
				OrderID:                     h.OrderID,
				AdminSetCanceled:            h.AdminSetCanceled,
				UserSetCanceled:             h.UserSetCanceled,
			},
		},
	}

	if err := handler.checkPowerRentalOrder(ctx); err != nil {
		return nil, wlog.WrapError(err)
	}
	if err := handler.getPowerRentalOrder(ctx); err != nil {
		return nil, wlog.WrapError(err)
	}
	handler.OrderOpHandler.OrderType = handler.powerRentalOrder.OrderType
	handler.OrderOpHandler.OrderState = handler.powerRentalOrder.OrderState
	if h.PaymentTransferCoinTypeID != nil || len(h.Balances) > 0 {
		if err := handler.PaymentUpdatable(); err != nil {
			return nil, wlog.WrapError(err)
		}
	}
	if h.UserSetCanceled != nil || h.AdminSetCanceled != nil {
		if err := handler.validateCancelParam(); err != nil {
			return nil, wlog.WrapError(err)
		}
		if err := handler.UserCancelable(); err != nil {
			return nil, wlog.WrapError(err)
		}
		if err := handler.getGoodBenefitTime(ctx); err != nil {
			return nil, wlog.WrapError(err)
		}
		if err := handler.getAppPowerRental(ctx); err != nil {
			return nil, wlog.WrapError(err)
		}
		handler.GoodCancelMode = handler.appPowerRental.CancelMode
		if err := handler.goodCancelable(); err != nil {
			return nil, wlog.WrapError(err)
		}
		if !handler.powerRentalOrder.Simulate {
			if err := handler.GetOrderCommissions(ctx); err != nil {
				return nil, wlog.WrapError(err)
			}
			handler.PrepareCommissionLockIDs()
		}
	}
	if err := handler.GetAppCoins(ctx, nil); err != nil {
		return nil, wlog.WrapError(err)
	}
	if err := handler.GetCoinUSDCurrencies(ctx); err != nil {
		return nil, wlog.WrapError(err)
	}
	handler.OrderOpHandler.PaymentAmountUSD, _ = decimal.NewFromString(handler.powerRentalOrder.PaymentAmountUSD)
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
	handler.constructPowerRentalOrderReq()
	if h.PaymentTransferCoinTypeID != nil || len(h.Balances) > 0 {
		if err := handler.ConstructOrderPayment(); err != nil {
			return nil, wlog.WrapError(err)
		}
		if err := handler.ResolvePaymentType(); err != nil {
			return nil, wlog.WrapError(err)
		}
		handler.PrepareLedgerLockID()
		handler.formalizePayment()
	}

	if err := handler.updatePowerRentalOrder(ctx); err != nil {
		return nil, wlog.WrapError(err)
	}

	return h.GetPowerRentalOrder(ctx)
}
