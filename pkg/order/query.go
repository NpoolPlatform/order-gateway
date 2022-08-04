package order

import (
	"context"
	"fmt"

	usercli "github.com/NpoolPlatform/appuser-middleware/pkg/client/user"
	billingcli "github.com/NpoolPlatform/cloud-hashing-billing/pkg/client"
	goodscli "github.com/NpoolPlatform/cloud-hashing-goods/pkg/client"
	couponcli "github.com/NpoolPlatform/inspire-middleware/pkg/client/coupon"
	npool "github.com/NpoolPlatform/message/npool/order/gw/v1/order"
	ordercli "github.com/NpoolPlatform/order-middleware/pkg/client/order"
	coininfocli "github.com/NpoolPlatform/sphinx-coininfo/pkg/client"

	billingpb "github.com/NpoolPlatform/message/npool/cloud-hashing-billing"
	goodspb "github.com/NpoolPlatform/message/npool/cloud-hashing-goods"
	coininfopb "github.com/NpoolPlatform/message/npool/coininfo"
	couponpb "github.com/NpoolPlatform/message/npool/inspire/mw/v1/inspire/coupon"

	cruder "github.com/NpoolPlatform/libent-cruder/pkg/cruder"

	"github.com/shopspring/decimal"

	"github.com/google/uuid"
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

	good, err := goodscli.GetGood(ctx, ord.GoodID)
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

func GetOrders(ctx context.Context, appID, userID string, offset, limit int32) ([]*npool.Order, error) { //nolint
	ords, err := ordercli.GetOrders(ctx, appID, userID, offset, limit)
	if err != nil {
		return nil, err
	}
	if len(ords) == 0 {
		return []*npool.Order{}, nil
	}

	user, err := usercli.GetUser(ctx, ords[0].AppID, ords[0].UserID)
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, fmt.Errorf("invalid user")
	}

	goods, err := goodscli.GetGoods(ctx)
	if err != nil {
		return nil, err
	}

	goodMap := map[string]*goodspb.GoodInfo{}
	for _, good := range goods {
		goodMap[good.ID] = good
	}

	coins, err := coininfocli.GetCoinInfos(ctx, cruder.NewFilterConds())
	if err != nil {
		return nil, err
	}

	coinMap := map[string]*coininfopb.CoinInfo{}
	for _, coin := range coins {
		coinMap[coin.ID] = coin
	}

	// TODO: get accounts with specific account ID
	accounts, err := billingcli.GetAccounts(ctx)
	if err != nil {
		return nil, err
	}

	accMap := map[string]*billingpb.CoinAccountInfo{}
	for _, acc := range accounts {
		accMap[acc.ID] = acc
	}

	ids := []string{}
	invalidID := uuid.UUID{}.String()

	for _, ord := range ords {
		if ord.FixAmountID == invalidID {
			continue
		}
		ids = append(ids, ord.FixAmountID)
	}

	coupons, err := couponcli.GetManyCoupons(ctx, ids, couponpb.CouponType_FixAmount)
	if err != nil {
		return nil, err
	}

	fixAmountMap := map[string]*couponpb.Coupon{}
	for _, coupon := range coupons {
		fixAmountMap[coupon.ID] = coupon
	}

	coupons, err = couponcli.GetManyCoupons(ctx, ids, couponpb.CouponType_Discount)
	if err != nil {
		return nil, err
	}

	discountMap := map[string]*couponpb.Coupon{}
	for _, coupon := range coupons {
		discountMap[coupon.ID] = coupon
	}

	coupons, err = couponcli.GetManyCoupons(ctx, ids, couponpb.CouponType_SpecialOffer)
	if err != nil {
		return nil, err
	}

	specialOfferMap := map[string]*couponpb.Coupon{}
	for _, coupon := range coupons {
		specialOfferMap[coupon.ID] = coupon
	}

	infos := []*npool.Order{}
	for _, ord := range ords {
		o := &npool.Order{
			ID:           ord.ID,
			UserID:       ord.UserID,
			EmailAddress: user.EmailAddress,
			PhoneNO:      user.PhoneNO,
			GoodID:       ord.GoodID,
			Units:        ord.Units,

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

		good, ok := goodMap[ord.GoodID]
		if !ok {
			return nil, fmt.Errorf("invalid good")
		}

		o.CoinTypeID = good.CoinInfoID

		coin, ok := coinMap[o.CoinTypeID]
		if !ok {
			return nil, fmt.Errorf("invalid coin")
		}

		o.CoinName = coin.Name
		o.CoinLogo = coin.Logo
		o.CoinUnit = coin.Unit

		coin, ok = coinMap[ord.PaymentCoinTypeID]
		if !ok {
			return nil, fmt.Errorf("invalid payment coin")
		}

		o.PaymentCoinName = coin.Name
		o.PaymentCoinLogo = coin.Logo
		o.PaymentCoinUnit = coin.Unit

		acc, ok := accMap[ord.PaymentAccountID]
		if !ok {
			return nil, fmt.Errorf("invalid account")
		}

		o.PaymentAddress = acc.Address

		if coupon, ok := fixAmountMap[ord.FixAmountID]; ok {
			o.FixAmountName = coupon.Name
			o.FixAmountAmount = coupon.Value
		}
		if coupon, ok := fixAmountMap[ord.DiscountID]; ok {
			o.DiscountName = coupon.Name
			percent, err := decimal.NewFromString(coupon.Value)
			if err != nil {
				return nil, err
			}
			o.DiscountPercent = uint32(percent.IntPart())
		}
		if coupon, ok := fixAmountMap[ord.SpecialOfferID]; ok {
			o.SpecialOfferAmount = coupon.Value
		}

		infos = append(infos, o)
	}

	return infos, nil
}
