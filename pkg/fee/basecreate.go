package fee

import (
	"context"

	timedef "github.com/NpoolPlatform/go-service-framework/pkg/const/time"
	wlog "github.com/NpoolPlatform/go-service-framework/pkg/wlog"
	appfeemwcli "github.com/NpoolPlatform/good-middleware/pkg/client/app/fee"
	goodcoinmwcli "github.com/NpoolPlatform/good-middleware/pkg/client/good/coin"
	cruder "github.com/NpoolPlatform/libent-cruder/pkg/cruder"
	goodtypes "github.com/NpoolPlatform/message/npool/basetypes/good/v1"
	types "github.com/NpoolPlatform/message/npool/basetypes/order/v1"
	basetypes "github.com/NpoolPlatform/message/npool/basetypes/v1"
	appfeemwpb "github.com/NpoolPlatform/message/npool/good/mw/v1/app/fee"
	appgoodmwpb "github.com/NpoolPlatform/message/npool/good/mw/v1/app/good"
	goodcoinmwpb "github.com/NpoolPlatform/message/npool/good/mw/v1/good/coin"
	feeordermwpb "github.com/NpoolPlatform/message/npool/order/mw/v1/fee"
	paymentmwpb "github.com/NpoolPlatform/message/npool/order/mw/v1/payment"
	powerrentalordermwpb "github.com/NpoolPlatform/message/npool/order/mw/v1/powerrental"
	constant "github.com/NpoolPlatform/order-gateway/pkg/const"
	ordercommon "github.com/NpoolPlatform/order-gateway/pkg/order/common"
	ordermwcli "github.com/NpoolPlatform/order-middleware/pkg/client/order"
	powerrentalordermwcli "github.com/NpoolPlatform/order-middleware/pkg/client/powerrental"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

type baseCreateHandler struct {
	*Handler
	*ordercommon.OrderCreateHandler
	parentOrder     *powerrentalordermwpb.PowerRentalOrder
	parentAppGood   *appgoodmwpb.Good
	parentGoodCoins []*goodcoinmwpb.GoodCoin
	appFees         map[string]*appfeemwpb.Fee
	feeOrderReqs    []*feeordermwpb.FeeOrderReq
}

func (h *baseCreateHandler) getParentOrder(ctx context.Context) error {
	info, err := ordermwcli.GetOrder(ctx, *h.ParentOrderID)
	if err != nil {
		return wlog.WrapError(err)
	}
	if info == nil {
		return wlog.Errorf("invalid parentorder")
	}
	switch info.GoodType {
	case goodtypes.GoodType_PowerRental:
	case goodtypes.GoodType_LegacyPowerRental:
	default:
		return wlog.Errorf("invalid parentorder goodtype")
	}
	info1, err := powerrentalordermwcli.GetPowerRentalOrder(ctx, *h.ParentOrderID)
	if err != nil {
		return wlog.WrapError(err)
	}
	if info1 == nil {
		return wlog.Errorf("invalid parentorder")
	}
	h.parentOrder = info1
	return nil
}

func (h *baseCreateHandler) getAppGoods(ctx context.Context) error {
	h.OrderCreateHandler.AppGoodIDs = append(h.OrderCreateHandler.AppGoodIDs, h.parentOrder.AppGoodID)
	if err := h.GetAppGoods(ctx); err != nil {
		return wlog.WrapError(err)
	}
	for appGoodID, appGood := range h.AppGoods {
		if appGoodID == h.parentOrder.AppGoodID {
			h.parentAppGood = appGood
			break
		}
	}
	if h.parentAppGood == nil {
		return wlog.Errorf("invalid parentappgood")
	}
	return nil
}

func (h *baseCreateHandler) getParentGoodCoins(ctx context.Context) error {
	offset := int32(0)
	limit := int32(constant.DefaultRowLimit)

	for {
		goodCoins, _, err := goodcoinmwcli.GetGoodCoins(ctx, &goodcoinmwpb.Conds{
			GoodID: &basetypes.StringVal{Op: cruder.EQ, Value: h.parentAppGood.GoodID},
		}, offset, limit)
		if err != nil {
			return wlog.WrapError(err)
		}
		if len(goodCoins) == 0 {
			return nil
		}
		h.parentGoodCoins = append(h.parentGoodCoins, goodCoins...)
		offset += limit
	}
}

func (h *baseCreateHandler) validateRequiredAppGoods() error {
	requireds, ok := h.RequiredAppGoods[h.parentAppGood.EntID]
	if !ok {
		return wlog.Errorf("invalid requiredappgood")
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
		if appGoodID == h.parentAppGood.EntID {
			continue
		}
		if _, ok := requireds[appGoodID]; !ok {
			return wlog.Errorf("invalid requiredappgood")
		}
	}
	return nil
}

func (h *baseCreateHandler) getAppFees(ctx context.Context) error {
	appFees, _, err := appfeemwcli.GetFees(ctx, &appfeemwpb.Conds{
		AppID:      &basetypes.StringVal{Op: cruder.EQ, Value: *h.OrderCheckHandler.AppID},
		AppGoodIDs: &basetypes.StringSliceVal{Op: cruder.IN, Value: h.OrderCreateHandler.AppGoodIDs},
	}, 0, int32(len(h.OrderCreateHandler.AppGoodIDs)))
	if err != nil {
		return wlog.WrapError(err)
	}
	h.appFees = map[string]*appfeemwpb.Fee{}
	for _, appFee := range appFees {
		h.appFees[appFee.AppGoodID] = appFee
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
	quantityUnits, err := decimal.NewFromString(h.parentOrder.Units)
	if err != nil {
		return value, wlog.WrapError(err)
	}
	durationUnits := *h.Handler.DurationSeconds
	switch appFee.DurationDisplayType {
	case goodtypes.GoodDurationType_GoodDurationByHour:
		durationUnits /= timedef.SecondsPerHour
	case goodtypes.GoodDurationType_GoodDurationByDay:
		durationUnits /= timedef.SecondsPerDay
	case goodtypes.GoodDurationType_GoodDurationByMonth:
		durationUnits /= timedef.SecondsPerMonth
	case goodtypes.GoodDurationType_GoodDurationByYear:
		durationUnits /= timedef.SecondsPerYear
	default:
		return value, wlog.Errorf("invalid appfee durationdisplaytype")
	}
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
	if len(h.feeOrderReqs) == 0 {
		paymentAmountUSD = decimal.NewFromInt(0)
		paymentType = types.PaymentType_PayWithOtherOrder
	}
	var promotionID *string
	topMostAppGood, ok := h.TopMostAppGoods[appFee.AppGoodID]
	if ok {
		promotionID = &topMostAppGood.TopMostID
	}
	req := &feeordermwpb.FeeOrderReq{
		EntID:         func() *string { s := uuid.NewString(); return &s }(),
		AppID:         h.Handler.OrderCheckHandler.AppID,
		UserID:        h.Handler.OrderCheckHandler.UserID,
		GoodID:        &appFee.GoodID,
		GoodType:      &appFee.GoodType,
		AppGoodID:     &appFee.AppGoodID,
		OrderID:       func() *string { s := uuid.NewString(); return &s }(),
		ParentOrderID: &h.parentOrder.OrderID,
		OrderType:     h.Handler.OrderType,
		PaymentType:   &paymentType,
		CreateMethod:  h.CreateMethod, // Admin or Purchase

		GoodValueUSD:      func() *string { s := goodValueUSD.String(); return &s }(),
		PaymentAmountUSD:  func() *string { s := paymentAmountUSD.String(); return &s }(),
		DiscountAmountUSD: func() *string { s := h.DeductAmountUSD.String(); return &s }(),
		PromotionID:       promotionID,
		DurationSeconds:   h.Handler.DurationSeconds,
		LedgerLockID:      h.BalanceLockID,
		PaymentID:         h.PaymentID,
		CouponIDs:         h.CouponIDs,
		PaymentBalances:   h.PaymentBalanceReqs,
		PaymentTransfers:  []*paymentmwpb.PaymentTransferReq{h.PaymentTransferReq},
	}
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
