//nolint:dupl
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
	npool "github.com/NpoolPlatform/message/npool/order/gw/v1/order"
	ordermwpb "github.com/NpoolPlatform/message/npool/order/mw/v1/order"
	ordermwcli "github.com/NpoolPlatform/order-middleware/pkg/client/order"
	ordermwsvcname "github.com/NpoolPlatform/order-middleware/pkg/servicename"

	"github.com/dtm-labs/dtm/client/dtmcli/dtmimp"
	"github.com/shopspring/decimal"
)

type createHandler struct {
	*baseCreateHandler
	appGood   *appgoodmwpb.Good
	promotion *topmostmwpb.TopMostGood
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

func (h *createHandler) checkUnitsLimit(ctx context.Context) error {
	if *h.OrderType != types.OrderType_Normal {
		return nil
	}
	units, err := decimal.NewFromString(h.Units)
	if err != nil {
		return err
	}
	if h.appGood.PurchaseLimit > 0 && units.Cmp(decimal.NewFromInt32(h.appGood.PurchaseLimit)) > 0 {
		return fmt.Errorf("too many units")
	}
	if !h.appGood.EnablePurchase {
		return fmt.Errorf("app good is not enabled purchase")
	}
	purchaseCountStr, err := ordermwcli.SumOrderUnits(ctx, &ordermwpb.Conds{
		AppID:      &basetypes.StringVal{Op: cruder.EQ, Value: *h.AppID},
		UserID:     &basetypes.StringVal{Op: cruder.EQ, Value: *h.UserID},
		AppGoodID:  &basetypes.StringVal{Op: cruder.EQ, Value: *h.AppGoodID},
		OrderState: &basetypes.Uint32Val{Op: cruder.NEQ, Value: uint32(types.OrderState_OrderStateCanceled)},
	})
	if err != nil {
		return err
	}
	purchaseCount, err := decimal.NewFromString(purchaseCountStr)
	if err != nil {
		return err
	}

	userPurchaseLimit, err := decimal.NewFromString(h.appGood.UserPurchaseLimit)
	if err != nil {
		return err
	}

	if userPurchaseLimit.Cmp(decimal.NewFromInt(0)) > 0 &&
		purchaseCount.Add(units).Cmp(userPurchaseLimit) > 0 {
		return fmt.Errorf("too many units")
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
	promotion, err := topmostmwcli.GetTopMostGoodOnly(ctx, &topmostmwpb.Conds{
		AppID:       &basetypes.StringVal{Op: cruder.EQ, Value: *h.AppID},
		AppGoodID:   &basetypes.StringVal{Op: cruder.EQ, Value: *h.AppGoodID},
		TopMostType: &basetypes.Uint32Val{Op: cruder.EQ, Value: uint32(goodtypes.GoodTopMostType_TopMostPromotion)},
		// TODO: Promotion time for now
		// TODO: Other topmost check
	})
	if err != nil {
		return err
	}
	h.promotion = promotion
	return nil
}

func (h *createHandler) calculateOrderUSDTPrice() error {
	units, err := decimal.NewFromString(h.Units)
	if err != nil {
		return err
	}
	amount, err := decimal.NewFromString(h.appGood.Price)
	if err != nil {
		return err
	}
	if amount.Cmp(decimal.NewFromInt(0)) <= 0 {
		return fmt.Errorf("invalid price")
	}
	if h.promotion == nil {
		h.goodValueUSDTAmount = amount.Mul(units)
		return nil
	}
	amount, err = decimal.NewFromString(h.promotion.Price)
	if err != nil {
		return err
	}
	if amount.Cmp(decimal.NewFromInt(0)) <= 0 {
		return fmt.Errorf("invalid price")
	}
	h.goodValueUSDTAmount = amount.Mul(units)
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
	goodValueUSDTAmount := h.goodValueUSDTAmount.String()
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
		GoodValueUSD:         &goodValueUSDTAmount,
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
	if h.promotion != nil {
		req.PromotionID = &h.promotion.ID
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
	if err := handler.checkUnitsLimit(ctx); err != nil {
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
	if err := handler.calculateOrderUSDTPrice(); err != nil {
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
	handler.prepareOrderAndLockIDs()

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
