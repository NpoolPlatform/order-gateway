package common

import (
	"context"
	"fmt"

	appmwcli "github.com/NpoolPlatform/appuser-middleware/pkg/client/app"
	usermwcli "github.com/NpoolPlatform/appuser-middleware/pkg/client/user"
	appcoinmwcli "github.com/NpoolPlatform/chain-middleware/pkg/client/app/coin"
	currencymwcli "github.com/NpoolPlatform/chain-middleware/pkg/client/coin/currency"
	appgoodmwcli "github.com/NpoolPlatform/good-middleware/pkg/client/app/good"
	requiredappgoodmwcli "github.com/NpoolPlatform/good-middleware/pkg/client/app/good/required"
	allocatedcouponmwcli "github.com/NpoolPlatform/inspire-middleware/pkg/client/coupon/allocated"
	appgoodscopemwcli "github.com/NpoolPlatform/inspire-middleware/pkg/client/coupon/app/scope"
	"github.com/NpoolPlatform/libent-cruder/pkg/cruder"
	appmwpb "github.com/NpoolPlatform/message/npool/appuser/mw/v1/app"
	usermwpb "github.com/NpoolPlatform/message/npool/appuser/mw/v1/user"
	inspiretypes "github.com/NpoolPlatform/message/npool/basetypes/inspire/v1"
	types "github.com/NpoolPlatform/message/npool/basetypes/order/v1"
	basetypes "github.com/NpoolPlatform/message/npool/basetypes/v1"
	appcoinmwpb "github.com/NpoolPlatform/message/npool/chain/mw/v1/app/coin"
	currencymwpb "github.com/NpoolPlatform/message/npool/chain/mw/v1/coin/currency"
	appgoodmwpb "github.com/NpoolPlatform/message/npool/good/mw/v1/app/good"
	requiredappgoodmwpb "github.com/NpoolPlatform/message/npool/good/mw/v1/app/good/required"
	allocatedcouponmwpb "github.com/NpoolPlatform/message/npool/inspire/mw/v1/coupon/allocated"
	appgoodscopemwpb "github.com/NpoolPlatform/message/npool/inspire/mw/v1/coupon/app/scope"
	paymentgwpb "github.com/NpoolPlatform/message/npool/order/gw/v1/payment"
	orderappconfigmwpb "github.com/NpoolPlatform/message/npool/order/mw/v1/app/config"
	feeordermwpb "github.com/NpoolPlatform/message/npool/order/mw/v1/fee"
	paymentmwpb "github.com/NpoolPlatform/message/npool/order/mw/v1/payment"
	powerrentalmwpb "github.com/NpoolPlatform/message/npool/order/mw/v1/powerrental"
	ordergwcommon "github.com/NpoolPlatform/order-gateway/pkg/common"
	constant "github.com/NpoolPlatform/order-gateway/pkg/const"
	orderappconfigmwcli "github.com/NpoolPlatform/order-middleware/pkg/client/app/config"
	feeordermwcli "github.com/NpoolPlatform/order-middleware/pkg/client/fee"
	powerrentalmwcli "github.com/NpoolPlatform/order-middleware/pkg/client/powerrental"
)

type OrderCreateHandler struct {
	ordergwcommon.AppGoodCheckHandler
	ordergwcommon.CoinCheckHandler
	ordergwcommon.AllocatedCouponCheckHandler
	DurationSeconds           *uint32
	PaymentBalances           []*paymentgwpb.PaymentBalance
	PaymentTransferCoinTypeID *string
	AllocatedCouponIDs        []string
	AppGoodIDs                []string

	allocatedCoupons  map[string]*allocatedcouponmwpb.Coupon
	coinUSDCurrencies map[string]*currencymwpb.Currency
	AppGoods          map[string]*appgoodmwpb.Good

	PaymentBalanceReqs  []*paymentmwpb.PaymentBalanceReq
	PaymentTransferReqs []*paymentmwpb.PaymentTransferReq

	OrderConfig      *orderappconfigmwpb.AppConfig
	App              *appmwpb.App
	User             *usermwpb.User
	AppCoins         map[string]*appcoinmwpb.Coin
	RequiredAppGoods map[string]map[string]*requiredappgoodmwpb.Required
}

func (h *OrderCreateHandler) GetAppConfig(ctx context.Context) (err error) {
	h.OrderConfig, err = orderappconfigmwcli.GetAppConfig(ctx, *h.AppGoodCheckHandler.AppID)
	return err
}

func (h *OrderCreateHandler) GetAllocatedCoupons(ctx context.Context) error {
	infos, _, err := allocatedcouponmwcli.GetCoupons(ctx, &allocatedcouponmwpb.Conds{
		AppID:  &basetypes.StringVal{Op: cruder.EQ, Value: *h.AppGoodCheckHandler.AppID},
		UserID: &basetypes.StringVal{Op: cruder.EQ, Value: *h.AppGoodCheckHandler.UserID},
		EntIDs: &basetypes.StringSliceVal{Op: cruder.IN, Value: h.AllocatedCouponIDs},
	}, 0, int32(len(h.AllocatedCouponIDs)))
	if err != nil {
		return err
	}
	if len(infos) != len(h.AllocatedCouponIDs) {
		return fmt.Errorf("invalid allocatedcoupons")
	}
	h.allocatedCoupons = map[string]*allocatedcouponmwpb.Coupon{}
	for _, info := range infos {
		h.allocatedCoupons[info.EntID] = info
	}
	return nil
}

func (h *OrderCreateHandler) GetAppCoins(ctx context.Context, parentGoodCoinTypeIDs []string) error {
	coinTypeIDs := func() (_coinTypeIDs []string) {
		for _, balance := range h.PaymentBalances {
			_coinTypeIDs = append(_coinTypeIDs, balance.CoinTypeID)
		}
		return
	}()
	coinTypeIDs = append(coinTypeIDs, parentGoodCoinTypeIDs...)
	if h.PaymentTransferCoinTypeID != nil {
		coinTypeIDs = append(coinTypeIDs, *h.PaymentTransferCoinTypeID)
	}
	coins, _, err := appcoinmwcli.GetCoins(ctx, &appcoinmwpb.Conds{
		AppID:  &basetypes.StringVal{Op: cruder.EQ, Value: *h.AppGoodCheckHandler.AppID},
		EntIDs: &basetypes.StringSliceVal{Op: cruder.IN, Value: coinTypeIDs},
	}, 0, int32(len(coinTypeIDs)))
	if err != nil {
		return err
	}
	h.AppCoins = map[string]*appcoinmwpb.Coin{}
	coinENV := ""
	for _, coin := range coins {
		if coinENV != "" && coin.ENV != coinENV {
			return fmt.Errorf("invalid appcoins")
		}
		h.AppCoins[coin.CoinTypeID] = coin
	}
	return nil
}

func (h *OrderCreateHandler) GetCoinUSDCurrencies(ctx context.Context) error {
	coinTypeIDs := func() (_coinTypeIDs []string) {
		for _, balance := range h.PaymentBalances {
			_coinTypeIDs = append(_coinTypeIDs, balance.CoinTypeID)
		}
		return
	}()
	if h.PaymentTransferCoinTypeID != nil {
		coinTypeIDs = append(coinTypeIDs, *h.PaymentTransferCoinTypeID)
	}
	infos, _, err := currencymwcli.GetCurrencies(ctx, &currencymwpb.Conds{
		CoinTypeIDs: &basetypes.StringSliceVal{Op: cruder.IN, Value: coinTypeIDs},
	}, 0, int32(len(coinTypeIDs)))
	if err != nil {
		return err
	}
	h.coinUSDCurrencies = map[string]*currencymwpb.Currency{}
	for _, info := range infos {
		h.coinUSDCurrencies[info.CoinTypeID] = info
	}
	return nil
}

func (h *OrderCreateHandler) GetAppGoods(ctx context.Context) error {
	appGoods, _, err := appgoodmwcli.GetGoods(ctx, &appgoodmwpb.Conds{
		AppID:  &basetypes.StringVal{Op: cruder.EQ, Value: *h.AppGoodCheckHandler.AppID},
		EntIDs: &basetypes.StringSliceVal{Op: cruder.IN, Value: h.AppGoodIDs},
	}, 0, int32(len(h.AppGoodIDs)))
	if err != nil {
		return err
	}
	if len(appGoods) != len(h.AppGoodIDs) {
		return fmt.Errorf("invalid appgoods")
	}
	h.AppGoods = map[string]*appgoodmwpb.Good{}
	for _, appGood := range appGoods {
		h.AppGoods[appGood.EntID] = appGood
	}
	return nil
}

func (h *OrderCreateHandler) GetApp(ctx context.Context) error {
	app, err := appmwcli.GetApp(ctx, *h.AppGoodCheckHandler.AppID)
	if err != nil {
		return err
	}
	if app == nil {
		return fmt.Errorf("invalid app")
	}
	h.App = app
	return nil
}

func (h *OrderCreateHandler) GetUser(ctx context.Context) error {
	user, err := usermwcli.GetUser(ctx, *h.AppGoodCheckHandler.AppID, *h.AppGoodCheckHandler.UserID)
	if err != nil {
		return err
	}
	if user == nil {
		return fmt.Errorf("invalid user")
	}
	h.User = user
	return nil
}

func (h *OrderCreateHandler) ValidateCouponScope(ctx context.Context, parentAppGoodID *string) error {
	reqs := []*appgoodscopemwpb.ScopeReq{}
	for _, allocatedCoupon := range h.allocatedCoupons {
		for appGoodID, appGood := range h.AppGoods {
			if parentAppGoodID != nil && *parentAppGoodID == appGoodID {
				continue
			}
			reqs = append(reqs, &appgoodscopemwpb.ScopeReq{
				AppID:       h.AppGoodCheckHandler.AppID,
				AppGoodID:   &appGoodID,
				GoodID:      &appGood.GoodID,
				CouponID:    &allocatedCoupon.CouponID,
				CouponScope: &allocatedCoupon.CouponScope,
			})
		}
	}
	return appgoodscopemwcli.VerifyCouponScopes(ctx, reqs)
}

func (h *OrderCreateHandler) ValidateCouponCount() error {
	discountCoupons := 0
	fixAmountCoupons := uint32(0)
	for _, coupon := range h.allocatedCoupons {
		switch coupon.CouponType {
		case inspiretypes.CouponType_Discount:
			discountCoupons++
			if discountCoupons > 1 {
				return fmt.Errorf("invalid discountcoupon")
			}
		case inspiretypes.CouponType_FixAmount:
			fixAmountCoupons++
			if h.OrderConfig == nil || h.OrderConfig.MaxTypedCouponsPerOrder == 0 {
				continue
			}
			if fixAmountCoupons > h.OrderConfig.MaxTypedCouponsPerOrder {
				return fmt.Errorf("invalid fixamountcoupon")
			}
		}
	}
	return nil
}

func (h *OrderCreateHandler) ValidateMaxUnpaidOrders(ctx context.Context) error {
	if h.OrderConfig == nil || h.OrderConfig.MaxUnpaidOrders == 0 {
		return nil
	}
	powerRentals, err := powerrentalmwcli.CountPowerRentalOrders(ctx, &powerrentalmwpb.Conds{
		AppID:        &basetypes.StringVal{Op: cruder.EQ, Value: *h.AppGoodCheckHandler.AppID},
		UserID:       &basetypes.StringVal{Op: cruder.EQ, Value: *h.AppGoodCheckHandler.UserID},
		OrderType:    &basetypes.Uint32Val{Op: cruder.EQ, Value: uint32(types.OrderType_Normal)},
		PaymentState: &basetypes.Uint32Val{Op: cruder.IN, Value: uint32(types.PaymentState_PaymentStateWait)},
	})
	if err != nil {
		return err
	}
	feeOrders, err := feeordermwcli.CountFeeOrders(ctx, &feeordermwpb.Conds{
		AppID:        &basetypes.StringVal{Op: cruder.EQ, Value: *h.AppGoodCheckHandler.AppID},
		UserID:       &basetypes.StringVal{Op: cruder.EQ, Value: *h.AppGoodCheckHandler.UserID},
		OrderType:    &basetypes.Uint32Val{Op: cruder.EQ, Value: uint32(types.OrderType_Normal)},
		PaymentState: &basetypes.Uint32Val{Op: cruder.IN, Value: uint32(types.PaymentState_PaymentStateWait)},
	})
	if err != nil {
		return err
	}
	if powerRentals+feeOrders >= h.OrderConfig.MaxUnpaidOrders {
		return fmt.Errorf("too many unpaid orders")
	}
	return nil
}

func (h *OrderCreateHandler) GetRequiredAppGoods(ctx context.Context) error {
	offset := int32(0)
	limit := int32(constant.DefaultRowLimit)

	for {
		requiredAppGoods, _, err := requiredappgoodmwcli.GetRequireds(ctx, &requiredappgoodmwpb.Conds{
			AppGoodIDs: &basetypes.StringSliceVal{Op: cruder.IN, Value: h.AppGoodIDs},
		}, offset, limit)
		if err != nil {
			return err
		}
		if len(requiredAppGoods) == 0 {
			return nil
		}
		h.RequiredAppGoods = map[string]map[string]*requiredappgoodmwpb.Required{}
		for _, requiredAppGood := range requiredAppGoods {
			requireds, ok := h.RequiredAppGoods[requiredAppGood.MainAppGoodID]
			if !ok {
				requireds = map[string]*requiredappgoodmwpb.Required{}
			}
			requireds[requiredAppGood.RequiredAppGoodID] = requiredAppGood
			h.RequiredAppGoods[requiredAppGood.MainAppGoodID] = requireds
		}
		offset += limit
	}
}
