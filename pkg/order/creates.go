package order

import (
	"context"
	"fmt"

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
		AppID: &basetypes.StringVal{Op: cruder.EQ, Value: *h.AppID},
		IDs:   &basetypes.StringSliceVal{Op: cruder.IN, Value: appGoodIDs},
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
		h.appGoods[good.ID] = good
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
		IDs: &basetypes.StringSliceVal{Op: cruder.IN, Value: goodIDs},
	}, int32(0), int32(len(goodIDs)))
	if err != nil {
		return err
	}
	if len(goods) < len(goodIDs) {
		return fmt.Errorf("invalid goods")
	}
	for _, good := range goods {
		h.goods[good.ID] = good
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
		AppID: &basetypes.StringVal{Op: cruder.EQ, Value: *h.AppID},
		IDs:   &basetypes.StringSliceVal{Op: cruder.IN, Value: coinTypeIDs},
	}, 0, int32(len(coinTypeIDs)))
	if err != nil {
		return err
	}
	for _, coin := range coins {
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

func (h *createsHandler) calculateOrderUSDPrice() error {
	for _, req := range h.orderReqs {
		appGood := h.appGoods[*req.AppGoodID]
		units, err := decimal.NewFromString(*req.Units)
		if err != nil {
			return err
		}
		unitPrice, err := decimal.NewFromString(appGood.Price)
		if err != nil {
			return err
		}
		if unitPrice.Cmp(decimal.NewFromInt(0)) <= 0 {
			return fmt.Errorf("invalid price")
		}

		goodValue := unitPrice.Mul(units).Div(h.coinCurrencyAmount).String()
		goodValueUSD := unitPrice.Mul(units).String()
		req.GoodValueUSD = &goodValueUSD
		req.GoodValue = &goodValue

		topMosts := h.topMostGoods[*req.AppGoodID]
		for _, topMost := range topMosts {
			price, err := decimal.NewFromString(topMost.Price)
			if err != nil {
				return err
			}
			if price.Cmp(decimal.NewFromInt(0)) < 0 {
				return fmt.Errorf("invalid topmostprice")
			}
			if unitPrice.Cmp(price) > 0 {
				unitPrice = price
				h.priceTopMostGoods[*req.AppGoodID] = topMost
			}
		}

		paymentUSDAmount := unitPrice.Mul(units)
		h.paymentUSDAmount = h.paymentUSDAmount.Add(paymentUSDAmount)
	}

	return nil
}

func (h *createsHandler) resolveStartMode() {
	for _, req := range h.orderReqs {
		mode := types.OrderStartMode_OrderStartConfirmed
		switch h.appGoods[*req.AppGoodID].StartMode {
		case goodtypes.GoodStartMode_GoodStartModeTBD:
			mode = types.OrderStartMode_OrderStartTBD
		case goodtypes.GoodStartMode_GoodStartModeConfirmed:
		}
		req.StartMode = &mode
	}
}

func (h *createsHandler) resolveStartEnd() {
	for _, req := range h.orderReqs {
		goodStartAt := h.parentAppGood.ServiceStartAt
		if goodStartAt == 0 {
			goodStartAt = h.parentAppGood.StartAt
		}
		goodDurationDays := uint32(h.parentAppGood.DurationDays)
		orderStartAt := uint32(h.tomorrowStart().Unix())
		if goodStartAt > orderStartAt {
			orderStartAt = goodStartAt
		}
		const secondsPerDay = timedef.SecondsPerDay
		endAt := orderStartAt + goodDurationDays*secondsPerDay
		req.StartAt = &orderStartAt
		if *req.ID == *h.ParentOrderID {
			req.EndAt = &endAt
			req.DurationDays = &goodDurationDays
			continue
		}
		childDurationDays := uint32(decimal.RequireFromString(*req.Units).IntPart())
		req.DurationDays = &childDurationDays
		childEndAt := orderStartAt + childDurationDays*secondsPerDay
		req.EndAt = &childEndAt
	}
}

func (h *createsHandler) withUpdateStock(dispose *dtmcli.SagaDispose) {
	for _, order := range h.Orders {
		if !order.Parent {
			continue
		}
		dispose.Add(
			goodmwsvcname.ServiceDomain,
			"good.middleware.app.good1.stock.v1.Middleware/Lock",
			"good.middleware.app.good1.stock.v1.Middleware/Unlock",
			&appgoodstockmwpb.LockRequest{
				ID:           h.parentAppGood.AppGoodStockID,
				AppID:        h.parentAppGood.AppID,
				GoodID:       h.parentAppGood.GoodID,
				AppGoodID:    *h.AppGoodID,
				Units:        order.Units,
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
		if *req.ID == *h.ParentOrderID {
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
		req.AppGoodID = &appGood.ID
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
			ID:                &id,
			AppID:             h.AppID,
			UserID:            h.UserID,
			AppGoodID:         &order.AppGoodID,
			Units:             &order.Units,
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
			h.ID = &id
		}
		h.IDs = append(h.IDs, id)
	}
	if h.AppGoodID == nil {
		return fmt.Errorf("invalid parentorder")
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
	if err := handler.getPaymentCoin(ctx); err != nil {
		return nil, err
	}
	if err := handler.getCoupons(ctx); err != nil {
		return nil, err
	}
	if err := handler.validateDiscountCoupon(); err != nil {
		return nil, err
	}
	if err := handler.constructOrderReqs(); err != nil {
		return nil, err
	}
	if err := handler.checkAppGoods(ctx); err != nil {
		return nil, err
	}
	if err := handler.checkGoods(ctx); err != nil {
		return nil, err
	}
	if err := handler.checkMaxUnpaidOrders(ctx); err != nil {
		return nil, err
	}
	if err := handler.checkAppGoodCoins(ctx); err != nil {
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
	handler.resolveStartEnd()

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

	return orders, nil
}
