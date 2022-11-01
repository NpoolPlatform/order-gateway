package order

import (
	"context"
	"fmt"

	"time"

	"github.com/NpoolPlatform/go-service-framework/pkg/logger"

	npool "github.com/NpoolPlatform/message/npool/order/gw/v1/order"

	appcli "github.com/NpoolPlatform/appuser-middleware/pkg/client/app"
	usercli "github.com/NpoolPlatform/appuser-middleware/pkg/client/user"
	billingcli "github.com/NpoolPlatform/cloud-hashing-billing/pkg/client"
	ordercli "github.com/NpoolPlatform/cloud-hashing-order/pkg/client"
	goodcli "github.com/NpoolPlatform/good-middleware/pkg/client/good"

	appgoodcli "github.com/NpoolPlatform/good-middleware/pkg/client/appgood"
	couponcli "github.com/NpoolPlatform/inspire-middleware/pkg/client/coupon"
	oraclecli "github.com/NpoolPlatform/oracle-manager/pkg/client"
	ordermwcli "github.com/NpoolPlatform/order-middleware/pkg/client/order"
	coininfocli "github.com/NpoolPlatform/sphinx-coininfo/pkg/client"
	sphinxproxycli "github.com/NpoolPlatform/sphinx-proxy/pkg/client"

	billingconst "github.com/NpoolPlatform/cloud-hashing-billing/pkg/const"
	orderconst "github.com/NpoolPlatform/cloud-hashing-order/pkg/const"
	oracleconst "github.com/NpoolPlatform/oracle-manager/pkg/const"

	goodmwpb "github.com/NpoolPlatform/message/npool/good/mw/v1/good"

	appgoodpb "github.com/NpoolPlatform/message/npool/good/mgr/v1/appgood"

	billingpb "github.com/NpoolPlatform/message/npool/cloud-hashing-billing"
	couponpb "github.com/NpoolPlatform/message/npool/inspire/mw/v1/inspire/coupon"
	ordermgrpb "github.com/NpoolPlatform/message/npool/order/mgr/v1/order/order"
	ordermwpb "github.com/NpoolPlatform/message/npool/order/mw/v1/order"
	sphinxproxypb "github.com/NpoolPlatform/message/npool/sphinxproxy"

	cruder "github.com/NpoolPlatform/libent-cruder/pkg/cruder"
	currency "github.com/NpoolPlatform/oracle-manager/pkg/middleware/currency"
	accountlock "github.com/NpoolPlatform/staker-manager/pkg/middleware/account"

	ledgermgrcli "github.com/NpoolPlatform/ledger-manager/pkg/client/general"
	ledgermgrpb "github.com/NpoolPlatform/message/npool/ledger/mgr/v1/ledger/general"

	commonpb "github.com/NpoolPlatform/message/npool"

	"google.golang.org/protobuf/types/known/structpb"

	"github.com/shopspring/decimal"
)

type OrderCreate struct {
	AppID            string
	UserID           string
	GoodID           string
	GoodStartAt      uint32
	GoodDurationDays uint32
	Units            uint32

	PaymentCoinID string
	BalanceAmount *string

	ParentOrderID *string
	OrderType     ordermgrpb.OrderType

	FixAmountID    *string
	DiscountID     *string
	SpecialOfferID *string

	goodPaymentID             string
	paymentCoinName           string
	paymentAmountUSD          decimal.Decimal
	paymentAmountCoin         decimal.Decimal
	paymentAddress            string
	paymentAddressStartAmount decimal.Decimal
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

func (o *OrderCreate) ValidateInit(ctx context.Context) error { //nolint
	app, err := appcli.GetApp(ctx, o.AppID)
	if err != nil {
		return err
	}
	if app == nil {
		return fmt.Errorf("invalid app")
	}

	user, err := usercli.GetUser(ctx, o.AppID, o.UserID)
	if err != nil {
		return err
	}
	if user == nil {
		return fmt.Errorf("invalid user")
	}

	good, err := goodcli.GetGood(ctx, o.GoodID)
	if err != nil {
		return err
	}
	if good == nil {
		return fmt.Errorf("invalid good")
	}

	o.GoodStartAt = good.StartAt
	o.GoodDurationDays = uint32(good.DurationDays)

	gcoin, err := coininfocli.GetCoinInfo(ctx, good.CoinTypeID)
	if err != nil {
		return err
	}
	if gcoin == nil {
		return fmt.Errorf("invalid good coin")
	}

	coin, err := coininfocli.GetCoinInfo(ctx, o.PaymentCoinID)
	if err != nil {
		return err
	}
	if coin == nil {
		return fmt.Errorf("invalid coin")
	}
	if coin.PreSale {
		return fmt.Errorf("presale coin won't for payment")
	}
	if !coin.ForPay {
		return fmt.Errorf("coin not for payment")
	}
	if coin.ENV != gcoin.ENV {
		return fmt.Errorf("good coin mismatch payment coin")
	}

	o.paymentCoinName = coin.Name

	if o.ParentOrderID != nil {
		order, err := ordercli.GetOrder(ctx, *o.ParentOrderID)
		if err != nil {
			return err
		}
		if order == nil {
			return fmt.Errorf("invalid parent order")
		}
	}

	appGood, _, err := appgoodcli.GetGoods(ctx, &appgoodpb.Conds{
		AppID: &commonpb.StringVal{
			Op:    cruder.EQ,
			Value: o.AppID,
		},
		GoodID: &commonpb.StringVal{
			Op:    cruder.EQ,
			Value: o.GoodID,
		},
	}, 0, 1)
	if err != nil {
		return err
	}
	if len(appGood) == 0 {
		return fmt.Errorf("invalid app good")
	}

	ag := appGood[0]

	if !ag.Online {
		return fmt.Errorf("good offline")
	}

	price, err := decimal.NewFromString(ag.Price)
	if err != nil {
		return err
	}
	if price.IntPart() <= 0 {
		return fmt.Errorf("invalid good price")
	}
	if ag.Price < good.Price {
		return fmt.Errorf("invalid app good price")
	}
	if ag.PurchaseLimit > 0 && o.Units > uint32(ag.PurchaseLimit) {
		return fmt.Errorf("too many units")
	}

	const maxUnpaidOrders = 3

	payments, err := ordercli.GetAppUserStatePayments(
		ctx,
		o.AppID, o.UserID,
		orderconst.PaymentStateWait,
	)
	if err != nil {
		return err
	}
	if len(payments) >= maxUnpaidOrders && (o.OrderType == ordermgrpb.OrderType_Normal || o.OrderType.String() == orderconst.OrderTypeNormal) {
		return fmt.Errorf("too many unpaid orders")
	}

	switch o.OrderType.String() {
	case ordermgrpb.OrderType_Normal.String():
	case ordermgrpb.OrderType_Offline.String():
	case ordermgrpb.OrderType_Airdrop.String():
	case orderconst.OrderTypeNormal:
	case orderconst.OrderTypeOffline:
	case orderconst.OrderTypeAirdrop:
	default:
		return fmt.Errorf("invalid order type")
	}

	// TODO: check app / user banned

	return nil
}

// nolint
func (o *OrderCreate) SetReduction(ctx context.Context) error {
	var fixAmount *couponpb.Coupon
	if o.FixAmountID != nil {
		ord, err := ordercli.GetCouponOrder(ctx, o.AppID, o.UserID, *o.FixAmountID, orderconst.FixAmountCoupon)
		if err != nil {
			return err
		}
		if ord != nil {
			return fmt.Errorf("used coupon")
		}

		fixAmount, err = couponcli.GetCoupon(ctx, *o.FixAmountID, couponpb.CouponType_FixAmount)
		if err != nil {
			return err
		}
	}
	if fixAmount != nil {
		if !fixAmount.Valid || fixAmount.Expired || fixAmount.AppID != o.AppID || fixAmount.UserID != o.UserID {
			return fmt.Errorf("invalid coupon")
		}
		amount, err := decimal.NewFromString(fixAmount.Value)
		if err != nil {
			return err
		}
		o.reductionAmount = o.reductionAmount.Add(amount)
	}

	var discount *couponpb.Coupon
	if o.DiscountID != nil {
		ord, err := ordercli.GetCouponOrder(ctx, o.AppID, o.UserID, *o.DiscountID, orderconst.DiscountCoupon)
		if err != nil {
			return err
		}
		if ord != nil {
			return fmt.Errorf("used coupon")
		}

		discount, err = couponcli.GetCoupon(ctx, *o.DiscountID, couponpb.CouponType_Discount)
		if err != nil {
			return err
		}
	}
	if discount != nil {
		if !discount.Valid || discount.Expired || discount.AppID != o.AppID || discount.UserID != o.UserID {
			return fmt.Errorf("invalid coupon")
		}

		percent, err := decimal.NewFromString(discount.Value)
		if err != nil {
			return err
		}
		o.reductionPercent = percent
	}
	if o.reductionPercent.Cmp(decimal.NewFromInt(100)) > 0 { //nolint
		return fmt.Errorf("invalid discount")
	}

	var specialOffer *couponpb.Coupon
	if o.SpecialOfferID != nil {
		ord, err := ordercli.GetCouponOrder(ctx, o.AppID, o.UserID, *o.SpecialOfferID, orderconst.UserSpecialReductionCoupon)
		if err != nil {
			return err
		}
		if ord != nil {
			return fmt.Errorf("used coupon")
		}

		specialOffer, err = couponcli.GetCoupon(ctx, *o.SpecialOfferID, couponpb.CouponType_SpecialOffer)
		if err != nil {
			return err
		}
	}
	if specialOffer != nil {
		if !specialOffer.Valid || specialOffer.Expired || specialOffer.AppID != o.AppID || specialOffer.UserID != o.UserID {
			return fmt.Errorf("invalid coupon")
		}
		amount, err := decimal.NewFromString(specialOffer.Value)
		if err != nil {
			return err
		}
		o.reductionAmount = o.reductionAmount.Add(amount)
	}

	return nil
}

func (o *OrderCreate) SetPrice(ctx context.Context) error {
	good, err := goodcli.GetGood(ctx, o.GoodID)
	if err != nil {
		return err
	}
	appGood, _, err := appgoodcli.GetGoods(ctx, &appgoodpb.Conds{
		AppID: &commonpb.StringVal{
			Op:    cruder.EQ,
			Value: o.AppID,
		},
		GoodID: &commonpb.StringVal{
			Op:    cruder.EQ,
			Value: o.GoodID,
		},
	}, 0, 1)
	if err != nil {
		return err
	}
	if len(appGood) == 0 {
		return fmt.Errorf("invalid app good")
	}

	ag := appGood[0]
	if !ag.Online {
		return fmt.Errorf("good offline")
	}
	price, err := decimal.NewFromString(ag.Price)
	if err != nil {
		return err
	}
	if price.Cmp(decimal.NewFromInt(0)) <= 0 {
		return fmt.Errorf("invalid good price")
	}
	if ag.Price < good.Price {
		return fmt.Errorf("invalid app good price")
	}

	o.price, err = decimal.NewFromString(ag.Price)
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
		o.price = promotionPrice
	}

	return nil
}

func (o *OrderCreate) SetCurrency(ctx context.Context) error {
	coin, err := coininfocli.GetCoinInfo(ctx, o.PaymentCoinID)
	if err != nil {
		return err
	}

	liveCurrency, err := currency.USDPrice(ctx, coin.Name)
	if err != nil {
		return err
	}

	o.liveCurrency = decimal.NewFromFloat(liveCurrency)

	if o.liveCurrency.Cmp(decimal.NewFromInt(0)) <= 0 {
		return fmt.Errorf("invalid live coin currency")
	}

	o.coinCurrency = o.liveCurrency

	pc, err := oraclecli.GetCurrencyOnly(ctx,
		cruder.NewFilterConds().
			WithCond(
				oracleconst.FieldAppID,
				cruder.EQ,
				structpb.NewStringValue(o.AppID),
			).
			WithCond(
				oracleconst.FieldCoinTypeID,
				cruder.EQ,
				structpb.NewStringValue(o.PaymentCoinID),
			))
	if err != nil {
		return err
	}
	if pc == nil {
		return nil
	}

	if pc.AppPriceVSUSDT > 0 {
		o.coinCurrency = decimal.NewFromFloat(pc.AppPriceVSUSDT)
	}
	if pc.PriceVSUSDT > 0 {
		o.localCurrency = decimal.NewFromFloat(pc.PriceVSUSDT)
	}

	return nil
}

func (o *OrderCreate) SetPaymentAmount(ctx context.Context) error {
	// TODO: also add sub good order payment amount

	o.paymentAmountUSD = o.price.Mul(decimal.NewFromInt(int64(o.Units)))
	logger.Sugar().Infow(
		"CreateOrder",
		"PaymentAmountUSD", o.paymentAmountUSD,
		"ReductionAmount", o.reductionAmount,
		"ReductionPercent", o.reductionPercent,
	)
	o.paymentAmountUSD = o.paymentAmountUSD.Sub(o.reductionAmount)

	if o.paymentAmountUSD.Cmp(decimal.NewFromInt(0)) < 0 {
		o.paymentAmountUSD = decimal.NewFromInt(0)
	}

	o.paymentAmountUSD = o.paymentAmountUSD.
		Sub(o.paymentAmountUSD.
			Mul(o.reductionPercent).
			Div(decimal.NewFromInt(100))) //nolint

	const accuracy = 1000000

	o.paymentAmountCoin = o.paymentAmountUSD.Div(o.coinCurrency)
	o.paymentAmountCoin = o.paymentAmountCoin.Mul(decimal.NewFromInt(accuracy))
	o.paymentAmountCoin = o.paymentAmountCoin.Ceil()
	o.paymentAmountCoin = o.paymentAmountCoin.Div(decimal.NewFromInt(accuracy))

	if o.BalanceAmount != nil {
		amount, err := decimal.NewFromString(*o.BalanceAmount)
		if err != nil {
			return err
		}
		if amount.Cmp(o.paymentAmountCoin) > 0 {
			amount = o.paymentAmountCoin
			amountStr := amount.String()
			o.BalanceAmount = &amountStr
		}
		o.paymentAmountCoin = o.paymentAmountCoin.Sub(amount)
	}

	return nil
}

func (o *OrderCreate) createAddresses(ctx context.Context) error {
	const createCount = 5
	successCreated := 0

	for i := 0; i < createCount; i++ {
		address, err := sphinxproxycli.CreateAddress(ctx, o.paymentCoinName)
		if err != nil {
			return err
		}
		if address == nil || address.Address == "" {
			return fmt.Errorf("invalid address")
		}

		account, err := billingcli.CreateAccount(ctx, &billingpb.CoinAccountInfo{
			CoinTypeID:             o.PaymentCoinID,
			Address:                address.Address,
			PlatformHoldPrivateKey: true,
		})
		if err != nil {
			return err
		}

		_, err = billingcli.CreateGoodPayment(ctx, &billingpb.GoodPayment{
			GoodID:            o.GoodID,
			PaymentCoinTypeID: o.PaymentCoinID,
			AccountID:         account.ID,
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

func (o *OrderCreate) peekAddress(ctx context.Context) (*billingpb.CoinAccountInfo, error) {
	payments, err := billingcli.GetIdleGoodPayments(ctx, o.GoodID, o.PaymentCoinID)
	if err != nil {
		return nil, err
	}

	var account *billingpb.GoodPayment

	for _, payment := range payments {
		if !payment.Idle {
			continue
		}

		if err := accountlock.Lock(payment.AccountID); err != nil {
			continue
		}

		info, err := billingcli.GetGoodPayment(ctx, payment.ID)
		if err != nil {
			accountlock.Unlock(payment.AccountID) //nolint
			return nil, err
		}

		if !info.Idle {
			accountlock.Unlock(payment.AccountID) //nolint
			continue
		}

		if info.AvailableAt >= uint32(time.Now().Unix()) {
			accountlock.Unlock(payment.AccountID) //nolint
			continue
		}

		info.Idle = false
		info.OccupiedBy = billingconst.TransactionForPaying
		info.UsedFor = billingconst.TransactionForPaying
		_, err = billingcli.UpdateGoodPayment(ctx, info)
		if err != nil {
			accountlock.Unlock(payment.AccountID) //nolint
			return nil, err
		}

		account = info
		accountlock.Unlock(payment.AccountID) //nolint
		break
	}

	if account == nil {
		return nil, nil
	}

	o.goodPaymentID = account.ID

	return billingcli.GetAccount(ctx, account.AccountID)
}

func (o *OrderCreate) PeekAddress(ctx context.Context) error {
	account, err := o.peekAddress(ctx)
	if err != nil {
		return err
	}
	if account != nil {
		o.paymentAddress = account.Address
		o.paymentAccountID = account.ID
		return nil
	}

	if err := o.createAddresses(ctx); err != nil {
		return err
	}

	account, err = o.peekAddress(ctx)
	if err != nil {
		return err
	}
	if account == nil {
		return fmt.Errorf("fail peek address")
	}

	o.paymentAddress = account.Address
	o.paymentAccountID = account.ID

	return nil
}

func (o *OrderCreate) ReleaseAddress(ctx context.Context) error {
	if err := accountlock.Lock(o.paymentAccountID); err != nil {
		return err
	}

	info, err := billingcli.GetGoodPayment(ctx, o.goodPaymentID)
	if err != nil {
		accountlock.Unlock(o.paymentAccountID) //nolint
		return err
	}

	info.Idle = true
	info.OccupiedBy = billingconst.TransactionForNotUsed
	_, err = billingcli.UpdateGoodPayment(ctx, info)

	accountlock.Unlock(o.paymentAccountID) //nolint
	return err
}

func (o *OrderCreate) SetBalance(ctx context.Context) error {
	balance, err := sphinxproxycli.GetBalance(ctx, &sphinxproxypb.GetBalanceRequest{
		Name:    o.paymentCoinName,
		Address: o.paymentAddress,
	})
	if err != nil {
		return err
	}
	if balance == nil {
		return fmt.Errorf("invalid balance")
	}

	o.paymentAddressStartAmount, err = decimal.NewFromString(balance.BalanceStr)

	return err
}

func (o *OrderCreate) createSubOrder(ctx context.Context) error { //nolint
	// TODO: create sub order according to good's must select sub good
	return nil
}

func (o *OrderCreate) LockStock(ctx context.Context) error {
	units := int32(o.Units)
	_, err := goodcli.UpdateGood(ctx, &goodmwpb.GoodReq{
		ID:     &o.GoodID,
		Locked: &units,
	})
	if err != nil {
		return err
	}
	return nil
}

func (o *OrderCreate) ReleaseStock(ctx context.Context) error {
	units := int32(o.Units) * -1
	_, err := goodcli.UpdateGood(ctx, &goodmwpb.GoodReq{
		ID:     &o.GoodID,
		Locked: &units,
	})
	if err != nil {
		return err
	}
	return nil
}

func (o *OrderCreate) LockBalance(ctx context.Context) error {
	if o.BalanceAmount == nil {
		return nil
	}

	ba, err := decimal.NewFromString(*o.BalanceAmount)
	if err != nil {
		return err
	}

	if ba.Cmp(decimal.NewFromInt(0)) <= 0 {
		return nil
	}

	general, err := ledgermgrcli.GetGeneralOnly(ctx, &ledgermgrpb.Conds{
		AppID: &commonpb.StringVal{
			Op:    cruder.EQ,
			Value: o.AppID,
		},
		UserID: &commonpb.StringVal{
			Op:    cruder.EQ,
			Value: o.UserID,
		},
		CoinTypeID: &commonpb.StringVal{
			Op:    cruder.EQ,
			Value: o.PaymentCoinID,
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

	spendableMinus := fmt.Sprintf("-%v", *o.BalanceAmount)

	_, err = ledgermgrcli.AddGeneral(ctx, &ledgermgrpb.GeneralReq{
		ID:         &general.ID,
		AppID:      &general.AppID,
		UserID:     &general.UserID,
		CoinTypeID: &general.CoinTypeID,
		Locked:     o.BalanceAmount,
		Spendable:  &spendableMinus,
	})

	return err
}

func (o *OrderCreate) ReleaseBalance(ctx context.Context) error {
	if o.BalanceAmount == nil {
		return nil
	}

	ba, err := decimal.NewFromString(*o.BalanceAmount)
	if err != nil {
		return err
	}

	if ba.Cmp(decimal.NewFromInt(0)) <= 0 {
		return nil
	}

	general, err := ledgermgrcli.GetGeneralOnly(ctx, &ledgermgrpb.Conds{
		AppID: &commonpb.StringVal{
			Op:    cruder.EQ,
			Value: o.AppID,
		},
		UserID: &commonpb.StringVal{
			Op:    cruder.EQ,
			Value: o.UserID,
		},
		CoinTypeID: &commonpb.StringVal{
			Op:    cruder.EQ,
			Value: o.PaymentCoinID,
		},
	})
	if err != nil {
		return err
	}
	if general == nil {
		return fmt.Errorf("insufficuent funds")
	}

	lockedMinus := fmt.Sprintf("-%v", o.BalanceAmount)

	_, err = ledgermgrcli.AddGeneral(ctx, &ledgermgrpb.GeneralReq{
		ID:         &general.ID,
		AppID:      &general.AppID,
		UserID:     &general.UserID,
		CoinTypeID: &general.CoinTypeID,
		Locked:     &lockedMinus,
		Spendable:  o.BalanceAmount,
	})

	return err
}

func tomorrowStart() time.Time {
	now := time.Now()
	y, m, d := now.Date()
	return time.Date(y, m, d+1, 0, 0, 0, 0, now.Location())
}

func (o *OrderCreate) Create(ctx context.Context) (*npool.Order, error) {
	switch o.OrderType.String() {
	case ordermgrpb.OrderType_Normal.String():
	case ordermgrpb.OrderType_Offline.String():
	case ordermgrpb.OrderType_Airdrop.String():
	case orderconst.OrderTypeNormal:
		o.OrderType = ordermgrpb.OrderType_Normal
	case orderconst.OrderTypeOffline:
		o.OrderType = ordermgrpb.OrderType_Offline
	case orderconst.OrderTypeAirdrop:
		o.OrderType = ordermgrpb.OrderType_Airdrop
	default:
		return nil, fmt.Errorf("invalid order type")
	}

	paymentAmount := o.paymentAmountCoin.String()
	startAmount := o.paymentAddressStartAmount.String()
	coinCurrency := o.coinCurrency.String()
	liveCurrency := o.liveCurrency.String()
	localCurrency := o.localCurrency.String()

	// Top order never pay with parent, only sub order may

	o.start = uint32(tomorrowStart().Unix())
	if o.GoodStartAt > o.start {
		o.start = o.GoodStartAt
	}
	const secondsPerDay = 24 * 60 * 60
	o.end = o.start + o.GoodDurationDays*secondsPerDay

	ord, err := ordermwcli.CreateOrder(ctx, &ordermwpb.OrderReq{
		AppID:     &o.AppID,
		UserID:    &o.UserID,
		GoodID:    &o.GoodID,
		Units:     &o.Units,
		OrderType: &o.OrderType,

		ParentOrderID: o.ParentOrderID,

		PaymentCoinID:             &o.PaymentCoinID,
		PayWithBalanceAmount:      o.BalanceAmount,
		PaymentAccountID:          &o.paymentAccountID,
		PaymentAmount:             &paymentAmount,
		PaymentAccountStartAmount: &startAmount,
		PaymentCoinUSDCurrency:    &coinCurrency,
		PaymentLiveUSDCurrency:    &liveCurrency,
		PaymentLocalUSDCurrency:   &localCurrency,

		FixAmountID:    o.FixAmountID,
		DiscountID:     o.DiscountID,
		SpecialOfferID: o.SpecialOfferID,

		Start: &o.start,
		End:   &o.end,

		PromotionID: o.promotionID,
	})
	if err != nil {
		return nil, err
	}

	return GetOrder(ctx, ord.ID)
}
