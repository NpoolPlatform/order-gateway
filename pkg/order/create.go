package order

import (
	"context"
	"fmt"
	"time"

	appcoinmwcli "github.com/NpoolPlatform/chain-middleware/pkg/client/app/coin"
	dtmcli "github.com/NpoolPlatform/dtm-cluster/pkg/dtm"
	timedef "github.com/NpoolPlatform/go-service-framework/pkg/const/time"

	redis2 "github.com/NpoolPlatform/go-service-framework/pkg/redis"
	appgoodmwcli "github.com/NpoolPlatform/good-middleware/pkg/client/app/good"
	topmostmwcli "github.com/NpoolPlatform/good-middleware/pkg/client/app/good/topmost/good"
	goodmwcli "github.com/NpoolPlatform/good-middleware/pkg/client/good"
	goodrequiredmwcli "github.com/NpoolPlatform/good-middleware/pkg/client/good/required"

	goodmwsvcname "github.com/NpoolPlatform/good-middleware/pkg/servicename"
	cruder "github.com/NpoolPlatform/libent-cruder/pkg/cruder"
	goodtypes "github.com/NpoolPlatform/message/npool/basetypes/good/v1"
	types "github.com/NpoolPlatform/message/npool/basetypes/order/v1"
	basetypes "github.com/NpoolPlatform/message/npool/basetypes/v1"
	appcoinmwpb "github.com/NpoolPlatform/message/npool/chain/mw/v1/app/coin"
	appgoodmwpb "github.com/NpoolPlatform/message/npool/good/mw/v1/app/good"
	appgoodstockmwpb "github.com/NpoolPlatform/message/npool/good/mw/v1/app/good/stock"
	topmostmwpb "github.com/NpoolPlatform/message/npool/good/mw/v1/app/good/topmost/good"
	goodmwpb "github.com/NpoolPlatform/message/npool/good/mw/v1/good"
	goodrequiredpb "github.com/NpoolPlatform/message/npool/good/mw/v1/good/required"
	allocatedmwpb "github.com/NpoolPlatform/message/npool/inspire/mw/v1/coupon/allocated"
	npool "github.com/NpoolPlatform/message/npool/order/gw/v1/order"
	ordermwpb "github.com/NpoolPlatform/message/npool/order/mw/v1/order"
	constant "github.com/NpoolPlatform/order-gateway/pkg/const"
	ordermwsvcname "github.com/NpoolPlatform/order-middleware/pkg/servicename"
	"github.com/dtm-labs/dtm/client/dtmcli/dtmimp"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

type createHandler struct {
	*baseCreateHandler
	appGood             *appgoodmwpb.Good
	parentAppGood       *appgoodmwpb.Good
	good                *goodmwpb.Good
	topMostGoods        []*topmostmwpb.TopMostGood
	priceTopMostGood    *topmostmwpb.TopMostGood
	orderStartMode      types.OrderStartMode
	orderStartAt        uint32
	orderEndAt          uint32
	goodValueUSDAmount  decimal.Decimal
	goodValueCoinAmount decimal.Decimal
}

func (h *createHandler) checkGood(ctx context.Context) error {
	good, err := goodmwcli.GetGood(ctx, h.appGood.GoodID)
	if err != nil {
		return err
	}
	if good == nil {
		return fmt.Errorf("invalid good")
	}
	h.good = good
	return nil
}

func (h *createHandler) getAppGood(ctx context.Context) error {
	good, err := appgoodmwcli.GetGood(ctx, *h.AppGoodID)
	if err != nil {
		return err
	}
	if good == nil {
		return fmt.Errorf("invalid good")
	}
	h.appGood = good
	return nil
}

func (h *createHandler) checkAppGoodCoin(ctx context.Context) error {
	goodCoin, err := appcoinmwcli.GetCoinOnly(ctx, &appcoinmwpb.Conds{
		AppID:      &basetypes.StringVal{Op: cruder.EQ, Value: *h.AppID},
		CoinTypeID: &basetypes.StringVal{Op: cruder.EQ, Value: h.appGood.CoinTypeID},
	})
	if err != nil {
		return err
	}
	if goodCoin == nil {
		return fmt.Errorf("invalid appgood coin")
	}
	h.goodCoinEnv = goodCoin.ENV
	if h.paymentCoin == nil {
		return nil
	}
	if h.paymentCoin.ENV != goodCoin.ENV {
		return fmt.Errorf("good coin mismatch payment coin")
	}

	return nil
}

func (h *createHandler) checkParentOrderGoodRequired(ctx context.Context) error {
	if h.ParentOrderID == nil {
		return nil
	}
	exist, err := goodrequiredmwcli.ExistRequiredConds(ctx, &goodrequiredpb.Conds{
		MainGoodID:     &basetypes.StringVal{Op: cruder.EQ, Value: h.parentOrder.GoodID},
		RequiredGoodID: &basetypes.StringVal{Op: cruder.EQ, Value: h.appGood.GoodID},
	})
	if err != nil {
		return err
	}
	if !exist {
		return fmt.Errorf("invalid goodrequired")
	}
	return nil
}

func (h *createHandler) checkParentGood(ctx context.Context) error {
	if h.ParentOrderID == nil {
		return nil
	}
	good, err := appgoodmwcli.GetGood(ctx, h.parentOrder.AppGoodID)
	if err != nil {
		return err
	}
	h.parentAppGood = good
	return nil
}

//nolint:gocyclo
func (h *createHandler) resolveUnits() error {
	if h.parentAppGood == nil {
		h.needCheckStock = true
		return nil
	}
	if h.parentAppGood.PackageWithRequireds {
		return fmt.Errorf("invalid parentappgood")
	}

	switch h.appGood.UnitType {
	case goodtypes.GoodUnitType_GoodUnitByDuration:
		switch h.appGood.DurationCalculateType {
		case goodtypes.GoodUnitCalculateType_GoodUnitCalculateByParent:
			return fmt.Errorf("invalid durationcalculatetype")
		case goodtypes.GoodUnitCalculateType_GoodUnitCalculateBySelf:
			if h.Duration == nil {
				return fmt.Errorf("invalid duration")
			}
		}
	case goodtypes.GoodUnitType_GoodUnitByQuantity:
		switch h.appGood.QuantityCalculateType {
		case goodtypes.GoodUnitCalculateType_GoodUnitCalculateByParent:
			h.Units = &h.parentOrder.Units
			h.needCheckStock = true
		case goodtypes.GoodUnitCalculateType_GoodUnitCalculateBySelf:
			return fmt.Errorf("invalid durationcalculatetype")
		}
	case goodtypes.GoodUnitType_GoodUnitByDurationAndQuantity:
		switch h.appGood.DurationCalculateType {
		case goodtypes.GoodUnitCalculateType_GoodUnitCalculateByParent:
			return fmt.Errorf("invalid durationcalculatetype")
		case goodtypes.GoodUnitCalculateType_GoodUnitCalculateBySelf:
			if h.Duration == nil {
				return fmt.Errorf("invalid duration")
			}
			h.needCheckStock = true
		}
		switch h.appGood.QuantityCalculateType {
		case goodtypes.GoodUnitCalculateType_GoodUnitCalculateByParent:
			h.Units = &h.parentOrder.Units
			h.needCheckStock = true
		case goodtypes.GoodUnitCalculateType_GoodUnitCalculateBySelf:
			return fmt.Errorf("invalid durationcalculatetype")
		}
	}
	return nil
}

func (h *createHandler) checkMainGood(ctx context.Context) error {
	exist, err := goodrequiredmwcli.ExistRequiredConds(ctx, &goodrequiredpb.Conds{
		RequiredGoodID: &basetypes.StringVal{Op: cruder.EQ, Value: h.appGood.GoodID},
	})
	if err != nil {
		return err
	}
	if exist && h.ParentOrderID == nil {
		return fmt.Errorf("invalid parentorderid")
	}
	return nil
}

func (h *createHandler) getAppGoodPromotion(ctx context.Context) error {
	offset := int32(0)
	limit := constant.DefaultRowLimit

	for {
		goods, _, err := topmostmwcli.GetTopMostGoods(ctx, &topmostmwpb.Conds{
			AppID:     &basetypes.StringVal{Op: cruder.EQ, Value: *h.AppID},
			AppGoodID: &basetypes.StringVal{Op: cruder.EQ, Value: *h.AppGoodID},
			// TODO: Promotion time for now
			// TODO: Other topmost check
		}, offset, limit)
		if err != nil {
			return err
		}
		if len(goods) == 0 {
			break
		}
		h.topMostGoods = append(h.topMostGoods, goods...)
		offset += limit
	}
	return nil
}

func (h *createHandler) topMostGoodPackagePrice() (decimal.Decimal, error) {
	price := decimal.NewFromInt(0)
	for _, topMost := range h.topMostGoods {
		packagePrice, err := decimal.NewFromString(topMost.PackagePrice)
		if err != nil {
			return decimal.Decimal{}, err
		}
		if packagePrice.Cmp(decimal.NewFromInt(0)) <= 0 {
			continue
		}
		if packagePrice.Cmp(price) < 0 {
			price = packagePrice
			h.priceTopMostGood = topMost
		}
	}
	return price, nil
}

func (h *createHandler) topMostGoodUnitPrice() (decimal.Decimal, error) {
	price := decimal.NewFromInt(0)
	for _, topMost := range h.topMostGoods {
		unitPrice, err := decimal.NewFromString(topMost.UnitPrice)
		if err != nil {
			return decimal.Decimal{}, err
		}
		if unitPrice.Cmp(decimal.NewFromInt(0)) <= 0 {
			continue
		}
		if unitPrice.Cmp(price) < 0 {
			price = unitPrice
			h.priceTopMostGood = topMost
		}
	}
	return price, nil
}

func (h *createHandler) goodPackagePrice() (decimal.Decimal, error) {
	if h.appGood.MinOrderDuration != h.appGood.MaxOrderDuration {
		return decimal.Decimal{}, nil
	}

	packagePrice, err := h.topMostGoodPackagePrice()
	if err != nil {
		return decimal.Decimal{}, err
	}
	if packagePrice.Cmp(decimal.NewFromInt(0)) > 0 {
		return packagePrice, nil
	}

	packagePrice, err = decimal.NewFromString(h.appGood.PackagePrice)
	if err != nil {
		return decimal.Decimal{}, err
	}
	return packagePrice, nil
}

func (h *createHandler) goodUnitPrice() (decimal.Decimal, error) {
	unitPrice, err := h.topMostGoodUnitPrice()
	if err != nil {
		return decimal.Decimal{}, err
	}
	if unitPrice.Cmp(decimal.NewFromInt(0)) > 0 {
		return unitPrice, nil
	}

	unitPrice, err = decimal.NewFromString(h.appGood.UnitPrice)
	if err != nil {
		return decimal.Decimal{}, err
	}
	return unitPrice, nil
}

// Here we get price which already calculate duration
//  GoodUnitByDuration: packagePrice or unitPrice * duration
//  GoodUnitByQuantity: packagePrice or unitPrice
//  GoodUnitByDurationAndQuantity: packagePrice or unitPrice * duration
func (h *createHandler) goodPrice() (decimal.Decimal, error) {
	packagePrice, err := h.goodPackagePrice()
	if err != nil {
		return decimal.Decimal{}, err
	}
	if packagePrice.Cmp(decimal.NewFromInt(0)) > 0 {
		return packagePrice, nil
	}

	unitPrice, err := h.goodUnitPrice()
	if err != nil {
		return decimal.Decimal{}, err
	}
	if unitPrice.Cmp(decimal.NewFromInt(0)) <= 0 {
		return decimal.Decimal{}, fmt.Errorf("invalid unitprice")
	}

	switch h.appGood.UnitType {
	case goodtypes.GoodUnitType_GoodUnitByDurationAndQuantity:
		fallthrough //nolint
	case goodtypes.GoodUnitType_GoodUnitByDuration:
		if h.Duration == nil {
			return decimal.Decimal{}, fmt.Errorf("invalid duration")
		}
		return unitPrice.Mul(decimal.NewFromInt(int64(*h.Duration))), nil
	case goodtypes.GoodUnitType_GoodUnitByQuantity:
		return unitPrice, nil
	default:
		return decimal.Decimal{}, fmt.Errorf("invalid unittype")
	}
}

func (h *createHandler) goodValue() (decimal.Decimal, error) {
	price, err := decimal.NewFromString(h.appGood.PackagePrice)
	if err != nil {
		return decimal.Decimal{}, err
	}
	if price.Cmp(decimal.NewFromInt(0)) <= 0 {
		price, err = decimal.NewFromString(h.appGood.UnitPrice)
		switch h.appGood.UnitType {
		case goodtypes.GoodUnitType_GoodUnitByDurationAndQuantity:
			fallthrough //nolint
		case goodtypes.GoodUnitType_GoodUnitByDuration:
			if h.Duration == nil {
				return decimal.Decimal{}, fmt.Errorf("invalid duration")
			}
			price = price.Mul(decimal.NewFromInt(int64(*h.Duration)))
		}
	}
	if err != nil {
		return decimal.Decimal{}, err
	}
	if price.Cmp(decimal.NewFromInt(0)) <= 0 {
		return decimal.Decimal{}, fmt.Errorf("invalid price")
	}
	units := decimal.NewFromInt(1)
	if h.Units != nil {
		units, err = decimal.NewFromString(*h.Units)
		if err != nil {
			return decimal.Decimal{}, err
		}
	}
	return price.Mul(units), nil
}

func (h *createHandler) goodPaymentUSDAmount() (decimal.Decimal, error) {
	price, err := h.goodPrice()
	if err != nil {
		return decimal.Decimal{}, err
	}
	units := decimal.NewFromInt(1)
	if h.Units != nil {
		units, err = decimal.NewFromString(*h.Units)
		if err != nil {
			return decimal.Decimal{}, err
		}
	}
	return price.Mul(units), nil
}

func (h *createHandler) calculateOrderUSDPrice() error {
	value, err := h.goodValue()
	if err != nil {
		return err
	}
	h.goodValueUSDAmount = value

	amount, err := h.goodPaymentUSDAmount()
	if err != nil {
		return err
	}
	h.paymentUSDAmount = amount
	return nil
}

func (h *createHandler) resolveStartMode() {
	switch h.appGood.StartMode {
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
	}
}

//nolint:gocyclo
func (h *createHandler) resolveStartEnd() error {
	durationUnitSeconds := timedef.SecondsPerHour
	switch h.appGood.DurationType {
	case goodtypes.GoodDurationType_GoodDurationByHour:
	case goodtypes.GoodDurationType_GoodDurationByDay:
		durationUnitSeconds = timedef.SecondsPerDay
	case goodtypes.GoodDurationType_GoodDurationByMonth:
		durationUnitSeconds = timedef.SecondsPerMonth
	case goodtypes.GoodDurationType_GoodDurationByYear:
		durationUnitSeconds = timedef.SecondsPerYear
	}

	goodStartAt := h.appGood.ServiceStartAt
	switch h.orderStartMode {
	case types.OrderStartMode_OrderStartPreset:
	case types.OrderStartMode_OrderStartInstantly:
		fallthrough //nolint
	case types.OrderStartMode_OrderStartNextDay:
		fallthrough //nolint
	case types.OrderStartMode_OrderStartTBD:
		if goodStartAt == 0 {
			goodStartAt = h.appGood.StartAt
		}
	}

	switch h.appGood.UnitType {
	case goodtypes.GoodUnitType_GoodUnitByDuration:
		fallthrough //nolint
	case goodtypes.GoodUnitType_GoodUnitByDurationAndQuantity:
		if h.appGood.MinOrderDuration == h.appGood.MaxOrderDuration && h.Duration == nil {
			h.Duration = &h.appGood.MinOrderDuration
		}
		if h.Duration == nil {
			return fmt.Errorf("invalid duration")
		}
		if *h.Duration < h.appGood.MinOrderDuration ||
			*h.Duration > h.appGood.MaxOrderDuration {
			return fmt.Errorf("invalid duration")
		}
	}

	now := uint32(time.Now().Unix())
	switch h.orderStartMode {
	case types.OrderStartMode_OrderStartTBD:
		fallthrough //nolint
	case types.OrderStartMode_OrderStartPreset:
		h.orderStartAt = goodStartAt
	case types.OrderStartMode_OrderStartInstantly:
		h.orderStartAt = now + timedef.SecondsPerMinute*10
	case types.OrderStartMode_OrderStartNextDay:
		h.orderStartAt = uint32(h.tomorrowStart().Unix())
	}

	if goodStartAt > h.orderStartAt {
		h.orderStartAt = goodStartAt
	}
	if h.orderStartAt < now {
		return fmt.Errorf("invalid startat")
	}

	durationSeconds := uint32(durationUnitSeconds) * *h.Duration
	h.orderEndAt = h.orderStartAt + durationSeconds
	return nil
}

func (h *createHandler) withUpdateStock(dispose *dtmcli.SagaDispose) {
	if !h.needCheckStock {
		return
	}
	dispose.Add(
		goodmwsvcname.ServiceDomain,
		"good.middleware.app.good1.stock.v1.Middleware/Lock",
		"good.middleware.app.good1.stock.v1.Middleware/Unlock",
		&appgoodstockmwpb.LockRequest{
			EntID:        h.appGood.AppGoodStockID,
			AppID:        h.appGood.AppID,
			GoodID:       h.appGood.GoodID,
			AppGoodID:    *h.AppGoodID,
			Units:        *h.Units,
			AppSpotUnits: decimal.NewFromInt(0).String(),
			LockID:       *h.stockLockID,
			Rollback:     true,
		},
	)
}

func (h *createHandler) withCreateOrder(dispose *dtmcli.SagaDispose) {
	goodValueCoinAmount := h.goodValueCoinAmount.String()
	goodValueUSDAmount := h.goodValueUSDAmount.String()
	paymentCoinAmount := h.paymentCoinAmount.String()
	discountCoinAmount := h.reductionCoinAmount.String()
	transferCoinAmount := h.transferCoinAmount.String()
	balanceCoinAmount := h.balanceCoinAmount.String()
	coinUSDCurrency := h.coinCurrencyAmount.String()
	localCoinUSDCurrency := h.localCurrencyAmount.String()
	liveCoinUSDCurrency := h.liveCurrencyAmount.String()

	req := &ordermwpb.OrderReq{
		EntID:                h.EntID,
		AppID:                h.AppID,
		UserID:               h.UserID,
		GoodID:               &h.appGood.GoodID,
		AppGoodID:            h.AppGoodID,
		ParentOrderID:        h.ParentOrderID,
		Units:                h.Units,
		GoodValue:            &goodValueCoinAmount,
		GoodValueUSD:         &goodValueUSDAmount,
		PaymentAmount:        &paymentCoinAmount,
		DiscountAmount:       &discountCoinAmount,
		Duration:             h.Duration,
		OrderType:            h.OrderType,
		InvestmentType:       h.InvestmentType,
		CouponIDs:            h.CouponIDs,
		PaymentType:          &h.paymentType,
		CoinTypeID:           &h.appGood.CoinTypeID,
		PaymentCoinTypeID:    h.PaymentCoinID,
		TransferAmount:       &transferCoinAmount,
		BalanceAmount:        &balanceCoinAmount,
		CoinUSDCurrency:      &coinUSDCurrency,
		LocalCoinUSDCurrency: &localCoinUSDCurrency,
		LiveCoinUSDCurrency:  &liveCoinUSDCurrency,
		StartAt:              &h.orderStartAt,
		EndAt:                &h.orderEndAt,
		StartMode:            &h.orderStartMode,
		AppGoodStockLockID:   h.stockLockID,
		LedgerLockID:         h.balanceLockID,
	}
	if h.priceTopMostGood != nil {
		req.PromotionID = &h.priceTopMostGood.TopMostID
	}
	if h.paymentAccount != nil {
		req.PaymentAccountID = &h.paymentAccount.AccountID
		paymentStartAmount := h.paymentStartAmount.String()
		req.PaymentStartAmount = &paymentStartAmount
	}
	dispose.Add(
		ordermwsvcname.ServiceDomain,
		"order.middleware.order1.v1.Middleware/CreateOrder",
		"order.middleware.order1.v1.Middleware/DeleteOrder",
		&ordermwpb.CreateOrderRequest{
			Info: req,
		},
	)
}

func (h *createHandler) calculateGoodValueCoinAmount() {
	if h.paymentCoin == nil {
		return
	}
	h.goodValueCoinAmount = h.goodValueUSDAmount.Div(h.coinCurrencyAmount)
}

//nolint:funlen,gocyclo
func (h *Handler) CreateOrder(ctx context.Context) (info *npool.Order, err error) {
	// 1 Check input
	//   1.1 Check user
	//   1.2 Check app good
	//   1.3 Check payment coin
	//   1.4 Check parent order (by middleware && handler)
	//   1.5 Check coupon ids (by handler)
	//   1.6 Check balance (by dtm lock)
	//   1.7 Check only one discount coupon
	//   1.8 Check max unpaid orders
	// 2 Calculate reduction
	//   2.1 Calculate amount of discount coupon
	//   2.2 Calculate amount of fix amount
	// 3 Calculate price
	//   3.1 Calculate USDT GoodValue - DiscountAmount
	//   3.2 Get currency
	//   3.3 Calculate payment coin amount
	// 4 Peek address
	//   4.1 Peek exist address
	//   4.2 If fail, create addresses them peek one
	//   4.3 Redis lock address
	//   4.4 Recheck address lock
	// DTM
	// 5 Lock balance
	// 6 Create order
	// 7 Lock address
	// 8 Created order notification (by scheduler)
	// 9 Order payment notification (by scheduler)
	// 10 Payment timeout notification (by scheduler)
	// 11 Reward (by scheduler)

	handler := &createHandler{
		baseCreateHandler: &baseCreateHandler{
			dtmHandler: &dtmHandler{
				Handler: h,
			},
			coupons: map[string]*allocatedmwpb.Coupon{},
		},
	}
	if err := handler.getApp(ctx); err != nil {
		return nil, err
	}
	if err := handler.getUser(ctx); err != nil {
		return nil, err
	}
	if err := handler.getAppGood(ctx); err != nil {
		return nil, err
	}
	if err := handler.checkGood(ctx); err != nil {
		return nil, err
	}
	if err := handler.getCoupons(ctx); err != nil {
		return nil, err
	}
	if err := handler.checkCouponWithdraw(ctx); err != nil {
		return nil, err
	}
	if err := handler.validateCouponScope(ctx, handler.good.EntID, *h.AppGoodID); err != nil {
		return nil, err
	}
	if err := handler.validateDiscountCoupon(); err != nil {
		return nil, err
	}
	if err := handler.checkMaxUnpaidOrders(ctx); err != nil {
		return nil, err
	}
	if err := handler.checkAppGoodCoin(ctx); err != nil {
		return nil, err
	}
	if err := handler.getPaymentCoin(ctx); err != nil {
		return nil, err
	}
	if err := handler.checkParentOrder(ctx); err != nil {
		return nil, err
	}
	if err := handler.checkParentOrderGoodRequired(ctx); err != nil {
		return nil, err
	}
	if err := handler.checkParentGood(ctx); err != nil {
		return nil, err
	}
	handler.resolveStartMode()
	if err := handler.resolveStartEnd(); err != nil {
		return nil, err
	}
	if err := handler.resolveUnits(); err != nil {
		return nil, err
	}
	if err := handler.checkUnitsLimit(ctx, handler.appGood); err != nil {
		return nil, err
	}
	if err := handler.checkMainGood(ctx); err != nil {
		return nil, err
	}
	if err := handler.getAppGoodPromotion(ctx); err != nil {
		return nil, err
	}
	if err := handler.calculateOrderUSDPrice(); err != nil {
		return nil, err
	}
	if err := handler.checkCouponConstraint(); err != nil {
		return nil, err
	}
	if err := handler.calculateDiscountCouponReduction(); err != nil {
		return nil, err
	}
	if err := handler.calculateFixAmountCouponReduction(); err != nil {
		return nil, err
	}
	if err := handler.checkPaymentCoinCurrency(ctx); err != nil {
		return nil, err
	}
	if err := handler.checkPaymentCoinAmount(); err != nil {
		return nil, err
	}
	handler.calculateGoodValueCoinAmount()
	if err := handler.checkTransferCoinAmount(); err != nil {
		return nil, err
	}

	handler.resolvePaymentType()
	handler.prepareStockAndLedgerLockIDs()

	id1 := uuid.NewString()
	if h.EntID == nil {
		h.EntID = &id1
	}

	if err := handler.acquirePaymentAddress(ctx); err != nil {
		return nil, err
	}
	defer handler.releasePaymentAddress()
	if err := handler.getPaymentStartAmount(ctx); err != nil {
		return nil, err
	}

	key := fmt.Sprintf("%v:%v:%v:%v", basetypes.Prefix_PrefixCreateOrder, *h.AppID, *h.UserID, *handler.EntID)
	if err := redis2.TryLock(key, 0); err != nil {
		return nil, err
	}
	defer func() {
		_ = redis2.Unlock(key)
	}()

	const timeoutSeconds = 10
	sagaDispose := dtmcli.NewSagaDispose(dtmimp.TransOptions{
		WaitResult:     true,
		RequestTimeout: timeoutSeconds,
		TimeoutToFail:  timeoutSeconds,
	})

	handler.withUpdateStock(sagaDispose)
	handler.withUpdateBalance(sagaDispose)
	handler.withLockPaymentAccount(sagaDispose)
	handler.withCreateOrder(sagaDispose)

	if err := handler.dtmDo(ctx, sagaDispose); err != nil {
		return nil, err
	}

	notifyCouponsUsed(handler.coupons, h.EntID)
	return h.GetOrder(ctx)
}
