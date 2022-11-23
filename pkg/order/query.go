//nolint:dupl
package order

import (
	"context"
	"fmt"

	usercli "github.com/NpoolPlatform/appuser-middleware/pkg/client/user"
	billingcli "github.com/NpoolPlatform/cloud-hashing-billing/pkg/client"
	"github.com/NpoolPlatform/go-service-framework/pkg/logger"
	goodscli "github.com/NpoolPlatform/good-middleware/pkg/client/good"
	couponcli "github.com/NpoolPlatform/inspire-middleware/pkg/client/coupon"
	npool "github.com/NpoolPlatform/message/npool/order/gw/v1/order"
	ordercli "github.com/NpoolPlatform/order-middleware/pkg/client/order"
	coininfocli "github.com/NpoolPlatform/sphinx-coininfo/pkg/client"

	goodspb "github.com/NpoolPlatform/message/npool/good/mw/v1/good"

	goodsmgepb "github.com/NpoolPlatform/message/npool/good/mgr/v1/good"

	userpb "github.com/NpoolPlatform/message/npool/appuser/mw/v1/user"
	billingpb "github.com/NpoolPlatform/message/npool/cloud-hashing-billing"

	coininfopb "github.com/NpoolPlatform/message/npool/coininfo"
	couponpb "github.com/NpoolPlatform/message/npool/inspire/mw/v1/inspire/coupon"
	ordermwpb "github.com/NpoolPlatform/message/npool/order/mw/v1/order"

	appgoodscli "github.com/NpoolPlatform/good-middleware/pkg/client/appgood"
	appgoodspb "github.com/NpoolPlatform/message/npool/good/mw/v1/appgood"

	appgoodsmgrpb "github.com/NpoolPlatform/message/npool/good/mgr/v1/appgood"

	npoolpb "github.com/NpoolPlatform/message/npool"

	cruder "github.com/NpoolPlatform/libent-cruder/pkg/cruder"

	"github.com/shopspring/decimal"

	"github.com/google/uuid"
)

var invalidID = uuid.UUID{}.String()

func GetOrder(ctx context.Context, id string) (*npool.Order, error) { //nolint
	ord, err := ordercli.GetOrder(ctx, id)
	if err != nil {
		return nil, err
	}
	if ord == nil {
		return nil, err
	}

	o := &npool.Order{
		ID:     ord.ID,
		AppID:  ord.AppID,
		UserID: ord.UserID,
		GoodID: ord.GoodID,
		Units:  ord.Units,

		ParentOrderID:     ord.ParentOrderID,
		ParentOrderGoodID: ord.ParentOrderGoodID,

		PaymentID:               ord.PaymentID,
		PaymentCoinTypeID:       ord.PaymentCoinTypeID,
		PaymentCoinUSDCurrency:  ord.PaymentCoinUSDCurrency,
		PaymentLiveUSDCurrency:  ord.PaymentLiveUSDCurrency,
		PaymentLocalUSDCurrency: ord.PaymentLocalUSDCurrency,
		PaymentAmount:           ord.PaymentAmount,
		PaymentStartAmount:      ord.PaymentStartAmount,
		PaymentFinishAmount:     ord.PaymentFinishAmount,
		PayWithParent:           ord.PayWithParent,
		PayWithBalanceAmount:    ord.PayWithBalanceAmount,

		FixAmountID:    ord.FixAmountID,
		DiscountID:     ord.DiscountID,
		SpecialOfferID: ord.SpecialOfferID,

		OrderType: ord.OrderType,
		CreatedAt: ord.CreatedAt,
		PaidAt:    ord.PaidAt,
		State:     ord.OrderState,

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

	appGood, err := appgoodscli.GetGood(ctx, ord.GoodID)
	if err != nil {
		return nil, err
	}

	if appGood == nil {
		return nil, fmt.Errorf("invalid app good")
	}

	o.GoodName = appGood.GoodName
	o.GoodUnit = appGood.Unit
	o.GoodServicePeriodDays = uint32(appGood.DurationDays)
	o.GoodUnitPrice = appGood.Price

	appGoodPrice, err := decimal.NewFromString(appGood.Price)
	if err != nil {
		return nil, err
	}
	o.GoodValue = appGoodPrice.Mul(decimal.NewFromInt32(int32(ord.Units))).String()
	if appGood.PromotionPrice != nil {
		appGoodPromotionPrice, err := decimal.NewFromString(*appGood.PromotionPrice)
		if err != nil {
			return nil, err
		}
		o.GoodValue = appGoodPromotionPrice.Mul(decimal.NewFromInt32(int32(ord.Units))).String()
	}

	o.GoodValue = appGood.Price

	coin, err := coininfocli.GetCoinInfo(ctx, appGood.CoinTypeID)
	if err != nil {
		return nil, err
	}

	if coin == nil {
		return nil, fmt.Errorf("invalid good coin")
	}

	o.CoinTypeID = appGood.CoinTypeID
	o.CoinName = coin.Name
	o.CoinLogo = coin.Logo
	o.CoinUnit = coin.Unit
	o.CoinPresale = coin.PreSale

	if ord.PaymentID != invalidID && ord.PaymentID != "" {
		coin, err = coininfocli.GetCoinInfo(ctx, ord.PaymentCoinTypeID)
		if err != nil {
			return nil, err
		}

		if coin == nil {
			return nil, fmt.Errorf("invalid payment coin")
		}
	}

	if coin != nil {
		o.PaymentCoinName = coin.Name
		o.PaymentCoinLogo = coin.Logo
		o.PaymentCoinUnit = coin.Unit
	}

	account, err := billingcli.GetAccount(ctx, ord.PaymentAccountID)
	if err != nil {
		return nil, err
	}
	if account != nil {
		o.PaymentAddress = account.Address
	}

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

func GetOrders(ctx context.Context, appID, userID string, offset, limit int32) ([]*npool.Order, uint32, error) {
	ords, total, err := ordercli.GetOrders(ctx, &ordermwpb.Conds{
		AppID: &npoolpb.StringVal{
			Op:    cruder.EQ,
			Value: appID,
		},
		UserID: &npoolpb.StringVal{
			Op:    cruder.EQ,
			Value: userID,
		},
	}, offset, limit)
	if err != nil {
		return nil, 0, err
	}
	if len(ords) == 0 {
		return []*npool.Order{}, 0, nil
	}

	orders, err := expand(ctx, ords)
	if err != nil {
		return nil, 0, err
	}

	return orders, total, nil
}

func GetAppOrders(ctx context.Context, appID string, offset, limit int32) ([]*npool.Order, uint32, error) {
	ords, total, err := ordercli.GetOrders(ctx, &ordermwpb.Conds{
		AppID: &npoolpb.StringVal{
			Op:    cruder.EQ,
			Value: appID,
		},
	}, offset, limit)
	if err != nil {
		return nil, 0, err
	}
	if len(ords) == 0 {
		return []*npool.Order{}, 0, nil
	}

	orders, err := expand(ctx, ords)
	if err != nil {
		return nil, 0, err
	}

	return orders, total, nil
}

// nolint
func expand(ctx context.Context, ords []*ordermwpb.Order) ([]*npool.Order, error) {
	if len(ords) == 0 {
		return []*npool.Order{}, nil
	}

	uids := []string{}
	for _, ord := range ords {
		uids = append(uids, ord.UserID)
	}

	users, _, err := usercli.GetManyUsers(ctx, uids)
	if err != nil {
		return nil, err
	}
	if len(users) == 0 {
		return nil, fmt.Errorf("invalid users")
	}

	userMap := map[string]*userpb.User{}
	for _, user := range users {
		userMap[user.ID] = user
	}

	goodIDs := []string{}
	for _, val := range ords {
		goodIDs = append(goodIDs, val.GetGoodID())
	}

	goods, _, err := goodscli.GetGoods(ctx, &goodsmgepb.Conds{
		IDs: &npoolpb.StringSliceVal{
			Op:    cruder.IN,
			Value: goodIDs,
		},
	}, 0, int32(len(goodIDs)))
	if err != nil {
		return nil, err
	}

	goodMap := map[string]*goodspb.Good{}
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

	ids = []string{}
	for _, ord := range ords {
		if ord.DiscountID == invalidID {
			continue
		}
		ids = append(ids, ord.DiscountID)
	}

	coupons, err = couponcli.GetManyCoupons(ctx, ids, couponpb.CouponType_Discount)
	if err != nil {
		return nil, err
	}

	discountMap := map[string]*couponpb.Coupon{}
	for _, coupon := range coupons {
		discountMap[coupon.ID] = coupon
	}

	ids = []string{}
	for _, ord := range ords {
		if ord.SpecialOfferID == invalidID {
			continue
		}
		ids = append(ids, ord.SpecialOfferID)
	}

	coupons, err = couponcli.GetManyCoupons(ctx, ids, couponpb.CouponType_SpecialOffer)
	if err != nil {
		return nil, err
	}

	specialOfferMap := map[string]*couponpb.Coupon{}
	for _, coupon := range coupons {
		specialOfferMap[coupon.ID] = coupon
	}

	appGoods, _, err := appgoodscli.GetGoods(ctx, &appgoodsmgrpb.Conds{
		GoodIDs: &npoolpb.StringSliceVal{
			Op:    cruder.IN,
			Value: goodIDs,
		},
	}, 0, int32(len(goodIDs)))
	if err != nil {
		return nil, err
	}

	appGoodMap := map[string]*appgoodspb.Good{}
	for _, appGood := range appGoods {
		appGoodMap[appGood.AppID+appGood.GoodID] = appGood
	}

	infos := []*npool.Order{}

	for _, ord := range ords {
		o := &npool.Order{
			ID:     ord.ID,
			AppID:  ord.AppID,
			UserID: ord.UserID,
			GoodID: ord.GoodID,
			Units:  ord.Units,

			ParentOrderID:     ord.ParentOrderID,
			ParentOrderGoodID: ord.ParentOrderGoodID,

			PaymentID:               ord.PaymentID,
			PaymentCoinTypeID:       ord.PaymentCoinTypeID,
			PaymentCoinUSDCurrency:  ord.PaymentCoinUSDCurrency,
			PaymentLiveUSDCurrency:  ord.PaymentLiveUSDCurrency,
			PaymentLocalUSDCurrency: ord.PaymentLocalUSDCurrency,
			PaymentAmount:           ord.PaymentAmount,
			PaymentStartAmount:      ord.PaymentStartAmount,
			PaymentFinishAmount:     ord.PaymentFinishAmount,
			PayWithParent:           ord.PayWithParent,
			PayWithBalanceAmount:    ord.PayWithBalanceAmount,

			FixAmountID:    ord.FixAmountID,
			DiscountID:     ord.DiscountID,
			SpecialOfferID: ord.SpecialOfferID,

			OrderType: ord.OrderType,
			CreatedAt: ord.CreatedAt,
			PaidAt:    ord.PaidAt,
			State:     ord.OrderState,

			Start: ord.Start,
			End:   ord.End,
		}

		user, ok := userMap[ord.UserID]
		if !ok {
			logger.Sugar().Warnw("expand", "UserID", ord.UserID, "OrderID", ord.ID)
		}

		if user != nil {
			o.EmailAddress = user.EmailAddress
			o.PhoneNO = user.PhoneNO
		}

		appGood, ok := appGoodMap[ord.AppID+ord.GoodID]
		if !ok {
			continue
		}

		o.CoinTypeID = appGood.CoinTypeID
		o.GoodName = appGood.GoodName
		o.GoodUnit = appGood.Unit
		o.GoodServicePeriodDays = uint32(appGood.DurationDays)
		o.GoodUnitPrice = appGood.Price

		appGoodPrice, err := decimal.NewFromString(appGood.Price)
		if err != nil {
			return nil, err
		}

		o.GoodValue = appGoodPrice.Mul(decimal.NewFromInt32(int32(ord.Units))).String()
		if appGood.PromotionPrice != nil {
			appGoodPromotionPrice, err := decimal.NewFromString(*appGood.PromotionPrice)
			if err != nil {
				return nil, err
			}
			o.GoodValue = appGoodPromotionPrice.Mul(decimal.NewFromInt32(int32(ord.Units))).String()
		}

		coin, ok := coinMap[o.CoinTypeID]
		if !ok {
			return nil, fmt.Errorf("invalid coin")
		}

		o.CoinName = coin.Name
		o.CoinLogo = coin.Logo
		o.CoinUnit = coin.Unit

		if ord.PaymentID != invalidID && ord.PaymentID != "" {
			coin, ok = coinMap[ord.PaymentCoinTypeID]
			if !ok {
				return nil, fmt.Errorf("invalid payment coin")
			}
		}

		if coin != nil {
			o.PaymentCoinName = coin.Name
			o.PaymentCoinLogo = coin.Logo
			o.PaymentCoinUnit = coin.Unit
		}

		acc, ok := accMap[ord.PaymentAccountID]
		if ok {
			o.PaymentAddress = acc.Address
		}

		if coupon, ok := fixAmountMap[ord.FixAmountID]; ok {
			o.FixAmountName = coupon.Name
			o.FixAmountAmount = coupon.Value
		}
		if coupon, ok := discountMap[ord.DiscountID]; ok {
			o.DiscountName = coupon.Name
			percent, err := decimal.NewFromString(coupon.Value)
			if err != nil {
				return nil, err
			}
			o.DiscountPercent = uint32(percent.IntPart())
		}
		if coupon, ok := specialOfferMap[ord.SpecialOfferID]; ok {
			o.SpecialOfferAmount = coupon.Value
		}

		infos = append(infos, o)
	}

	return infos, nil
}
