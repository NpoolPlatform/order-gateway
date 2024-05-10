package powerrental

import (
	"context"
	"time"

	dtmcli "github.com/NpoolPlatform/dtm-cluster/pkg/dtm"
	logger "github.com/NpoolPlatform/go-service-framework/pkg/logger"
	wlog "github.com/NpoolPlatform/go-service-framework/pkg/wlog"
	appfeemwcli "github.com/NpoolPlatform/good-middleware/pkg/client/app/fee"
	apppowerrentalmwcli "github.com/NpoolPlatform/good-middleware/pkg/client/app/powerrental"
	goodcoinmwcli "github.com/NpoolPlatform/good-middleware/pkg/client/good/coin"
	cruder "github.com/NpoolPlatform/libent-cruder/pkg/cruder"
	goodtypes "github.com/NpoolPlatform/message/npool/basetypes/good/v1"
	types "github.com/NpoolPlatform/message/npool/basetypes/order/v1"
	basetypes "github.com/NpoolPlatform/message/npool/basetypes/v1"
	appfeemwpb "github.com/NpoolPlatform/message/npool/good/mw/v1/app/fee"
	apppowerrentalmwpb "github.com/NpoolPlatform/message/npool/good/mw/v1/app/powerrental"
	goodcoinmwpb "github.com/NpoolPlatform/message/npool/good/mw/v1/good/coin"
	feeordermwpb "github.com/NpoolPlatform/message/npool/order/mw/v1/fee"
	paymentmwpb "github.com/NpoolPlatform/message/npool/order/mw/v1/payment"
	powerrentalordermwpb "github.com/NpoolPlatform/message/npool/order/mw/v1/powerrental"
	ordergwcommon "github.com/NpoolPlatform/order-gateway/pkg/common"
	constant "github.com/NpoolPlatform/order-gateway/pkg/const"
	ordercommon "github.com/NpoolPlatform/order-gateway/pkg/order/common"
	ordermwsvcname "github.com/NpoolPlatform/order-middleware/pkg/servicename"
	"github.com/dtm-labs/dtm/client/dtmcli/dtmimp"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

type baseCreateHandler struct {
	*Handler
	*ordercommon.OrderOpHandler
	appPowerRental      *apppowerrentalmwpb.PowerRental
	goodCoins           []*goodcoinmwpb.GoodCoin
	appFees             map[string]*appfeemwpb.Fee
	powerRentalOrderReq *powerrentalordermwpb.PowerRentalOrderReq
	feeOrderReqs        []*feeordermwpb.FeeOrderReq
}

func (h *baseCreateHandler) getAppGoods(ctx context.Context) error {
	if err := h.GetAppGoods(ctx); err != nil {
		return wlog.WrapError(err)
	}
	return nil
}

func (h *baseCreateHandler) getGoodCoins(ctx context.Context) error {
	offset := int32(0)
	limit := int32(constant.DefaultRowLimit)

	for {
		goodCoins, _, err := goodcoinmwcli.GetGoodCoins(ctx, &goodcoinmwpb.Conds{
			GoodID: &basetypes.StringVal{Op: cruder.EQ, Value: h.appPowerRental.GoodID},
		}, offset, limit)
		if err != nil {
			return wlog.WrapError(err)
		}
		if len(goodCoins) == 0 {
			return nil
		}
		h.goodCoins = append(h.goodCoins, goodCoins...)
		offset += limit
	}
}

func (h *baseCreateHandler) validateRequiredAppGoods() error {
	requireds, ok := h.RequiredAppGoods[*h.Handler.AppGoodID]
	if !ok {
		return nil
	}
	for _, required := range requireds {
		if !required.Must {
			continue
		}
		if _, ok := h.AppGoods[required.RequiredAppGoodID]; !ok {
			return wlog.Errorf("miss requiredappgood")
		}
	}
	for appGoodID, _ := range h.AppGoods {
		if _, ok := requireds[appGoodID]; !ok {
			return wlog.Errorf("invalid requiredappgood")
		}
	}
	return nil
}

func (h *baseCreateHandler) getAppFees(ctx context.Context) error {
	appFees, _, err := appfeemwcli.GetFees(ctx, &appfeemwpb.Conds{
		AppID:      &basetypes.StringVal{Op: cruder.EQ, Value: *h.OrderCheckHandler.AppID},
		AppGoodIDs: &basetypes.StringSliceVal{Op: cruder.IN, Value: h.Handler.FeeAppGoodIDs},
	}, 0, int32(len(h.Handler.FeeAppGoodIDs)))
	if err != nil {
		return wlog.WrapError(err)
	}
	h.appFees = map[string]*appfeemwpb.Fee{}
	for _, appFee := range appFees {
		h.appFees[appFee.AppGoodID] = appFee
	}
	return nil
}

func (h *baseCreateHandler) getAppPowerRental(ctx context.Context) (err error) {
	h.appPowerRental, err = apppowerrentalmwcli.GetPowerRental(ctx, *h.Handler.AppGoodID)
	return wlog.WrapError(err)
}

func (h *baseCreateHandler) calculateFeeOrderValueUSD(appGoodID string) (value decimal.Decimal, err error) {
	appFee, ok := h.appFees[appGoodID]
	if !ok {
		return value, wlog.Errorf("invalid appfee")
	}
	unitValue, err := decimal.NewFromString(appFee.UnitValue)
	if err != nil {
		return value, wlog.WrapError(err)
	}
	quantityUnits := *h.Handler.Units
	durationUnits, _ := ordergwcommon.GoodDurationDisplayType2Unit(
		appFee.DurationDisplayType, *h.Handler.DurationSeconds,
	)
	return unitValue.Mul(quantityUnits).Mul(decimal.NewFromInt(int64(durationUnits))), nil
}

func (h *baseCreateHandler) calculateTotalGoodValueUSD() error {
	for _, appFee := range h.appFees {
		if appFee.SettlementType != goodtypes.GoodSettlementType_GoodSettledByPaymentAmount {
			return wlog.Errorf("invalid appfee settlementtype")
		}
		goodValueUSD, err := h.calculateFeeOrderValueUSD(appFee.AppGoodID)
		if err != nil {
			return wlog.WrapError(err)
		}
		h.TotalGoodValueUSD = h.TotalGoodValueUSD.Add(goodValueUSD)
	}
	return nil
}

func (h *baseCreateHandler) constructFeeOrderReq(appGoodID string) error {
	appFee, ok := h.appFees[appGoodID]
	if !ok {
		return wlog.Errorf("invalid appfee")
	}
	goodValueUSD, err := h.calculateFeeOrderValueUSD(appGoodID)
	if err != nil {
		return wlog.WrapError(err)
	}
	paymentAmountUSD := h.PaymentAmountUSD
	paymentType := h.PaymentType
	if len(h.feeOrderReqs) > 0 {
		paymentAmountUSD = decimal.NewFromInt(0)
		paymentType = types.PaymentType_PayWithOtherOrder
	}
	var promotionID *string
	topMostAppGood, ok := h.TopMostAppGoods[appFee.AppGoodID]
	if ok {
		promotionID = &topMostAppGood.TopMostID
	}
	req := &feeordermwpb.FeeOrderReq{
		EntID:        func() *string { s := uuid.NewString(); return &s }(),
		AppID:        h.Handler.OrderCheckHandler.AppID,
		UserID:       h.Handler.OrderCheckHandler.UserID,
		GoodID:       &appFee.GoodID,
		GoodType:     &appFee.GoodType,
		AppGoodID:    &appFee.AppGoodID,
		OrderID:      func() *string { s := uuid.NewString(); return &s }(),
		OrderType:    h.Handler.OrderType,
		PaymentType:  &paymentType,
		CreateMethod: h.CreateMethod, // Admin or Purchase

		GoodValueUSD:      func() *string { s := goodValueUSD.String(); return &s }(),
		PaymentAmountUSD:  func() *string { s := paymentAmountUSD.String(); return &s }(),
		DiscountAmountUSD: func() *string { s := h.DeductAmountUSD.String(); return &s }(),
		PromotionID:       promotionID,
		DurationSeconds:   h.Handler.DurationSeconds,
		LedgerLockID:      h.BalanceLockID,
		PaymentID:         h.PaymentID,
		CouponIDs:         h.CouponIDs,
	}
	if len(h.feeOrderReqs) == 0 {
		req.PaymentBalances = h.PaymentBalanceReqs
		if h.PaymentTransferReq != nil {
			req.PaymentTransfers = []*paymentmwpb.PaymentTransferReq{h.PaymentTransferReq}
		}
		h.OrderID = req.OrderID
	}
	h.OrderIDs = append(h.OrderIDs, *req.OrderID)
	h.feeOrderReqs = append(h.feeOrderReqs, req)
	return nil
}

func (h *baseCreateHandler) constructFeeOrderReqs() error {
	for _, appFee := range h.appFees {
		if err := h.constructFeeOrderReq(appFee.AppGoodID); err != nil {
			return wlog.WrapError(err)
		}
	}
	return nil
}

func (h *baseCreateHandler) formalizePayment() {
	h.feeOrderReqs[0].PaymentType = &h.PaymentType
	h.feeOrderReqs[0].PaymentBalances = h.PaymentBalanceReqs
	if h.PaymentTransferReq != nil {
		h.feeOrderReqs[0].PaymentTransfers = []*paymentmwpb.PaymentTransferReq{h.PaymentTransferReq}
	}
	h.feeOrderReqs[0].PaymentAmountUSD = func() *string { s := h.PaymentAmountUSD.String(); return &s }()
	h.feeOrderReqs[0].DiscountAmountUSD = func() *string { s := h.DeductAmountUSD.String(); return &s }()
	h.feeOrderReqs[0].LedgerLockID = h.BalanceLockID
}

func (h *baseCreateHandler) dtmDo(ctx context.Context, dispose *dtmcli.SagaDispose) error {
	start := time.Now()
	_ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	err := dtmcli.WithSaga(_ctx, dispose)
	dtmElapsed := time.Since(start)
	logger.Sugar().Infow(
		"CreatePowerRentalOrderWithFees",
		"OrderID", *h.feeOrderReqs[0].OrderID,
		"Start", start,
		"DtmElapsed", dtmElapsed,
		"Error", err,
	)
	return wlog.WrapError(err)
}

func (h *baseCreateHandler) notifyCouponUsed() {

}

func (h *baseCreateHandler) _createPowerRentalOrderWithFees(dispose *dtmcli.SagaDispose) {
	dispose.Add(
		ordermwsvcname.ServiceDomain,
		"order.middleware.fee.v1.Middleware/CreatePowerRentalOrderWithFees",
		"",
		&powerrentalordermwpb.CreatePowerRentalOrderWithFeesRequest{
			PowerRentalOrder: h.powerRentalOrderReq,
			FeeOrders:        h.feeOrderReqs,
		},
	)
}

func (h *baseCreateHandler) createPowerRentalOrder(ctx context.Context) error {
	sagaDispose := dtmcli.NewSagaDispose(dtmimp.TransOptions{
		WaitResult:     true,
		RequestTimeout: 10,
		TimeoutToFail:  10,
	})
	h.LockBalances(sagaDispose)
	h.LockPaymentTransferAccount(sagaDispose)
	h._createPowerRentalOrderWithFees(sagaDispose)
	defer h.notifyCouponUsed()
	return h.dtmDo(ctx, sagaDispose)
}
