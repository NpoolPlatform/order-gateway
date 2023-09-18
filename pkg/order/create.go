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
	appGood          *appgoodmwpb.Good
	topMostGoods     []*topmostmwpb.TopMostGood
	priceTopMostGood *topmostmwpb.TopMostGood
	orderStartMode   types.OrderStartMode
	orderStartAt     uint32
	orderEndAt       uint32
}

func (h *createHandler) checkGood(ctx context.Context) error {
	good, err := goodmwcli.GetGood(ctx, h.appGood.GoodID)
	if err != nil {
		return err
	}
	if good == nil {
		return fmt.Errorf("invalid good")
	}
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

func (h *createHandler) calculateOrderUSDPrice() error {
	units, err := decimal.NewFromString(h.Units)
	if err != nil {
		return err
	}
	unitPrice, err := decimal.NewFromString(h.appGood.Price)
	if err != nil {
		return err
	}
	if unitPrice.Cmp(decimal.NewFromInt(0)) <= 0 {
		return fmt.Errorf("invalid price")
	}
	h.goodValueUSDAmount = unitPrice.Mul(units)
	for _, topMost := range h.topMostGoods {
		price, err := decimal.NewFromString(topMost.Price)
		if err != nil {
			return err
		}
		if price.Cmp(decimal.NewFromInt(0)) <= 0 {
			return fmt.Errorf("invalid price")
		}
		if unitPrice.Cmp(price) > 0 {
			unitPrice = price
			h.priceTopMostGood = topMost
		}
	}
	h.paymentUSDAmount = unitPrice.Mul(units)
	return nil
}

func (h *createHandler) resolveStartMode() {
	if h.appGood.StartMode == goodtypes.GoodStartMode_GoodStartModeTBD {
		h.orderStartMode = types.OrderStartMode_OrderStartTBD
		return
	}
	h.orderStartMode = types.OrderStartMode_OrderStartConfirmed
}

func (h *createHandler) resolveStartEnd() {
	goodStartAt := h.appGood.ServiceStartAt
	if h.appGood.ServiceStartAt == 0 {
		goodStartAt = h.appGood.StartAt
	}
	goodDurationDays := uint32(h.appGood.DurationDays)
	h.orderStartAt = uint32(h.tomorrowStart().Unix())
	if goodStartAt > h.orderStartAt {
		h.orderStartAt = goodStartAt
	}
	const secondsPerDay = timedef.SecondsPerDay
	h.orderEndAt = h.orderStartAt + goodDurationDays*secondsPerDay
}

func (h *createHandler) withUpdateStock(dispose *dtmcli.SagaDispose) {
	dispose.Add(
		goodmwsvcname.ServiceDomain,
		"good.middleware.app.good1.stock.v1.Middleware/Lock",
		"good.middleware.app.good1.stock.v1.Middleware/Unlock",
		&appgoodstockmwpb.LockRequest{
			ID:           h.appGood.AppGoodStockID,
			AppID:        h.appGood.AppID,
			GoodID:       h.appGood.GoodID,
			AppGoodID:    *h.AppGoodID,
			Units:        h.Units,
			AppSpotUnits: decimal.NewFromInt(0).String(),
			LockID:       h.stockLockID,
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
	goodDurationDays := uint32(h.appGood.DurationDays)

	req := &ordermwpb.OrderReq{
		ID:                   h.ID,
		AppID:                h.AppID,
		UserID:               h.UserID,
		GoodID:               &h.appGood.GoodID,
		AppGoodID:            h.AppGoodID,
		ParentOrderID:        h.ParentOrderID,
		Units:                &h.Units,
		GoodValue:            &goodValueCoinAmount,
		GoodValueUSD:         &goodValueUSDAmount,
		PaymentAmount:        &paymentCoinAmount,
		DiscountAmount:       &discountCoinAmount,
		DurationDays:         &goodDurationDays,
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
		AppGoodStockLockID:   &h.stockLockID,
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
			Handler: h,
			coupons: map[string]*allocatedmwpb.Coupon{},
		},
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
	if err := handler.checkMaxUnpaidOrders(ctx); err != nil {
		return nil, err
	}
	if err := handler.getAppGood(ctx); err != nil {
		return nil, err
	}
	if err := handler.checkGood(ctx); err != nil {
		return nil, err
	}
	if err := handler.checkAppGoodCoin(ctx); err != nil {
		return nil, err
	}
	if err := handler.checkUnitsLimit(ctx, handler.appGood); err != nil {
		return nil, err
	}
	if err := handler.checkParentOrder(ctx); err != nil {
		return nil, err
	}
	if err := handler.checkParentOrderGoodRequired(ctx); err != nil {
		return nil, err
	}
	if err := handler.getAppGoodPromotion(ctx); err != nil {
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
	if err := handler.checkPaymentCoinCurrency(ctx); err != nil {
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
	handler.prepareStockAndLedgerLockIDs()

	id1 := uuid.NewString()
	if h.ID == nil {
		h.ID = &id1
	}

	if err := handler.acquirePaymentAddress(ctx); err != nil {
		return nil, err
	}
	defer handler.releasePaymentAddress()
	if err := handler.getPaymentStartAmount(ctx); err != nil {
		return nil, err
	}

	key := fmt.Sprintf("%v:%v:%v:%v", basetypes.Prefix_PrefixCreateOrder, *h.AppID, *h.UserID, *handler.ID)
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
	})

	handler.withUpdateStock(sagaDispose)
	handler.withUpdateBalance(sagaDispose)
	handler.withCreateOrder(sagaDispose)
	handler.withLockPaymentAccount(sagaDispose)

	if err := dtmcli.WithSaga(ctx, sagaDispose); err != nil {
		return nil, err
	}

	return h.GetOrder(ctx)
}
