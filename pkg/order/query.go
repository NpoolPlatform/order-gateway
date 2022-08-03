package order

import (
	"context"
	"fmt"

	usercli "github.com/NpoolPlatform/appuser-middleware/pkg/client/user"
	billingcli "github.com/NpoolPlatform/cloud-hashing-billing/pkg/client"
	goodcli "github.com/NpoolPlatform/cloud-hashing-goods/pkg/client"
	couponcli "github.com/NpoolPlatform/inspire-middleware/pkg/client/coupon"
	npool "github.com/NpoolPlatform/message/npool/order/gw/v1/order"
	ordercli "github.com/NpoolPlatform/order-middleware/pkg/client/order"
	coininfocli "github.com/NpoolPlatform/sphinx-coininfo/pkg/client"

	couponpb "github.com/NpoolPlatform/message/npool/inspire/mw/v1/inspire/coupon"

	"github.com/shopspring/decimal"
)

func GetOrder(ctx context.Context, id string) (*npool.Order, error) { //nolint
	ord, err := ordercli.GetOrder(ctx, id)
	if err != nil {
		return nil, err
	}

	o := &npool.Order{
		ID:     ord.ID,
		UserID: ord.UserID,
		GoodID: ord.GoodID,
		Units:  ord.Units,

		ParentOrderID:     ord.ParentOrderID,
		ParentOrderGoodID: ord.ParentOrderGoodID,

		PaymentID:               ord.PaymentID,
		PaymentCoinTypeID:       ord.PaymentCoinTypeID,
		PaymentCoinUSDCurrency:  ord.PaymentCoinUSDCurrency,
		PaymentLiveUSDCurrency:  ord.PaymentLiveCoinUSDCurrency,
		PaymentLocalUSDCurrency: ord.PaymentLocalCoinUSDCurrency,
		PaymentAmount:           ord.PaymentAmount,
		PayWithParent:           ord.PayWithParent,
		PayWithBalanceAmount:    ord.PayWithBalanceAmount,

		FixAmountID:    ord.FixAmountID,
		DiscountID:     ord.DiscountID,
		SpecialOfferID: ord.SpecialOfferID,

		CreatedAt: ord.CreatedAt,
		PaidAt:    ord.PaidAt,
		State:     ord.State,

		Start: ord.Start,
		End:   ord.End,
	}

	user, err := usercli.GetUser(ctx, ord.AppID, ord.UserID)
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, fmt.Errorf("invalid user")
	}

	o.EmailAddress = user.EmailAddress
	o.PhoneNO = user.PhoneNO

	good, err := goodcli.GetGood(ctx, ord.GoodID)
	if err != nil {
		return nil, err
	}
	if good == nil {
		return nil, fmt.Errorf("invalid good")
	}

	o.GoodName = good.Title

	coin, err := coininfocli.GetCoinInfo(ctx, good.CoinInfoID)
	if err != nil {
		return nil, err
	}
	if coin == nil {
		return nil, fmt.Errorf("invalid good coin")
	}

	o.CoinTypeID = good.CoinInfoID
	o.CoinName = coin.Name
	o.CoinLogo = coin.Logo
	o.CoinUnit = coin.Unit

	coin, err = coininfocli.GetCoinInfo(ctx, ord.PaymentCoinTypeID)
	if err != nil {
		return nil, err
	}
	if coin == nil {
		return nil, fmt.Errorf("invalid payment coin")
	}

	o.PaymentCoinName = coin.Name
	o.PaymentCoinLogo = coin.Logo
	o.PaymentCoinUnit = coin.Unit

	account, err := billingcli.GetAccount(ctx, ord.PaymentAccountID)
	if err != nil {
		return nil, err
	}
	// TODO: for old placeholder payment
	if account == nil {
		return nil, fmt.Errorf("invalid account")
	}

	o.PaymentAddress = account.Address

	coupon, err := couponcli.GetCoupon(ctx, ord.FixAmountID, couponpb.CouponType_FixAmount)
	if err != nil {
		return nil, err
	}
	if coupon != nil {
		o.FixAmountName = coupon.Name
		o.FixAmountAmount = coupon.Value
	}

	coupon, err = couponcli.GetCoupon(ctx, ord.DiscountID, couponpb.CouponType_Discount)
	if err != nil {
		return nil, err
	}
	if coupon != nil {
		o.DiscountName = coupon.Name
		v, err := decimal.NewFromString(coupon.Value)
		if err != nil {
			return nil, err
		}
		o.DiscountPercent = uint32(v.IntPart())
	}

	coupon, err = couponcli.GetCoupon(ctx, ord.SpecialOfferID, couponpb.CouponType_SpecialOffer)
	if err != nil {
		return nil, err
	}
	if coupon != nil {
		o.SpecialOfferAmount = coupon.Value
	}

	return o, nil
}
