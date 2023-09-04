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
	user           *usermwpb.User
	appGood        []*appgoodmwpb.Good
	paymentAccount *payaccmwpb.Account
	coupons        []*inspiremwpb.Coupon
	currency       *currencymwpb.Currency
}

func tomorrowStart() time.Time {
	now := time.Now()
	y, m, d := now.Date()
	return time.Date(y, m, d+1, 0, 0, 0, 0, now.Location())
}

func (h *createHandler) validateInit(ctx context.Context) error {
	coin, err := coininfocli.GetCoin(ctx, *h.PaymentCoinID)
	if err != nil {
		return err
	}
	if coin == nil {
		return fmt.Errorf("invalid coin")
	}
	if coin.Presale {
		return fmt.Errorf("presale coin won't for payment")
	}
	if !coin.ForPay {
		return fmt.Errorf("coin not for payment")
	}
	h.paymentCoinName = coin.Name

	for _, goodReq := range h.Goods {
		good := h.orderGood.goods[goodReq.GoodID]
		gcoin, err := coininfocli.GetCoin(ctx, good.CoinTypeID)
		if err != nil {
			return err
		}
		if gcoin == nil {
			return fmt.Errorf("invalid good coin")
		}
		if coin.ENV != gcoin.ENV {
			return fmt.Errorf("good coin mismatch payment coin")
		}

		appgood := h.orderGood.appgoods[*h.AppID+*h.GoodID]
		goodStartAt := appgood.ServiceStartAt
		if appgood.ServiceStartAt == 0 {
			goodStartAt = good.StartAt
		}
		goodDurationDays := uint32(good.DurationDays)
		startAt := uint32(tomorrowStart().Unix())
		if goodStartAt > startAt {
			startAt = goodStartAt
		}
		const secondsPerDay = 24 * 60 * 60
		endAt := startAt + goodDurationDays*secondsPerDay
		h.startAts[*h.AppID+goodReq.GoodID] = startAt
		h.endAts[*h.AppID+goodReq.GoodID] = endAt
	}

	const maxUnpaidOrders = 5
	orders, _, err := ordermwcli.GetOrders(ctx, &ordermwpb.Conds{
		AppID:        &basetypes.StringVal{Op: cruder.EQ, Value: *h.AppID},
		UserID:       &basetypes.StringVal{Op: cruder.EQ, Value: *h.UserID},
		PaymentState: &basetypes.Uint32Val{Op: cruder.EQ, Value: uint32(ordertypes.PaymentState_PaymentStateWait)},
	}, 0, maxUnpaidOrders)
	if err != nil {
		return err
	}
	if len(orders) >= maxUnpaidOrders && *h.OrderType == ordertypes.OrderType_Normal {
		return fmt.Errorf("too many unpaid orders")
	}

	return nil
}

func (h *createHandler) checkGoodRequests() error {
	goodRequiredSet := make(map[string]struct{})
	goodSet := make(map[string]struct{})
	for _, goodRequired := range h.orderGood.goodRequireds {
		goodRequiredSet[goodRequired.RequiredGoodID] = struct{}{}
	}

	for _, goodReq := range h.Goods {
		if !goodReq.Parent {
			if _, ok := goodRequiredSet[goodReq.GoodID]; !ok {
				return fmt.Errorf("invalid goodrequired")
			}
			goodSet[goodReq.GoodID] = struct{}{}
		}
	}

	for _, goodRequired := range h.orderGood.goodRequireds {
		if goodRequired.Must {
			if _, ok := goodSet[goodRequired.RequiredGoodID]; !ok {
				return fmt.Errorf("invalid goodrequired must")
			}
		}
	}
	return nil
}

// nolint
func (h *createHandler) SetReduction(ctx context.Context) error {
	if len(h.CouponIDs) == 0 {
		return nil
	}

	coupons, _, err := allocatedmwcli.GetCoupons(ctx, &allocatedmwpb.Conds{
		AppID:  &basetypes.StringVal{Op: cruder.EQ, Value: *h.AppID},
		UserID: &basetypes.StringVal{Op: cruder.EQ, Value: *h.UserID},
		IDs:    &basetypes.StringSliceVal{Op: cruder.IN, Value: h.CouponIDs},
	}, int32(0), int32(len(h.CouponIDs)))
	if err != nil {
		return err
	}
	if len(coupons) != len(h.CouponIDs) {
		return fmt.Errorf("invalid coupon")
	}

	couponTypeDiscountNum := 0
	for _, coup := range coupons {
		if !coup.Valid || coup.Expired || coup.AppID != *h.AppID || coup.UserID != *h.UserID {
			return fmt.Errorf("invalid coupon")
		}
		switch coup.CouponType {
		case inspiretypes.CouponType_FixAmount:
			fallthrough //nolint
		case inspiretypes.CouponType_SpecialOffer:
			amount, err := decimal.NewFromString(coup.Denomination)
			if err != nil {
				return err
			}
			h.reductionAmount = h.reductionAmount.Add(amount)
		case inspiretypes.CouponType_Discount:
			if couponTypeDiscountNum > 1 {
				return fmt.Errorf("invalid discount")
			}
			percent, err := decimal.NewFromString(coup.Denomination)
			if err != nil {
				return err
			}
			if percent.Cmp(decimal.NewFromInt(100)) >= 0 {
				return fmt.Errorf("invalid discount")
			}
			h.reductionPercent = percent
			couponTypeDiscountNum++
		default:
			return fmt.Errorf("unknown coupon type")
		}
	}

	return nil
}

func (h *createHandler) SetPrice(ctx context.Context) error {
	for _, goodReq := range h.Goods {
		appgood := h.orderGood.appgoods[*h.AppID+goodReq.GoodID]
		topmostGood := h.orderGood.topMostGoods[*h.AppID+goodReq.GoodID]
		price, err := decimal.NewFromString(appgood.Price)
		if err != nil {
			return err
		}

		if topmostGood != nil {
			promotionPrice, err := decimal.NewFromString(topmostGood.GetPrice())
			if err != nil {
				return err
			}
			if promotionPrice.Cmp(decimal.NewFromInt(0)) <= 0 {
				return fmt.Errorf("invalid price")
			}
			price = promotionPrice
		}
		h.goodPrices[*h.AppID+goodReq.GoodID] = price
	}

	return nil
}

func (h *createHandler) SetCurrency(ctx context.Context) error {
	curr, err := currvalmwcli.GetCurrencyOnly(ctx, &currvalmwpb.Conds{
		CoinTypeID: &basetypes.StringVal{Op: cruder.EQ, Value: *h.PaymentCoinID},
	})
	if err != nil {
		return err
	}
	if curr == nil {
		return fmt.Errorf("invalid coin currency")
	}

	const maxElapsed = uint32(10 * 60)
	if curr.UpdatedAt+maxElapsed < uint32(time.Now().Unix()) {
		return fmt.Errorf("stale coin currency")
	}

	val, err := decimal.NewFromString(curr.MarketValueLow)
	if err != nil {
		return err
	}
	if val.Cmp(decimal.NewFromInt(0)) <= 0 {
		return fmt.Errorf("invalid market value")
	}

	h.liveCurrency = val
	h.coinCurrency = val

	apc, err := appcoinmwcli.GetCoinOnly(ctx, &appcoinmwpb.Conds{
		AppID:      &basetypes.StringVal{Op: cruder.EQ, Value: *h.AppID},
		CoinTypeID: &basetypes.StringVal{Op: cruder.EQ, Value: *h.PaymentCoinID},
	})
	if err != nil {
		return err
	}
	if apc == nil {
		return nil
	}

	currVal, err := decimal.NewFromString(apc.SettleValue)
	if err != nil {
		return err
	}
	if currVal.Cmp(decimal.NewFromInt(0)) > 0 {
		h.coinCurrency = currVal
	}

	currVal, err = decimal.NewFromString(apc.MarketValue)
	if err != nil {
		return err
	}
	if currVal.Cmp(decimal.NewFromInt(0)) > 0 {
		h.localCurrency = currVal
	}

	return nil
}

func getAccuracy(coin decimal.Decimal) decimal.Decimal {
	const accuracy = 1000000
	coin = coin.Mul(decimal.NewFromInt(accuracy))
	coin = coin.Ceil()
	coin = coin.Div(decimal.NewFromInt(accuracy))
	return coin
}

func (h *createHandler) SetPaymentAmount(ctx context.Context) error {
	totalPaymentAmountUSD := decimal.NewFromInt(0)
	for _, goodReq := range h.Goods {
		price := h.goodPrices[*h.AppID+goodReq.GoodID]
		units, err := decimal.NewFromString(goodReq.Units)
		if err != nil {
			return err
		}
		paymentAmountUSD := price.Mul(units)
		totalPaymentAmountUSD = totalPaymentAmountUSD.Add(paymentAmountUSD)
		h.goodValueUSDs[goodReq.GoodID] = paymentAmountUSD

		goodValueCoin := paymentAmountUSD.Div(h.coinCurrency)
		goodValueCoin = getAccuracy(goodValueCoin)
		h.paymentAmountCoin = h.paymentAmountCoin.Add(goodValueCoin)
		h.goodValueCoins[goodReq.GoodID] = goodValueCoin
	}

	logger.Sugar().Infow(
		"CreateOrder",
		"PaymentAmountUSD", totalPaymentAmountUSD,
		"ReductionAmount", h.reductionAmount,
		"ReductionPercent", h.reductionPercent,
	)

	discountAmountUSD := decimal.NewFromInt(0)
	if h.reductionPercent != decimal.NewFromInt(0) {
		discountAmountUSD = totalPaymentAmountUSD.
			Mul(h.reductionPercent).
			Div(decimal.NewFromInt(100)) //nolint
	}
	discountAmountUSD = discountAmountUSD.Add(h.reductionAmount)

	h.discountAmountCoin = discountAmountUSD.Div(h.coinCurrency)
	h.discountAmountCoin = getAccuracy(h.discountAmountCoin)

	if *h.OrderType == ordertypes.OrderType_Airdrop {
		h.paymentAmountCoin = decimal.NewFromInt(0)
	}

	h.paymentAmountCoin = h.paymentAmountCoin.Sub(h.discountAmountCoin)
	if h.paymentAmountCoin.Cmp(decimal.NewFromInt(0)) < 0 {
		h.paymentAmountCoin = decimal.NewFromInt(0)
	}

	h.paymentTransferAmount = h.paymentAmountCoin
	if h.BalanceAmount != nil {
		amount, err := decimal.NewFromString(*h.BalanceAmount)
		if err != nil {
			return err
		}
		if amount.Cmp(h.paymentTransferAmount) > 0 {
			amount = h.paymentTransferAmount
			amountStr := amount.String()
			h.BalanceAmount = &amountStr
		}
		h.paymentTransferAmount = h.paymentTransferAmount.Sub(amount)
	}

	return nil
}

func (h *createHandler) paymentType() (*ordertypes.PaymentType, error) {
	switch *h.OrderType {
	case ordertypes.OrderType_Normal:
		if h.BalanceAmount != nil {
			if h.paymentTransferAmount.Cmp(decimal.NewFromInt(0)) > 0 {
				return ordertypes.PaymentType_PayWithTransferAndBalance.Enum(), nil
			}
			return ordertypes.PaymentType_PayWithBalanceOnly.Enum(), nil
		}
		return ordertypes.PaymentType_PayWithTransferOnly.Enum(), nil
	case ordertypes.OrderType_Offline:
		return ordertypes.PaymentType_PayWithOffline.Enum(), nil
	case ordertypes.OrderType_Airdrop:
		return ordertypes.PaymentType_PayWithNoPayment.Enum(), nil
	default:
		return nil, fmt.Errorf("invalid ordertype")
	}
}

func (h *createHandler) SetPaymentType(ctx context.Context) error {
	paymentType, err := h.paymentType()
	if err != nil {
		return err
	}
	h.mainPaymentType = paymentType
	return nil
}

func (h *createHandler) createAddresses(ctx context.Context) error {
	const createCount = 5
	successCreated := 0

	for i := 0; i < createCount; i++ {
		address, err := sphinxproxycli.CreateAddress(ctx, h.paymentCoinName)
		if err != nil {
			return err
		}
		if address == nil || address.Address == "" {
			return fmt.Errorf("invalid address")
		}

		_, err = payaccmwcli.CreateAccount(ctx, &payaccmwpb.AccountReq{
			CoinTypeID: h.PaymentCoinID,
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

	return nil
}

func (h *createHandler) peekAddress(ctx context.Context) (*payaccmwpb.Account, error) {
	payments, _, err := payaccmwcli.GetAccounts(ctx, &payaccmwpb.Conds{
		CoinTypeID:  &basetypes.StringVal{Op: cruder.EQ, Value: *h.PaymentCoinID},
		Active:      &basetypes.BoolVal{Op: cruder.EQ, Value: true},
		Locked:      &basetypes.BoolVal{Op: cruder.EQ, Value: false},
		Blocked:     &basetypes.BoolVal{Op: cruder.EQ, Value: false},
		AvailableAt: &basetypes.Uint32Val{Op: cruder.LTE, Value: uint32(time.Now().Unix())},
	}, 0, 5) //nolint
	if err != nil {
		return nil, err
	}

	var account *payaccmwpb.Account

	for _, payment := range payments {
		info, err := payaccmwcli.GetAccount(ctx, payment.ID)
		if err != nil {
			return nil, err
		}

		if info.Locked || !info.Active || info.Blocked {
			continue
		}

		if info.AvailableAt > uint32(time.Now().Unix()) {
			continue
		}
		account = info
		break
	}

	if account == nil {
		return nil, nil
	}

	h.paymentAccount = account

	return account, nil
}

func (h *createHandler) withLockPaymentAccount(dispose *dtmcli.SagaDispose) {
	switch *h.mainPaymentType {
	case ordertypes.PaymentType_PayWithTransferOnly:
		fallthrough //nolint
	case ordertypes.PaymentType_PayWithTransferAndBalance:
		fallthrough //nolint
	case ordertypes.PaymentType_PayWithOffline:
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
}

func (h *createHandler) PeekAddress(ctx context.Context) error {
	account, err := h.peekAddress(ctx)
	if err != nil {
		return err
	}
	if account != nil {
		h.paymentAccount = account
		return nil
	}

	if err := h.createAddresses(ctx); err != nil {
		return err
	}

	account, err = h.peekAddress(ctx)
	if err != nil {
		return err
	}
	if account == nil {
		return fmt.Errorf("fail peek address")
	}

	h.paymentAccount = account
	return nil
}

func (h *createHandler) SetAddressBalance(ctx context.Context) error {
	balance, err := sphinxproxycli.GetBalance(ctx, &sphinxproxypb.GetBalanceRequest{
		Name:    h.paymentCoinName,
		Address: h.paymentAccount.Address,
	})
	if err != nil {
		return err
	}
	if balance == nil {
		return fmt.Errorf("invalid balance")
	}

	h.paymentAddressStartAmount, err = decimal.NewFromString(balance.BalanceStr)
	return err
}

func (h *createHandler) withUpdateStock(dispose *dtmcli.SagaDispose) {
	for _, goodReq := range h.Goods {
		req := &appgoodstockmwpb.StockReq{
			AppID:  h.AppID,
			GoodID: &goodReq.GoodID,
			Locked: &goodReq.Units,
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
}

func (h *createHandler) withUpdateBalance(dispose *dtmcli.SagaDispose) {
	if h.BalanceAmount != nil {
		return
	}
	req := &ledgermwpb.LedgerReq{
		AppID:      h.AppID,
		UserID:     h.UserID,
		CoinTypeID: &h.paymentAccount.CoinTypeID,
		Spendable:  h.BalanceAmount,
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

func (h *createHandler) orderReqs() []*ordermwpb.OrderReq {
	paymentAmount := h.paymentAmountCoin.String()
	startAmount := h.paymentAddressStartAmount.String()
	paymentTransferAmount := h.paymentTransferAmount.String()
	coinCurrency := h.coinCurrency.String()
	liveCurrency := h.liveCurrency.String()
	localCurrency := h.localCurrency.String()
	discountAmountCoin := h.discountAmountCoin.String()
	childPaymentType := ordertypes.PaymentType_PayWithParentOrder
	zeroAmount := "0"
	h.mainOrderID = uuid.NewString()

	orderReqs := []*ordermwpb.OrderReq{}
	for _, goodReq := range h.Goods {
		goodValue := h.goodValueCoins[goodReq.GoodID].String()
		goodValueUSD := h.goodValueUSDs[goodReq.GoodID].String()
		appgood := h.orderGood.appgoods[*h.AppID+goodReq.GoodID]
		good := h.orderGood.goods[goodReq.GoodID]
		goodDurationDays := uint32(good.DurationDays)
		startAt := h.startAts[*h.AppID+goodReq.GoodID]
		endAt := h.endAts[*h.AppID+goodReq.GoodID]

		logger.Sugar().Infow(
			"CreateOrder",
			"PaymentAmountCoin", h.paymentAmountCoin,
			"DiscountAmountCoin", h.discountAmountCoin,
			"BalanceAmount", h.BalanceAmount,
			"ReductionAmount", h.reductionAmount,
			"ReductionPercent", h.reductionPercent,
			"PaymentAddressStartAmount", h.paymentAddressStartAmount,
			"CoinCurrency", h.coinCurrency,
			"LiveCurrency", h.liveCurrency,
			"LocalCurrency", h.localCurrency,
		)
		orderReq := &ordermwpb.OrderReq{
			AppID:                h.AppID,
			UserID:               h.UserID,
			GoodID:               &goodReq.GoodID,
			AppGoodID:            &appgood.ID,
			Units:                &goodReq.Units,
			GoodValue:            &goodValue,
			GoodValueUSD:         &goodValueUSD,
			DurationDays:         &goodDurationDays,
			OrderType:            h.OrderType,
			InvestmentType:       h.InvestmentType,
			CoinTypeID:           &good.CoinTypeID,
			PaymentCoinTypeID:    h.PaymentCoinID,
			CoinUSDCurrency:      &coinCurrency,
			LiveCoinUSDCurrency:  &liveCurrency,
			LocalCoinUSDCurrency: &localCurrency,
			StartAt:              &startAt,
			EndAt:                &endAt,
		}

		if !goodReq.Parent && h.ParentOrderID == nil {
			id := uuid.NewString()
			// batch child order
			orderReq.ID = &id
			orderReq.ParentOrderID = &h.mainOrderID
			orderReq.PaymentAmount = &zeroAmount
			orderReq.DiscountAmount = &zeroAmount
			orderReq.PaymentType = &childPaymentType
			orderReq.TransferAmount = &zeroAmount
			orderReq.BalanceAmount = &zeroAmount
			h.IDs = append(h.IDs, id)
		} else {
			// parent order or single order
			orderReq.ID = &h.mainOrderID
			orderReq.ParentOrderID = h.ParentOrderID
			orderReq.PaymentAmount = &paymentAmount
			orderReq.DiscountAmount = &discountAmountCoin
			orderReq.CouponIDs = h.CouponIDs
			orderReq.PaymentType = h.mainPaymentType
			orderReq.TransferAmount = &paymentTransferAmount
			orderReq.BalanceAmount = h.BalanceAmount
			orderReq.PaymentAccountID = &h.paymentAccount.AccountID
			orderReq.PaymentStartAmount = &startAmount
			h.IDs = append(h.IDs, h.mainOrderID)
		}

		topmostGood := h.orderGood.topMostGoods[*h.AppID+goodReq.GoodID]
		if topmostGood != nil {
			orderReq.PromotionID = &topmostGood.TopMostID
		}
		orderReqs = append(orderReqs, orderReq)
	}
	return orderReqs
}

func (h *createHandler) withCreateOrder(dispose *dtmcli.SagaDispose, req *ordermwpb.OrderReq) {
	dispose.Add(
		ordermwsvcname.ServiceDomain,
		"order.middleware.order1.v1.Middleware/CreateOrder",
		"order.middleware.order1.v1.Middleware/DeleteOrder",
		&ordermwpb.CreateOrderRequest{
			Info: req,
		},
	)
}

func (h *createHandler) withCreateOrders(dispose *dtmcli.SagaDispose, reqs []*ordermwpb.OrderReq) {
	dispose.Add(
		ordermwsvcname.ServiceDomain,
		"order.middleware.order1.v1.Middleware/CreateOrders",
		"order.middleware.order1.v1.Middleware/DeleteOrders",
		&ordermwpb.CreateOrdersRequest{
			Infos: reqs,
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

	handler := &createHandler{
		Handler: h,
	}
	if err := handler.getUser(ctx); err != nil {
		return err
	}
	if err := handler.getPaymentCoin(ctx); err != nil {
		return err
	}
	if err := handler.getCoupons(ctx); err != nil {
		return err
	}
	if err := handler.validateDiscountCoupon(ctx); err != nil {
		return err
	}
	if err := handler.calculateDiscountCouponReduction(); err != nil {
		return err
	}
	if err := handler.calculateFixAmountCouponReduction(); err != nil {
		return err
	}
	if err := handler.getAppGood(ctx); err != nil {
		return err
	}
	if err := handler.calculateOrderUSDTPrice(ctx); err != nil {
		return err
	}
	if err := handler.getPaymentCoinCurrency(ctx); err != nil {
		return err
	}
	if err := handler.calculatePaymentCoinAmount(ctx); err != nil {
		return err
	}
	if err := handler.peekExistAddress(ctx); err != nil {
		if err := handler.peekNewAddress(ctx); err != nil {
			return err
		}
	}
	if err := accountlock.Lock(handler.paymentAccount.AccountID); err != nil {
		return err
	}
	if err := handler.recheckPaymentAccount(ctx); err != nil {
		return err
	}
	defer func() {
		_ = accountlock.Unlock(handler.paymentAccount.AccountID)
	}()

	id := uuid.NewString()
	if h.ID == nil {
		h.ID = &id
	}

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

//nolint:funlen,gocyclo
func (h *Handler) CreateOrders(ctx context.Context) (infos []*npool.Order, err error) {
	orderGood, err := h.ToOrderGoods(ctx)
	if err != nil {
		return nil, err
	}
	handler := &createHandler{
		Handler:   h,
		orderGood: orderGood,
	}
	if err := handler.validateInit(ctx); err != nil {
		return nil, err
	}

	if err := handler.checkGoodRequests(); err != nil {
		return nil, err
	}

	if err := handler.SetReduction(ctx); err != nil {
		return nil, err
	}

	if err := handler.SetPrice(ctx); err != nil {
		return nil, err
	}

	if err := handler.SetCurrency(ctx); err != nil {
		return nil, err
	}

	if err := handler.SetPaymentAmount(ctx); err != nil {
		return nil, err
	}

	if err := handler.SetPaymentType(ctx); err != nil {
		return nil, err
	}

	switch *handler.mainPaymentType {
	case ordertypes.PaymentType_PayWithTransferOnly:
		fallthrough //nolint
	case ordertypes.PaymentType_PayWithTransferAndBalance:
		fallthrough //nolint
	case ordertypes.PaymentType_PayWithOffline:
		for i := 0; i < 5; i++ {
			if err := handler.PeekAddress(ctx); err != nil {
				return nil, err
			}
			if err := accountlock.Lock(handler.paymentAccount.AccountID); err != nil {
				continue
			}
			break
		}
		defer func() {
			accountlock.Unlock(handler.paymentAccount.AccountID) //nolint
		}()
		if err := handler.SetAddressBalance(ctx); err != nil {
			return nil, err
		}
	}

	createReqs := handler.orderReqs()
	lockKey := fmt.Sprintf("%v:%v:%v:%v", basetypes.Prefix_PrefixCreateOrder, *h.AppID, *h.UserID, handler.mainOrderID)
	if err := redis2.TryLock(lockKey, 0); err != nil {
		return nil, err
	}
	defer func() {
		_ = redis2.Unlock(lockKey)
	}()

	sagaDispose := dtmcli.NewSagaDispose(dtmimp.TransOptions{
		WaitResult:     true,
		RequestTimeout: handler.RequestTimeoutSeconds,
	})

	handler.withUpdateStock(sagaDispose)
	handler.withUpdateBalance(sagaDispose)
	handler.withCreateOrders(sagaDispose, createReqs)
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
