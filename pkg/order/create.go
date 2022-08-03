package order

import (
	"context"
	"fmt"
	"time"

	// npool "github.com/NpoolPlatform/message/npool/order/gw/v1/order"

	appcli "github.com/NpoolPlatform/appuser-middleware/pkg/client/app"
	usercli "github.com/NpoolPlatform/appuser-middleware/pkg/client/user"
	billingcli "github.com/NpoolPlatform/cloud-hashing-billing/pkg/client"
	goodcli "github.com/NpoolPlatform/cloud-hashing-goods/pkg/client"
	ordercli "github.com/NpoolPlatform/cloud-hashing-order/pkg/client"
	couponcli "github.com/NpoolPlatform/inspire-middleware/pkg/client/coupon"
	oraclecli "github.com/NpoolPlatform/oracle-manager/pkg/client"
	coininfocli "github.com/NpoolPlatform/sphinx-coininfo/pkg/client"
	sphinxproxycli "github.com/NpoolPlatform/sphinx-proxy/pkg/client"

	billingconst "github.com/NpoolPlatform/cloud-hashing-billing/pkg/const"
	orderconst "github.com/NpoolPlatform/cloud-hashing-order/pkg/const"
	oracleconst "github.com/NpoolPlatform/oracle-manager/pkg/const"

	billingpb "github.com/NpoolPlatform/message/npool/cloud-hashing-billing"
	couponpb "github.com/NpoolPlatform/message/npool/inspire/mw/v1/inspire/coupon"

	cruder "github.com/NpoolPlatform/libent-cruder/pkg/cruder"
	currency "github.com/NpoolPlatform/oracle-manager/pkg/middleware/currency"
	accountlock "github.com/NpoolPlatform/staker-manager/pkg/middleware/account"

	"google.golang.org/protobuf/types/known/structpb"

	"github.com/shopspring/decimal"
)

type OrderCreate struct {
	AppID  string
	UserID string
	GoodID string
	Units  uint32

	PaymentCoinID string
	BalanceAmount *string

	ParentOrderID *string

	FixAmountID    *string
	DiscountID     *string
	SpecialOfferID *string

	paymentCoinName   string
	paymentAmountUSD  decimal.Decimal
	paymentAmountCoin decimal.Decimal
	paymentAddress    string
	paymentAccountID  string

	promotionID string

	price decimal.Decimal

	liveCurrency  decimal.Decimal
	localCurrency decimal.Decimal
	coinCurrency  decimal.Decimal

	reductionAmount  decimal.Decimal
	reductionPercent decimal.Decimal
}

func (o *OrderCreate) ValidateInit(ctx context.Context) error {
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

	gcoin, err := coininfocli.GetCoinInfo(ctx, good.CoinInfoID)
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

	ag, err := goodcli.GetAppGood(ctx, o.AppID, o.GoodID)
	if err != nil {
		return err
	}
	if err != nil {
		return err
	}
	if ag == nil {
		return fmt.Errorf("permission denied")
	}
	if !ag.Online {
		return fmt.Errorf("good offline")
	}
	if ag.Price <= 0 {
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
	if len(payments) >= maxUnpaidOrders {
		return fmt.Errorf("too many unpaid orders")
	}

	// TODO: check app / user banned

	return nil
}

func (o *OrderCreate) SetReduction(ctx context.Context) error {
	var err error

	var fixAmount *couponpb.Coupon
	if o.FixAmountID != nil {
		fixAmount, err = couponcli.GetCoupon(ctx, *o.FixAmountID, couponpb.CouponType_FixAmount)
		if err != nil {
			return err
		}
	}
	if fixAmount != nil {
		amount, err := decimal.NewFromString(fixAmount.Value)
		if err != nil {
			return err
		}
		o.reductionAmount = o.reductionAmount.Add(amount)
	}

	var discount *couponpb.Coupon
	if o.DiscountID != nil {
		discount, err = couponcli.GetCoupon(ctx, *o.DiscountID, couponpb.CouponType_Discount)
		if err != nil {
			return err
		}
	}
	if discount != nil {
		percent, err := decimal.NewFromString(discount.Value)
		if err != nil {
			return err
		}
		o.reductionPercent = percent
	}
	if o.reductionPercent.Cmp(decimal.NewFromInt(100)) > 0 {
		return fmt.Errorf("invalid discount")
	}

	var specialOffer *couponpb.Coupon
	if o.SpecialOfferID != nil {
		specialOffer, err = couponcli.GetCoupon(ctx, *o.SpecialOfferID, couponpb.CouponType_SpecialOffer)
		if err != nil {
			return err
		}
	}
	if specialOffer != nil {
		amount, err := decimal.NewFromString(specialOffer.Value)
		if err != nil {
			return err
		}
		o.reductionAmount = o.reductionAmount.Add(amount)
	}

	return nil
}

func (o *OrderCreate) setPrice(ctx context.Context) error {
	good, err := goodcli.GetGood(ctx, o.GoodID)
	if err != nil {
		return err
	}

	ag, err := goodcli.GetAppGood(ctx, o.AppID, o.GoodID)
	if err != nil {
		return err
	}
	if !ag.Online {
		return fmt.Errorf("good offline")
	}
	if ag.Price <= 0 {
		return fmt.Errorf("invalid good price")
	}
	if ag.Price < good.Price {
		return fmt.Errorf("invalid app good price")
	}

	o.price = decimal.NewFromFloat(ag.Price)

	promotion, err := goodcli.GetCurrentPromotion(ctx, o.AppID, o.GoodID, uint32(time.Now().Unix()))
	if err != nil {
		return err
	}
	if promotion != nil {
		o.promotionID = promotion.ID
	}

	if promotion.Price <= 0 {
		return fmt.Errorf("invalid price")
	}
	if promotion != nil {
		o.price = decimal.NewFromFloat(promotion.Price)
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

	o.coinCurrency = decimal.NewFromFloat(liveCurrency)

	if o.coinCurrency.Cmp(decimal.NewFromInt(0)) <= 0 ||
		o.liveCurrency.Cmp(decimal.NewFromInt(0)) <= 0 {
		return fmt.Errorf("invalid coin currency")
	}

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
	o.paymentAmountUSD = o.price.Mul(decimal.NewFromInt(int64(o.Units)))
	o.paymentAmountUSD = o.paymentAmountUSD.Sub(o.reductionAmount)
	if o.paymentAmountUSD.Cmp(decimal.NewFromInt(0)) < 0 {
		o.paymentAmountUSD = decimal.NewFromInt(0)
	}

	o.paymentAmountUSD = o.paymentAmountUSD.
		Mul(o.reductionPercent).
		Div(decimal.NewFromInt(100))

	const accuracy = 1000000

	o.paymentAmountCoin = o.paymentAmountUSD.Div(o.coinCurrency)
	o.paymentAmountCoin = o.paymentAmountCoin.Mul(decimal.NewFromInt(accuracy))
	o.paymentAmountCoin = o.paymentAmountCoin.Ceil()
	o.paymentAmountCoin = o.paymentAmountCoin.Div(decimal.NewFromInt(accuracy))

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
	payments, err := billingcli.GetIdleGoodPayments(ctx, o.AppID, o.GoodID)
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
			accountlock.Unlock(payment.AccountID) //nilint
			return nil, err
		}

		if !info.Idle {
			accountlock.Unlock(payment.AccountID) //nilint
			continue
		}

		info.Idle = false
		info.OccupiedBy = billingconst.TransactionForPaying
		_, err = billingcli.UpdateGoodPayment(ctx, info)
		if err != nil {
			accountlock.Unlock(payment.AccountID) //nilint
			return nil, err
		}

		account = info
		accountlock.Unlock(payment.AccountID) //nilint
		break
	}

	if account == nil {
		return nil, nil
	}

	return billingcli.GetAccount(ctx, account.AccountID)
}

func (o *OrderCreate) PeekAddress(ctx context.Context) error {
	account, err := o.peekAddress(ctx)
	if err != nil {
		return err
	}
	if account != nil {
		o.paymentAddress = account.Address
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
	return nil
}

/*
func (o *OrderCreate) checkBalance(ctx context.Context) error {

}

func (o *OrderCreate) lockStock(ctx context) error {

}

func (o *OrderCreate) create(ctx context.Context) (*npool.Order, error) {

}
*/
