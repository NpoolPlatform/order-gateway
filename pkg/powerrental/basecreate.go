package powerrental

import (
	"context"
	"time"

	dtmcli "github.com/NpoolPlatform/dtm-cluster/pkg/dtm"
	timedef "github.com/NpoolPlatform/go-service-framework/pkg/const/time"
	wlog "github.com/NpoolPlatform/go-service-framework/pkg/wlog"
	apppowerrentalmwcli "github.com/NpoolPlatform/good-middleware/pkg/client/app/powerrental"
	apppowerrentalsimulatemwcli "github.com/NpoolPlatform/good-middleware/pkg/client/app/powerrental/simulate"
	goodcoinmwcli "github.com/NpoolPlatform/good-middleware/pkg/client/good/coin"
	goodmwsvcname "github.com/NpoolPlatform/good-middleware/pkg/servicename"
	cruder "github.com/NpoolPlatform/libent-cruder/pkg/cruder"
	goodtypes "github.com/NpoolPlatform/message/npool/basetypes/good/v1"
	types "github.com/NpoolPlatform/message/npool/basetypes/order/v1"
	basetypes "github.com/NpoolPlatform/message/npool/basetypes/v1"
	appfeemwpb "github.com/NpoolPlatform/message/npool/good/mw/v1/app/fee"
	appgoodstockmwpb "github.com/NpoolPlatform/message/npool/good/mw/v1/app/good/stock"
	apppowerrentalmwpb "github.com/NpoolPlatform/message/npool/good/mw/v1/app/powerrental"
	apppowerrentalsimulatemwpb "github.com/NpoolPlatform/message/npool/good/mw/v1/app/powerrental/simulate"
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
	*dtmHandler
	*ordercommon.OrderOpHandler
	appPowerRental         *apppowerrentalmwpb.PowerRental
	appPowerRentalSimulate *apppowerrentalsimulatemwpb.Simulate
	goodCoins              []*goodcoinmwpb.GoodCoin
	appFees                map[string]*appfeemwpb.Fee
	powerRentalOrderReq    *powerrentalordermwpb.PowerRentalOrderReq
	feeOrderReqs           []*feeordermwpb.FeeOrderReq
	appGoodStockLockID     *string
	orderStartMode         types.OrderStartMode
	orderStartAt           uint32
}

func (h *baseCreateHandler) getAppGoods(ctx context.Context) error {
	if err := h.GetAppGoods(ctx); err != nil {
		return wlog.WrapError(err)
	}
	return nil
}

func (h *baseCreateHandler) getGoodCoins(ctx context.Context) error {
	offset := int32(0)
	limit := constant.DefaultRowLimit

	for {
		goodCoins, _, err := goodcoinmwcli.GetGoodCoins(ctx, &goodcoinmwpb.Conds{
			GoodID: &basetypes.StringVal{Op: cruder.EQ, Value: h.appPowerRental.GoodID},
		}, offset, limit)
		if err != nil {
			return wlog.WrapError(err)
		}
		if len(goodCoins) == 0 {
			break
		}
		h.goodCoins = append(h.goodCoins, goodCoins...)
		offset += limit
	}
	if len(h.goodCoins) == 0 {
		return wlog.Errorf("invalid goodcoins")
	}
	return nil
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
	for _, appGoodID := range h.FeeAppGoodIDs {
		if _, ok := requireds[appGoodID]; !ok {
			return wlog.Errorf("invalid requiredappgood")
		}
	}
	return nil
}

func (h *baseCreateHandler) formalizeFeeAppGoodIDs() {
	if !h.appPowerRental.PackageWithRequireds {
		return
	}
	requireds, ok := h.RequiredAppGoods[*h.Handler.AppGoodID]
	if !ok {
		return
	}
	for _, required := range requireds {
		if !required.Must {
			continue
		}
		h.Handler.FeeAppGoodIDs = append(h.Handler.FeeAppGoodIDs, required.RequiredAppGoodID)
	}
}

func (h *baseCreateHandler) getAppFees(ctx context.Context) (err error) {
	h.appFees, err = ordergwcommon.GetAppFees(ctx, h.Handler.FeeAppGoodIDs)
	return wlog.WrapError(err)
}

func (h *baseCreateHandler) getAppPowerRental(ctx context.Context) (err error) {
	h.appPowerRental, err = apppowerrentalmwcli.GetPowerRental(ctx, *h.Handler.AppGoodID)
	if err != nil {
		return wlog.WrapError(err)
	}
	if h.appPowerRental == nil || !h.appPowerRental.AppGoodOnline || !h.appPowerRental.GoodOnline {
		return wlog.Errorf("invalid apppowerrental")
	}

	if !h.appPowerRental.AppGoodPurchasable || !h.appPowerRental.GoodPurchasable {
		if *h.CreateMethod != types.OrderCreateMethod_OrderCreatedByAdmin {
			return wlog.Errorf("invalid apppowerrental")
		}
	}
	return nil
}

func (h *baseCreateHandler) getAppPowerRentalSimulate(ctx context.Context) (err error) {
	h.appPowerRentalSimulate, err = apppowerrentalsimulatemwcli.GetSimulate(ctx, *h.Handler.AppGoodID)
	if err != nil {
		return wlog.WrapError(err)
	}
	if h.appPowerRentalSimulate == nil {
		return wlog.Errorf("invalid apppowerrentalsimulate")
	}
	return nil
}

func (h *baseCreateHandler) formalizeSimulateOrder() error {
	units, err := decimal.NewFromString(h.appPowerRentalSimulate.OrderUnits)
	if err != nil {
		return wlog.WrapError(err)
	}
	h.Units = &units
	h.DurationSeconds = &h.appPowerRentalSimulate.OrderDurationSeconds
	return nil
}

func (h *baseCreateHandler) validateOrderDuration() error {
	if h.appPowerRental.FixedDuration {
		h.Handler.DurationSeconds = &h.appPowerRental.MinOrderDurationSeconds
		return nil
	}
	if h.Handler.DurationSeconds == nil {
		return wlog.Errorf("invalid durationseconds")
	}
	if *h.Handler.DurationSeconds < h.appPowerRental.MinOrderDurationSeconds ||
		*h.Handler.DurationSeconds > h.appPowerRental.MaxOrderDurationSeconds {
		return wlog.Errorf("invalid durationseconds")
	}
	return nil
}

func (h *baseCreateHandler) validateOrderUnits() error {
	minOrderAmount, err := decimal.NewFromString(h.appPowerRental.MinOrderAmount)
	if err != nil {
		return wlog.WrapError(err)
	}
	if h.Units.LessThan(minOrderAmount) {
		return wlog.Errorf("invalid orderunits")
	}
	maxOrderAmount, err := decimal.NewFromString(h.appPowerRental.MaxOrderAmount)
	if err != nil {
		return wlog.WrapError(err)
	}
	if h.Units.GreaterThan(maxOrderAmount) {
		return wlog.Errorf("invalid orderunits")
	}
	maxUserAmount, err := decimal.NewFromString(h.appPowerRental.MaxUserAmount)
	if err != nil {
		return wlog.WrapError(err)
	}
	if h.Units.GreaterThan(maxUserAmount) {
		return wlog.Errorf("invalid orderunits")
	}
	return nil
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
	if h.Handler.FeeDurationSeconds == nil {
		return decimal.NewFromInt(0), wlog.Errorf("invalid feedurationseconds")
	}
	durationUnits, _ := ordergwcommon.GoodDurationDisplayType2Unit(
		appFee.DurationDisplayType, *h.Handler.FeeDurationSeconds,
	)
	*h.Handler.FeeDurationSeconds = ordergwcommon.GoodDurationDisplayType2Seconds(appFee.DurationDisplayType) * durationUnits
	return unitValue.Mul(quantityUnits).Mul(decimal.NewFromInt(int64(durationUnits))), nil
}

func (h *baseCreateHandler) checkEnableSimulateOrder() error {
	if h.OrderOpHandler.Simulate && h.OrderConfig != nil && !h.OrderConfig.EnableSimulateOrder {
		return wlog.Errorf("permission denied")
	}
	return nil
}

func (h *baseCreateHandler) calculatePowerRentalOrderValueUSD() (value decimal.Decimal, err error) {
	unitValue, err := decimal.NewFromString(h.appPowerRental.UnitPrice)
	if err != nil {
		return value, wlog.WrapError(err)
	}
	if h.appPowerRental.FixedDuration {
		return unitValue, nil
	}
	quantityUnits := *h.Handler.Units
	durationUnits, _ := ordergwcommon.GoodDurationDisplayType2Unit(
		h.appPowerRental.DurationDisplayType, *h.Handler.DurationSeconds,
	)
	*h.Handler.DurationSeconds = ordergwcommon.GoodDurationDisplayType2Seconds(h.appPowerRental.DurationDisplayType) * durationUnits
	return unitValue.Mul(quantityUnits).Mul(decimal.NewFromInt(int64(durationUnits))), nil
}

func (h *baseCreateHandler) calculateTotalGoodValueUSD() (err error) {
	h.TotalGoodValueUSD, err = h.calculatePowerRentalOrderValueUSD()
	if err != nil {
		return err
	}
	if h.appPowerRental.PackageWithRequireds {
		return nil
	}
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
		PaymentType:  func() *types.PaymentType { e := types.PaymentType_PayWithParentOrder; return &e }(),
		CreateMethod: h.CreateMethod, // Admin or Purchase

		GoodValueUSD: func() *string {
			s := goodValueUSD.String()
			if h.appPowerRental.PackageWithRequireds {
				s = decimal.NewFromInt(0).String()
			}
			return &s
		}(),
		PaymentAmountUSD:  func() *string { s := decimal.NewFromInt(0).String(); return &s }(),
		DiscountAmountUSD: func() *string { s := decimal.NewFromInt(0).String(); return &s }(),
		PromotionID:       promotionID,
		DurationSeconds: func() *uint32 {
			if h.appPowerRental.PackageWithRequireds {
				return h.Handler.DurationSeconds
			} else {
				return h.Handler.FeeDurationSeconds
			}
		}(),
		PaymentID: h.PaymentID,
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

func (h *baseCreateHandler) resolveStartMode() error {
	switch h.appPowerRental.AppGoodStartMode {
	case goodtypes.GoodStartMode_GoodStartModeTBD:
		h.orderStartMode = types.OrderStartMode_OrderStartTBD
	case goodtypes.GoodStartMode_GoodStartModeConfirmed:
		h.orderStartMode = types.OrderStartMode_OrderStartNextDay
	case goodtypes.GoodStartMode_GoodStartModeInstantly:
		h.orderStartMode = types.OrderStartMode_OrderStartInstantly
	case goodtypes.GoodStartMode_GoodStartModeNextDay:
		h.orderStartMode = types.OrderStartMode_OrderStartNextDay
	case goodtypes.GoodStartMode_GoodStartModePreset:
		h.orderStartMode = types.OrderStartMode_OrderStartPreset
	default:
		return wlog.Errorf("invalid goodstartmode")
	}
	return nil
}

func (h *baseCreateHandler) resolveStartAt() error {
	now := uint32(time.Now().Unix())
	switch h.orderStartMode {
	case types.OrderStartMode_OrderStartTBD:
		fallthrough //nolint
	case types.OrderStartMode_OrderStartPreset:
		h.orderStartAt = h.appPowerRental.AppGoodServiceStartAt
	case types.OrderStartMode_OrderStartInstantly:
		h.orderStartAt = now + timedef.SecondsPerMinute*10
	case types.OrderStartMode_OrderStartNextDay:
		h.orderStartAt = uint32(timedef.TomorrowStart().Unix())
	}

	if h.appPowerRental.AppGoodServiceStartAt > h.orderStartAt {
		h.orderStartAt = h.appPowerRental.AppGoodServiceStartAt
	}
	if h.orderStartAt < now {
		return wlog.Errorf("invalid orderstartat")
	}
	return nil
}

func (h *baseCreateHandler) constructPowerRentalOrderReq() error {
	if err := h.resolveStartMode(); err != nil {
		return wlog.WrapError(err)
	}
	if err := h.resolveStartAt(); err != nil {
		return wlog.WrapError(err)
	}
	goodValueUSD, err := h.calculatePowerRentalOrderValueUSD()
	if err != nil {
		return wlog.WrapError(err)
	}
	var promotionID *string
	topMostAppGood, ok := h.TopMostAppGoods[*h.Handler.AppGoodID]
	if ok {
		promotionID = &topMostAppGood.TopMostID
	}
	h.appGoodStockLockID = func() *string { s := uuid.NewString(); return &s }()
	req := &powerrentalordermwpb.PowerRentalOrderReq{
		EntID:        func() *string { s := uuid.NewString(); return &s }(),
		AppID:        h.Handler.OrderCheckHandler.AppID,
		UserID:       h.Handler.OrderCheckHandler.UserID,
		GoodID:       &h.appPowerRental.GoodID,
		GoodType:     &h.appPowerRental.GoodType,
		AppGoodID:    &h.appPowerRental.AppGoodID,
		OrderID:      func() *string { s := uuid.NewString(); return &s }(),
		OrderType:    h.Handler.OrderType,
		CreateMethod: h.CreateMethod, // Admin or Purchase
		Simulate:     h.Handler.Simulate,

		AppGoodStockID:    h.AppGoodStockID,
		Units:             func() *string { s := h.Handler.Units.String(); return &s }(),
		GoodValueUSD:      func() *string { s := goodValueUSD.String(); return &s }(),
		PaymentAmountUSD:  func() *string { s := h.PaymentAmountUSD.String(); return &s }(),
		DiscountAmountUSD: func() *string { s := h.DeductAmountUSD.String(); return &s }(),
		PromotionID:       promotionID,
		DurationSeconds:   h.Handler.DurationSeconds,
		InvestmentType:    h.Handler.InvestmentType,

		StartMode: &h.orderStartMode,
		StartAt:   &h.orderStartAt,

		AppGoodStockLockID: h.appGoodStockLockID,
		LedgerLockID:       h.BalanceLockID,
		CouponIDs:          h.CouponIDs,
		PaymentID:          h.PaymentID,
	}
	req.PaymentBalances = h.PaymentBalanceReqs
	if h.PaymentTransferReq != nil {
		req.PaymentTransfers = []*paymentmwpb.PaymentTransferReq{h.PaymentTransferReq}
	}
	h.OrderID = req.OrderID
	h.dtmHandler.OrderID = req.OrderID
	h.OrderIDs = append(h.OrderIDs, *req.OrderID)
	h.powerRentalOrderReq = req
	return nil
}

func (h *baseCreateHandler) formalizePayment() {
	h.powerRentalOrderReq.PaymentType = h.PaymentType
	h.powerRentalOrderReq.PaymentBalances = h.PaymentBalanceReqs
	if h.PaymentTransferReq != nil {
		h.powerRentalOrderReq.PaymentTransfers = []*paymentmwpb.PaymentTransferReq{h.PaymentTransferReq}
	}
	h.powerRentalOrderReq.PaymentAmountUSD = func() *string { s := h.PaymentAmountUSD.String(); return &s }()
	h.powerRentalOrderReq.DiscountAmountUSD = func() *string { s := h.DeductAmountUSD.String(); return &s }()
	h.powerRentalOrderReq.LedgerLockID = h.BalanceLockID
}

func (h *baseCreateHandler) notifyCouponUsed() {

}

func (h *baseCreateHandler) withCreatePowerRentalOrderWithFees(dispose *dtmcli.SagaDispose) {
	dispose.Add(
		ordermwsvcname.ServiceDomain,
		"order.middleware.powerrental.v1.Middleware/CreatePowerRentalOrderWithFees",
		"",
		&powerrentalordermwpb.CreatePowerRentalOrderWithFeesRequest{
			PowerRentalOrder: h.powerRentalOrderReq,
			FeeOrders:        h.feeOrderReqs,
		},
	)
}

func (h *baseCreateHandler) withLockStock(dispose *dtmcli.SagaDispose) {
	dispose.Add(
		goodmwsvcname.ServiceDomain,
		"good.middleware.app.good1.stock.v1.Middleware/Lock",
		"good.middleware.app.good1.stock.v1.Middleware/Unlock",
		&appgoodstockmwpb.LockRequest{
			EntID:     *h.AppGoodStockID,
			AppGoodID: *h.Handler.AppGoodID,
			Units:     h.Units.String(),
			AppSpotUnits: func() string {
				if h.AppSpotUnits != nil {
					return h.AppSpotUnits.String()
				}
				return decimal.NewFromInt(0).String()
			}(),
			LockID:   *h.appGoodStockLockID,
			Rollback: true,
		},
	)
}

func (h *baseCreateHandler) createPowerRentalOrder(ctx context.Context) error {
	sagaDispose := dtmcli.NewSagaDispose(dtmimp.TransOptions{
		WaitResult:     true,
		RequestTimeout: 10,
		TimeoutToFail:  10,
	})
	if !h.OrderOpHandler.Simulate {
		if h.AppGoodStockID == nil {
			return wlog.Errorf("invalid appgoodstockid")
		}
		h.withLockStock(sagaDispose)
		h.WithLockBalances(sagaDispose)
		h.WithLockPaymentTransferAccount(sagaDispose)
	}
	h.withCreatePowerRentalOrderWithFees(sagaDispose)
	defer h.notifyCouponUsed()
	return h.dtmDo(ctx, sagaDispose)
}
