package order

import (
	"context"
	"fmt"

	// npool "github.com/NpoolPlatform/message/npool/order/gw/v1/order"

	appcli "github.com/NpoolPlatform/appuser-middleware/pkg/client/app"
	usercli "github.com/NpoolPlatform/appuser-middleware/pkg/client/user"
	goodcli "github.com/NpoolPlatform/cloud-hashing-goods/pkg/client"
	ordercli "github.com/NpoolPlatform/cloud-hashing-order/pkg/client"
	couponcli "github.com/NpoolPlatform/inspire-middleware/pkg/client/coupon"
	coininfocli "github.com/NpoolPlatform/sphinx-coininfo/pkg/client"

	orderconst "github.com/NpoolPlatform/cloud-hashing-order/pkg/const"

	couponpb "github.com/NpoolPlatform/message/npool/inspire/mw/v1/inspire/coupon"

	// accountlock "github.com/NpoolPlatform/staker-manager/pkg/middleware/account"

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

	paymentAmount    decimal.Decimal
	paymentAddress   string
	paymentAccountID string

	promotionID string

	price decimal.Decimal

	liveCurrency  decimal.Decimal
	localCurrency decimal.Decimal
	coinCurrency  decimal.Decimal

	reduction decimal.Decimal
}

func (o *OrderCreate) Validate(ctx context.Context) error {
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
	o.reduction = decimal.NewFromInt(0)
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
		o.reduction = o.reduction.Add(amount)
	}

	var discount *couponpb.Coupon
	if o.DiscountID != nil {
		discount, err = couponcli.GetCoupon(ctx, *o.DiscountID, couponpb.CouponType_Discount)
		if err != nil {
			return err
		}
	}
	if discount != nil {
		amount, err := decimal.NewFromString(discount.Value)
		if err != nil {
			return err
		}
		o.reduction = o.reduction.Add(amount)
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
		o.reduction = o.reduction.Add(amount)
	}

	return nil
}

/*
func (o *OrderCreate) setPrice(ctx context.Context) error {
	ag, err := goodcli.GetAppGood(ctx, in.GetAppID(), in.GetGoodID())
	if err != nil {
		return  err
	}
	if ag == nil {
		return  fmt.Errorf("permission denied")
	}
	if !ag.Online {
		return  fmt.Errorf("good offline")
	}
	if ag.Price <= 0 {
		return  fmt.Errorf("invalid good price")
	}
	if ag.Price < good.Price {
		return  fmt.Errorf("invalid app good price")
	}
	if ag.PurchaseLimit > 0 && in.GetUnits() > uint32(ag.PurchaseLimit) {
		return  fmt.Errorf("too many units")
	}

	promotionID := uuid.UUID{}.String()
	promotion, err := goodcli.GetCurrentPromotion(
		ctx,
		in.GetAppID(), in.GetGoodID(),
		uint32(time.Now().Unix()),
	)
	if err != nil {
		return  err
	}
	if promotion != nil {
		promotionID = promotion.ID
	}

	price := ag.Price
	if promotion != nil {
		price = promotion.Price
	}
	if promotion.Price == 0 {
		return  fmt.Errorf("invalid price")
	}
}

func (o *OrderCreate) setCurrency(ctx context.Context) error {
	liveCurrency, err := currency.USDPrice(ctx, coin.Name)
	if err != nil {
		return  err
	}
	coinCurrency := liveCurrency
	localCurrency := 0.0

	pc, err := oraclecli.GetCurrencyOnly(ctx,
		cruder.NewFilterConds().
			WithCond(
				oracleconst.FieldAppID, cruder.EQ, structpb.NewStringValue(appID),
			).
			WithCond(
				oracleconst.FieldCoinTypeID, cruder.EQ, structpb.NewStringValue(paymentCoinID),
			))
	if err != nil {
		return  err
	}
	if pc != nil {
		if pc.AppPriceVSUSDT {
			coinCurrency = pc.AppPriceVSUSDT
		}
		if pc.PriceVSUSDT > 0 {
			localCurrency = pc.PriceVSUSDT
		}
	}
	if coinCurrency <= 0 || liveCurrency <= 0 {
		return  fmt.Errorf("invalid coin currency")
	}
}

func (o *OrderCreate) setAddress(ctx context.Context) error {

}

func (o *OrderCreate) setPaymentAmount(ctx context.Context) error {

}

func (o *OrderCreate) checkBalance(ctx context.Context) error {

}

func (o *OrderCreate) lockStock(ctx context) error {

}

func (o *OrderCreate) create(ctx context.Context) (*npool.Order, error) {

}
*/
