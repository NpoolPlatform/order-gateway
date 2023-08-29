package order

import (
	"context"
	"fmt"
	"time"

	dtmcli "github.com/NpoolPlatform/dtm-cluster/pkg/dtm"
	"github.com/NpoolPlatform/go-service-framework/pkg/logger"
	"github.com/dtm-labs/dtm/client/dtmcli/dtmimp"

	cruder "github.com/NpoolPlatform/libent-cruder/pkg/cruder"
	ordermwsvcname "github.com/NpoolPlatform/order-middleware/pkg/servicename"

	appgoodstockmwcli "github.com/NpoolPlatform/good-middleware/pkg/client/app/good/stock"
	appgoodstockmwpb "github.com/NpoolPlatform/message/npool/good/mw/v1/app/good/stock"

	appcoinmwcli "github.com/NpoolPlatform/chain-middleware/pkg/client/app/coin"
	coininfocli "github.com/NpoolPlatform/chain-middleware/pkg/client/coin"
	currvalmwcli "github.com/NpoolPlatform/chain-middleware/pkg/client/coin/currency"
	appcoinmwpb "github.com/NpoolPlatform/message/npool/chain/mw/v1/app/coin"
	currvalmwpb "github.com/NpoolPlatform/message/npool/chain/mw/v1/coin/currency"

	sphinxproxypb "github.com/NpoolPlatform/message/npool/sphinxproxy"
	sphinxproxycli "github.com/NpoolPlatform/sphinx-proxy/pkg/client"

	payaccmwcli "github.com/NpoolPlatform/account-middleware/pkg/client/payment"
	accountlock "github.com/NpoolPlatform/account-middleware/pkg/lock"
	payaccmwpb "github.com/NpoolPlatform/message/npool/account/mw/v1/payment"

	ledgermwcli "github.com/NpoolPlatform/ledger-middleware/pkg/client/ledger"
	ledgermwpb "github.com/NpoolPlatform/message/npool/ledger/mw/v2/ledger"

	ordermwpb "github.com/NpoolPlatform/message/npool/order/mw/v1/order"
	ordermwcli "github.com/NpoolPlatform/order-middleware/pkg/client/order"

	allocatedmwcli "github.com/NpoolPlatform/inspire-middleware/pkg/client/coupon/allocated"
	allocatedmwpb "github.com/NpoolPlatform/message/npool/inspire/mw/v1/coupon/allocated"

	inspiretypes "github.com/NpoolPlatform/message/npool/basetypes/inspire/v1"
	ordertypes "github.com/NpoolPlatform/message/npool/basetypes/order/v1"
	basetypes "github.com/NpoolPlatform/message/npool/basetypes/v1"
	npool "github.com/NpoolPlatform/message/npool/order/gw/v1/order"

	"github.com/shopspring/decimal"
)

type createHandler struct {
	*Handler
	orderGood *OrderGood

	discountAmount    decimal.Decimal
	paymentAmountUSD  decimal.Decimal
	paymentAmountCoin decimal.Decimal

	promotionID               string
	paymentCoinName           string
	paymentAddress            string
	paymentAddressStartAmount decimal.Decimal
	goodPaymentID             string
	paymentAccountID          string

	goodPrices            map[string]decimal.Decimal
	goodValues            map[string]decimal.Decimal
	goodValueUSDs         map[string]decimal.Decimal
	paymentType           ordertypes.PaymentType
	paymentTransferAmount decimal.Decimal

	liveCurrency  decimal.Decimal
	localCurrency decimal.Decimal
	coinCurrency  decimal.Decimal

	reductionAmount  decimal.Decimal
	reductionPercent decimal.Decimal

	startAts map[string]uint32
	endAts   map[string]uint32
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

	h.paymentCoinName = coin.Name

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

	exist, err := ordermwcli.ExistOrderConds(ctx, &ordermwpb.Conds{
		CouponIDs: &basetypes.StringSliceVal{Op: cruder.EQ, Value: h.CouponIDs},
	})
	if err != nil {
		return err
	}
	if exist {
		return fmt.Errorf("coupon already used")
	}

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
			percent, err := decimal.NewFromString(coup.Denomination)
			if err != nil {
				return err
			}
			if percent.Cmp(decimal.NewFromInt(100)) >= 0 {
				return fmt.Errorf("invalid discount")
			}
			h.reductionPercent = percent
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

func (h *createHandler) SetPaymentAmount(ctx context.Context) error {
	const accuracy = 1000000
	totalPaymentAmountUSD := decimal.NewFromInt(0)
	for _, goodReq := range h.Goods {
		price := h.goodPrices[*h.AppID+goodReq.GoodID]
		units, err := decimal.NewFromString(goodReq.Units)
		if err != nil {
			return err
		}
		paymentAmountUSD := price.Mul(units)
		h.goodValueUSDs[goodReq.GoodID] = paymentAmountUSD
		totalPaymentAmountUSD = totalPaymentAmountUSD.Add(paymentAmountUSD)

		goodValue := paymentAmountUSD.Div(h.coinCurrency)
		goodValue = goodValue.Mul(decimal.NewFromInt(accuracy))
		goodValue = goodValue.Ceil()
		goodValue = goodValue.Div(decimal.NewFromInt(accuracy))
		h.goodValues[goodReq.GoodID] = goodValue
	}

	h.paymentAmountUSD = totalPaymentAmountUSD
	logger.Sugar().Infow(
		"CreateOrder",
		"PaymentAmountUSD", h.paymentAmountUSD,
		"ReductionAmount", h.reductionAmount,
		"ReductionPercent", h.reductionPercent,
	)

	h.discountAmount = h.paymentAmountUSD.
		Mul(h.reductionPercent).
		Div(decimal.NewFromInt(100)). //nolint
		Add(h.reductionAmount)

	h.paymentAmountUSD = totalPaymentAmountUSD.Sub(h.discountAmount)

	if h.paymentAmountUSD.Cmp(decimal.NewFromInt(0)) < 0 {
		h.paymentAmountUSD = decimal.NewFromInt(0)
	}

	h.paymentAmountCoin = h.paymentAmountUSD.Div(h.coinCurrency)
	h.paymentAmountCoin = h.paymentAmountCoin.Mul(decimal.NewFromInt(accuracy))
	h.paymentAmountCoin = h.paymentAmountCoin.Ceil()
	h.paymentAmountCoin = h.paymentAmountCoin.Div(decimal.NewFromInt(accuracy))

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
		if err := accountlock.Lock(payment.AccountID); err != nil {
			logger.Sugar().Infow("peekAddress", "payment", payment.Address, "ID", payment.ID, "error", err)
			continue
		}

		info, err := payaccmwcli.GetAccount(ctx, payment.ID)
		if err != nil {
			accountlock.Unlock(payment.AccountID) //nolint
			logger.Sugar().Infow("peekAddress", "payment", payment.Address, "ID", payment.ID, "error", err)
			return nil, err
		}

		if info.Locked || !info.Active || info.Blocked {
			accountlock.Unlock(payment.AccountID) //nolint
			continue
		}

		if info.AvailableAt > uint32(time.Now().Unix()) {
			accountlock.Unlock(payment.AccountID) //nolint
			continue
		}

		locked := true
		lockedBy := basetypes.AccountLockedBy_Payment

		info, err = payaccmwcli.UpdateAccount(ctx, &payaccmwpb.AccountReq{
			ID:       &payment.ID,
			Locked:   &locked,
			LockedBy: &lockedBy,
		})
		if err != nil {
			accountlock.Unlock(payment.AccountID) //nolint
			logger.Sugar().Infow("peekAddress", "payment", info.Address, "error", err)
			return nil, err
		}

		account = info
		accountlock.Unlock(payment.AccountID) //nolint
		break
	}

	if account == nil {
		return nil, nil
	}

	h.goodPaymentID = account.ID

	return account, nil
}

func (h *createHandler) PeekAddress(ctx context.Context) error {
	account, err := h.peekAddress(ctx)
	if err != nil {
		return err
	}
	if account != nil {
		h.paymentAddress = account.Address
		h.paymentAccountID = account.AccountID
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

	h.paymentAddress = account.Address
	h.paymentAccountID = account.AccountID

	return nil
}

func (h *createHandler) ReleaseAddress(ctx context.Context) error {
	if err := accountlock.Lock(h.paymentAccountID); err != nil {
		return err
	}

	locked := false

	_, err := payaccmwcli.UpdateAccount(ctx, &payaccmwpb.AccountReq{
		ID:     &h.goodPaymentID,
		Locked: &locked,
	})

	accountlock.Unlock(h.paymentAccountID) //nolint
	return err
}

func (h *createHandler) SetBalance(ctx context.Context) error {
	balance, err := sphinxproxycli.GetBalance(ctx, &sphinxproxypb.GetBalanceRequest{
		Name:    h.paymentCoinName,
		Address: h.paymentAddress,
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
			"good.middleware.app.good1.stock.v1.Middleware/SubStock",
			"good.middleware.app.good1.stock.v1.Middleware/AddStock",
			&appgoodstockmwpb.AddStockRequest{
				Info: req,
			},
		)
	}
}

func (h *createHandler) LockStock(ctx context.Context) error {
	_, err := appgoodstockmwcli.AddStock(ctx, &appgoodstockmwpb.StockReq{
		AppID:  h.AppID,
		GoodID: h.GoodID,
		Locked: &h.Units,
	})
	if err != nil {
		return err
	}
	return nil
}

func (h *createHandler) ReleaseStock(ctx context.Context) error {
	_, err := appgoodstockmwcli.SubStock(ctx, &appgoodstockmwpb.StockReq{
		AppID:  h.AppID,
		GoodID: h.GoodID,
		Locked: &h.Units,
	})
	if err != nil {
		return err
	}
	return nil
}

func (h *createHandler) LockBalance(ctx context.Context) error {
	if h.BalanceAmount == nil {
		return nil
	}

	ba, err := decimal.NewFromString(*h.BalanceAmount)
	if err != nil {
		return err
	}

	if ba.Cmp(decimal.NewFromInt(0)) <= 0 {
		return nil
	}

	general, err := ledgermwcli.GetLedgerOnly(ctx, &ledgermwpb.Conds{
		AppID:      &basetypes.StringVal{Op: cruder.EQ, Value: *h.AppID},
		UserID:     &basetypes.StringVal{Op: cruder.EQ, Value: *h.UserID},
		CoinTypeID: &basetypes.StringVal{Op: cruder.EQ, Value: *h.PaymentCoinID},
	})
	if err != nil {
		return err
	}
	if general == nil {
		return fmt.Errorf("insufficient balance")
	}

	spendable, err := decimal.NewFromString(general.Spendable)
	if err != nil {
		return err
	}

	if spendable.Cmp(ba) < 0 {
		return fmt.Errorf("insufficient balance")
	}

	_, err = ledgermwcli.SubBalance(ctx, &ledgermwpb.LedgerReq{
		ID:         &general.ID,
		AppID:      &general.AppID,
		UserID:     &general.UserID,
		CoinTypeID: &general.CoinTypeID,
		Locked:     h.BalanceAmount,
		Spendable:  h.BalanceAmount,
	})

	return err
}

func (h *createHandler) ReleaseBalance(ctx context.Context) error {
	if h.BalanceAmount == nil {
		return nil
	}

	ba, err := decimal.NewFromString(*h.BalanceAmount)
	if err != nil {
		return err
	}

	if ba.Cmp(decimal.NewFromInt(0)) <= 0 {
		return nil
	}

	general, err := ledgermwcli.GetLedgerOnly(ctx, &ledgermwpb.Conds{
		AppID:      &basetypes.StringVal{Op: cruder.EQ, Value: *h.AppID},
		UserID:     &basetypes.StringVal{Op: cruder.EQ, Value: *h.UserID},
		CoinTypeID: &basetypes.StringVal{Op: cruder.EQ, Value: *h.PaymentCoinID},
	})
	if err != nil {
		return err
	}
	if general == nil {
		return fmt.Errorf("insufficuent funds")
	}

	lockedMinus := fmt.Sprintf("-%v", h.BalanceAmount)

	_, err = ledgermwcli.AddBalance(ctx, &ledgermwpb.LedgerReq{
		ID:         &general.ID,
		AppID:      &general.AppID,
		UserID:     &general.UserID,
		CoinTypeID: &general.CoinTypeID,
		Locked:     &lockedMinus,
		Spendable:  h.BalanceAmount,
	})

	return err
}

func tomorrowStart() time.Time {
	now := time.Now()
	y, m, d := now.Date()
	return time.Date(y, m, d+1, 0, 0, 0, 0, now.Location())
}

func (h *createHandler) create(ctx context.Context) (*npool.Order, error) {
	paymentAmount := h.paymentAmountCoin.String()
	startAmount := h.paymentAddressStartAmount.String()
	coinCurrency := h.coinCurrency.String()
	liveCurrency := h.liveCurrency.String()
	localCurrency := h.localCurrency.String()
	goodValue := h.goodValues[*h.GoodID].String()
	goodValueUSD := h.goodValueUSDs[*h.GoodID].String()
	discountAmount := h.discountAmount.String()
	paymentTransferAmount := h.paymentTransferAmount.String()

	// Top order never pay with parent, only sub order may
	topmostGood := h.orderGood.topMostGoods[*h.AppID+*h.GoodID]
	if topmostGood != nil {
		h.promotionID = topmostGood.TopMostID
	}

	appgood := h.orderGood.appgoods[*h.AppID+*h.GoodID]
	good := h.orderGood.goods[*h.GoodID]
	goodDurationDays := uint32(good.DurationDays)
	startAt := h.startAts[*h.AppID+*h.GoodID]
	endAt := h.endAts[*h.AppID+*h.GoodID]

	logger.Sugar().Infow(
		"CreateOrder",
		"PaymentAmountUSD", h.paymentAmountUSD,
		"PaymentAmountCoin", h.paymentAmountCoin,
		"BalanceAmount", h.BalanceAmount,
		"ReductionAmount", h.reductionAmount,
		"ReductionPercent", h.reductionPercent,
		"PaymentAddressStartAmount", h.paymentAddressStartAmount,
		"CoinCurrency", h.coinCurrency,
		"LiveCurrency", h.liveCurrency,
		"LocalCurrency", h.localCurrency,
	)

	ord, err := ordermwcli.CreateOrder(ctx, &ordermwpb.OrderReq{
		AppID:                h.AppID,
		UserID:               h.UserID,
		GoodID:               h.GoodID,
		AppGoodID:            &appgood.ID,
		ParentOrderID:        h.ParentOrderID,
		Units:                &h.Units,
		GoodValue:            &goodValue,
		GoodValueUSD:         &goodValueUSD,
		PaymentAmount:        &paymentAmount,
		DiscountAmount:       &discountAmount,
		PromotionID:          &h.promotionID,
		DurationDays:         &goodDurationDays,
		OrderType:            h.OrderType,
		InvestmentType:       h.InvestmentType,
		CouponIDs:            h.CouponIDs,
		PaymentType:          &h.paymentType,
		PaymentAccountID:     &h.paymentAccountID,
		CoinTypeID:           &good.CoinTypeID,
		PaymentCoinTypeID:    h.PaymentCoinID,
		PaymentStartAmount:   &startAmount,
		TransferAmount:       &paymentTransferAmount,
		BalanceAmount:        h.BalanceAmount,
		CoinUSDCurrency:      &coinCurrency,
		LiveCoinUSDCurrency:  &liveCurrency,
		LocalCoinUSDCurrency: &localCurrency,
		StartAt:              &startAt,
		EndAt:                &endAt,
	})
	if err != nil {
		return nil, err
	}
	h.ID = &ord.ID

	return h.GetOrder(ctx)
}

func (h *Handler) CreateOrder(ctx context.Context) (info *npool.Order, err error) {
	orderGood, err := h.ToOrderGood(ctx)
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

	if err := handler.PeekAddress(ctx); err != nil {
		return nil, err
	}

	if err := handler.SetBalance(ctx); err != nil {
		_ = handler.ReleaseAddress(ctx)
		return nil, err
	}

	if err := handler.LockStock(ctx); err != nil {
		_ = handler.ReleaseAddress(ctx)
		return nil, err
	}

	if err := handler.LockBalance(ctx); err != nil {
		_ = handler.ReleaseAddress(ctx)
		_ = handler.ReleaseStock(ctx)
		return nil, err
	}

	ord, err := handler.create(ctx)
	if err != nil {
		_ = handler.ReleaseAddress(ctx)
		_ = handler.ReleaseStock(ctx)
		_ = handler.ReleaseBalance(ctx)
		return nil, err
	}

	return ord, nil
}

func (h *createHandler) creates(ctx context.Context) ([]*npool.Order, error) {
	paymentAmount := h.paymentAmountCoin.String()
	startAmount := h.paymentAddressStartAmount.String()
	paymentTransferAmount := h.paymentTransferAmount.String()
	coinCurrency := h.coinCurrency.String()
	liveCurrency := h.liveCurrency.String()
	localCurrency := h.localCurrency.String()
	discountAmount := h.discountAmount.String()

	orderReqs := []*ordermwpb.OrderReq{}
	for _, goodReq := range h.Goods {
		goodValue := h.goodValues[goodReq.GoodID].String()
		goodValueUSD := h.goodValueUSDs[goodReq.GoodID].String()
		// Top order never pay with parent, only sub order may
		topmostGood := h.orderGood.topMostGoods[*h.AppID+goodReq.GoodID]
		if topmostGood != nil {
			h.promotionID = topmostGood.TopMostID
		}

		appgood := h.orderGood.appgoods[*h.AppID+goodReq.GoodID]
		good := h.orderGood.goods[goodReq.GoodID]
		goodDurationDays := uint32(good.DurationDays)
		startAt := h.startAts[*h.AppID+goodReq.GoodID]
		endAt := h.endAts[*h.AppID+goodReq.GoodID]

		logger.Sugar().Infow(
			"CreateOrder",
			"PaymentAmountUSD", h.paymentAmountUSD,
			"PaymentAmountCoin", h.paymentAmountCoin,
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
			GoodID:               h.GoodID,
			AppGoodID:            &appgood.ID,
			ParentOrderID:        h.ParentOrderID,
			Units:                &h.Units,
			GoodValue:            &goodValue,
			GoodValueUSD:         &goodValueUSD,
			PaymentAmount:        &paymentAmount,
			DiscountAmount:       &discountAmount,
			PromotionID:          &h.promotionID,
			DurationDays:         &goodDurationDays,
			OrderType:            h.OrderType,
			InvestmentType:       h.InvestmentType,
			CouponIDs:            h.CouponIDs,
			PaymentType:          &h.paymentType,
			CoinTypeID:           &good.CoinTypeID,
			PaymentCoinTypeID:    h.PaymentCoinID,
			TransferAmount:       &paymentTransferAmount,
			BalanceAmount:        h.BalanceAmount,
			CoinUSDCurrency:      &coinCurrency,
			LiveCoinUSDCurrency:  &liveCurrency,
			LocalCoinUSDCurrency: &localCurrency,
			StartAt:              &startAt,
			EndAt:                &endAt,
		}
		if goodReq.Parent {
			orderReq.PaymentAccountID = &h.paymentAccountID
			orderReq.PaymentStartAmount = &startAmount
		}
		orderReqs = append(orderReqs, orderReq)
	}

	orders, err := ordermwcli.CreateOrders(ctx, orderReqs)
	if err != nil {
		return nil, err
	}
	for key := range orders {
		h.IDs = append(h.IDs, orders[key].ID)
	}

	infos, _, err := h.GetOrders(ctx)
	if err != nil {
		return nil, err
	}
	return infos, nil
}

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

	if err := handler.PeekAddress(ctx); err != nil {
		return nil, err
	}

	if err := handler.SetBalance(ctx); err != nil {
		_ = handler.ReleaseAddress(ctx)
		return nil, err
	}

	sagaDispose := dtmcli.NewSagaDispose(dtmimp.TransOptions{
		WaitResult:     true,
		RequestTimeout: handler.RequestTimeoutSeconds,
	})

	handler.withUpdateStock(sagaDispose)

	if err := dtmcli.WithSaga(ctx, sagaDispose); err != nil {
		_ = handler.ReleaseAddress(ctx)
		return nil, err
	}

	if err := handler.LockBalance(ctx); err != nil {
		_ = handler.ReleaseAddress(ctx)
		return nil, err
	}

	orders, err := handler.creates(ctx)
	if err != nil {
		_ = handler.ReleaseAddress(ctx)
		_ = handler.ReleaseBalance(ctx)
		return nil, err
	}

	return orders, nil
}
