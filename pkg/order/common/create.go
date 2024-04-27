package common

import (
	"context"

	currencymwcli "github.com/NpoolPlatform/chain-middleware/pkg/client/coin/currency"
	appgoodmwcli "github.com/NpoolPlatform/good-middleware/pkg/client/app/good"
	allocatedcouponmwcli "github.com/NpoolPlatform/inspire-middleware/pkg/client/coupon/allocated"
	"github.com/NpoolPlatform/libent-cruder/pkg/cruder"
	basetypes "github.com/NpoolPlatform/message/npool/basetypes/v1"
	currencymwpb "github.com/NpoolPlatform/message/npool/chain/mw/v1/coin/currency"
	appgoodmwpb "github.com/NpoolPlatform/message/npool/good/mw/v1/app/good"
	allocatedcouponmwpb "github.com/NpoolPlatform/message/npool/inspire/mw/v1/coupon/allocated"
	paymentgwpb "github.com/NpoolPlatform/message/npool/order/gw/v1/payment"
	paymentmwpb "github.com/NpoolPlatform/message/npool/order/mw/v1/payment"
	ordergwcommon "github.com/NpoolPlatform/order-gateway/pkg/common"
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
	appGoods          map[string]*appgoodmwpb.Good

	PaymentBalanceReqs  []*paymentmwpb.PaymentBalanceReq
	PaymentTransferReqs []*paymentmwpb.PaymentTransferReq
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
	h.allocatedCoupons = map[string]*allocatedcouponmwpb.Coupon{}
	for _, info := range infos {
		h.allocatedCoupons[info.EntID] = info
	}
	return nil
}

func (h *OrderCreateHandler) getCoinUSDCurrencies(ctx context.Context) error {
	coinTypeIDs := func() (_coinTypeIDs []string) {
		for _, balance := range h.PaymentBalances {
			_coinTypeIDs = append(_coinTypeIDs, balance.CoinTypeID)
		}
		return
	}()
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

func (h *OrderCreateHandler) getAppGoods(ctx context.Context) error {
	appGoods, _, err := appgoodmwcli.GetGoods(ctx, &appgoodmwpb.Conds{
		AppID:  &basetypes.StringVal{Op: cruder.EQ, Value: *h.AppGoodCheckHandler.AppID},
		EntIDs: &basetypes.StringSliceVal{Op: cruder.IN, Value: h.AppGoodIDs},
	}, 0, int32(len(h.AppGoodIDs)))
	if err != nil {
		return err
	}
	h.appGoods = map[string]*appgoodmwpb.Good{}
	for _, appGood := range appGoods {
		h.appGoods[appGood.EntID] = appGood
	}
	return nil
}
