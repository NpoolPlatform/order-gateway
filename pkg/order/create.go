package order

import (
	"context"
	"fmt"
	"time"

	"github.com/NpoolPlatform/go-service-framework/pkg/logger"
	"github.com/google/uuid"

	payaccmwcli "github.com/NpoolPlatform/account-middleware/pkg/client/payment"

	goodmwcli "github.com/NpoolPlatform/good-middleware/pkg/client/good"

	appcoinmwcli "github.com/NpoolPlatform/chain-middleware/pkg/client/app/coin"
	coininfocli "github.com/NpoolPlatform/chain-middleware/pkg/client/coin"
	currvalmwcli "github.com/NpoolPlatform/chain-middleware/pkg/client/coin/currency"
	appgoodmwcli "github.com/NpoolPlatform/good-middleware/pkg/client/appgood"
	sphinxproxycli "github.com/NpoolPlatform/sphinx-proxy/pkg/client"

	goodmwpb "github.com/NpoolPlatform/message/npool/good/mw/v1/good"
	paymentmwpb "github.com/NpoolPlatform/message/npool/order/mw/v1/payment"
	paymentmwcli "github.com/NpoolPlatform/order-middleware/pkg/client/payment"

	appcoinmwpb "github.com/NpoolPlatform/message/npool/chain/mw/v1/app/coin"
	appgoodpb "github.com/NpoolPlatform/message/npool/good/mgr/v1/appgood"

	payaccmwpb "github.com/NpoolPlatform/message/npool/account/mw/v1/payment"
	currvalmwpb "github.com/NpoolPlatform/message/npool/chain/mw/v1/coin/currency"
	sphinxproxypb "github.com/NpoolPlatform/message/npool/sphinxproxy"

	accountlock "github.com/NpoolPlatform/account-middleware/pkg/lock"
	cruder "github.com/NpoolPlatform/libent-cruder/pkg/cruder"

	ledgermgrcli "github.com/NpoolPlatform/ledger-middleware/pkg/client/ledger/v2"
	ledgermgrpb "github.com/NpoolPlatform/message/npool/ledger/mgr/v1/ledger/general"

	commonpb "github.com/NpoolPlatform/message/npool"
	basetypes "github.com/NpoolPlatform/message/npool/basetypes/v1"
	ordermwpb "github.com/NpoolPlatform/message/npool/order/mw/v1/order"
	ordermwcli "github.com/NpoolPlatform/order-middleware/pkg/client/order"

	allocatedmwcli "github.com/NpoolPlatform/inspire-middleware/pkg/client/coupon/allocated"
	inspiretypes "github.com/NpoolPlatform/message/npool/basetypes/inspire/v1"
	ordertypes "github.com/NpoolPlatform/message/npool/basetypes/order/v1"
	allocatedmwpb "github.com/NpoolPlatform/message/npool/inspire/mw/v1/coupon/allocated"
	npool "github.com/NpoolPlatform/message/npool/order/gw/v1/order"

	"github.com/shopspring/decimal"
)

type createHandler struct {
	*Handler
	GoodStartAt               uint32
	GoodDurationDays          uint32
	BalanceAmount             *string
	paymentCoinName           string
	paymentAmountUSD          decimal.Decimal
	paymentAmountCoin         decimal.Decimal
	paymentAddress            string
	paymentAddressStartAmount decimal.Decimal
	goodPaymentID             string
	paymentAccountID          string

	promotionID *string

	price decimal.Decimal

	liveCurrency  decimal.Decimal
	localCurrency decimal.Decimal
	coinCurrency  decimal.Decimal

	reductionAmount  decimal.Decimal
	reductionPercent decimal.Decimal

	start uint32
	end   uint32
}

func (h *createHandler) validateInit(ctx context.Context) error { //nolint
	if h.AppID == nil {
		return fmt.Errorf("invalid appid")
	}
	if h.UserID == nil {
		return fmt.Errorf("invalid userid")
	}
	if h.GoodID == nil {
		return fmt.Errorf("invalid goodid")
	}
	if h.Units == "" {
		return fmt.Errorf("invalid units")
	}
	units, err := decimal.NewFromString(h.Units)
	if err != nil {
		return err
	}
	if units.Cmp(decimal.NewFromInt32(0)) <= 0 {
		return fmt.Errorf("units is 0")
	}
	if h.PaymentCoinID == nil {
		return fmt.Errorf("invalid paymentcoinid")
	}
	if h.CouponIDs != nil {
		for _, id := range h.CouponIDs {
			if _, err := uuid.Parse(id); err != nil {
				logger.Sugar().Errorw("CreateOrder", "error", err)
				return fmt.Errorf("invalid couponids")
			}
		}
	}

	good, err := goodmwcli.GetGood(ctx, *h.GoodID)
	if err != nil {
		return err
	}
	if good == nil {
		return fmt.Errorf("invalid good")
	}

	ag, err := appgoodmwcli.GetGoodOnly(ctx, &appgoodpb.Conds{
		AppID: &commonpb.StringVal{
			Op:    cruder.EQ,
			Value: *h.AppID,
		},
		GoodID: &commonpb.StringVal{
			Op:    cruder.EQ,
			Value: *h.GoodID,
		},
	})
	if err != nil {
		return err
	}
	if ag == nil {
		return fmt.Errorf("invalid app good")
	}

	h.GoodStartAt = ag.ServiceStartAt

	if ag.ServiceStartAt == 0 {
		h.GoodStartAt = good.StartAt
	}

	h.GoodDurationDays = uint32(good.DurationDays)

	gcoin, err := coininfocli.GetCoin(ctx, good.CoinTypeID)
	if err != nil {
		return err
	}
	if gcoin == nil {
		return fmt.Errorf("invalid good coin")
	}

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
	if coin.ENV != gcoin.ENV {
		return fmt.Errorf("good coin mismatch payment coin")
	}

	h.paymentCoinName = coin.Name

	if h.ParentOrderID != nil {
		order, err := ordermwcli.GetOrder(ctx, *h.ParentOrderID)
		if err != nil {
			return err
		}
		if order == nil {
			return fmt.Errorf("invalid parent order")
		}
	}

	if !ag.Online {
		return fmt.Errorf("good offline")
	}

	agPrice, err := decimal.NewFromString(ag.Price)
	if err != nil {
		return err
	}
	if agPrice.IntPart() <= 0 {
		return fmt.Errorf("invalid good price")
	}
	price, err := decimal.NewFromString(good.Price)
	if err != nil {
		return err
	}
	if agPrice.Cmp(price) < 0 {
		return fmt.Errorf("invalid app good price")
	}

	const maxUnpaidOrders = 5

	payments, _, err := paymentmwcli.GetPayments(ctx, &paymentmwpb.Conds{
		AppID: &basetypes.StringVal{
			Op:    cruder.EQ,
			Value: *h.AppID,
		},
		UserID: &basetypes.StringVal{
			Op:    cruder.EQ,
			Value: *h.UserID,
		},
		State: &basetypes.Uint32Val{
			Op:    cruder.EQ,
			Value: uint32(ordertypes.PaymentState_PaymentStateWait),
		},
	}, 0, maxUnpaidOrders)
	if err != nil {
		return err
	}
	if len(payments) >= maxUnpaidOrders && *h.OrderType == ordertypes.OrderType_Normal {
		return fmt.Errorf("too many unpaid orders")
	}

	switch *h.OrderType {
	case ordertypes.OrderType_Normal:
	case ordertypes.OrderType_Offline:
	case ordertypes.OrderType_Airdrop:
	default:
		return fmt.Errorf("invalid order type")
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
	good, err := goodmwcli.GetGood(ctx, *h.GoodID)
	if err != nil {
		return err
	}
	ag, err := appgoodmwcli.GetGoodOnly(ctx, &appgoodpb.Conds{
		AppID: &commonpb.StringVal{
			Op:    cruder.EQ,
			Value: *h.AppID,
		},
		GoodID: &commonpb.StringVal{
			Op:    cruder.EQ,
			Value: *h.GoodID,
		},
	})
	if err != nil {
		return err
	}
	if ag == nil {
		return fmt.Errorf("invalid app good")
	}

	if !ag.Online {
		return fmt.Errorf("good offline")
	}
	agPrice, err := decimal.NewFromString(ag.Price)
	if err != nil {
		return err
	}
	if agPrice.Cmp(decimal.NewFromInt(0)) <= 0 {
		return fmt.Errorf("invalid good price")
	}
	price, err := decimal.NewFromString(good.Price)
	if err != nil {
		return err
	}
	if agPrice.Cmp(price) < 0 {
		return fmt.Errorf("invalid app good price")
	}

	h.price, err = decimal.NewFromString(ag.Price)
	if err != nil {
		return err
	}

	if ag.PromotionPrice != nil {
		promotionPrice, err := decimal.NewFromString(ag.GetPromotionPrice())
		if err != nil {
			return err
		}
		if promotionPrice.Cmp(decimal.NewFromInt(0)) <= 0 {
			return fmt.Errorf("invalid price")
		}
		h.price = promotionPrice
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
		AppID: &basetypes.StringVal{
			Op:    cruder.EQ,
			Value: *h.AppID,
		},
		CoinTypeID: &basetypes.StringVal{
			Op:    cruder.EQ,
			Value: *h.PaymentCoinID,
		},
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
	// TODO: also add sub good order payment amount
	units, err := decimal.NewFromString(h.Units)
	if err != nil {
		return err
	}
	h.paymentAmountUSD = h.price.Mul(units)
	logger.Sugar().Infow(
		"CreateOrder",
		"PaymentAmountUSD", h.paymentAmountUSD,
		"ReductionAmount", h.reductionAmount,
		"ReductionPercent", h.reductionPercent,
	)

	h.paymentAmountUSD = h.price.Mul(units).
		Sub(h.paymentAmountUSD.
			Mul(h.reductionPercent).
			Div(decimal.NewFromInt(100))) //nolint

	h.paymentAmountUSD = h.paymentAmountUSD.Sub(h.reductionAmount)

	if h.paymentAmountUSD.Cmp(decimal.NewFromInt(0)) < 0 {
		h.paymentAmountUSD = decimal.NewFromInt(0)
	}

	const accuracy = 1000000

	h.paymentAmountCoin = h.paymentAmountUSD.Div(h.coinCurrency)
	h.paymentAmountCoin = h.paymentAmountCoin.Mul(decimal.NewFromInt(accuracy))
	h.paymentAmountCoin = h.paymentAmountCoin.Ceil()
	h.paymentAmountCoin = h.paymentAmountCoin.Div(decimal.NewFromInt(accuracy))

	if h.BalanceAmount != nil {
		amount, err := decimal.NewFromString(*h.BalanceAmount)
		if err != nil {
			return err
		}
		if amount.Cmp(h.paymentAmountCoin) > 0 {
			amount = h.paymentAmountCoin
			amountStr := amount.String()
			h.BalanceAmount = &amountStr
		}
		h.paymentAmountCoin = h.paymentAmountCoin.Sub(amount)
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
		CoinTypeID: &basetypes.StringVal{
			Op:    cruder.EQ,
			Value: *h.PaymentCoinID,
		},
		Active: &basetypes.BoolVal{
			Op:    cruder.EQ,
			Value: true,
		},
		Locked: &basetypes.BoolVal{
			Op:    cruder.EQ,
			Value: false,
		},
		Blocked: &basetypes.BoolVal{
			Op:    cruder.EQ,
			Value: false,
		},
		AvailableAt: &basetypes.Uint32Val{
			Op:    cruder.LTE,
			Value: uint32(time.Now().Unix()),
		},
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

func (h *createHandler) createSubOrder(ctx context.Context) error { //nolint
	// TODO: create sub order according to good's must select sub good
	return nil
}

func (h *createHandler) LockStock(ctx context.Context) error {
	_, err := goodmwcli.UpdateGood(ctx, &goodmwpb.GoodReq{
		ID:     h.GoodID,
		Locked: &h.Units,
	})
	if err != nil {
		return err
	}
	return nil
}

func (h *createHandler) ReleaseStock(ctx context.Context) error {
	units, err := decimal.NewFromString(h.Units)
	if err != nil {
		return err
	}
	unitsStr := units.Neg().String()
	_, err = goodmwcli.UpdateGood(ctx, &goodmwpb.GoodReq{
		ID:     h.GoodID,
		Locked: &unitsStr,
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

	general, err := ledgermgrcli.GetGeneralOnly(ctx, &ledgermgrpb.Conds{
		AppID: &commonpb.StringVal{
			Op:    cruder.EQ,
			Value: *h.AppID,
		},
		UserID: &commonpb.StringVal{
			Op:    cruder.EQ,
			Value: *h.UserID,
		},
		CoinTypeID: &commonpb.StringVal{
			Op:    cruder.EQ,
			Value: *h.PaymentCoinID,
		},
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

	spendableMinus := fmt.Sprintf("-%v", *h.BalanceAmount)

	_, err = ledgermgrcli.AddGeneral(ctx, &ledgermgrpb.GeneralReq{
		ID:         &general.ID,
		AppID:      &general.AppID,
		UserID:     &general.UserID,
		CoinTypeID: &general.CoinTypeID,
		Locked:     h.BalanceAmount,
		Spendable:  &spendableMinus,
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

	general, err := ledgermgrcli.GetGeneralOnly(ctx, &ledgermgrpb.Conds{
		AppID: &commonpb.StringVal{
			Op:    cruder.EQ,
			Value: *h.AppID,
		},
		UserID: &commonpb.StringVal{
			Op:    cruder.EQ,
			Value: *h.UserID,
		},
		CoinTypeID: &commonpb.StringVal{
			Op:    cruder.EQ,
			Value: *h.PaymentCoinID,
		},
	})
	if err != nil {
		return err
	}
	if general == nil {
		return fmt.Errorf("insufficuent funds")
	}

	lockedMinus := fmt.Sprintf("-%v", h.BalanceAmount)

	_, err = ledgermgrcli.AddGeneral(ctx, &ledgermgrpb.GeneralReq{
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
	switch *h.OrderType {
	case ordertypes.OrderType_Normal:
	case ordertypes.OrderType_Offline:
	case ordertypes.OrderType_Airdrop:
	default:
		return nil, fmt.Errorf("invalid order type")
	}

	paymentAmount := h.paymentAmountCoin.String()
	startAmount := h.paymentAddressStartAmount.String()
	coinCurrency := h.coinCurrency.String()
	liveCurrency := h.liveCurrency.String()
	localCurrency := h.localCurrency.String()

	// Top order never pay with parent, only sub order may

	h.start = uint32(tomorrowStart().Unix())
	if h.GoodStartAt > h.start {
		h.start = h.GoodStartAt
	}
	const secondsPerDay = 24 * 60 * 60
	h.end = h.start + h.GoodDurationDays*secondsPerDay

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
		AppID:                     h.AppID,
		UserID:                    h.UserID,
		GoodID:                    h.GoodID,
		Units:                     &h.Units,
		OrderType:                 h.OrderType,
		ParentOrderID:             h.ParentOrderID,
		PaymentCoinID:             h.PaymentCoinID,
		PayWithBalanceAmount:      h.BalanceAmount,
		PaymentAccountID:          &h.paymentAccountID,
		PaymentAmount:             &paymentAmount,
		PaymentAccountStartAmount: &startAmount,
		PaymentCoinUSDCurrency:    &coinCurrency,
		PaymentLiveUSDCurrency:    &liveCurrency,
		PaymentLocalUSDCurrency:   &localCurrency,
		FixAmountID:               h.FixAmountID,
		DiscountID:                h.DiscountID,
		SpecialOfferID:            h.SpecialOfferID,
		Start:                     &h.start,
		End:                       &h.end,
		PromotionID:               h.promotionID,
		CouponIDs:                 h.CouponIDs,
	})
	if err != nil {
		return nil, err
	}
	h.ID = &ord.ID

	return h.GetOrder(ctx)
}

func (h *Handler) CreateOrder(ctx context.Context) (info *npool.Order, err error) {
	handler := &createHandler{
		Handler: h,
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
