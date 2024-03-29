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
	coinmwcli "github.com/NpoolPlatform/chain-middleware/pkg/client/coin"
	currencymwcli "github.com/NpoolPlatform/chain-middleware/pkg/client/coin/currency"
	dtmcli "github.com/NpoolPlatform/dtm-cluster/pkg/dtm"
	"github.com/NpoolPlatform/go-service-framework/pkg/logger"
	allocatedmwcli "github.com/NpoolPlatform/inspire-middleware/pkg/client/coupon/allocated"
	appgoodscopemwcli "github.com/NpoolPlatform/inspire-middleware/pkg/client/coupon/app/scope"
	couponwithdrawmwcli "github.com/NpoolPlatform/ledger-middleware/pkg/client/withdraw/coupon"
	ledgermwsvcname "github.com/NpoolPlatform/ledger-middleware/pkg/servicename"
	cruder "github.com/NpoolPlatform/libent-cruder/pkg/cruder"
	payaccmwpb "github.com/NpoolPlatform/message/npool/account/mw/v1/payment"
	appmwpb "github.com/NpoolPlatform/message/npool/appuser/mw/v1/app"
	usermwpb "github.com/NpoolPlatform/message/npool/appuser/mw/v1/user"
	inspiretypes "github.com/NpoolPlatform/message/npool/basetypes/inspire/v1"
	ledgertypes "github.com/NpoolPlatform/message/npool/basetypes/ledger/v1"
	types "github.com/NpoolPlatform/message/npool/basetypes/order/v1"
	basetypes "github.com/NpoolPlatform/message/npool/basetypes/v1"
	appcoinmwpb "github.com/NpoolPlatform/message/npool/chain/mw/v1/app/coin"
	coinmwpb "github.com/NpoolPlatform/message/npool/chain/mw/v1/coin"
	currencymwpb "github.com/NpoolPlatform/message/npool/chain/mw/v1/coin/currency"
	appgoodmwpb "github.com/NpoolPlatform/message/npool/good/mw/v1/app/good"
	appsimulategoodmwpb "github.com/NpoolPlatform/message/npool/good/mw/v1/app/good/simulate"
	allocatedmwpb "github.com/NpoolPlatform/message/npool/inspire/mw/v1/coupon/allocated"
	appgoodscopemwpb "github.com/NpoolPlatform/message/npool/inspire/mw/v1/coupon/app/scope"
	ledgermwpb "github.com/NpoolPlatform/message/npool/ledger/mw/v2/ledger"
	couponwithdrawmwpb "github.com/NpoolPlatform/message/npool/ledger/mw/v2/withdraw/coupon"
	ordermwpb "github.com/NpoolPlatform/message/npool/order/mw/v1/order"
	configmwpb "github.com/NpoolPlatform/message/npool/order/mw/v1/simulate/config"
	sphinxproxypb "github.com/NpoolPlatform/message/npool/sphinxproxy"
	ordermwcli "github.com/NpoolPlatform/order-middleware/pkg/client/order"
	configmwcli "github.com/NpoolPlatform/order-middleware/pkg/client/simulate/config"
	sphinxproxycli "github.com/NpoolPlatform/sphinx-proxy/pkg/client"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

type baseCreateHandler struct {
	*dtmHandler
	app                     *appmwpb.App
	user                    *usermwpb.User
	parentOrder             *ordermwpb.Order
	paymentCoin             *appcoinmwpb.Coin
	paymentAccount          *payaccmwpb.Account
	simulateConfig          *configmwpb.SimulateConfig
	paymentAccountLockStart time.Time
	paymentStartAmount      decimal.Decimal
	coupons                 map[string]*allocatedmwpb.Coupon
	appSimulateGoods        map[string]*appsimulategoodmwpb.Simulate
	paymentCoinAmount       decimal.Decimal
	paymentUSDAmount        decimal.Decimal
	reductionUSDAmount      decimal.Decimal
	reductionCoinAmount     decimal.Decimal
	liveCurrencyAmount      decimal.Decimal
	coinCurrencyAmount      decimal.Decimal
	localCurrencyAmount     decimal.Decimal
	balanceCoinAmount       decimal.Decimal
	transferCoinAmount      decimal.Decimal
	paymentType             types.PaymentType
	stockLockID             *string
	balanceLockID           *string
	goodCoinEnv             string
	needCheckStock          bool
}

func (h *baseCreateHandler) getUser(ctx context.Context) error {
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

func (h *baseCreateHandler) getApp(ctx context.Context) error {
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

func (h *baseCreateHandler) getStableUSDCoin(ctx context.Context) error {
	coinName := "usdttrc20"
	if h.goodCoinEnv == "test" {
		coinName = "tusdttrc20"
	}
	coin, err := coinmwcli.GetCoinOnly(ctx, &coinmwpb.Conds{
		Name: &basetypes.StringVal{Op: cruder.EQ, Value: coinName},
		ENV:  &basetypes.StringVal{Op: cruder.EQ, Value: h.goodCoinEnv},
	})
	if err != nil {
		return err
	}
	if coin == nil {
		return fmt.Errorf("invalid stablecoin")
	}
	coin1, err := appcoinmwcli.GetCoinOnly(ctx, &appcoinmwpb.Conds{
		AppID:      &basetypes.StringVal{Op: cruder.EQ, Value: *h.AppID},
		CoinTypeID: &basetypes.StringVal{Op: cruder.EQ, Value: coin.EntID},
	})
	if err != nil {
		return err
	}
	if coin1 == nil {
		return fmt.Errorf("invalid appcoin")
	}
	h.paymentCoin = coin1
	h.PaymentCoinID = &coin.EntID
	return nil
}

func (h *baseCreateHandler) getPaymentCoin(ctx context.Context) error {
	if h.Simulate != nil && *h.Simulate {
		return nil
	}
	if h.PaymentCoinID == nil {
		return h.getStableUSDCoin(ctx)
	}
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

func (h *baseCreateHandler) getCoupons(ctx context.Context) error {
	if h.Simulate != nil && *h.Simulate {
		return nil
	}
	if len(h.CouponIDs) == 0 {
		return nil
	}
	coupons, _, err := allocatedmwcli.GetCoupons(ctx, &allocatedmwpb.Conds{
		AppID:  &basetypes.StringVal{Op: cruder.EQ, Value: *h.AppID},
		UserID: &basetypes.StringVal{Op: cruder.EQ, Value: *h.UserID},
		EntIDs: &basetypes.StringSliceVal{Op: cruder.IN, Value: h.CouponIDs},
		Used:   &basetypes.BoolVal{Op: cruder.EQ, Value: false},
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
		h.coupons[coupon.EntID] = coupon
	}
	return nil
}

func (h *baseCreateHandler) checkCouponWithdraw(ctx context.Context) error {
	if h.Simulate != nil && *h.Simulate {
		return nil
	}
	for _, coupon := range h.coupons {
		cw, err := couponwithdrawmwcli.GetCouponWithdrawOnly(ctx, &couponwithdrawmwpb.Conds{
			AppID:       &basetypes.StringVal{Op: cruder.EQ, Value: *h.AppID},
			UserID:      &basetypes.StringVal{Op: cruder.EQ, Value: *h.UserID},
			AllocatedID: &basetypes.StringVal{Op: cruder.EQ, Value: coupon.EntID},
			State:       &basetypes.Uint32Val{Op: cruder.EQ, Value: uint32(ledgertypes.WithdrawState_Reviewing)},
		})
		if err != nil {
			return err
		}
		if cw != nil {
			return fmt.Errorf("coupon is withdrawing")
		}
	}
	return nil
}

func (h *baseCreateHandler) validateCouponScope(ctx context.Context, goodID, appGoodID string) error {
	if h.Simulate != nil && *h.Simulate {
		return nil
	}
	if len(h.CouponIDs) == 0 {
		return nil
	}
	reqs := []*appgoodscopemwpb.ScopeReq{}
	for _, coupon := range h.coupons {
		reqs = append(reqs, &appgoodscopemwpb.ScopeReq{
			AppID:       h.AppID,
			AppGoodID:   &appGoodID,
			GoodID:      &goodID,
			CouponID:    &coupon.CouponID,
			CouponScope: &coupon.CouponScope,
		})
	}

	if err := appgoodscopemwcli.VerifyCouponScopes(ctx, reqs); err != nil {
		return err
	}
	return nil
}

func (h *baseCreateHandler) validateDiscountCoupon() error {
	if h.Simulate != nil && *h.Simulate {
		return nil
	}
	discountCoupons := 0
	fixAmountCoupons := uint32(0)
	for _, coupon := range h.coupons {
		switch coupon.CouponType {
		case inspiretypes.CouponType_Discount:
			discountCoupons++
		case inspiretypes.CouponType_FixAmount:
			fixAmountCoupons++
		}
	}
	if discountCoupons > 1 {
		return fmt.Errorf("invalid discountcoupon")
	}
	if fixAmountCoupons > h.app.MaxTypedCouponsPerOrder {
		return fmt.Errorf("invalid fixamountcoupon")
	}
	return nil
}

func (h *baseCreateHandler) checkMaxUnpaidOrders(ctx context.Context) error {
	if h.Simulate != nil && *h.Simulate {
		return nil
	}
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

func (h *baseCreateHandler) checkParentOrder(ctx context.Context) error {
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
	if order.AppID != *h.AppID || order.UserID != *h.UserID {
		return fmt.Errorf("invalid parentorder")
	}
	h.parentOrder = order
	return nil
}

func (h *baseCreateHandler) checkCouponConstraint() error {
	for _, coupon := range h.coupons {
		if coupon.CouponConstraint == inspiretypes.CouponConstraint_PaymentThreshold {
			threshold, err := decimal.NewFromString(coupon.Threshold)
			if err != nil {
				return fmt.Errorf("threshold not available")
			}
			if h.paymentUSDAmount.Cmp(threshold) < 0 {
				return fmt.Errorf("payment amount needs to be greater than  %v", threshold)
			}
		}
	}
	return nil
}

func (h *baseCreateHandler) calculateDiscountCouponReduction() error {
	for _, coupon := range h.coupons {
		if coupon.CouponType != inspiretypes.CouponType_Discount {
			continue
		}
		discount, err := decimal.NewFromString(coupon.Denomination)
		if err != nil {
			return err
		}
		h.reductionUSDAmount = h.reductionUSDAmount.Add(
			h.paymentUSDAmount.Mul(discount).Div(decimal.NewFromInt(100)), //nolint
		)
	}
	return nil
}

func (h *baseCreateHandler) calculateFixAmountCouponReduction() error {
	for _, coupon := range h.coupons {
		switch coupon.CouponType {
		case inspiretypes.CouponType_FixAmount:
		default:
			continue
		}
		amount, err := decimal.NewFromString(coupon.Denomination)
		if err != nil {
			return err
		}
		h.reductionUSDAmount = h.reductionUSDAmount.Add(amount)
	}
	return nil
}

func (h *baseCreateHandler) checkPaymentCoinCurrency(ctx context.Context) error {
	if h.paymentCoin == nil {
		return nil
	}
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

func (h *baseCreateHandler) checkPaymentCoinAmount() error {
	if h.paymentCoin == nil {
		return nil
	}

	amount := h.paymentUSDAmount.
		Sub(h.reductionUSDAmount).
		Div(h.coinCurrencyAmount)
	if amount.Cmp(decimal.NewFromInt(0)) <= 0 {
		return fmt.Errorf("invalid price")
	}
	h.paymentCoinAmount = amount
	h.reductionCoinAmount = h.reductionUSDAmount.Div(h.coinCurrencyAmount)
	return nil
}

func (h *baseCreateHandler) checkTransferCoinAmount() error {
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

func (h *baseCreateHandler) resolvePaymentType() {
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

func (h *baseCreateHandler) peekExistAddress(ctx context.Context) (*payaccmwpb.Account, error) {
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
		if err := accountlock.Lock(account.AccountID); err != nil {
			continue
		}
		usable, err := h.recheckPaymentAccount(ctx, account.EntID)
		if err != nil {
			_ = accountlock.Unlock(account.AccountID)
			return nil, err
		}
		if !usable {
			_ = accountlock.Unlock(account.AccountID)
			continue
		}
		return account, nil
	}
	return nil, fmt.Errorf("invalid address")
}

func (h *baseCreateHandler) peekNewAddress(ctx context.Context) (*payaccmwpb.Account, error) {
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

func (h *baseCreateHandler) acquirePaymentAddress(ctx context.Context) error {
	switch h.paymentType {
	case types.PaymentType_PayWithBalanceOnly:
		fallthrough //nolint
	case types.PaymentType_PayWithOffline:
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
	h.paymentAccountLockStart = time.Now()
	logger.Sugar().Infow(
		"acquirePaymentAddress",
		"OrderID", *h.EntID,
		"AccountID", account.AccountID,
		"LockAt", h.paymentAccountLockStart,
	)
	return nil
}

func (h *baseCreateHandler) releasePaymentAddress() {
	if h.paymentAccount != nil {
		_ = accountlock.Unlock(h.paymentAccount.AccountID)
		logger.Sugar().Infow(
			"releasePaymentAddress",
			"OrderID", *h.EntID,
			"AccountID", h.paymentAccount.AccountID,
			"LockElapsed", time.Since(h.paymentAccountLockStart),
		)
	}
}

/**
 * paymentAccountID: ID of account_manager.payments
 */
func (h *baseCreateHandler) recheckPaymentAccount(ctx context.Context, paymentAccountID string) (bool, error) {
	account, err := payaccmwcli.GetAccount(ctx, paymentAccountID)
	if err != nil {
		return false, err
	}
	if account == nil {
		return false, fmt.Errorf("invalid account")
	}
	if account.Locked || !account.Active || account.Blocked {
		return false, nil
	}
	if account.AvailableAt > uint32(time.Now().Unix()) {
		return false, nil
	}
	return true, nil
}

func (h *baseCreateHandler) getPaymentStartAmount(ctx context.Context) error {
	if h.paymentAccount == nil {
		return nil
	}
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

func (h *baseCreateHandler) withUpdateBalance(dispose *dtmcli.SagaDispose) {
	if h.balanceCoinAmount.Cmp(decimal.NewFromInt(0)) <= 0 {
		return
	}

	dispose.Add(
		ledgermwsvcname.ServiceDomain,
		"ledger.middleware.ledger.v2.Middleware/LockBalance",
		"ledger.middleware.ledger.v2.Middleware/UnlockBalance",
		&ledgermwpb.LockBalanceRequest{
			AppID:      *h.AppID,
			UserID:     *h.UserID,
			CoinTypeID: *h.PaymentCoinID,
			Amount:     h.balanceCoinAmount.String(),
			LockID:     *h.balanceLockID,
			Rollback:   true,
		},
	)
}

func (h *baseCreateHandler) tomorrowStart() time.Time {
	now := time.Now()
	y, m, d := now.Date()
	return time.Date(y, m, d+1, 0, 0, 0, 0, now.Location())
}

func (h *baseCreateHandler) withLockPaymentAccount(dispose *dtmcli.SagaDispose) {
	if h.paymentAccount == nil {
		return
	}
	dispose.Add(
		accountmwsvcname.ServiceDomain,
		"account.middleware.payment.v1.Middleware/LockAccount",
		"account.middleware.payment.v1.Middleware/UnlockAccount",
		&payaccmwpb.LockAccountRequest{
			ID:       h.paymentAccount.ID,
			LockedBy: basetypes.AccountLockedBy_Payment,
		},
	)
}

func (h *baseCreateHandler) prepareStockAndLedgerLockIDs() {
	if h.Simulate != nil && *h.Simulate {
		return
	}
	if h.needCheckStock {
		id := uuid.NewString()
		h.stockLockID = &id
	}
	if h.balanceCoinAmount.Cmp(decimal.NewFromInt(0)) > 0 {
		id1 := uuid.NewString()
		h.balanceLockID = &id1
	}
}

func (h *baseCreateHandler) checkSimulateRepeated(ctx context.Context) error {
	if h.Simulate == nil {
		return nil
	}
	if !*h.Simulate {
		return nil
	}
	simulate := true
	exist, err := ordermwcli.ExistOrderConds(ctx, &ordermwpb.Conds{
		AppID:      &basetypes.StringVal{Op: cruder.EQ, Value: *h.AppID},
		UserID:     &basetypes.StringVal{Op: cruder.EQ, Value: *h.UserID},
		AppGoodID:  &basetypes.StringVal{Op: cruder.EQ, Value: *h.AppGoodID},
		Simulate:   &basetypes.BoolVal{Op: cruder.EQ, Value: simulate},
		OrderState: &basetypes.Uint32Val{Op: cruder.NEQ, Value: uint32(types.OrderState_OrderStateCanceled)},
	})
	if err != nil {
		return err
	}
	if exist {
		return fmt.Errorf("repeated simulate order")
	}
	return nil
}

//nolint:gocyclo
func (h *baseCreateHandler) checkUnitsLimit(ctx context.Context, appGood *appgoodmwpb.Good) error {
	if h.parentOrder != nil {
		return nil
	}
	if *h.OrderType != types.OrderType_Normal {
		return nil
	}
	if appGood.EntID != *h.AppGoodID {
		return fmt.Errorf("mismatch appgoodid")
	}
	if h.Units == nil {
		return nil
	}
	units, err := decimal.NewFromString(*h.Units)
	if err != nil {
		return err
	}

	max, err := decimal.NewFromString(appGood.MaxOrderAmount)
	if err != nil {
		return err
	}
	min, err := decimal.NewFromString(appGood.MinOrderAmount)
	if err != nil {
		return err
	}
	if (min.Cmp(decimal.NewFromInt(0)) > 0 && units.Cmp(min) < 0) ||
		(max.Cmp(decimal.NewFromInt(0)) > 0 && units.Cmp(max) > 0) {
		return fmt.Errorf("too many | less units")
	}
	if !appGood.EnablePurchase {
		return fmt.Errorf("permission denied")
	}
	if h.Simulate != nil && *h.Simulate {
		return nil
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
	userPurchaseLimit, err := decimal.NewFromString(appGood.MaxUserAmount)
	if err != nil {
		return err
	}
	if userPurchaseLimit.Cmp(decimal.NewFromInt(0)) > 0 &&
		purchaseCount.Add(units).Cmp(userPurchaseLimit) > 0 {
		return fmt.Errorf("too many units")
	}
	return nil
}

func (h *baseCreateHandler) getSimulateConfig(ctx context.Context) error {
	if h.Simulate == nil {
		return nil
	}
	if !*h.Simulate {
		return nil
	}
	enabled := true
	simulateConfig, err := configmwcli.GetSimulateConfigOnly(ctx, &configmwpb.Conds{
		AppID:   &basetypes.StringVal{Op: cruder.EQ, Value: *h.AppID},
		Enabled: &basetypes.BoolVal{Op: cruder.EQ, Value: enabled},
	})
	if err != nil {
		return err
	}
	if simulateConfig == nil {
		return fmt.Errorf("not support simulate order")
	}
	h.simulateConfig = simulateConfig

	return nil
}
