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
	timedef "github.com/NpoolPlatform/go-service-framework/pkg/const/time"
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
	goodrequiredpb "github.com/NpoolPlatform/message/npool/good/mw/v1/good/required"
	allocatedmwpb "github.com/NpoolPlatform/message/npool/inspire/mw/v1/coupon/allocated"
	ledgermwpb "github.com/NpoolPlatform/message/npool/ledger/mw/v2/ledger"
	npool "github.com/NpoolPlatform/message/npool/order/gw/v1/order"
	ordermwpb "github.com/NpoolPlatform/message/npool/order/mw/v1/order"
	sphinxproxypb "github.com/NpoolPlatform/message/npool/sphinxproxy"
	ordermwcli "github.com/NpoolPlatform/order-middleware/pkg/client/order"
	ordermwsvcname "github.com/NpoolPlatform/order-middleware/pkg/servicename"
	sphinxproxycli "github.com/NpoolPlatform/sphinx-proxy/pkg/client"
	"github.com/dtm-labs/dtm/client/dtmcli/dtmimp"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

type createHandler struct {
	*Handler
	app                 *appmwpb.App
	user                *usermwpb.User
	appGood             *appgoodmwpb.Good
	parentOrder         *ordermwpb.Order
	paymentCoin         *appcoinmwpb.Coin
	paymentAccount      *payaccmwpb.Account
	paymentStartAmount  decimal.Decimal
	coupons             map[string]*allocatedmwpb.Coupon
	promotion           *topmostmwpb.TopMostGood
	goodValueUSDTAmount decimal.Decimal
	goodValueCoinAmount decimal.Decimal
	paymentCoinAmount   decimal.Decimal
	reductionUSDTAmount decimal.Decimal
	reductionCoinAmount decimal.Decimal
	liveCurrencyAmount  decimal.Decimal
	coinCurrencyAmount  decimal.Decimal
	localCurrencyAmount decimal.Decimal
	balanceCoinAmount   decimal.Decimal
	transferCoinAmount  decimal.Decimal
	paymentType         types.PaymentType
	orderStartMode      types.OrderStartMode
	orderStartAt        uint32
	orderEndAt          uint32
	stockLockID         *string
	balanceLockID       *string
}

func (h *createHandler) getUser(ctx context.Context) error {
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

func (h *createHandler) getApp(ctx context.Context) error {
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

func (h *createHandler) getPaymentCoin(ctx context.Context) error {
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

func (h *createHandler) getCoupons(ctx context.Context) error {
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

func (h *createHandler) validateDiscountCoupon() error {
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

func (h *createHandler) checkMaxUnpaidOrders(ctx context.Context) error {
	const maxUnpaidOrders = uint32(10000)
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

	if userPurchaseLimit.Cmp(decimal.NewFromInt(0)) > 0 && purchaseCount.Add(units).Cmp(userPurchaseLimit) > 0 {
		return fmt.Errorf("too many units")
	}

	return nil
}

func (h *createHandler) getParentOrder(ctx context.Context) error {
	if h.ParentOrderID == nil {
		return nil
	}
	order, err := ordermwcli.GetOrder(ctx, *h.ParentOrderID)
	if err != nil {
		return err
	}
	if order == nil {
		return fmt.Errorf("invalid parentorderid")
	}
	h.parentOrder = order
	return nil
}

func (h *createHandler) checkParentOrderGoodRequired(ctx context.Context) error {
	if h.ParentOrderID == nil {
		return nil
	}
	goodRequired, err := goodrequiredmwcli.GetRequiredOnly(ctx, &goodrequiredpb.Conds{
		MainGoodID:     &basetypes.StringVal{Op: cruder.EQ, Value: h.parentOrder.GoodID},
		RequiredGoodID: &basetypes.StringVal{Op: cruder.EQ, Value: h.appGood.GoodID},
	})
	if err != nil {
		return err
	}
	if goodRequired == nil {
		return fmt.Errorf("invalid goodrequired")
	}
	return nil
}

func (h *createHandler) checkGoodRequestMust(ctx context.Context) error {
	if h.ParentOrderID != nil {
		return nil
	}
	goodRequireds, _, err := goodrequiredmwcli.GetRequireds(ctx, &goodrequiredpb.Conds{
		MainGoodID: &basetypes.StringVal{Op: cruder.EQ, Value: h.appGood.GoodID},
	}, 0, 0)
	if err != nil {
		return err
	}
	for _, goodRequired := range goodRequireds {
		if goodRequired.Must {
			return fmt.Errorf("invalid must goodrequired")
		}
	}
	return nil
}

func (h *createHandler) checkGoodRequest(ctx context.Context) error {
	if h.ParentOrderID != nil {
		return nil
	}

	goodRequireds, _, err := goodrequiredmwcli.GetRequireds(ctx, &goodrequiredpb.Conds{
		RequiredGoodID: &basetypes.StringVal{Op: cruder.EQ, Value: h.appGood.GoodID},
	}, 0, 1)
	if err != nil {
		return err
	}
	if len(goodRequireds) > 0 {
		return fmt.Errorf("parentorderid is empty")
	}

	return nil
}

func (h *createHandler) getAppGoodPromotion(ctx context.Context) error {
	promotion, err := topmostmwcli.GetTopMostGoodOnly(ctx, &topmostmwpb.Conds{
		AppID:       &basetypes.StringVal{Op: cruder.EQ, Value: *h.AppID},
		AppGoodID:   &basetypes.StringVal{Op: cruder.EQ, Value: *h.AppGoodID},
		TopMostType: &basetypes.Uint32Val{Op: cruder.EQ, Value: uint32(goodtypes.GoodTopMostType_TopMostPromotion)},
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

func (h *createHandler) calculateDiscountCouponReduction() error {
	for _, coupon := range h.coupons {
		if coupon.CouponType == inspiretypes.CouponType_Discount {
			discount, err := decimal.NewFromString(coupon.Denomination)
			if err != nil {
				return err
			}
			h.reductionUSDTAmount = h.reductionUSDTAmount.
				Add(h.goodValueUSDTAmount.Mul(discount).Div(decimal.NewFromInt(100))) //nolint
		}
	}
	return nil
}

func (h *createHandler) calculateFixAmountCouponReduction() error {
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
func (h *createHandler) checkPaymentCoinCurrency(ctx context.Context) error {
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

func (h *createHandler) getAccuracy(amount decimal.Decimal) decimal.Decimal {
	const accuracy = 1000000
	amount = amount.Mul(decimal.NewFromInt(accuracy))
	amount = amount.Ceil()
	amount = amount.Div(decimal.NewFromInt(accuracy))
	return amount
}

func (h *createHandler) checkPaymentCoinAmount() error {
	amount := h.goodValueUSDTAmount.
		Sub(h.reductionUSDTAmount).
		Div(h.coinCurrencyAmount)
	if amount.Cmp(decimal.NewFromInt(0)) <= 0 {
		return fmt.Errorf("invalid price")
	}
	h.paymentCoinAmount = h.getAccuracy(amount)
	h.goodValueCoinAmount = h.goodValueUSDTAmount.Div(h.coinCurrencyAmount)
	h.goodValueCoinAmount = h.getAccuracy(h.goodValueCoinAmount)
	h.reductionCoinAmount = h.reductionUSDTAmount.Div(h.coinCurrencyAmount)
	h.reductionCoinAmount = h.getAccuracy(h.reductionCoinAmount)
	return nil
}

//nolint:dupl
func (h *createHandler) checkTransferCoinAmount() error {
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

func (h *createHandler) resolvePaymentType() {
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

func (h *createHandler) resolveStartMode() {
	if h.appGood.StartMode == goodtypes.GoodStartMode_GoodStartModeTBD {
		h.orderStartMode = types.OrderStartMode_OrderStartTBD
		return
	}
	h.orderStartMode = types.OrderStartMode_OrderStartConfirmed
}

//nolint:dupl
func (h *createHandler) peekExistAddress(ctx context.Context) (*payaccmwpb.Account, error) {
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

func (h *createHandler) peekNewAddress(ctx context.Context) (*payaccmwpb.Account, error) {
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

func (h *createHandler) peekPaymentAddress(ctx context.Context) error {
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

func (h *createHandler) recheckPaymentAccount(ctx context.Context) error {
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

func (h *createHandler) getPaymentStartAmount(ctx context.Context) error {
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

func (h *createHandler) withUpdateStock(dispose *dtmcli.SagaDispose) {
	req := &appgoodstockmwpb.StockReq{
		ID:        &h.appGood.AppGoodStockID,
		GoodID:    &h.appGood.GoodID,
		AppGoodID: h.AppGoodID,
		Locked:    &h.Units,
		LockID:    h.stockLockID,
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

func (h *createHandler) withUpdateBalance(dispose *dtmcli.SagaDispose) {
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

func (h *createHandler) tomorrowStart() time.Time {
	now := time.Now()
	y, m, d := now.Date()
	return time.Date(y, m, d+1, 0, 0, 0, 0, now.Location())
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
		AppGoodStockLockID:   h.stockLockID,
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

func (h *createHandler) withLockPaymentAccount(dispose *dtmcli.SagaDispose) {
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
		Handler: h,
		coupons: map[string]*allocatedmwpb.Coupon{},
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
	if err := handler.getParentOrder(ctx); err != nil {
		return nil, err
	}
	if err := handler.checkParentOrderGoodRequired(ctx); err != nil {
		return nil, err
	}
	if err := handler.checkGoodRequestMust(ctx); err != nil {
		return nil, err
	}
	if err := handler.checkGoodRequest(ctx); err != nil {
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

	id1 := uuid.NewString()
	if h.ID == nil {
		h.ID = &id1
	}
	id2 := uuid.NewString()
	handler.stockLockID = &id2
	if handler.balanceCoinAmount.Cmp(decimal.NewFromInt(0)) > 0 {
		id3 := uuid.NewString()
		handler.balanceLockID = &id3
	}

	key := fmt.Sprintf("%v:%v:%v:%v", basetypes.Prefix_PrefixCreateOrder, *h.AppID, *h.UserID, id1)
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
