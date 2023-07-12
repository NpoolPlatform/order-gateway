//nolint:dupl
package order

import (
	"context"
	"fmt"

	"github.com/NpoolPlatform/go-service-framework/pkg/logger"

	payaccmwcli "github.com/NpoolPlatform/account-middleware/pkg/client/payment"
	usermwcli "github.com/NpoolPlatform/appuser-middleware/pkg/client/user"
	appcoinmwcli "github.com/NpoolPlatform/chain-middleware/pkg/client/app/coin"
	coininfocli "github.com/NpoolPlatform/chain-middleware/pkg/client/coin"
	allocatedmwcli "github.com/NpoolPlatform/inspire-middleware/pkg/client/coupon/allocated"
	npool "github.com/NpoolPlatform/message/npool/order/gw/v1/order"
	ordercli "github.com/NpoolPlatform/order-middleware/pkg/client/order"

	usermwpb "github.com/NpoolPlatform/message/npool/appuser/mw/v1/user"

	payaccmwpb "github.com/NpoolPlatform/message/npool/account/mw/v1/payment"
	appcoinmwpb "github.com/NpoolPlatform/message/npool/chain/mw/v1/app/coin"
	allocatedmgrpb "github.com/NpoolPlatform/message/npool/inspire/mgr/v1/coupon/allocated"
	allocatedmwpb "github.com/NpoolPlatform/message/npool/inspire/mw/v1/coupon/allocated"
	ordermwpb "github.com/NpoolPlatform/message/npool/order/mw/v1/order"

	appgoodscli "github.com/NpoolPlatform/good-middleware/pkg/client/appgood"
	appgoodspb "github.com/NpoolPlatform/message/npool/good/mw/v1/appgood"

	appgoodsmgrpb "github.com/NpoolPlatform/message/npool/good/mgr/v1/appgood"

	cruder "github.com/NpoolPlatform/libent-cruder/pkg/cruder"

	"github.com/shopspring/decimal"

	commonpb "github.com/NpoolPlatform/message/npool"
	basetypes "github.com/NpoolPlatform/message/npool/basetypes/v1"

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

	user, err := usermwcli.GetUser(ctx, ord.AppID, ord.UserID)
	if err != nil {
		return nil, err
	}

	if user == nil {
		return nil, fmt.Errorf("invalid user")
	}

	o.EmailAddress = user.EmailAddress
	o.PhoneNO = user.PhoneNO

	appGood, err := appgoodscli.GetGoodOnly(ctx, &appgoodsmgrpb.Conds{
		AppID: &commonpb.StringVal{
			Op:    cruder.EQ,
			Value: ord.AppID,
		},
		GoodID: &commonpb.StringVal{
			Op:    cruder.EQ,
			Value: ord.GoodID,
		},
	})
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
	units, err := decimal.NewFromString(ord.Units)
	if err != nil {
		return nil, err
	}
	o.GoodValue = appGoodPrice.Mul(units).String()
	if appGood.PromotionPrice != nil {
		appGoodPromotionPrice, err := decimal.NewFromString(*appGood.PromotionPrice)
		if err != nil {
			return nil, err
		}
		o.GoodValue = appGoodPromotionPrice.Mul(units).String()
	}

	coin, err := coininfocli.GetCoin(ctx, appGood.CoinTypeID)
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
	o.CoinPresale = coin.Presale

	if ord.PaymentID != invalidID && ord.PaymentID != "" {
		coin, err = coininfocli.GetCoin(ctx, ord.PaymentCoinTypeID)
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

	account, err := payaccmwcli.GetAccountOnly(ctx, &payaccmwpb.Conds{
		AccountID: &basetypes.StringVal{
			Op:    cruder.EQ,
			Value: ord.PaymentAccountID,
		},
	})
	if err != nil {
		return nil, err
	}
	if account != nil {
		o.PaymentAddress = account.Address
	}

	coupons, _, err := allocatedmwcli.GetCoupons(ctx, &allocatedmgrpb.Conds{
		AppID: &commonpb.StringVal{
			Op:    cruder.EQ,
			Value: ord.AppID,
		},
		IDs: &commonpb.StringSliceVal{
			Op:    cruder.IN,
			Value: ord.CouponIDs,
		},
	}, int32(0), int32(len(ord.CouponIDs)))
	if err != nil {
		return nil, err
	}

	couponMap := map[string]*allocatedmwpb.Coupon{}
	for _, coup := range coupons {
		couponMap[coup.ID] = coup
	}

	for _, id := range ord.CouponIDs {
		coup, ok := couponMap[id]
		if !ok {
			continue
		}

		o.Coupons = append(o.Coupons, &npool.Coupon{
			CouponID:    id,
			CouponType:  coup.CouponType,
			CouponName:  coup.CouponName,
			CouponValue: coup.Value,
		})
	}

	return o, nil
}

func GetOrders(ctx context.Context, appID, userID string, offset, limit int32) ([]*npool.Order, uint32, error) {
	ords, total, err := ordercli.GetOrders(ctx, &ordermwpb.Conds{
		AppID: &commonpb.StringVal{
			Op:    cruder.EQ,
			Value: appID,
		},
		UserID: &commonpb.StringVal{
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

	orders, err := expand(ctx, ords, appID)
	if err != nil {
		return nil, 0, err
	}

	return orders, total, nil
}

func GetAppOrders(ctx context.Context, appID string, offset, limit int32) ([]*npool.Order, uint32, error) {
	ords, total, err := ordercli.GetOrders(ctx, &ordermwpb.Conds{
		AppID: &commonpb.StringVal{
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

	orders, err := expand(ctx, ords, appID)
	if err != nil {
		return nil, 0, err
	}

	return orders, total, nil
}

// nolint
func expand(ctx context.Context, ords []*ordermwpb.Order, appID string) ([]*npool.Order, error) {
	if len(ords) == 0 {
		return []*npool.Order{}, nil
	}

	uids := []string{}
	for _, ord := range ords {
		uids = append(uids, ord.UserID)
	}

	users, _, err := usermwcli.GetUsers(ctx, &usermwpb.Conds{
		IDs: &basetypes.StringSliceVal{Op: cruder.IN, Value: uids},
	}, 0, int32(len(uids)))
	if err != nil {
		return nil, err
	}
	if len(users) == 0 {
		return nil, fmt.Errorf("invalid users")
	}

	userMap := map[string]*usermwpb.User{}
	for _, user := range users {
		userMap[user.ID] = user
	}

	goodIDs := []string{}
	for _, val := range ords {
		goodIDs = append(goodIDs, val.GetGoodID())
	}

	accIDs := []string{}
	for _, ord := range ords {
		if _, err := uuid.Parse(ord.PaymentAccountID); err != nil {
			continue
		}
		accIDs = append(accIDs, ord.PaymentAccountID)
	}

	accounts, _, err := payaccmwcli.GetAccounts(ctx, &payaccmwpb.Conds{
		AccountIDs: &basetypes.StringSliceVal{
			Op:    cruder.IN,
			Value: accIDs,
		},
	}, 0, int32(len(accIDs)))
	if err != nil {
		return nil, err
	}

	accMap := map[string]*payaccmwpb.Account{}
	for _, acc := range accounts {
		accMap[acc.AccountID] = acc
	}

	ids := []string{}
	for _, ord := range ords {
		ids = append(ids, ord.CouponIDs...)
	}

	coupons, _, err := allocatedmwcli.GetCoupons(ctx, &allocatedmgrpb.Conds{
		AppID: &commonpb.StringVal{
			Op:    cruder.EQ,
			Value: appID,
		},
		IDs: &commonpb.StringSliceVal{
			Op:    cruder.IN,
			Value: ids,
		},
	}, int32(0), int32(len(ids)))
	if err != nil {
		return nil, err
	}

	couponMap := map[string]*allocatedmwpb.Coupon{}
	for _, coup := range coupons {
		couponMap[coup.ID] = coup
	}

	appGoods, _, err := appgoodscli.GetGoods(ctx, &appgoodsmgrpb.Conds{
		GoodIDs: &commonpb.StringSliceVal{
			Op:    cruder.IN,
			Value: goodIDs,
		},
		AppID: &commonpb.StringVal{
			Op:    cruder.IN,
			Value: appID,
		},
	}, 0, int32(len(goodIDs)))
	if err != nil {
		return nil, err
	}

	fmt.Printf("goodIDs: %v, appID %v, appGoods %v | %v\n", goodIDs, appID, len(appGoods), appGoods)

	appGoodMap := map[string]*appgoodspb.Good{}
	for _, appGood := range appGoods {
		appGoodMap[appGood.AppID+appGood.GoodID] = appGood
	}

	coinTypeIDs := []string{}
	for _, val := range ords {
		if _, err := uuid.Parse(val.PaymentCoinTypeID); err != nil {
			continue
		}
		coinTypeIDs = append(coinTypeIDs, val.PaymentCoinTypeID)
	}
	for _, val := range appGoods {
		if _, err := uuid.Parse(val.CoinTypeID); err != nil {
			continue
		}
		coinTypeIDs = append(coinTypeIDs, val.CoinTypeID)
	}

	coins, _, err := appcoinmwcli.GetCoins(ctx, &appcoinmwpb.Conds{
		AppID: &basetypes.StringVal{
			Op:    cruder.EQ,
			Value: appID,
		},
		CoinTypeIDs: &basetypes.StringSliceVal{
			Op:    cruder.IN,
			Value: coinTypeIDs,
		},
	}, 0, int32(len(coinTypeIDs)))
	if err != nil {
		return nil, err
	}

	coinMap := map[string]*appcoinmwpb.Coin{}
	for _, coin := range coins {
		coinMap[coin.CoinTypeID] = coin
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
			logger.Sugar().Warnw("expand", "AppID", ord.AppID, "GoodID", ord.GoodID)
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

		units, err := decimal.NewFromString(ord.Units)
		if err != nil {
			return nil, err
		}
		o.GoodValue = appGoodPrice.Mul(units).String()
		if appGood.PromotionPrice != nil {
			appGoodPromotionPrice, err := decimal.NewFromString(*appGood.PromotionPrice)
			if err != nil {
				return nil, err
			}
			o.GoodValue = appGoodPromotionPrice.Mul(units).String()
		}

		coin, ok := coinMap[o.CoinTypeID]
		if !ok {
			logger.Sugar().Warnw("expand", "AppID", o.AppID, "CoinTypeID", o.CoinTypeID)
			continue
		}

		o.CoinName = coin.Name
		o.CoinLogo = coin.Logo
		o.CoinUnit = coin.Unit

		if ord.PaymentID != invalidID && ord.PaymentID != "" {
			coin, ok = coinMap[ord.PaymentCoinTypeID]
			if !ok {
				logger.Sugar().Warnw("expand", "AppID", o.AppID, "PaymentCoinTypeID", o.PaymentCoinTypeID)
				continue
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

		for _, id := range ord.CouponIDs {
			coup, ok := couponMap[id]
			if !ok {
				continue
			}

			o.Coupons = append(o.Coupons, &npool.Coupon{
				CouponID:    id,
				CouponType:  coup.CouponType,
				CouponName:  coup.CouponName,
				CouponValue: coup.Value,
			})
		}

		infos = append(infos, o)
	}

	return infos, nil
}
