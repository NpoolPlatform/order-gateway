//nolint:dupl
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

type createsHandler struct {
	*baseCreateHandler
	orderReqs         []*ordermwpb.OrderReq
	appGoods          map[string]*appgoodmwpb.Good
	goods             map[string]*goodmwpb.Good
	parentAppGood     *appgoodmwpb.Good
	parentGood        *goodmwpb.Good
	requiredGoods     map[string]*goodrequiredpb.Required
	topMostGoods      map[string][]*topmostmwpb.TopMostGood
	priceTopMostGoods map[string]*topmostmwpb.TopMostGood
}

func (h *createsHandler) checkAppGoods(ctx context.Context) error {
	var appGoodIDs []string
	for _, order := range h.Orders {
		appGoodIDs = append(appGoodIDs, order.AppGoodID)
	}
	goods, _, err := appgoodmwcli.GetGoods(ctx, &appgoodmwpb.Conds{
		AppID:  &basetypes.StringVal{Op: cruder.EQ, Value: *h.AppID},
		EntIDs: &basetypes.StringSliceVal{Op: cruder.IN, Value: appGoodIDs},
	}, int32(0), int32(len(appGoodIDs)))
	if err != nil {
		return err
	}
	if len(goods) < len(appGoodIDs) {
		return fmt.Errorf("invalid appgoods")
	}
	for _, good := range goods {
		if !good.EnablePurchase {
			return fmt.Errorf("permission denied")
		}
		h.appGoods[good.EntID] = good
	}
	h.parentAppGood = h.appGoods[*h.AppGoodID]
	if h.parentAppGood == nil {
		return fmt.Errorf("invalid parentgood")
	}
	return nil
}

func (h *createsHandler) checkGoods(ctx context.Context) error {
	var goodIDs []string
	for _, appGood := range h.appGoods {
		goodIDs = append(goodIDs, appGood.GoodID)
	}
	goods, _, err := goodmwcli.GetGoods(ctx, &goodmwpb.Conds{
		EntIDs: &basetypes.StringSliceVal{Op: cruder.IN, Value: goodIDs},
	}, int32(0), int32(len(goodIDs)))
	if err != nil {
		return err
	}
	if len(goods) < len(goodIDs) {
		return fmt.Errorf("invalid goods")
	}
	for _, good := range goods {
		h.goods[good.EntID] = good
	}
	h.parentGood = h.goods[h.parentAppGood.GoodID]
	if h.parentGood == nil {
		return fmt.Errorf("invalid parent good")
	}
	return nil
}

func (h *createsHandler) checkAppGoodCoins(ctx context.Context) error {
	coinTypeIDs := []string{}
	for _, good := range h.appGoods {
		coinTypeIDs = append(coinTypeIDs, good.CoinTypeID)
	}
	coins, _, err := appcoinmwcli.GetCoins(ctx, &appcoinmwpb.Conds{
		AppID:  &basetypes.StringVal{Op: cruder.EQ, Value: *h.AppID},
		EntIDs: &basetypes.StringSliceVal{Op: cruder.IN, Value: coinTypeIDs},
	}, 0, int32(len(coinTypeIDs)))
	if err != nil {
		return err
	}
	for _, coin := range coins {
		h.goodCoinEnv = coin.ENV
		if h.paymentCoin.ENV != coin.ENV {
			return fmt.Errorf("mismatch coin environment")
		}
	}
	return nil
}

func (h *createsHandler) getAppGoodPromotions(ctx context.Context) error {
	appGoodIDs := []string{}
	for _, order := range h.Orders {
		appGoodIDs = append(appGoodIDs, order.AppGoodID)
	}

	offset := int32(0)
	limit := constant.DefaultRowLimit

	for {
		goods, _, err := topmostmwcli.GetTopMostGoods(ctx, &topmostmwpb.Conds{
			AppID:      &basetypes.StringVal{Op: cruder.EQ, Value: *h.AppID},
			AppGoodIDs: &basetypes.StringSliceVal{Op: cruder.IN, Value: appGoodIDs},
		}, offset, limit)
		if err != nil {
			return err
		}
		if len(goods) == 0 {
			break
		}
		for _, good := range goods {
			goods, ok := h.topMostGoods[good.AppGoodID]
			if !ok {
				goods = []*topmostmwpb.TopMostGood{}
			}
			goods = append(goods, good)
			h.topMostGoods[good.AppGoodID] = goods
		}
		offset += limit
	}
	return nil
}

func (h *createsHandler) topMostGoodPackagePrice(req *ordermwpb.OrderReq) (decimal.Decimal, error) {
	price := decimal.NewFromInt(0)
	topMosts := h.topMostGoods[*req.AppGoodID]
	for _, topMost := range topMosts {
		packagePrice, err := decimal.NewFromString(topMost.PackagePrice)
		if err != nil {
			return decimal.Decimal{}, err
		}
		if packagePrice.Cmp(decimal.NewFromInt(0)) <= 0 {
			continue
		}
		if packagePrice.Cmp(price) < 0 {
			price = packagePrice
			h.priceTopMostGoods[*req.AppGoodID] = topMost
		}
	}
	return price, nil
}

func (h *createsHandler) topMostGoodUnitPrice(req *ordermwpb.OrderReq) (decimal.Decimal, error) {
	price := decimal.NewFromInt(0)
	topMosts := h.topMostGoods[*req.AppGoodID]
	for _, topMost := range topMosts {
		unitPrice, err := decimal.NewFromString(topMost.UnitPrice)
		if err != nil {
			return decimal.Decimal{}, err
		}
		if unitPrice.Cmp(decimal.NewFromInt(0)) <= 0 {
			continue
		}
		if unitPrice.Cmp(price) < 0 {
			price = unitPrice
			h.priceTopMostGoods[*req.AppGoodID] = topMost
		}
	}
	return price, nil
}

func (h *createsHandler) goodPackagePrice(req *ordermwpb.OrderReq) (decimal.Decimal, error) {
	good := h.appGoods[*req.AppGoodID]
	if good.MinOrderDuration != good.MaxOrderDuration {
		return decimal.Decimal{}, nil
	}

	packagePrice, err := h.topMostGoodPackagePrice(req)
	if err != nil {
		return decimal.Decimal{}, err
	}
	if packagePrice.Cmp(decimal.NewFromInt(0)) > 0 {
		return packagePrice, nil
	}

	packagePrice, err = decimal.NewFromString(good.PackagePrice)
	if err != nil {
		return decimal.Decimal{}, err
	}
	return packagePrice, nil
}

func (h *createsHandler) goodUnitPrice(req *ordermwpb.OrderReq) (decimal.Decimal, error) {
	unitPrice, err := h.topMostGoodUnitPrice(req)
	if err != nil {
		return decimal.Decimal{}, err
	}
	if unitPrice.Cmp(decimal.NewFromInt(0)) > 0 {
		return unitPrice, nil
	}

	good := h.appGoods[*req.AppGoodID]
	unitPrice, err = decimal.NewFromString(good.UnitPrice)
	if err != nil {
		return decimal.Decimal{}, err
	}
	return unitPrice, nil
}

// Here we get price which already calculate duration
//  GoodUnitByDuration: packagePrice or unitPrice * duration
//  GoodUnitByQuantity: packagePrice or unitPrice
//  GoodUnitByDurationAndQuantity: packagePrice or unitPrice * duration
func (h *createsHandler) goodPrice(req *ordermwpb.OrderReq) (decimal.Decimal, error) {
	packagePrice, err := h.goodPackagePrice(req)
	if err != nil {
		return decimal.Decimal{}, err
	}
	if packagePrice.Cmp(decimal.NewFromInt(0)) > 0 {
		return packagePrice, nil
	}

	unitPrice, err := h.goodUnitPrice(req)
	if err != nil {
		return decimal.Decimal{}, err
	}
	if unitPrice.Cmp(decimal.NewFromInt(0)) <= 0 {
		return decimal.Decimal{}, fmt.Errorf("invalid unitprice")
	}

	good := h.appGoods[*req.AppGoodID]
	switch good.UnitType {
	case goodtypes.GoodUnitType_GoodUnitByDurationAndQuantity:
		fallthrough //nolint
	case goodtypes.GoodUnitType_GoodUnitByDuration:
		if req.Duration == nil {
			return decimal.Decimal{}, fmt.Errorf("invalid duration")
		}
		return unitPrice.Mul(decimal.NewFromInt(int64(*req.Duration))), nil
	case goodtypes.GoodUnitType_GoodUnitByQuantity:
		return unitPrice, nil
	default:
		return decimal.Decimal{}, fmt.Errorf("invalid unittype")
	}
}

func (h *createsHandler) goodValue(req *ordermwpb.OrderReq) (decimal.Decimal, error) {
	good := h.appGoods[*req.AppGoodID]
	price, err := decimal.NewFromString(good.PackagePrice)
	if err != nil {
		return decimal.Decimal{}, err
	}
	if price.Cmp(decimal.NewFromInt(0)) <= 0 {
		price, err = decimal.NewFromString(good.UnitPrice)
		switch good.UnitType {
		case goodtypes.GoodUnitType_GoodUnitByDurationAndQuantity:
			fallthrough //nolint
		case goodtypes.GoodUnitType_GoodUnitByDuration:
			if req.Duration == nil {
				return decimal.Decimal{}, fmt.Errorf("invalid duration")
			}
			price = price.Mul(decimal.NewFromInt(int64(*req.Duration)))
		}
	}
	if err != nil {
		return decimal.Decimal{}, err
	}
	if price.Cmp(decimal.NewFromInt(0)) <= 0 {
		return decimal.Decimal{}, fmt.Errorf("invalid price")
	}
	units := decimal.NewFromInt(1)
	if req.Units != nil {
		units, err = decimal.NewFromString(*req.Units)
		if err != nil {
			return decimal.Decimal{}, err
		}
	}
	return price.Mul(units), nil
}

func (h *createsHandler) goodPaymentUSDAmount(req *ordermwpb.OrderReq) (decimal.Decimal, error) {
	price, err := h.goodPrice(req)
	if err != nil {
		return decimal.Decimal{}, err
	}
	units := decimal.NewFromInt(1)
	if req.Units != nil {
		units, err = decimal.NewFromString(*req.Units)
		if err != nil {
			return decimal.Decimal{}, err
		}
	}
	return price.Mul(units), nil
}

func (h *createsHandler) calculateOrderUSDPrice() error {
	parentPackagePrice, err := decimal.NewFromString(h.parentAppGood.PackagePrice)
	if err != nil {
		return err
	}
	packageWithRequireds := h.parentAppGood.PackageWithRequireds &&
		parentPackagePrice.Cmp(decimal.NewFromInt(0)) > 0
	valueZero := decimal.NewFromInt(0).String()

	for _, req := range h.orderReqs {
		if packageWithRequireds && *req.EntID != *h.ParentOrderID {
			req.GoodValueUSD = &valueZero
			req.GoodValue = &valueZero
			continue
		}
		goodValueUSD, err := h.goodValue(req)
		if err != nil {
			return err
		}
		paymentUSDAmount, err := h.goodPaymentUSDAmount(req)
		if err != nil {
			return err
		}
		goodValue := goodValueUSD.String()
		if h.coinCurrencyAmount.Cmp(decimal.NewFromInt(0)) > 0 {
			goodValue = goodValueUSD.Div(h.coinCurrencyAmount).String()
		}
		_goodValueUSD := goodValueUSD.String()
		req.GoodValueUSD = &_goodValueUSD
		req.GoodValue = &goodValue
		h.paymentUSDAmount = h.paymentUSDAmount.Add(paymentUSDAmount)
	}
	return nil
}

func (h *createsHandler) resolveStartMode() {
	for _, req := range h.orderReqs {
		mode := types.OrderStartMode_OrderStartConfirmed
		switch h.parentAppGood.StartMode {
		case goodtypes.GoodStartMode_GoodStartModeTBD:
			mode = types.OrderStartMode_OrderStartTBD
		case goodtypes.GoodStartMode_GoodStartModeConfirmed:
			mode = types.OrderStartMode_OrderStartNextDay
		case goodtypes.GoodStartMode_GoodStartModeInstantly:
			mode = types.OrderStartMode_OrderStartInstantly
		case goodtypes.GoodStartMode_GoodStartModeNextDay:
			mode = types.OrderStartMode_OrderStartNextDay
		case goodtypes.GoodStartMode_GoodStartModePreset:
			mode = types.OrderStartMode_OrderStartPreset
		}
		req.StartMode = &mode
	}
}

//nolint:gocyclo
func (h *createsHandler) resolveStartEnd() error {
	durationUnitSeconds := timedef.SecondsPerHour
	for _, req := range h.orderReqs {
		switch h.parentAppGood.DurationType {
		case goodtypes.GoodDurationType_GoodDurationByHour:
		case goodtypes.GoodDurationType_GoodDurationByDay:
			durationUnitSeconds = timedef.SecondsPerDay
		case goodtypes.GoodDurationType_GoodDurationByMonth:
			durationUnitSeconds = timedef.SecondsPerMonth
		case goodtypes.GoodDurationType_GoodDurationByYear:
			durationUnitSeconds = timedef.SecondsPerYear
		}

		goodStartAt := h.parentAppGood.ServiceStartAt
		switch *req.StartMode {
		case types.OrderStartMode_OrderStartPreset:
		case types.OrderStartMode_OrderStartInstantly:
			fallthrough //nolint
		case types.OrderStartMode_OrderStartNextDay:
			fallthrough //nolint
		case types.OrderStartMode_OrderStartTBD:
			if goodStartAt == 0 {
				goodStartAt = h.parentAppGood.StartAt
			}
		}

		switch h.parentAppGood.UnitType {
		case goodtypes.GoodUnitType_GoodUnitByDuration:
			fallthrough //nolint
		case goodtypes.GoodUnitType_GoodUnitByDurationAndQuantity:
			if req.Duration == nil {
				return fmt.Errorf("invalid duration")
			}
			if h.parentAppGood.MinOrderDuration == h.parentAppGood.MaxOrderDuration {
				*req.Duration = h.parentAppGood.MinOrderDuration
			}
			if *req.Duration < h.parentAppGood.MinOrderDuration ||
				*req.Duration > h.parentAppGood.MaxOrderDuration {
				return fmt.Errorf("invalid duration")
			}
		}

		switch *req.StartMode {
		case types.OrderStartMode_OrderStartTBD:
			fallthrough //nolint
		case types.OrderStartMode_OrderStartPreset:
			req.StartAt = &goodStartAt
		case types.OrderStartMode_OrderStartInstantly:
			now := uint32(time.Now().Unix())
			req.StartAt = &now
		case types.OrderStartMode_OrderStartNextDay:
			startAt := uint32(h.tomorrowStart().Unix())
			req.StartAt = &startAt
		}

		if goodStartAt > *req.StartAt {
			req.StartAt = &goodStartAt
		}

		durationSeconds := uint32(durationUnitSeconds) * *req.Duration
		endAt := *req.StartAt + durationSeconds
		req.EndAt = &endAt
	}
	return nil
}

func (h *createsHandler) withUpdateStock(dispose *dtmcli.SagaDispose) {
	for _, order := range h.Orders {
		if !order.Parent {
			continue
		}
		if order.Units == nil {
			continue
		}
		dispose.Add(
			goodmwsvcname.ServiceDomain,
			"good.middleware.app.good1.stock.v1.Middleware/Lock",
			"good.middleware.app.good1.stock.v1.Middleware/Unlock",
			&appgoodstockmwpb.LockRequest{
				EntID:        h.parentAppGood.AppGoodStockID,
				AppID:        h.parentAppGood.AppID,
				GoodID:       h.parentAppGood.GoodID,
				AppGoodID:    *h.AppGoodID,
				Units:        *order.Units,
				AppSpotUnits: decimal.NewFromInt(0).String(),
				LockID:       h.stockLockID,
				Rollback:     true,
			},
		)
	}
}

func (h *createsHandler) withCreateOrders(dispose *dtmcli.SagaDispose) {
	paymentCoinAmount := h.paymentCoinAmount.String()
	discountCoinAmount := h.reductionCoinAmount.String()
	transferCoinAmount := h.transferCoinAmount.String()
	balanceCoinAmount := h.balanceCoinAmount.String()
	coinUSDCurrency := h.coinCurrencyAmount.String()
	localCoinUSDCurrency := h.localCurrencyAmount.String()
	liveCoinUSDCurrency := h.liveCurrencyAmount.String()

	for _, req := range h.orderReqs {
		req.OrderType = h.OrderType
		req.CoinUSDCurrency = &coinUSDCurrency
		req.LocalCoinUSDCurrency = &localCoinUSDCurrency
		req.LiveCoinUSDCurrency = &liveCoinUSDCurrency
		if *req.EntID == *h.ParentOrderID {
			if topMost, ok := h.priceTopMostGoods[*req.AppGoodID]; ok {
				req.PromotionID = &topMost.TopMostID
			}
			req.CoinTypeID = &h.parentAppGood.CoinTypeID
			req.AppGoodStockLockID = &h.stockLockID
			req.PaymentAmount = &paymentCoinAmount
			req.DiscountAmount = &discountCoinAmount
			req.PaymentType = &h.paymentType
			req.TransferAmount = &transferCoinAmount
			req.BalanceAmount = &balanceCoinAmount
			req.CouponIDs = h.CouponIDs
			req.LedgerLockID = h.balanceLockID
			if h.paymentAccount != nil {
				req.PaymentAccountID = &h.paymentAccount.AccountID
				paymentStartAmount := h.paymentStartAmount.String()
				req.PaymentStartAmount = &paymentStartAmount
			}
		} else {
			req.ParentOrderID = h.ParentOrderID
			childPaymentType := types.PaymentType_PayWithParentOrder
			req.PaymentType = &childPaymentType
			invalidID := uuid.Nil.String()
			req.CoinTypeID = &invalidID
		}
		appGood := h.appGoods[*req.AppGoodID]
		req.GoodID = &appGood.GoodID
		req.AppGoodID = &appGood.EntID
	}

	dispose.Add(
		ordermwsvcname.ServiceDomain,
		"order.middleware.order1.v1.Middleware/CreateOrders",
		"order.middleware.order1.v1.Middleware/DeleteOrders",
		&ordermwpb.CreateOrdersRequest{
			Infos: h.orderReqs,
		},
	)
}

func (h *createsHandler) getRequiredGoods(ctx context.Context) error {
	offset := int32(0)
	limit := constant.DefaultRowLimit

	for {
		goods, _, err := goodrequiredmwcli.GetRequireds(ctx, &goodrequiredpb.Conds{
			MainGoodID: &basetypes.StringVal{Op: cruder.EQ, Value: h.parentAppGood.GoodID},
		}, offset, limit)
		if err != nil {
			return err
		}
		if len(goods) == 0 {
			break
		}
		for _, good := range goods {
			h.requiredGoods[good.RequiredGoodID] = good
		}
		offset += limit
	}
	return nil
}

func (h *createsHandler) validateOrderGoods() error {
	for _, order := range h.Orders {
		if order.Parent {
			continue
		}
		appgood := h.appGoods[order.AppGoodID]
		if _, ok := h.requiredGoods[appgood.GoodID]; !ok {
			return fmt.Errorf("invalid requiredgood")
		}
	}
	return nil
}

func (h *createsHandler) validateRequiredOrders() error {
	orderGoodIDs := map[string]struct{}{}
	for _, order := range h.Orders {
		appgood := h.appGoods[order.AppGoodID]
		orderGoodIDs[appgood.GoodID] = struct{}{}
	}
	for _, good := range h.requiredGoods {
		if !good.Must {
			continue
		}
		if _, ok := orderGoodIDs[good.RequiredGoodID]; !ok {
			return fmt.Errorf("miss requiredgood")
		}
	}
	return nil
}

func (h *createsHandler) constructOrderReqs() error {
	for _, order := range h.Orders {
		id := uuid.NewString()
		h.orderReqs = append(h.orderReqs, &ordermwpb.OrderReq{
			EntID:             &id,
			AppID:             h.AppID,
			UserID:            h.UserID,
			AppGoodID:         &order.AppGoodID,
			Units:             order.Units,
			Duration:          order.Duration,
			OrderType:         h.OrderType,
			InvestmentType:    h.InvestmentType,
			PaymentCoinTypeID: h.PaymentCoinID,
		})
		if order.Parent {
			if h.AppGoodID != nil {
				return fmt.Errorf("invalid parentorder")
			}
			h.AppGoodID = &order.AppGoodID
			h.ParentOrderID = &id
			h.Units = order.Units
			h.Duration = order.Duration
			h.EntID = &id
		}
		h.EntIDs = append(h.EntIDs, id)
	}
	if h.AppGoodID == nil {
		return fmt.Errorf("invalid parentorder")
	}
	for _, req := range h.orderReqs {
		if *req.EntID == *h.EntID { // Parent order
			continue
		}
		good := h.appGoods[*req.AppGoodID]
		if good.QuantityCalculateType == goodtypes.GoodUnitCalculateType_GoodUnitCalculateByParent {
			req.Units = h.Units
		}
		if good.DurationCalculateType == goodtypes.GoodUnitCalculateType_GoodUnitCalculateByParent {
			req.Duration = h.Duration
		}
	}
	return nil
}

//nolint:funlen,gocyclo
func (h *Handler) CreateOrders(ctx context.Context) (infos []*npool.Order, err error) {
	handler := &createsHandler{
		baseCreateHandler: &baseCreateHandler{
			dtmHandler: &dtmHandler{
				Handler: h,
			},
			coupons: map[string]*allocatedmwpb.Coupon{},
		},
		appGoods:          map[string]*appgoodmwpb.Good{},
		goods:             map[string]*goodmwpb.Good{},
		requiredGoods:     map[string]*goodrequiredpb.Required{},
		topMostGoods:      map[string][]*topmostmwpb.TopMostGood{},
		priceTopMostGoods: map[string]*topmostmwpb.TopMostGood{},
	}

	if err := handler.getApp(ctx); err != nil {
		return nil, err
	}
	if err := handler.getUser(ctx); err != nil {
		return nil, err
	}
	if err := handler.checkAppGoods(ctx); err != nil {
		return nil, err
	}
	if err := handler.constructOrderReqs(); err != nil {
		return nil, err
	}
	if err := handler.checkGoods(ctx); err != nil {
		return nil, err
	}
	if err := handler.getCoupons(ctx); err != nil {
		return nil, err
	}
	if err := handler.validateCouponScope(ctx, handler.parentGood.EntID, handler.parentAppGood.EntID); err != nil {
		return nil, err
	}
	if err := handler.validateDiscountCoupon(); err != nil {
		return nil, err
	}
	if err := handler.checkMaxUnpaidOrders(ctx); err != nil {
		return nil, err
	}
	if err := handler.checkAppGoodCoins(ctx); err != nil {
		return nil, err
	}
	if err := handler.getPaymentCoin(ctx); err != nil {
		return nil, err
	}
	if err := handler.checkUnitsLimit(ctx, handler.parentAppGood); err != nil {
		return nil, err
	}
	if err := handler.getRequiredGoods(ctx); err != nil {
		return nil, err
	}
	if err := handler.validateOrderGoods(); err != nil {
		return nil, err
	}
	if err := handler.validateRequiredOrders(); err != nil {
		return nil, err
	}
	if err := handler.getAppGoodPromotions(ctx); err != nil {
		return nil, err
	}
	if err := handler.checkPaymentCoinCurrency(ctx); err != nil {
		return nil, err
	}
	if err := handler.resolveStartEnd(); err != nil {
		return nil, err
	}
	if err := handler.calculateOrderUSDPrice(); err != nil {
		return nil, err
	}
	if err := handler.calculateDiscountCouponReduction(); err != nil {
		return nil, err
	}
	if err := handler.calculateFixAmountCouponReduction(); err != nil {
		return nil, err
	}
	if err := handler.checkPaymentCoinAmount(); err != nil {
		return nil, err
	}
	if err := handler.checkTransferCoinAmount(); err != nil {
		return nil, err
	}
	handler.resolvePaymentType()
	handler.resolveStartMode()

	if err := handler.acquirePaymentAddress(ctx); err != nil {
		return nil, err
	}
	defer handler.releasePaymentAddress()
	if err := handler.getPaymentStartAmount(ctx); err != nil {
		return nil, err
	}

	handler.prepareStockAndLedgerLockIDs()
	key := fmt.Sprintf("%v:%v:%v:%v", basetypes.Prefix_PrefixCreateOrder, *h.AppID, *h.UserID, h.ParentOrderID)
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
	handler.withCreateOrders(sagaDispose)

	if err := handler.dtmDo(ctx, sagaDispose); err != nil {
		return nil, err
	}

	orders, _, err := h.GetOrders(ctx)
	if err != nil {
		return nil, err
	}

	notifyCouponsUsed(handler.coupons, h.ParentOrderID)
	return orders, nil
}
