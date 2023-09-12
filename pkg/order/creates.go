//nolint:dupl
package order

import (
	"context"
	"fmt"
	"time"

	payaccmwcli "github.com/NpoolPlatform/account-middleware/pkg/client/payment"
	accountlock "github.com/NpoolPlatform/account-middleware/pkg/lock"
	accountmwsvcname "github.com/NpoolPlatform/account-middleware/pkg/servicename"
	appmwcli "github.com/NpoolPlatform/appuser-middleware/pkg/client/app"
	usermwcli "github.com/NpoolPlatform/appuser-middleware/pkg/client/user"
	appcoinmwcli "github.com/NpoolPlatform/chain-middleware/pkg/client/app/coin"
	currencymwcli "github.com/NpoolPlatform/chain-middleware/pkg/client/coin/currency"
	dtmcli "github.com/NpoolPlatform/dtm-cluster/pkg/dtm"
	redis2 "github.com/NpoolPlatform/go-service-framework/pkg/redis"
	appgoodmwcli "github.com/NpoolPlatform/good-middleware/pkg/client/app/good"
	topmostmwcli "github.com/NpoolPlatform/good-middleware/pkg/client/app/good/topmost/good"
	goodmwcli "github.com/NpoolPlatform/good-middleware/pkg/client/good"
	goodrequiredmwcli "github.com/NpoolPlatform/good-middleware/pkg/client/good/required"
	goodmwsvcname "github.com/NpoolPlatform/good-middleware/pkg/servicename"
	allocatedmwcli "github.com/NpoolPlatform/inspire-middleware/pkg/client/coupon/allocated"
	ledgermwsvcname "github.com/NpoolPlatform/ledger-middleware/pkg/servicename"
	cruder "github.com/NpoolPlatform/libent-cruder/pkg/cruder"
	payaccmwpb "github.com/NpoolPlatform/message/npool/account/mw/v1/payment"
	appmwpb "github.com/NpoolPlatform/message/npool/appuser/mw/v1/app"
	usermwpb "github.com/NpoolPlatform/message/npool/appuser/mw/v1/user"
	goodtypes "github.com/NpoolPlatform/message/npool/basetypes/good/v1"
	inspiretypes "github.com/NpoolPlatform/message/npool/basetypes/inspire/v1"
	types "github.com/NpoolPlatform/message/npool/basetypes/order/v1"
	basetypes "github.com/NpoolPlatform/message/npool/basetypes/v1"
	appcoinmwpb "github.com/NpoolPlatform/message/npool/chain/mw/v1/app/coin"
	currencymwpb "github.com/NpoolPlatform/message/npool/chain/mw/v1/coin/currency"
	appgoodmwpb "github.com/NpoolPlatform/message/npool/good/mw/v1/app/good"
	appgoodstockmwpb "github.com/NpoolPlatform/message/npool/good/mw/v1/app/good/stock"
	topmostmwpb "github.com/NpoolPlatform/message/npool/good/mw/v1/app/good/topmost/good"
	goodmwpb "github.com/NpoolPlatform/message/npool/good/mw/v1/good"
	goodrequiredpb "github.com/NpoolPlatform/message/npool/good/mw/v1/good/required"
	allocatedmwpb "github.com/NpoolPlatform/message/npool/inspire/mw/v1/coupon/allocated"
	ledgermwpb "github.com/NpoolPlatform/message/npool/ledger/mw/v2/ledger"
	npool "github.com/NpoolPlatform/message/npool/order/gw/v1/order"
	ordermwpb "github.com/NpoolPlatform/message/npool/order/mw/v1/order"
	sphinxproxypb "github.com/NpoolPlatform/message/npool/sphinxproxy"
	constant "github.com/NpoolPlatform/order-gateway/pkg/const"
	ordermwcli "github.com/NpoolPlatform/order-middleware/pkg/client/order"
	ordermwsvcname "github.com/NpoolPlatform/order-middleware/pkg/servicename"
	sphinxproxycli "github.com/NpoolPlatform/sphinx-proxy/pkg/client"
	"github.com/dtm-labs/dtm/client/dtmcli/dtmimp"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

type createsHandler struct {
	*Handler
	ids                 map[string]*string
	app                 *appmwpb.App
	user                *usermwpb.User
	appGoods            map[string]*appgoodmwpb.Good
	goods               map[string]*goodmwpb.Good
	parentAppGood       *appgoodmwpb.Good
	parentGood          *goodmwpb.Good
	requiredGoods       map[string]*goodrequiredpb.Required
	paymentCoin         *appcoinmwpb.Coin
	paymentAccount      *payaccmwpb.Account
	paymentStartAmount  decimal.Decimal
	coupons             map[string]*allocatedmwpb.Coupon
	promotions          map[string]*topmostmwpb.TopMostGood
	paymentUSDTAmount   decimal.Decimal
	goodValueUSDTAmount map[string]decimal.Decimal
	goodValueCoinAmount map[string]decimal.Decimal
	paymentCoinAmount   decimal.Decimal
	reductionUSDTAmount decimal.Decimal
	reductionCoinAmount decimal.Decimal
	liveCurrencyAmount  decimal.Decimal
	coinCurrencyAmount  decimal.Decimal
	localCurrencyAmount decimal.Decimal
	balanceCoinAmount   decimal.Decimal
	transferCoinAmount  decimal.Decimal
	paymentType         types.PaymentType
	orderStartMode      map[string]types.OrderStartMode
	orderStartAt        map[string]uint32
	orderEndAt          map[string]uint32
	stockLockIDs        map[string]*string
	balanceLockID       *string
}

func (h *createsHandler) tomorrowStart() time.Time {
	now := time.Now()
	y, m, d := now.Date()
	return time.Date(y, m, d+1, 0, 0, 0, 0, now.Location())
}

func (h *createsHandler) getApp(ctx context.Context) error {
	app, err := appmwcli.GetApp(ctx, *h.AppID)
	if err != nil {
		return err
	}
	if app == nil {
		return fmt.Errorf("invalid app")
	}
	h.app = app
	return nil
}

func (h *createsHandler) getUser(ctx context.Context) error {
	user, err := usermwcli.GetUser(ctx, *h.AppID, *h.UserID)
	if err != nil {
		return err
	}
	if user == nil {
		return fmt.Errorf("invalid user")
	}
	h.user = user
	return nil
}

func (h *createsHandler) getPaymentCoin(ctx context.Context) error {
	coin, err := appcoinmwcli.GetCoinOnly(ctx, &appcoinmwpb.Conds{
		AppID:      &basetypes.StringVal{Op: cruder.EQ, Value: *h.AppID},
		CoinTypeID: &basetypes.StringVal{Op: cruder.EQ, Value: *h.PaymentCoinID},
	})
	if err != nil {
		return err
	}
	if coin == nil {
		return fmt.Errorf("invalid paymentcoin")
	}
	if coin.Presale {
		return fmt.Errorf("invalid paymentcoin")
	}
	if !coin.ForPay {
		return fmt.Errorf("invalid paymentcoin")
	}
	h.paymentCoin = coin
	return nil
}

func (h *createsHandler) getCoupons(ctx context.Context) error {
	coupons, _, err := allocatedmwcli.GetCoupons(ctx, &allocatedmwpb.Conds{
		AppID: &basetypes.StringVal{Op: cruder.EQ, Value: *h.AppID},
		IDs:   &basetypes.StringSliceVal{Op: cruder.IN, Value: h.CouponIDs},
	}, int32(0), int32(len(h.CouponIDs)))
	if err != nil {
		return err
	}
	if len(coupons) < len(h.CouponIDs) {
		return fmt.Errorf("invalid coupon")
	}
	for _, coupon := range coupons {
		if !coupon.Valid || coupon.Expired {
			return fmt.Errorf("invalid coupon")
		}
		h.coupons[coupon.ID] = coupon
	}
	return nil
}

func (h *createsHandler) validateDiscountCoupon() error {
	discountCoupons := 0
	fixAmountCoupons := uint32(0)
	specialOfferCoupons := uint32(0)
	for _, coupon := range h.coupons {
		switch coupon.CouponType {
		case inspiretypes.CouponType_Discount:
			discountCoupons++
		case inspiretypes.CouponType_FixAmount:
			fixAmountCoupons++
		case inspiretypes.CouponType_SpecialOffer:
			specialOfferCoupons++
		}
	}
	if discountCoupons > 1 {
		return fmt.Errorf("invalid discountcoupon")
	}
	if fixAmountCoupons > h.app.MaxTypedCouponsPerOrder || specialOfferCoupons > h.app.MaxTypedCouponsPerOrder {
		return fmt.Errorf("invalid fixamountcoupon")
	}
	return nil
}

func (h *createsHandler) checkMaxUnpaidOrders(ctx context.Context) error {
	const maxUnpaidOrders = uint32(5)
	orderCount, err := ordermwcli.CountOrders(ctx, &ordermwpb.Conds{
		AppID:        &basetypes.StringVal{Op: cruder.EQ, Value: *h.AppID},
		UserID:       &basetypes.StringVal{Op: cruder.EQ, Value: *h.UserID},
		AppGoodID:    &basetypes.StringVal{Op: cruder.EQ, Value: *h.AppGoodID},
		OrderType:    &basetypes.Uint32Val{Op: cruder.EQ, Value: uint32(types.OrderType_Normal)},
		PaymentState: &basetypes.Uint32Val{Op: cruder.EQ, Value: uint32(types.PaymentState_PaymentStateWait)},
	})
	if err != nil {
		return err
	}
	if orderCount >= maxUnpaidOrders && *h.OrderType == types.OrderType_Normal {
		return fmt.Errorf("too many unpaid orders")
	}
	return nil
}

func (h *createsHandler) getAppGoods(ctx context.Context) error {
	var appGoodIDs []string
	parentAppGoodID := uuid.Nil.String()
	for _, order := range h.Orders {
		appGoodIDs = append(appGoodIDs, order.AppGoodID)
		if order.Parent {
			if parentAppGoodID != uuid.Nil.String() {
				return fmt.Errorf("too many parents")
			}
			parentAppGoodID = order.AppGoodID
		}
	}
	goods, _, err := appgoodmwcli.GetGoods(ctx, &appgoodmwpb.Conds{
		IDs: &basetypes.StringSliceVal{Op: cruder.IN, Value: appGoodIDs},
	}, int32(0), int32(len(appGoodIDs)))
	if err != nil {
		return err
	}
	if len(goods) < len(appGoodIDs) {
		return fmt.Errorf("invalid appgoods")
	}
	for _, good := range goods {
		h.appGoods[good.ID] = good
		if good.ID == parentAppGoodID {
			h.parentAppGood = good
		}
	}
	if h.parentAppGood == nil {
		return fmt.Errorf("invalid parent appgood")
	}
	return nil
}

func (h *createsHandler) getGoods(ctx context.Context) error {
	var goodIDs []string
	var parentGoodID string
	for _, appGood := range h.appGoods {
		goodIDs = append(goodIDs, appGood.GoodID)
		if appGood.ID == h.parentAppGood.ID {
			parentGoodID = appGood.GoodID
		}
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
		if good.ID == parentGoodID {
			h.parentGood = good
		}
	}
	if h.parentGood == nil {
		return fmt.Errorf("invalid parent good")
	}
	return nil
}

func (h *createsHandler) checkAppGoodCoin(ctx context.Context) error {
	for _, good := range h.appGoods {
		goodCoin, err := appcoinmwcli.GetCoinOnly(ctx, &appcoinmwpb.Conds{
			AppID:      &basetypes.StringVal{Op: cruder.EQ, Value: *h.AppID},
			CoinTypeID: &basetypes.StringVal{Op: cruder.EQ, Value: good.CoinTypeID},
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
	}

	return nil
}

func (h *createsHandler) checkUnitsLimit(ctx context.Context) error {
	if *h.OrderType != types.OrderType_Normal {
		return nil
	}
	for _, order := range h.Orders {
		appGood := h.appGoods[order.AppGoodID]
		units, err := decimal.NewFromString(order.Units)
		if err != nil {
			return err
		}
		if appGood.PurchaseLimit > 0 && units.Cmp(decimal.NewFromInt32(appGood.PurchaseLimit)) > 0 {
			return fmt.Errorf("too many units")
		}
		if !appGood.EnablePurchase {
			return fmt.Errorf("app good is not enabled purchase")
		}
		purchaseCountStr, err := ordermwcli.SumOrderUnits(
			ctx,
			&ordermwpb.Conds{
				AppID:      &basetypes.StringVal{Op: cruder.EQ, Value: *h.AppID},
				UserID:     &basetypes.StringVal{Op: cruder.EQ, Value: *h.UserID},
				AppGoodID:  &basetypes.StringVal{Op: cruder.EQ, Value: *h.AppGoodID},
				OrderState: &basetypes.Uint32Val{Op: cruder.NEQ, Value: uint32(types.OrderState_OrderStateCanceled)},
			},
		)
		if err != nil {
			return err
		}
		purchaseCount, err := decimal.NewFromString(purchaseCountStr)
		if err != nil {
			return err
		}

		userPurchaseLimit, err := decimal.NewFromString(appGood.UserPurchaseLimit)
		if err != nil {
			return err
		}

		if userPurchaseLimit.Cmp(decimal.NewFromInt(0)) > 0 && purchaseCount.Add(units).Cmp(userPurchaseLimit) > 0 {
			return fmt.Errorf("too many units")
		}
	}
	return nil
}

func (h *createsHandler) getAppGoodPromotions(ctx context.Context) error {
	for _, order := range h.Orders {
		promotion, err := topmostmwcli.GetTopMostGoodOnly(ctx, &topmostmwpb.Conds{
			AppID:       &basetypes.StringVal{Op: cruder.EQ, Value: *h.AppID},
			AppGoodID:   &basetypes.StringVal{Op: cruder.EQ, Value: order.AppGoodID},
			TopMostType: &basetypes.Uint32Val{Op: cruder.EQ, Value: uint32(goodtypes.GoodTopMostType_TopMostPromotion)},
		})
		if err != nil {
			return err
		}
		h.promotions[order.AppGoodID] = promotion
	}
	return nil
}

func (h *createsHandler) getAccuracy(amount decimal.Decimal) decimal.Decimal {
	const accuracy = 1000000
	amount = amount.Mul(decimal.NewFromInt(accuracy))
	amount = amount.Ceil()
	amount = amount.Div(decimal.NewFromInt(accuracy))
	return amount
}

func (h *createsHandler) calculateOrderUSDTPrice() error {
	for _, order := range h.Orders {
		appGood := h.appGoods[order.AppGoodID]
		units, err := decimal.NewFromString(order.Units)
		if err != nil {
			return err
		}
		amount, err := decimal.NewFromString(appGood.Price)
		if err != nil {
			return err
		}
		if amount.Cmp(decimal.NewFromInt(0)) <= 0 {
			return fmt.Errorf("invalid price")
		}
		promotion := h.promotions[order.AppGoodID]
		if promotion == nil {
			h.goodValueUSDTAmount[order.AppGoodID] = amount.Mul(units)
			h.goodValueCoinAmount[order.AppGoodID] = h.goodValueUSDTAmount[order.AppGoodID].Div(h.coinCurrencyAmount)
			h.goodValueCoinAmount[order.AppGoodID] = h.getAccuracy(h.goodValueCoinAmount[order.AppGoodID])
			h.paymentUSDTAmount = h.paymentUSDTAmount.Add(h.goodValueUSDTAmount[order.AppGoodID])
			continue
		}
		amount, err = decimal.NewFromString(promotion.Price)
		if err != nil {
			return err
		}
		if amount.Cmp(decimal.NewFromInt(0)) <= 0 {
			return fmt.Errorf("invalid price")
		}
		h.goodValueUSDTAmount[order.AppGoodID] = amount.Mul(units)
		h.goodValueCoinAmount[order.AppGoodID] = h.goodValueUSDTAmount[order.AppGoodID].Div(h.coinCurrencyAmount)
		h.goodValueCoinAmount[order.AppGoodID] = h.getAccuracy(h.goodValueCoinAmount[order.AppGoodID])
		h.paymentUSDTAmount = h.paymentUSDTAmount.Add(h.goodValueUSDTAmount[order.AppGoodID])
	}

	return nil
}

func (h *createsHandler) calculateDiscountCouponReduction() error {
	for _, coupon := range h.coupons {
		if coupon.CouponType == inspiretypes.CouponType_Discount {
			discount, err := decimal.NewFromString(coupon.Denomination)
			if err != nil {
				return err
			}
			h.reductionUSDTAmount = h.reductionUSDTAmount.
				Add(h.paymentUSDTAmount.Mul(discount).Div(decimal.NewFromInt(100))) //nolint
		}
	}
	return nil
}

func (h *createsHandler) calculateFixAmountCouponReduction() error {
	for _, coupon := range h.coupons {
		switch coupon.CouponType {
		case inspiretypes.CouponType_FixAmount:
			fallthrough //nolint
		case inspiretypes.CouponType_SpecialOffer:
			amount, err := decimal.NewFromString(coupon.Denomination)
			if err != nil {
				return err
			}
			h.reductionUSDTAmount = h.reductionUSDTAmount.Add(amount)
		}
	}
	return nil
}

//nolint:dupl
func (h *createsHandler) checkPaymentCoinCurrency(ctx context.Context) error {
	currency, err := currencymwcli.GetCurrencyOnly(ctx, &currencymwpb.Conds{
		CoinTypeID: &basetypes.StringVal{Op: cruder.EQ, Value: h.paymentCoin.CoinTypeID},
	})
	if err != nil {
		return err
	}
	if currency == nil {
		return fmt.Errorf("invalid currency")
	}
	const maxElapsed = uint32(10 * 60)
	if currency.UpdatedAt+maxElapsed < uint32(time.Now().Unix()) {
		return fmt.Errorf("stale coin currency")
	}
	amount, err := decimal.NewFromString(currency.MarketValueLow)
	if err != nil {
		return err
	}
	if amount.Cmp(decimal.NewFromInt(0)) <= 0 {
		return fmt.Errorf("invalid market value")
	}

	h.liveCurrencyAmount = amount
	h.coinCurrencyAmount = amount

	amount, err = decimal.NewFromString(h.paymentCoin.SettleValue)
	if err != nil {
		return err
	}
	if amount.Cmp(decimal.NewFromInt(0)) > 0 {
		h.coinCurrencyAmount = amount
	}

	amount, err = decimal.NewFromString(h.paymentCoin.MarketValue)
	if err != nil {
		return err
	}
	if amount.Cmp(decimal.NewFromInt(0)) > 0 {
		h.localCurrencyAmount = amount
	}
	return nil
}

func (h *createsHandler) checkPaymentCoinAmount() error {
	amount := h.paymentUSDTAmount.
		Sub(h.reductionUSDTAmount).
		Div(h.coinCurrencyAmount)
	if amount.Cmp(decimal.NewFromInt(0)) <= 0 {
		return fmt.Errorf("invalid price")
	}
	h.paymentCoinAmount = h.getAccuracy(amount)
	h.reductionCoinAmount = h.reductionUSDTAmount.Div(h.coinCurrencyAmount)
	h.reductionCoinAmount = h.getAccuracy(h.reductionCoinAmount)
	return nil
}

//nolint:dupl
func (h *createsHandler) checkTransferCoinAmount() error {
	if h.BalanceAmount == nil {
		h.transferCoinAmount = h.paymentCoinAmount
		return nil
	}

	balanceCoinAmount, err := decimal.NewFromString(*h.BalanceAmount)
	if err != nil {
		return err
	}
	if balanceCoinAmount.Cmp(decimal.NewFromInt(0)) <= 0 {
		return fmt.Errorf("invalid balanceamount")
	}
	h.balanceCoinAmount = balanceCoinAmount
	h.transferCoinAmount = h.paymentCoinAmount.Sub(balanceCoinAmount)
	if h.transferCoinAmount.Cmp(decimal.NewFromInt(0)) < 0 {
		h.balanceCoinAmount = h.paymentCoinAmount
		h.transferCoinAmount = decimal.NewFromInt(0)
	}
	return nil
}

func (h *createsHandler) resolvePaymentType() {
	switch *h.OrderType {
	case types.OrderType_Offline:
		h.paymentType = types.PaymentType_PayWithOffline
		return
	case types.OrderType_Airdrop:
		h.paymentType = types.PaymentType_PayWithNoPayment
		return
	}
	if h.transferCoinAmount.Cmp(decimal.NewFromInt(0)) == 0 &&
		h.balanceCoinAmount.Cmp(decimal.NewFromInt(0)) == 0 {
		h.paymentType = types.PaymentType_PayWithNoPayment
		return
	}
	if h.transferCoinAmount.Cmp(h.paymentCoinAmount) == 0 {
		h.paymentType = types.PaymentType_PayWithTransferOnly
		return
	}
	if h.balanceCoinAmount.Cmp(h.paymentCoinAmount) == 0 {
		h.paymentType = types.PaymentType_PayWithBalanceOnly
		return
	}
	h.paymentType = types.PaymentType_PayWithTransferAndBalance
}

func (h *createsHandler) resolveStartMode() {
	for _, order := range h.Orders {
		appGood := h.appGoods[order.AppGoodID]
		if appGood.StartMode == goodtypes.GoodStartMode_GoodStartModeTBD {
			h.orderStartMode[order.AppGoodID] = types.OrderStartMode_OrderStartTBD
			return
		}
		h.orderStartMode[order.AppGoodID] = types.OrderStartMode_OrderStartConfirmed
	}
}

//nolint:dupl
func (h *createsHandler) peekExistAddress(ctx context.Context) (*payaccmwpb.Account, error) {
	const batchAccounts = int32(5)
	accounts, _, err := payaccmwcli.GetAccounts(ctx, &payaccmwpb.Conds{
		CoinTypeID:  &basetypes.StringVal{Op: cruder.EQ, Value: h.paymentCoin.CoinTypeID},
		Active:      &basetypes.BoolVal{Op: cruder.EQ, Value: true},
		Locked:      &basetypes.BoolVal{Op: cruder.EQ, Value: false},
		Blocked:     &basetypes.BoolVal{Op: cruder.EQ, Value: false},
		AvailableAt: &basetypes.Uint32Val{Op: cruder.LTE, Value: uint32(time.Now().Unix())},
	}, int32(0), batchAccounts)
	if err != nil {
		return nil, err
	}
	for _, account := range accounts {
		if account.Locked || !account.Active || account.Blocked {
			continue
		}
		if account.AvailableAt > uint32(time.Now().Unix()) {
			continue
		}
		return account, nil
	}
	return nil, fmt.Errorf("invalid address")
}

func (h *createsHandler) peekNewAddress(ctx context.Context) (*payaccmwpb.Account, error) {
	const createCount = 5
	successCreated := 0

	for i := 0; i < createCount; i++ {
		address, err := sphinxproxycli.CreateAddress(ctx, h.paymentCoin.CoinName)
		if err != nil {
			return nil, err
		}
		if address == nil || address.Address == "" {
			return nil, fmt.Errorf("invalid address")
		}
		_, err = payaccmwcli.CreateAccount(ctx, &payaccmwpb.AccountReq{
			CoinTypeID: &h.paymentCoin.CoinTypeID,
			Address:    &address.Address,
		})
		if err != nil {
			return nil, err
		}
		successCreated++
	}
	if successCreated == 0 {
		return nil, fmt.Errorf("fail create addresses")
	}

	return h.peekExistAddress(ctx)
}

func (h *createsHandler) peekPaymentAddress(ctx context.Context) error {
	switch h.paymentType {
	case types.PaymentType_PayWithBalanceOnly:
		fallthrough //nolint
	case types.PaymentType_PayWithNoPayment:
		return nil
	}

	account, err := h.peekExistAddress(ctx)
	if err != nil {
		account, err = h.peekNewAddress(ctx)
		if err != nil {
			return err
		}
	}
	h.paymentAccount = account
	return nil
}

func (h *createsHandler) recheckPaymentAccount(ctx context.Context) error {
	account, err := payaccmwcli.GetAccount(ctx, h.paymentAccount.ID)
	if err != nil {
		return err
	}
	if account == nil {
		return fmt.Errorf("invalid account")
	}
	if account.Locked || !account.Active || account.Blocked {
		return fmt.Errorf("invalid account")
	}
	if account.AvailableAt > uint32(time.Now().Unix()) {
		return fmt.Errorf("invalid account")
	}
	return nil
}

func (h *createsHandler) getPaymentStartAmount(ctx context.Context) error {
	balance, err := sphinxproxycli.GetBalance(ctx, &sphinxproxypb.GetBalanceRequest{
		Name:    h.paymentCoin.CoinName,
		Address: h.paymentAccount.Address,
	})
	if err != nil {
		return err
	}
	if balance == nil {
		return fmt.Errorf("invalid balance")
	}

	h.paymentStartAmount, err = decimal.NewFromString(balance.BalanceStr)
	return err
}

func (h *createsHandler) resolveStartEnd() {
	for _, order := range h.Orders {
		appGood := h.appGoods[order.AppGoodID]
		goodStartAt := appGood.ServiceStartAt
		if appGood.ServiceStartAt == 0 {
			goodStartAt = appGood.StartAt
		}
		goodDurationDays := uint32(appGood.DurationDays)
		orderStartAt := uint32(h.tomorrowStart().Unix())
		if goodStartAt > orderStartAt {
			orderStartAt = goodStartAt
		}
		const secondsPerDay = 24 * 60 * 60
		h.orderEndAt[order.AppGoodID] = orderStartAt + goodDurationDays*secondsPerDay
		h.orderStartAt[order.AppGoodID] = orderStartAt
	}
}

func (h *createsHandler) withUpdateStock(dispose *dtmcli.SagaDispose) {
	for _, order := range h.Orders {
		appGood := h.appGoods[order.AppGoodID]
		req := &appgoodstockmwpb.StockReq{
			ID:        &appGood.AppGoodStockID,
			GoodID:    &appGood.GoodID,
			AppGoodID: &order.AppGoodID,
			Locked:    &order.Units,
			LockID:    h.stockLockIDs[order.AppGoodID],
		}
		dispose.Add(
			goodmwsvcname.ServiceDomain,
			"good.middleware.app.good1.stock.v1.Middleware/AddStock",
			"good.middleware.app.good1.stock.v1.Middleware/SubStock",
			&appgoodstockmwpb.AddStockRequest{
				Info: req,
			},
		)
	}
}

func (h *createsHandler) withUpdateBalance(dispose *dtmcli.SagaDispose) {
	if h.balanceCoinAmount.Cmp(decimal.NewFromInt(0)) <= 0 {
		return
	}

	amount := h.balanceCoinAmount.String()
	req := &ledgermwpb.LedgerReq{
		AppID:      h.AppID,
		UserID:     h.UserID,
		CoinTypeID: h.PaymentCoinID,
		Spendable:  &amount,
		LockID:     h.balanceLockID,
	}
	dispose.Add(
		ledgermwsvcname.ServiceDomain,
		"ledger.middleware.ledger.v2.Middleware/SubBalance",
		"ledger.middleware.ledger.v2.Middleware/AddBalance",
		&ledgermwpb.AddBalanceRequest{
			Info: req,
		},
	)
}

func (h *createsHandler) withCreateOrders(dispose *dtmcli.SagaDispose) {
	paymentCoinAmount := h.paymentCoinAmount.String()
	discountCoinAmount := h.reductionCoinAmount.String()
	transferCoinAmount := h.transferCoinAmount.String()
	balanceCoinAmount := h.balanceCoinAmount.String()
	coinUSDCurrency := h.coinCurrencyAmount.String()
	localCoinUSDCurrency := h.localCurrencyAmount.String()
	liveCoinUSDCurrency := h.liveCurrencyAmount.String()

	reqs := []*ordermwpb.OrderReq{}
	for _, order := range h.Orders {
		goodValueCoinAmount := h.goodValueCoinAmount[order.AppGoodID].String()
		goodValueUSDTAmount := h.goodValueUSDTAmount[order.AppGoodID].String()
		orderStartAt := h.orderStartAt[order.AppGoodID]
		orderEndAt := h.orderEndAt[order.AppGoodID]
		startMode := h.orderStartMode[order.AppGoodID]
		goodDurationDays := uint32(h.appGoods[order.AppGoodID].DurationDays)

		req := &ordermwpb.OrderReq{
			ID:                   h.ids[order.AppGoodID],
			AppID:                h.AppID,
			UserID:               h.UserID,
			GoodID:               &h.appGoods[order.AppGoodID].GoodID,
			AppGoodID:            &order.AppGoodID,
			Units:                &order.Units,
			GoodValue:            &goodValueCoinAmount,
			GoodValueUSD:         &goodValueUSDTAmount,
			DurationDays:         &goodDurationDays,
			OrderType:            h.OrderType,
			InvestmentType:       h.InvestmentType,
			CoinTypeID:           &h.appGoods[order.AppGoodID].CoinTypeID,
			PaymentCoinTypeID:    h.PaymentCoinID,
			CoinUSDCurrency:      &coinUSDCurrency,
			LocalCoinUSDCurrency: &localCoinUSDCurrency,
			LiveCoinUSDCurrency:  &liveCoinUSDCurrency,
			StartAt:              &orderStartAt,
			EndAt:                &orderEndAt,
			StartMode:            &startMode,
			AppGoodStockLockID:   h.stockLockIDs[order.AppGoodID],
		}
		if h.promotions[order.AppGoodID] != nil {
			req.PromotionID = &h.promotions[order.AppGoodID].ID
		}
		if order.Parent {
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
		}
		reqs = append(reqs, req)
	}

	dispose.Add(
		ordermwsvcname.ServiceDomain,
		"order.middleware.order1.v1.Middleware/CreateOrders",
		"order.middleware.order1.v1.Middleware/DeleteOrders",
		&ordermwpb.CreateOrdersRequest{
			Infos: reqs,
		},
	)
}

func (h *createsHandler) withLockPaymentAccount(dispose *dtmcli.SagaDispose) {
	if h.paymentAccount == nil {
		return
	}

	locked := true
	lockedBy := basetypes.AccountLockedBy_Payment
	req := &payaccmwpb.AccountReq{
		ID:       &h.paymentAccount.ID,
		Locked:   &locked,
		LockedBy: &lockedBy,
	}
	dispose.Add(
		accountmwsvcname.ServiceDomain,
		"account.middleware.payment.v1.Middleware/UpdateAccount",
		"",
		&payaccmwpb.UpdateAccountRequest{
			Info: req,
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

func (h *createsHandler) validateRequiredGoods() error {
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

//nolint:funlen,gocyclo
func (h *Handler) CreateOrders(ctx context.Context) (infos []*npool.Order, err error) {
	handler := &createsHandler{
		Handler:             h,
		ids:                 map[string]*string{},
		appGoods:            map[string]*appgoodmwpb.Good{},
		goods:               map[string]*goodmwpb.Good{},
		requiredGoods:       map[string]*goodrequiredpb.Required{},
		coupons:             map[string]*allocatedmwpb.Coupon{},
		promotions:          map[string]*topmostmwpb.TopMostGood{},
		goodValueUSDTAmount: map[string]decimal.Decimal{},
		goodValueCoinAmount: map[string]decimal.Decimal{},
		orderStartMode:      map[string]types.OrderStartMode{},
		orderStartAt:        map[string]uint32{},
		orderEndAt:          map[string]uint32{},
		stockLockIDs:        map[string]*string{},
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
	if err := handler.getAppGoods(ctx); err != nil {
		return nil, err
	}
	if err := handler.getGoods(ctx); err != nil {
		return nil, err
	}
	if err := handler.checkAppGoodCoin(ctx); err != nil {
		return nil, err
	}
	if err := handler.checkUnitsLimit(ctx); err != nil {
		return nil, err
	}
	if err := handler.getRequiredGoods(ctx); err != nil {
		return nil, err
	}
	if err := handler.validateOrderGoods(); err != nil {
		return nil, err
	}
	if err := handler.validateRequiredGoods(); err != nil {
		return nil, err
	}
	if err := handler.getAppGoodPromotions(ctx); err != nil {
		return nil, err
	}
	if err := handler.checkPaymentCoinCurrency(ctx); err != nil {
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
	if err := handler.checkPaymentCoinAmount(); err != nil {
		return nil, err
	}
	if err := handler.checkTransferCoinAmount(); err != nil {
		return nil, err
	}
	handler.resolvePaymentType()
	handler.resolveStartMode()
	handler.resolveStartEnd()

	if err := handler.peekPaymentAddress(ctx); err != nil {
		return nil, err
	}
	if handler.paymentAccount != nil {
		if err := accountlock.Lock(handler.paymentAccount.AccountID); err != nil {
			return nil, err
		}
		if err := handler.recheckPaymentAccount(ctx); err != nil {
			return nil, err
		}
		defer func() {
			_ = accountlock.Unlock(handler.paymentAccount.AccountID)
		}()
		if err := handler.getPaymentStartAmount(ctx); err != nil {
			return nil, err
		}
	}

	for _, order := range h.Orders {
		id1 := uuid.NewString()
		handler.ids[order.AppGoodID] = &id1
		if order.Parent {
			h.ParentOrderID = &id1
		}
		h.IDs = append(h.IDs, id1)

		id2 := uuid.NewString()
		handler.stockLockIDs[order.AppGoodID] = &id2
	}
	if handler.balanceCoinAmount.Cmp(decimal.NewFromInt(0)) > 0 {
		id := uuid.NewString()
		handler.balanceLockID = &id
	}

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
	})

	handler.withUpdateStock(sagaDispose)
	handler.withUpdateBalance(sagaDispose)
	handler.withCreateOrders(sagaDispose)
	handler.withLockPaymentAccount(sagaDispose)

	if err := dtmcli.WithSaga(ctx, sagaDispose); err != nil {
		return nil, err
	}

	orders, _, err := h.GetOrders(ctx)
	if err != nil {
		return nil, err
	}

	return orders, nil
}
