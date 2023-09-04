package order

import (
	"context"
	"fmt"
	"time"

	payaccmwcli "github.com/NpoolPlatform/account-middleware/pkg/client/payment"
	accountlock "github.com/NpoolPlatform/account-middleware/pkg/lock"
	appcoinmwcli "github.com/NpoolPlatform/chain-middleware/pkg/client/app/coin"
	coininfocli "github.com/NpoolPlatform/chain-middleware/pkg/client/coin"
	currvalmwcli "github.com/NpoolPlatform/chain-middleware/pkg/client/coin/currency"
	dtmcli "github.com/NpoolPlatform/dtm-cluster/pkg/dtm"
	"github.com/NpoolPlatform/go-service-framework/pkg/logger"
	redis2 "github.com/NpoolPlatform/go-service-framework/pkg/redis"
	allocatedmwcli "github.com/NpoolPlatform/inspire-middleware/pkg/client/coupon/allocated"
	cruder "github.com/NpoolPlatform/libent-cruder/pkg/cruder"
	payaccmwpb "github.com/NpoolPlatform/message/npool/account/mw/v1/payment"
	inspiretypes "github.com/NpoolPlatform/message/npool/basetypes/inspire/v1"
	ordertypes "github.com/NpoolPlatform/message/npool/basetypes/order/v1"
	basetypes "github.com/NpoolPlatform/message/npool/basetypes/v1"
	appcoinmwpb "github.com/NpoolPlatform/message/npool/chain/mw/v1/app/coin"
	currvalmwpb "github.com/NpoolPlatform/message/npool/chain/mw/v1/coin/currency"
	appgoodstockmwpb "github.com/NpoolPlatform/message/npool/good/mw/v1/app/good/stock"
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
	user                *usermwpb.User
	appGood             *appgoodmwpb.Good
	paymentCoin         *appcoinmwpb.Coin
	paymentAccount      *payaccmwpb.Account
	paymentStartAmount  decimal.Decimal // TODO
	coupons             map[string]*allocatedmwpb.Coupon
	currency            *currencymwpb.Currency
	promotion           *topmostmwpb.TopMost
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
	orderStartAt        uint32 // TODO
	orderEndAt          uint32 // TODO
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

func (h *creatHandler) getPaymentCoin(ctx context.Context) error {
	coin, err := appcoinmwcli.GetCoinOnly(ctx, &appcoinmwpb.Conds{
		AppID:      &basetypes.StringVal{Op: cruder.EQ, Value: *h.AppID},
		CoinTypeID: &basetypes.StringVal{Op: cruder.EQ, Value: *h.PaymentCoinTypeID},
	})
	if err != nil {
		return err
	}
	if coin == nil {
		return fmt.Errorf("invalid paymentcoin")
	}
	if coin.PreSale {
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

func (h *creatHandler) validateDiscountCoupon() error {
	discountCoupons := 0
	for _, coupon := range h.coupons {
		if coupon.CouponType == inspiretypes.CouponType_Discount {
			discountCoupons++
		}
	}
	if discountCoupons > 1 {
		return fmt.Errorf("invalid discountcoupon")
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

func (h *createHandler) getAppGoodPromotion(ctx context.Context) error {
	promotion, err := topmostmwcli.GetTopMostOnly(ctx, &topmostmwpb.Conds{
		AppID:     &basetypes.StringVal{Op: cruder.EQ, Value: *h.AppID},
		AppGoodID: &basetypes.StringVal{Op: cruder.EQ, Value: *h.AppGoodID},
		// TODO: Add topmost type of promotion
		// TODO: One good is added to multiple topmost
	})
	if err != nil {
		return err
	}
	h.promotion = promotion
	return nil
}

func (h *createHandler) calculateOrderUSDTPrice() error {
	units, err := decimal.NewFromString(*h.Units)
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
		h.paymentUSDTAmount = amount.Mul(units)
		return nil
	}
	amount, err = decimal.NewFromString(h.promotion.Price)
	if err != nil {
		return err
	}
	if amount.Cmp(decimal.NewFromInt(0)) <= 0 {
		return fmt.Errorf("invalid price")
	}
	h.paymentUSDTAmount = amount.Mul(units)
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
				Add(h.paymentUSDTAmount.Mul(discount).Div(decimal.NewFromInt(100))) //nolint
			return nil
		}
	}
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
		h.coinCurrency = amount
	}

	amount, err = decimal.NewFromString(h.paymentCoin.MarketValue)
	if err != nil {
		return err
	}
	if amount.Cmp(decimal.NewFromInt(0)) > 0 {
		h.localCurrency = amount
	}
	return nil
}

func (h *createHandler) checkPaymentCoinAmount() error {
	amount := h.goodValueUSDTAmount.
		Sub(h.reductionUSDTAmount).
		Div(h.coinCurrency)
	if amount.Cmp(decimal.NewFromInt(0)) <= 0 {
		return fmt.Errorf("invalid price")
	}
	h.paymentCoinAmount = amount
	h.goodValueCoinAmount = h.goodValueUSDTAmount.Div(h.coinCurrency)
	h.reductionCoinAmount = h.reductionUSDTAmount.Div(h.coinCurrency)
	return nil
}

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
	if h.transferCoinAmount.Cmp(decimal.NewFromInt(0)) == 0 &&
		h.transferCoinAmount.Cmp(decimal.NewFromInt(0)) == 0 {
		h.paymentType = types.PaymentType_PaymentTypeNoPayment
		return
	}
	if h.transferCoinAmount.Cmp(h.paymentCoinAmount) == 0 {
		h.paymentType = types.PaymentType_PaymentTypeTransferOnly
		return
	}
	if h.balanceCoinAmount.Cmp(h.paymentCoinAmount) == 0 {
		h.paymentType = types.PaymentType_PaymentTypeBalanceOnly
		return
	}
	h.paymentType = types.PaymentType_PaymentTypeTransferAndBalance
}

func (h *createHandler) peekExistAddress(ctx context.Context) (*payaccmwpb.Account, error) {
	const batchAccounts = int32(5)
	accounts, _, err := payaccmwcli.GetAccounts(ctx, &payaccmwcli.Conds{
		CoinTypeID:  &basetypes.StringVal{Op: cruder.EQ, Value: h.PaymentCoin.CoinTypeID},
		Active:      &basetypes.BoolVal{Op: cruder.EQ, Value: true},
		Locked:      &basetypes.BoolVal{Op: cruder.EQ, Value: false},
		Blocked:     &basetypes.BoolVal{Op: cruder.EQ, Value: false},
		AvailableAt: &basetypes.Uint32Val{Op: cruder.LTE, Value: uint32(time.Now().Unix())},
	}, int32(0), batchAccounts)
	if err != nil {
		return err
	}
	for _, account := range accounts {
		if info.Locked || !info.Active || info.Blocked {
			continue
		}
		if info.AvailableAt > uint32(time.Now().Unix()) {
			continue
		}
		return account
	}
	return nil
}

func (h *createHandler) peekNewAddress(ctx context.Context) (*payaccmwpb.Account, error) {
	const createCount = 5
	successCreated := 0

	for i := 0; i < createCount; i++ {
		address, err := sphinxproxycli.CreateAddress(ctx, h.paymentCoin.CoinName)
		if err != nil {
			return err
		}
		if address == nil || address.Address == "" {
			return fmt.Errorf("invalid address")
		}
		_, err = payaccmwcli.CreateAccount(ctx, &payaccmwpb.AccountReq{
			CoinTypeID: h.paymentCoin.CoinTypeID,
			Address:    &address.Address,
		})
		if err != nil {
			return err
		}
		successCreated++
	}
	if successCreated == 0 {
		return fmt.Errorf("fail create addresses")
	}

	return h.peekExistAddress(ctx)
}

func (h *createHandler) peekPaymentAddress(ctx context.Context) error {
	switch h.paymentType {
	case types.PaymentType_PaymentTypeBalanceOnly:
		fallthrough //nolint
	case types.PaymentType_PaymentTypeNoPayment:
		return nil
	}

	account, err := handler.peekExistAddress(ctx)
	if err != nil {
		account, err = handler.peekNewAddress(ctx)
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

func (h *createHandler) withUpdateStock(dispose *dtmcli.SagaDispose) {
	req := &appgoodstockmwpb.StockReq{
		AppGoodID: h.AppGoodID,
		Locked:    h.Units,
	}
	dispose.Add(
		ordermwsvcname.ServiceDomain,
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
		CoinTypeID: h.PaymentCoinTypeID,
		Spendable:  &amount,
	}
	dispose.Add(
		ordermwsvcname.ServiceDomain,
		"ledger.middleware.ledger.v2.Middleware/SubBalance",
		"ledger.middleware.ledger.v2.Middleware/AddBalance",
		&ledgermwpb.AddBalanceRequest{
			Info: req,
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
	coinUSDCurrency := h.coinCurrency.String()
	localCoinUSDCurrency := h.localCurrency.String()
	liveCoinUSDCurrency := h.liveCurrency.String()

	req := &ordermwpb.OrderReq{
		ID:                   h.ID,
		AppID:                h.AppID,
		UserID:               h.UserID,
		GoodID:               h.GoodID,
		AppGoodID:            h.AppGoodID,
		ParentOrderID:        h.ParentOrderID,
		Units:                h.Units,
		GoodValue:            &goodValueCoinAmount,
		GoodValueUSD:         &goodValueUSDTAmount,
		PaymentAmount:        &paymentCoinAmount,
		DiscountAmount:       &discountCoinAmount,
		DurationDays:         &h.appGood.DurationDays,
		OrderType:            h.OrderType,
		InvestmentType:       h.InvestmentType,
		CouponIDs:            h.CouponIDs,
		PaymentType:          &h.paymentType,
		CoinTypeID:           &h.appGood.CoinTypeID,
		PaymentCoinTypeID:    h.PaymentCoinTypeID,
		TransferAmount:       &transferCoinAmount,
		BalanceAmount:        &balanceCoinAmount,
		CoinUSDCurrency:      &coinUSDCurrency,
		LocalCoinUSDCurrency: &localCoinUSDCurrency,
		LiveCoinUSDCurrency:  &liveCoinUSDCurrency,
		StartAt:              &h.orderStartAt,
		EndAt:                &h.orderEndAt,
		StartMode:            &h.appGood.StartMode,
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
		ordermwsvcname.ServiceDomain,
		"account.middleware.payment.v1.Middleware/UpdateAccount",
		"",
		&payaccmwpb.UpdateAccountRequest{
			Info: req,
		},
	)
}

func (h *Handler) CreateOrder(ctx context.Context) (info *npool.Order, err error) {
	// 1 Check input
	//   1.1 Check user
	//   1.2 Check app good
	//   1.3 Check payment coin
	//   1.4 Check parent order (by middleware && handler)
	//   1.5 Check coupon ids (by handler)
	//   1.6 Check balance (by dtm lock)
	//   1.7 Check only one discount coupon
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
	}
	if err := handler.getUser(ctx); err != nil {
		return err
	}
	if err := handler.checkPaymentCoin(ctx); err != nil {
		return err
	}
	if err := handler.getCoupons(ctx); err != nil {
		return err
	}
	if err := handler.validateDiscountCoupon(ctx); err != nil {
		return err
	}
	if err := handler.getAppGood(ctx); err != nil {
		return err
	}
	if err := handler.getAppGoodPromotion(ctx); err != nil {
		return err
	}
	if err := handler.calculateOrderUSDTPrice(ctx); err != nil {
		return err
	}
	if err := handler.calculateDiscountCouponReduction(); err != nil {
		return err
	}
	if err := handler.calculateFixAmountCouponReduction(); err != nil {
		return err
	}
	if err := handler.checkPaymentCoinCurrency(ctx); err != nil {
		return err
	}
	if err := handler.checkPaymentCoinAmount(); err != nil {
		return err
	}
	if err := handler.checkTransferCoinAmount(); err != nil {
		return err
	}
	handler.resolvePaymentType()

	if err := handler.peekPaymentAddress(ctx); err != nil {
		return err
	}
	if handler.paymentAccount != nil {
		if err := accountlock.Lock(handler.paymentAccount.AccountID); err != nil {
			return err
		}
		if err := handler.recheckPaymentAccount(ctx); err != nil {
			return err
		}
		defer func() {
			_ = accountlock.Unlock(handler.paymentAccount.AccountID)
		}()
	}

	id := uuid.NewString()
	if h.ID == nil {
		h.ID = &id
	}

	key := fmt.Sprintf("%v:%v:%v:%v", basetypes.Prefix_PrefixCreateOrder, *h.AppID, *h.UserID, id)
	if err := redis2.TryLock(key, 0); err != nil {
		return err
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
