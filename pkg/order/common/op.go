package common

import (
	"context"
	"time"

	paymentaccountmwcli "github.com/NpoolPlatform/account-middleware/pkg/client/payment"
	accountlock "github.com/NpoolPlatform/account-middleware/pkg/lock"
	accountmwsvcname "github.com/NpoolPlatform/account-middleware/pkg/servicename"
	appmwcli "github.com/NpoolPlatform/appuser-middleware/pkg/client/app"
	usermwcli "github.com/NpoolPlatform/appuser-middleware/pkg/client/user"
	appcoinmwcli "github.com/NpoolPlatform/chain-middleware/pkg/client/app/coin"
	currencymwcli "github.com/NpoolPlatform/chain-middleware/pkg/client/coin/currency"
	dtmcli "github.com/NpoolPlatform/dtm-cluster/pkg/dtm"
	timedef "github.com/NpoolPlatform/go-service-framework/pkg/const/time"
	wlog "github.com/NpoolPlatform/go-service-framework/pkg/wlog"
	appgoodmwcli "github.com/NpoolPlatform/good-middleware/pkg/client/app/good"
	requiredappgoodmwcli "github.com/NpoolPlatform/good-middleware/pkg/client/app/good/required"
	topmostgoodmwcli "github.com/NpoolPlatform/good-middleware/pkg/client/app/good/topmost/good"
	allocatedcouponmwcli "github.com/NpoolPlatform/inspire-middleware/pkg/client/coupon/allocated"
	appgoodscopemwcli "github.com/NpoolPlatform/inspire-middleware/pkg/client/coupon/app/scope"
	ledgerstatementmwcli "github.com/NpoolPlatform/ledger-middleware/pkg/client/ledger/statement"
	ledgermwsvcname "github.com/NpoolPlatform/ledger-middleware/pkg/servicename"
	"github.com/NpoolPlatform/libent-cruder/pkg/cruder"
	paymentaccountmwpb "github.com/NpoolPlatform/message/npool/account/mw/v1/payment"
	appmwpb "github.com/NpoolPlatform/message/npool/appuser/mw/v1/app"
	usermwpb "github.com/NpoolPlatform/message/npool/appuser/mw/v1/user"
	goodtypes "github.com/NpoolPlatform/message/npool/basetypes/good/v1"
	inspiretypes "github.com/NpoolPlatform/message/npool/basetypes/inspire/v1"
	ledgertypes "github.com/NpoolPlatform/message/npool/basetypes/ledger/v1"
	types "github.com/NpoolPlatform/message/npool/basetypes/order/v1"
	basetypes "github.com/NpoolPlatform/message/npool/basetypes/v1"
	appcoinmwpb "github.com/NpoolPlatform/message/npool/chain/mw/v1/app/coin"
	currencymwpb "github.com/NpoolPlatform/message/npool/chain/mw/v1/coin/currency"
	appgoodmwpb "github.com/NpoolPlatform/message/npool/good/mw/v1/app/good"
	requiredappgoodmwpb "github.com/NpoolPlatform/message/npool/good/mw/v1/app/good/required"
	topmostgoodmwpb "github.com/NpoolPlatform/message/npool/good/mw/v1/app/good/topmost/good"
	allocatedcouponmwpb "github.com/NpoolPlatform/message/npool/inspire/mw/v1/coupon/allocated"
	appgoodscopemwpb "github.com/NpoolPlatform/message/npool/inspire/mw/v1/coupon/app/scope"
	ledgermwpb "github.com/NpoolPlatform/message/npool/ledger/mw/v2/ledger"
	ledgerstatementmwpb "github.com/NpoolPlatform/message/npool/ledger/mw/v2/ledger/statement"
	orderappconfigmwpb "github.com/NpoolPlatform/message/npool/order/mw/v1/app/config"
	feeordermwpb "github.com/NpoolPlatform/message/npool/order/mw/v1/fee"
	orderlockmwpb "github.com/NpoolPlatform/message/npool/order/mw/v1/order/lock"
	paymentmwpb "github.com/NpoolPlatform/message/npool/order/mw/v1/payment"
	powerrentalmwpb "github.com/NpoolPlatform/message/npool/order/mw/v1/powerrental"
	sphinxproxypb "github.com/NpoolPlatform/message/npool/sphinxproxy"
	ordergwcommon "github.com/NpoolPlatform/order-gateway/pkg/common"
	constant "github.com/NpoolPlatform/order-gateway/pkg/const"
	orderappconfigmwcli "github.com/NpoolPlatform/order-middleware/pkg/client/app/config"
	feeordermwcli "github.com/NpoolPlatform/order-middleware/pkg/client/fee"
	powerrentalmwcli "github.com/NpoolPlatform/order-middleware/pkg/client/powerrental"
	ordermwsvcname "github.com/NpoolPlatform/order-middleware/pkg/servicename"
	sphinxproxycli "github.com/NpoolPlatform/sphinx-proxy/pkg/client"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

type OrderOpHandler struct {
	ordergwcommon.AppGoodCheckHandler
	ordergwcommon.CoinCheckHandler
	ordergwcommon.AllocatedCouponCheckHandler
	DurationSeconds           *uint32
	PaymentTransferCoinTypeID *string
	AllocatedCouponIDs        []string
	AppGoodIDs                []string
	OrderType                 types.OrderType
	Simulate                  bool

	allocatedCoupons  map[string]*allocatedcouponmwpb.Coupon
	coinUSDCurrencies map[string]*currencymwpb.Currency
	AppGoods          map[string]*appgoodmwpb.Good

	PaymentBalanceReqs         []*paymentmwpb.PaymentBalanceReq
	PaymentTransferReq         *paymentmwpb.PaymentTransferReq
	PaymentType                *types.PaymentType
	PaymentTransferAccount     *paymentaccountmwpb.Account
	PaymentTransferStartAmount decimal.Decimal
	BalanceLockID              *string
	PaymentID                  *string

	OrderID                    *string
	OrderState                 types.OrderState
	AdminSetCanceled           *bool
	UserSetCanceled            *bool
	GoodCancelMode             goodtypes.CancelMode
	CommissionLedgerStatements []*ledgerstatementmwpb.Statement
	CommissionLockIDs          map[string]string

	DeductAmountUSD   decimal.Decimal
	PaymentAmountUSD  decimal.Decimal
	TotalGoodValueUSD decimal.Decimal

	OrderConfig      *orderappconfigmwpb.AppConfig
	App              *appmwpb.App
	User             *usermwpb.User
	AppCoins         map[string]*appcoinmwpb.Coin
	RequiredAppGoods map[string]map[string]*requiredappgoodmwpb.Required
	TopMostAppGoods  map[string]*topmostgoodmwpb.TopMostGood
}

func (h *OrderOpHandler) GetAppConfig(ctx context.Context) (err error) {
	h.OrderConfig, err = orderappconfigmwcli.GetAppConfig(ctx, *h.AllocatedCouponCheckHandler.AppID)
	return wlog.WrapError(err)
}

func (h *OrderOpHandler) GetAllocatedCoupons(ctx context.Context) error {
	infos, _, err := allocatedcouponmwcli.GetCoupons(ctx, &allocatedcouponmwpb.Conds{
		AppID:  &basetypes.StringVal{Op: cruder.EQ, Value: *h.AllocatedCouponCheckHandler.AppID},
		UserID: &basetypes.StringVal{Op: cruder.EQ, Value: *h.AllocatedCouponCheckHandler.UserID},
		EntIDs: &basetypes.StringSliceVal{Op: cruder.IN, Value: h.AllocatedCouponIDs},
	}, 0, int32(len(h.AllocatedCouponIDs)))
	if err != nil {
		return wlog.WrapError(err)
	}
	if len(infos) != len(h.AllocatedCouponIDs) {
		return wlog.Errorf("invalid allocatedcoupons")
	}
	h.allocatedCoupons = map[string]*allocatedcouponmwpb.Coupon{}
	for _, info := range infos {
		h.allocatedCoupons[info.EntID] = info
	}
	return nil
}

func (h *OrderOpHandler) GetAppCoins(ctx context.Context, parentGoodCoinTypeIDs []string) error {
	coinTypeIDs := func() (_coinTypeIDs []string) {
		for _, balance := range h.PaymentBalanceReqs {
			_coinTypeIDs = append(_coinTypeIDs, *balance.CoinTypeID)
		}
		return
	}()
	coinTypeIDs = append(coinTypeIDs, parentGoodCoinTypeIDs...)
	if h.PaymentTransferCoinTypeID != nil {
		coinTypeIDs = append(coinTypeIDs, *h.PaymentTransferCoinTypeID)
	}
	coins, _, err := appcoinmwcli.GetCoins(ctx, &appcoinmwpb.Conds{
		AppID:       &basetypes.StringVal{Op: cruder.EQ, Value: *h.AppGoodCheckHandler.AppID},
		CoinTypeIDs: &basetypes.StringSliceVal{Op: cruder.IN, Value: coinTypeIDs},
	}, 0, int32(len(coinTypeIDs)))
	if err != nil {
		return wlog.WrapError(err)
	}
	h.AppCoins = map[string]*appcoinmwpb.Coin{}
	coinENV := ""
	for _, coin := range coins {
		if coinENV != "" && coin.ENV != coinENV {
			return wlog.Errorf("invalid appcoins")
		}
		h.AppCoins[coin.CoinTypeID] = coin
	}
	return nil
}

func (h *OrderOpHandler) GetCoinUSDCurrencies(ctx context.Context) error {
	coinTypeIDs := func() (_coinTypeIDs []string) {
		for _, balance := range h.PaymentBalanceReqs {
			_coinTypeIDs = append(_coinTypeIDs, *balance.CoinTypeID)
		}
		return
	}()
	if h.PaymentTransferCoinTypeID != nil {
		coinTypeIDs = append(coinTypeIDs, *h.PaymentTransferCoinTypeID)
	}
	infos, _, err := currencymwcli.GetCurrencies(ctx, &currencymwpb.Conds{
		CoinTypeIDs: &basetypes.StringSliceVal{Op: cruder.IN, Value: coinTypeIDs},
	}, 0, int32(len(coinTypeIDs)))
	if err != nil {
		return wlog.WrapError(err)
	}
	h.coinUSDCurrencies = map[string]*currencymwpb.Currency{}
	now := uint32(time.Now().Unix())
	for _, info := range infos {
		if info.CreatedAt+timedef.SecondsPerMinute*10 < now {
			return wlog.Errorf("stale coincurrency")
		}
		h.coinUSDCurrencies[info.CoinTypeID] = info
	}
	return nil
}

func (h *OrderOpHandler) GetAppGoods(ctx context.Context) error {
	appGoods, _, err := appgoodmwcli.GetGoods(ctx, &appgoodmwpb.Conds{
		AppID:  &basetypes.StringVal{Op: cruder.EQ, Value: *h.AppGoodCheckHandler.AppID},
		EntIDs: &basetypes.StringSliceVal{Op: cruder.IN, Value: h.AppGoodIDs},
	}, 0, int32(len(h.AppGoodIDs)))
	if err != nil {
		return wlog.WrapError(err)
	}
	if len(appGoods) != len(h.AppGoodIDs) {
		return wlog.Errorf("invalid appgoods")
	}
	h.AppGoods = map[string]*appgoodmwpb.Good{}
	for _, appGood := range appGoods {
		h.AppGoods[appGood.EntID] = appGood
	}
	return nil
}

func (h *OrderOpHandler) GetApp(ctx context.Context) error {
	app, err := appmwcli.GetApp(ctx, *h.AppGoodCheckHandler.AppID)
	if err != nil {
		return wlog.WrapError(err)
	}
	if app == nil {
		return wlog.Errorf("invalid app")
	}
	h.App = app
	return nil
}

func (h *OrderOpHandler) GetUser(ctx context.Context) error {
	user, err := usermwcli.GetUser(ctx, *h.AppGoodCheckHandler.AppID, *h.AppGoodCheckHandler.UserID)
	if err != nil {
		return wlog.WrapError(err)
	}
	if user == nil {
		return wlog.Errorf("invalid user")
	}
	h.User = user
	return nil
}

func (h *OrderOpHandler) ValidateCouponScope(ctx context.Context, parentAppGoodID *string) error {
	if len(h.allocatedCoupons) == 0 {
		return nil
	}
	reqs := []*appgoodscopemwpb.ScopeReq{}
	for _, allocatedCoupon := range h.allocatedCoupons {
		for appGoodID, appGood := range h.AppGoods {
			if parentAppGoodID != nil && *parentAppGoodID != appGoodID {
				continue
			}
			_appGoodID := appGoodID
			reqs = append(reqs, &appgoodscopemwpb.ScopeReq{
				AppID:       h.AppGoodCheckHandler.AppID,
				AppGoodID:   &_appGoodID,
				GoodID:      &appGood.GoodID,
				CouponID:    &allocatedCoupon.CouponID,
				CouponScope: &allocatedCoupon.CouponScope,
			})
		}
	}
	return wlog.WrapError(appgoodscopemwcli.VerifyCouponScopes(ctx, reqs))
}

func (h *OrderOpHandler) ValidateCouponCount() error {
	discountCoupons := 0
	fixAmountCoupons := uint32(0)
	for _, coupon := range h.allocatedCoupons {
		switch coupon.CouponType {
		case inspiretypes.CouponType_Discount:
			discountCoupons++
			if discountCoupons > 1 {
				return wlog.Errorf("invalid discountcoupon")
			}
		case inspiretypes.CouponType_FixAmount:
			fixAmountCoupons++
			if h.OrderConfig == nil || h.OrderConfig.MaxTypedCouponsPerOrder == 0 {
				continue
			}
			if fixAmountCoupons > h.OrderConfig.MaxTypedCouponsPerOrder {
				return wlog.Errorf("invalid fixamountcoupon")
			}
		}
	}
	return nil
}

func (h *OrderOpHandler) ValidateMaxUnpaidOrders(ctx context.Context) error {
	if h.OrderConfig == nil || h.OrderConfig.MaxUnpaidOrders == 0 {
		return nil
	}
	powerRentals, err := powerrentalmwcli.CountPowerRentalOrders(ctx, &powerrentalmwpb.Conds{
		AppID:        &basetypes.StringVal{Op: cruder.EQ, Value: *h.AppGoodCheckHandler.AppID},
		UserID:       &basetypes.StringVal{Op: cruder.EQ, Value: *h.AppGoodCheckHandler.UserID},
		OrderType:    &basetypes.Uint32Val{Op: cruder.EQ, Value: uint32(types.OrderType_Normal)},
		PaymentState: &basetypes.Uint32Val{Op: cruder.IN, Value: uint32(types.PaymentState_PaymentStateWait)},
	})
	if err != nil {
		return wlog.WrapError(err)
	}
	feeOrders, err := feeordermwcli.CountFeeOrders(ctx, &feeordermwpb.Conds{
		AppID:        &basetypes.StringVal{Op: cruder.EQ, Value: *h.AppGoodCheckHandler.AppID},
		UserID:       &basetypes.StringVal{Op: cruder.EQ, Value: *h.AppGoodCheckHandler.UserID},
		OrderType:    &basetypes.Uint32Val{Op: cruder.EQ, Value: uint32(types.OrderType_Normal)},
		PaymentState: &basetypes.Uint32Val{Op: cruder.IN, Value: uint32(types.PaymentState_PaymentStateWait)},
	})
	if err != nil {
		return wlog.WrapError(err)
	}
	if powerRentals+feeOrders >= h.OrderConfig.MaxUnpaidOrders {
		return wlog.Errorf("too many unpaid orders")
	}
	return nil
}

func (h *OrderOpHandler) GetRequiredAppGoods(ctx context.Context, mainAppGoodID string) error {
	offset := int32(0)
	limit := constant.DefaultRowLimit
	h.RequiredAppGoods = map[string]map[string]*requiredappgoodmwpb.Required{}

	for {
		requiredAppGoods, _, err := requiredappgoodmwcli.GetRequireds(ctx, &requiredappgoodmwpb.Conds{
			AppID:         &basetypes.StringVal{Op: cruder.EQ, Value: *h.AppGoodCheckHandler.AppID},
			MainAppGoodID: &basetypes.StringVal{Op: cruder.EQ, Value: mainAppGoodID},
			AppGoodIDs:    &basetypes.StringSliceVal{Op: cruder.IN, Value: h.AppGoodIDs},
		}, offset, limit)
		if err != nil {
			return wlog.WrapError(err)
		}
		if len(requiredAppGoods) == 0 {
			return nil
		}
		for _, requiredAppGood := range requiredAppGoods {
			requireds, ok := h.RequiredAppGoods[requiredAppGood.MainAppGoodID]
			if !ok {
				requireds = map[string]*requiredappgoodmwpb.Required{}
			}
			requireds[requiredAppGood.RequiredAppGoodID] = requiredAppGood
			h.RequiredAppGoods[requiredAppGood.MainAppGoodID] = requireds
		}
		offset += limit
	}
}

func (h *OrderOpHandler) GetTopMostAppGoods(ctx context.Context) error {
	offset := int32(0)
	limit := constant.DefaultRowLimit
	h.TopMostAppGoods = map[string]*topmostgoodmwpb.TopMostGood{}

	for {
		topMostGoods, _, err := topmostgoodmwcli.GetTopMostGoods(ctx, &topmostgoodmwpb.Conds{
			AppID:      &basetypes.StringVal{Op: cruder.EQ, Value: *h.AppGoodCheckHandler.AppID},
			AppGoodIDs: &basetypes.StringSliceVal{Op: cruder.IN, Value: h.AppGoodIDs},
		}, offset, limit)
		if err != nil {
			return wlog.WrapError(err)
		}
		if len(topMostGoods) == 0 {
			return nil
		}
		for _, topMostGood := range topMostGoods {
			unitPrice, err := decimal.NewFromString(topMostGood.UnitPrice)
			if err != nil {
				return wlog.WrapError(err)
			}
			unitPrice1 := decimal.NewFromInt(0)
			existTopMostGood, ok := h.TopMostAppGoods[topMostGood.AppGoodID]
			if ok {
				unitPrice1, err = decimal.NewFromString(existTopMostGood.UnitPrice)
				if err != nil {
					return wlog.WrapError(err)
				}
			}
			if unitPrice1.Equal(decimal.NewFromInt(0)) || unitPrice.LessThan(unitPrice1) {
				h.TopMostAppGoods[topMostGood.AppGoodID] = topMostGood
			}
		}
		offset += limit
	}
}

func (h *OrderOpHandler) CalculateDeductAmountUSD() error {
	if h.TotalGoodValueUSD.Equal(decimal.NewFromInt(0)) {
		return wlog.Errorf("invalid totalgoodvalueusd")
	}
	for _, allocatedCoupon := range h.allocatedCoupons {
		switch allocatedCoupon.CouponType {
		case inspiretypes.CouponType_Discount:
			discount, err := decimal.NewFromString(allocatedCoupon.Denomination)
			if err != nil {
				return wlog.WrapError(err)
			}
			discount = discount.Div(decimal.NewFromInt(100)) //nolint
			h.DeductAmountUSD = h.DeductAmountUSD.Add(h.TotalGoodValueUSD.Mul(discount))
		case inspiretypes.CouponType_FixAmount:
			amount, err := decimal.NewFromString(allocatedCoupon.Denomination)
			if err != nil {
				return wlog.WrapError(err)
			}
			h.DeductAmountUSD = h.DeductAmountUSD.Add(amount)
		default:
			return wlog.Errorf("invalid coupontype")
		}
	}
	return nil
}

func (h *OrderOpHandler) CalculatePaymentAmountUSD() {
	h.PaymentAmountUSD = h.TotalGoodValueUSD.Sub(h.DeductAmountUSD)
	if h.PaymentAmountUSD.Cmp(decimal.NewFromInt(0)) < 0 {
		h.PaymentAmountUSD = decimal.NewFromInt(0)
	}
}

func (h *OrderOpHandler) getCoinUSDCurrency(coinTypeID string) (cur decimal.Decimal, live, local *string, err error) {
	currency, ok := h.coinUSDCurrencies[coinTypeID]
	if !ok {
		return cur, live, local, wlog.Errorf("invalid currency")
	}
	amount, err := decimal.NewFromString(currency.MarketValueLow)
	if err != nil {
		return cur, live, local, wlog.WrapError(err)
	}

	cur = amount
	live = func() *string { s := amount.String(); return &s }()

	appCoin, ok := h.AppCoins[coinTypeID]
	if !ok {
		return cur, live, local, wlog.Errorf("invalid coin")
	}

	amount, err = decimal.NewFromString(appCoin.SettleValue)
	if err != nil {
		return cur, live, local, wlog.WrapError(err)
	}
	if amount.GreaterThan(decimal.NewFromInt(0)) {
		cur = amount
	}

	amount, err = decimal.NewFromString(appCoin.MarketValue)
	if err != nil {
		return cur, live, local, wlog.WrapError(err)
	}
	if amount.GreaterThan(decimal.NewFromInt(0)) {
		local = func() *string { s := amount.String(); return &s }()
	}
	if cur.Cmp(decimal.NewFromInt(0)) <= 0 {
		return cur, live, local, wlog.Errorf("invalid currency")
	}

	return cur, live, local, nil
}

func (h *OrderOpHandler) ConstructOrderPayment() error {
	switch h.OrderType {
	case types.OrderType_Offline:
		fallthrough //nolint
	case types.OrderType_Airdrop:
		return nil
	}
	if h.Simulate {
		return nil
	}

	remainAmountUSD := h.PaymentAmountUSD

	for _, balance := range h.PaymentBalanceReqs {
		cur, live, local, err := h.getCoinUSDCurrency(*balance.CoinTypeID)
		if err != nil {
			return wlog.WrapError(err)
		}
		amount, err := decimal.NewFromString(*balance.Amount)
		if err != nil {
			return wlog.WrapError(err)
		}
		if amount.Cmp(decimal.NewFromInt(0)) <= 0 {
			return wlog.Errorf("invalid paymentbalanceamount")
		}
		amountUSD := amount.Mul(cur)
		if remainAmountUSD.Cmp(amountUSD) < 0 {
			amountUSD = remainAmountUSD
		}
		balance.CoinUSDCurrency = func() *string { s := cur.String(); return &s }()
		balance.LiveCoinUSDCurrency = live
		balance.LocalCoinUSDCurrency = local
		remainAmountUSD = remainAmountUSD.Sub(amountUSD)
		if remainAmountUSD.Cmp(decimal.NewFromInt(0)) <= 0 {
			return nil
		}
	}
	if h.PaymentTransferCoinTypeID == nil {
		return wlog.Errorf("invalid paymenttransfercointypeid")
	}
	if h.PaymentTransferAccount == nil {
		return wlog.Errorf("invalid paymenttransferaccount")
	}
	cur, live, local, err := h.getCoinUSDCurrency(*h.PaymentTransferCoinTypeID)
	if err != nil {
		return wlog.WrapError(err)
	}
	remainAmountCoin := remainAmountUSD.Div(cur)
	h.PaymentTransferReq = &paymentmwpb.PaymentTransferReq{
		CoinTypeID:           h.PaymentTransferCoinTypeID,
		Amount:               func() *string { s := remainAmountCoin.String(); return &s }(),
		AccountID:            &h.PaymentTransferAccount.AccountID,
		StartAmount:          func() *string { s := h.PaymentTransferStartAmount.String(); return &s }(),
		CoinUSDCurrency:      func() *string { s := cur.String(); return &s }(),
		LiveCoinUSDCurrency:  live,
		LocalCoinUSDCurrency: local,
	}
	return nil
}

func (h *OrderOpHandler) ValidateCouponConstraint() error {
	for _, allocatedCoupon := range h.allocatedCoupons {
		if allocatedCoupon.CouponConstraint != inspiretypes.CouponConstraint_PaymentThreshold {
			continue
		}
		thresholdAmount, err := decimal.NewFromString(allocatedCoupon.Threshold)
		if err != nil {
			return wlog.WrapError(err)
		}
		if h.PaymentAmountUSD.LessThan(thresholdAmount) {
			return wlog.Errorf("not enough payment amount")
		}
	}
	return nil
}

func (h *OrderOpHandler) ResolvePaymentType() error {
	if h.PaymentTransferReq == nil && len(h.PaymentBalanceReqs) == 0 {
		if !h.Simulate {
			switch h.OrderType {
			case types.OrderType_Offline:
			case types.OrderType_Airdrop:
			default:
				return wlog.Errorf("invalid paymenttype")
			}
		}
		h.PaymentType = types.PaymentType_PayWithNoPayment.Enum()
	}
	if h.PaymentTransferReq == nil {
		h.PaymentType = types.PaymentType_PayWithBalanceOnly.Enum()
		return nil
	}
	if len(h.PaymentBalanceReqs) == 0 {
		h.PaymentType = types.PaymentType_PayWithTransferOnly.Enum()
		return nil
	}
	h.PaymentType = types.PaymentType_PayWithTransferAndBalance.Enum()
	return nil
}

/**
 * paymentAccountID: ID of account_manager.payments
 */
func (h *OrderOpHandler) recheckPaymentAccount(ctx context.Context, paymentAccountID string) (bool, error) {
	account, err := paymentaccountmwcli.GetAccount(ctx, paymentAccountID)
	if err != nil {
		return false, wlog.WrapError(err)
	}
	if account == nil {
		return false, wlog.Errorf("invalid account")
	}
	if account.Locked || !account.Active || account.Blocked {
		return false, nil
	}
	if account.AvailableAt > uint32(time.Now().Unix()) {
		return false, nil
	}
	return true, nil
}

func (h *OrderOpHandler) peekExistPaymentAccount(ctx context.Context) (*paymentaccountmwpb.Account, error) {
	accounts, _, err := paymentaccountmwcli.GetAccounts(ctx, &paymentaccountmwpb.Conds{
		CoinTypeID:  &basetypes.StringVal{Op: cruder.EQ, Value: *h.PaymentTransferCoinTypeID},
		Active:      &basetypes.BoolVal{Op: cruder.EQ, Value: true},
		Locked:      &basetypes.BoolVal{Op: cruder.EQ, Value: false},
		Blocked:     &basetypes.BoolVal{Op: cruder.EQ, Value: false},
		AvailableAt: &basetypes.Uint32Val{Op: cruder.LTE, Value: uint32(time.Now().Unix())},
	}, 0, 5) //nolint
	if err != nil {
		return nil, wlog.WrapError(err)
	}
	for _, account := range accounts {
		if err := accountlock.Lock(account.AccountID); err != nil {
			continue
		}
		usable, err := h.recheckPaymentAccount(ctx, account.EntID)
		if err != nil {
			_ = accountlock.Unlock(account.AccountID)
			return nil, wlog.WrapError(err)
		}
		if !usable {
			_ = accountlock.Unlock(account.AccountID)
			continue
		}
		return account, nil
	}
	return nil, wlog.Errorf("invalid paymentaccount")
}

func (h *OrderOpHandler) peekNewPaymentAccount(ctx context.Context) (*paymentaccountmwpb.Account, error) {
	paymentTransferCoin, ok := h.AppCoins[*h.PaymentTransferCoinTypeID]
	if !ok {
		return nil, wlog.Errorf("invalid paymenttransfercoin")
	}
	for i := 0; i < 5; i++ {
		address, err := sphinxproxycli.CreateAddress(ctx, paymentTransferCoin.CoinName)
		if err != nil {
			return nil, wlog.WrapError(err)
		}
		if address == nil || address.Address == "" {
			return nil, wlog.Errorf("invalid address")
		}
		_, err = paymentaccountmwcli.CreateAccount(ctx, &paymentaccountmwpb.AccountReq{
			CoinTypeID: &paymentTransferCoin.CoinTypeID,
			Address:    &address.Address,
		})
		if err != nil {
			return nil, wlog.WrapError(err)
		}
	}
	return h.peekExistPaymentAccount(ctx)
}

func (h *OrderOpHandler) AcquirePaymentTransferAccount(ctx context.Context) error {
	if h.PaymentTransferCoinTypeID == nil {
		return nil
	}
	account, err := h.peekExistPaymentAccount(ctx)
	if err != nil {
		account, err = h.peekNewPaymentAccount(ctx)
		if err != nil {
			return wlog.WrapError(err)
		}
	}
	h.PaymentTransferAccount = account
	return nil
}

func (h *OrderOpHandler) ReleasePaymentTransferAccount() {
	if h.PaymentTransferAccount == nil {
		return
	}
	_ = accountlock.Unlock(h.PaymentTransferAccount.AccountID)
}

func (h *OrderOpHandler) GetPaymentTransferStartAmount(ctx context.Context) error {
	if h.PaymentTransferAccount == nil {
		return nil
	}
	paymentTransferCoin, ok := h.AppCoins[*h.PaymentTransferCoinTypeID]
	if !ok {
		return wlog.Errorf("invalid paymenttransfercoin")
	}
	balance, err := sphinxproxycli.GetBalance(ctx, &sphinxproxypb.GetBalanceRequest{
		Name:    paymentTransferCoin.CoinName,
		Address: h.PaymentTransferAccount.Address,
	})
	if err != nil {
		return wlog.WrapError(err)
	}
	if balance == nil {
		return wlog.Errorf("invalid balance")
	}
	h.PaymentTransferStartAmount, err = decimal.NewFromString(balance.BalanceStr)
	return err
}

func (h *OrderOpHandler) PrepareLedgerLockID() {
	if len(h.PaymentBalanceReqs) == 0 {
		return
	}
	h.BalanceLockID = func() *string { s := uuid.NewString(); return &s }()
}

func (h *OrderOpHandler) PreparePaymentID() {
	if h.PaymentTransferReq == nil && len(h.PaymentBalanceReqs) == 0 {
		return
	}
	h.PaymentID = func() *string { s := uuid.NewString(); return &s }()
}

func (h *OrderOpHandler) WithLockBalances(dispose *dtmcli.SagaDispose) {
	if len(h.PaymentBalanceReqs) == 0 {
		return
	}
	balances := []*ledgermwpb.LockBalancesRequest_XBalance{}
	for _, req := range h.PaymentBalanceReqs {
		balances = append(balances, &ledgermwpb.LockBalancesRequest_XBalance{
			CoinTypeID: *req.CoinTypeID,
			Amount:     *req.Amount,
		})
	}
	dispose.Add(
		ledgermwsvcname.ServiceDomain,
		"ledger.middleware.ledger.v2.Middleware/LockBalances",
		"ledger.middleware.ledger.v2.Middleware/UnlockBalances",
		&ledgermwpb.LockBalancesRequest{
			AppID:    *h.AllocatedCouponCheckHandler.AppID,
			UserID:   *h.AllocatedCouponCheckHandler.UserID,
			LockID:   *h.BalanceLockID,
			Rollback: true,
			Balances: balances,
		},
	)
}

func (h *OrderOpHandler) WithLockPaymentTransferAccount(dispose *dtmcli.SagaDispose) {
	if h.PaymentTransferAccount == nil {
		return
	}
	dispose.Add(
		accountmwsvcname.ServiceDomain,
		"account.middleware.payment.v1.Middleware/LockAccount",
		"account.middleware.payment.v1.Middleware/UnlockAccount",
		&paymentaccountmwpb.LockAccountRequest{
			ID:       h.PaymentTransferAccount.ID,
			LockedBy: basetypes.AccountLockedBy_Payment,
		},
	)
}

func (h *OrderOpHandler) PaymentUpdatable() error {
	switch h.OrderState {
	case types.OrderState_OrderStateCreated:
	case types.OrderState_OrderStateWaitPayment:
	default:
		return wlog.Errorf("permission denied")
	}
	return nil
}

func (h *OrderOpHandler) ValidateCancelParam() error {
	if h.UserSetCanceled != nil && !*h.UserSetCanceled {
		return wlog.Errorf("permission denied")
	}
	if h.AdminSetCanceled != nil && !*h.AdminSetCanceled {
		return wlog.Errorf("permission denied")
	}
	return nil
}

func (h *OrderOpHandler) UserCancelable() error {
	switch h.OrderType {
	case types.OrderType_Normal:
		switch h.OrderState {
		case types.OrderState_OrderStateWaitPayment:
			if h.AdminSetCanceled != nil {
				return wlog.Errorf("permission denied")
			}
		case types.OrderState_OrderStatePaid:
		case types.OrderState_OrderStateInService:
		default:
			return wlog.Errorf("permission denied")
		}
	case types.OrderType_Offline:
		fallthrough //nolint
	case types.OrderType_Airdrop:
		if h.UserSetCanceled != nil {
			return wlog.Errorf("permission denied")
		}
		switch h.OrderState {
		case types.OrderState_OrderStatePaid:
		case types.OrderState_OrderStateInService:
		default:
			return wlog.Errorf("permission denied")
		}
	default:
		return wlog.Errorf("permission denied")
	}
	return nil
}

func (h *OrderOpHandler) GoodCancelable() error {
	switch h.GoodCancelMode {
	case goodtypes.CancelMode_Uncancellable:
		return wlog.Errorf("permission denied")
	case goodtypes.CancelMode_CancellableBeforeStart:
		switch h.OrderState {
		case types.OrderState_OrderStateWaitPayment:
		case types.OrderState_OrderStatePaid:
		default:
			return wlog.Errorf("permission denied")
		}
	case goodtypes.CancelMode_CancellableBeforeBenefit:
		switch h.OrderState {
		case types.OrderState_OrderStateWaitPayment:
		case types.OrderState_OrderStatePaid:
		case types.OrderState_OrderStateInService:
			// This should be checked by upper layer
		default:
			return wlog.Errorf("permission denied")
		}
	default:
		return wlog.Errorf("invalid cancelmode")
	}
	return nil
}

func (h *OrderOpHandler) GetOrderCommissions(ctx context.Context) error {
	offset := int32(0)
	limit := constant.DefaultRowLimit
	for {
		infos, _, err := ledgerstatementmwcli.GetStatements(ctx, &ledgerstatementmwpb.Conds{
			AppID:     &basetypes.StringVal{Op: cruder.EQ, Value: *h.AllocatedCouponCheckHandler.AppID},
			IOType:    &basetypes.Uint32Val{Op: cruder.EQ, Value: uint32(ledgertypes.IOType_Incoming)},
			IOSubType: &basetypes.Uint32Val{Op: cruder.EQ, Value: uint32(ledgertypes.IOSubType_Commission)},
			IOExtra:   &basetypes.StringVal{Op: cruder.LIKE, Value: *h.OrderID},
		}, offset, limit)
		if err != nil {
			return wlog.WrapError(err)
		}
		if len(infos) == 0 {
			return nil
		}
		h.CommissionLedgerStatements = append(h.CommissionLedgerStatements, infos...)
	}
}

func (h *OrderOpHandler) PrepareCommissionLockIDs() {
	for _, statement := range h.CommissionLedgerStatements {
		if _, ok := h.CommissionLockIDs[statement.UserID]; ok {
			continue
		}
		h.CommissionLockIDs[statement.UserID] = uuid.NewString()
	}
}

func (h *OrderOpHandler) WithCreateOrderCommissionLocks(dispose *dtmcli.SagaDispose) {
	reqs := func() (_reqs []*orderlockmwpb.OrderLockReq) {
		for userID, commissionLockID := range h.CommissionLockIDs {
			_userID := userID
			_commissionLockID := commissionLockID
			_reqs = append(_reqs, &orderlockmwpb.OrderLockReq{
				EntID:    &_commissionLockID,
				UserID:   &_userID,
				OrderID:  h.OrderID,
				LockType: types.OrderLockType_LockCommission.Enum(),
			})
		}
		return
	}()
	dispose.Add(
		ordermwsvcname.ServiceDomain,
		"order.middleware.order1.orderlock.v1.Middleware/CreateOrderLocks",
		"order.middleware.order1.orderlock.v1.Middleware/DeleteOrderLocks",
		&orderlockmwpb.CreateOrderLocksRequest{
			Infos: reqs,
		},
	)
}

func (h *OrderOpHandler) WithLockCommissions(dispose *dtmcli.SagaDispose) {
	balances := map[string][]*ledgermwpb.LockBalancesRequest_XBalance{}
	for _, statement := range h.CommissionLedgerStatements {
		balances[statement.UserID] = append(balances[statement.UserID], &ledgermwpb.LockBalancesRequest_XBalance{
			CoinTypeID: statement.CoinTypeID,
			Amount:     statement.Amount,
		})
	}
	for userID, userBalances := range balances {
		dispose.Add(
			ledgermwsvcname.ServiceDomain,
			"ledger.middleware.ledger.v2.Middleware/LockBalances",
			"ledger.middleware.ledger.v2.Middleware/UnlockBalances",
			&ledgermwpb.LockBalancesRequest{
				AppID:    *h.AllocatedCouponCheckHandler.AppID,
				UserID:   userID,
				LockID:   h.CommissionLockIDs[userID],
				Rollback: true,
				Balances: userBalances,
			},
		)
	}
}
