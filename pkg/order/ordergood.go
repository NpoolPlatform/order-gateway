package order

import (
	"context"
	"fmt"

	cruder "github.com/NpoolPlatform/libent-cruder/pkg/cruder"
	basetypes "github.com/NpoolPlatform/message/npool/basetypes/v1"

	coininfocli "github.com/NpoolPlatform/chain-middleware/pkg/client/coin"
	appgoodmwcli "github.com/NpoolPlatform/good-middleware/pkg/client/app/good"
	topmostgoodmwcli "github.com/NpoolPlatform/good-middleware/pkg/client/app/good/topmost/good"
	goodmwcli "github.com/NpoolPlatform/good-middleware/pkg/client/good"
	goodrequiredmwcli "github.com/NpoolPlatform/good-middleware/pkg/client/good/required"
	coinpb "github.com/NpoolPlatform/message/npool/chain/mw/v1/coin"
	appgoodpb "github.com/NpoolPlatform/message/npool/good/mw/v1/app/good"
	topmostgoodpb "github.com/NpoolPlatform/message/npool/good/mw/v1/app/good/topmost/good"
	goodpb "github.com/NpoolPlatform/message/npool/good/mw/v1/good"
	goodrequiredpb "github.com/NpoolPlatform/message/npool/good/mw/v1/good/required"
	npool "github.com/NpoolPlatform/message/npool/order/gw/v1/order"

	ordertypes "github.com/NpoolPlatform/message/npool/basetypes/order/v1"
	ordermwpb "github.com/NpoolPlatform/message/npool/order/mw/v1/order"
	ordermwcli "github.com/NpoolPlatform/order-middleware/pkg/client/order"

	"github.com/shopspring/decimal"
)

type OrderGood struct {
	goods         map[string]*goodpb.Good
	appgoods      map[string]*appgoodpb.Good
	goodCoins     map[string]*coinpb.Coin
	topMostGoods  map[string]*topmostgoodpb.TopMostGood
	goodRequireds []*goodrequiredpb.Required
}

func (h *Handler) ToOrderGood(ctx context.Context) (*OrderGood, error) {
	parent := true
	if h.ParentOrderID != nil {
		parent = false
	}
	goodReq := &npool.CreateOrdersRequest_Good{
		GoodID: *h.GoodID,
		Units:  h.Units,
		Parent: parent,
	}
	h.Goods = append(h.Goods, goodReq)
	return h.ToOrderGoods(ctx)
}

//nolint:funlen,gocyclo
func (h *Handler) ToOrderGoods(ctx context.Context) (*OrderGood, error) {
	ordergood := &OrderGood{
		goods:        map[string]*goodpb.Good{},
		appgoods:     map[string]*appgoodpb.Good{},
		goodCoins:    map[string]*coinpb.Coin{},
		topMostGoods: map[string]*topmostgoodpb.TopMostGood{},
	}
	parentGoodNum := 0
	for _, goodReq := range h.Goods {
		if goodReq.Parent {
			parentGoodNum++
			if parentGoodNum != 1 {
				return nil, fmt.Errorf("invalid parent")
			}
			goodRequireds, _, err := goodrequiredmwcli.GetRequireds(ctx, &goodrequiredpb.Conds{
				MainGoodID: &basetypes.StringVal{Op: cruder.EQ, Value: goodReq.GoodID},
			}, 0, 0)
			if err != nil {
				return nil, err
			}
			ordergood.goodRequireds = goodRequireds
		}
		good, err := goodmwcli.GetGood(ctx, goodReq.GoodID)
		if err != nil {
			return nil, err
		}
		if good == nil {
			return nil, fmt.Errorf("invalid good")
		}
		ordergood.goods[goodReq.GoodID] = good

		appgood, err := appgoodmwcli.GetGoodOnly(ctx, &appgoodpb.Conds{
			AppID:  &basetypes.StringVal{Op: cruder.EQ, Value: *h.AppID},
			GoodID: &basetypes.StringVal{Op: cruder.EQ, Value: goodReq.GoodID},
		})
		if err != nil {
			return nil, err
		}
		if appgood == nil {
			return nil, fmt.Errorf("invalid app good")
		}
		if !appgood.Online {
			return nil, fmt.Errorf("good offline")
		}

		agPrice, err := decimal.NewFromString(appgood.Price)
		if err != nil {
			return nil, err
		}
		if agPrice.Cmp(decimal.NewFromInt(0)) <= 0 {
			return nil, fmt.Errorf("invalid good price")
		}
		price, err := decimal.NewFromString(good.Price)
		if err != nil {
			return nil, err
		}
		if agPrice.Cmp(price) < 0 {
			return nil, fmt.Errorf("invalid app good price")
		}

		if *h.OrderType == ordertypes.OrderType_Normal {
			units, err := decimal.NewFromString(h.Units)
			if err != nil {
				return nil, err
			}
			if appgood.PurchaseLimit > 0 && units.Cmp(decimal.NewFromInt32(appgood.PurchaseLimit)) > 0 {
				return nil, fmt.Errorf("too many units")
			}
			if !appgood.EnablePurchase {
				return nil, fmt.Errorf("app good is not enabled purchase")
			}
			purchaseCountStr, err := ordermwcli.SumOrderUnits(
				ctx,
				&ordermwpb.Conds{
					AppID:  &basetypes.StringVal{Op: cruder.EQ, Value: *h.AppID},
					UserID: &basetypes.StringVal{Op: cruder.EQ, Value: *h.UserID},
					GoodID: &basetypes.StringVal{Op: cruder.EQ, Value: *h.GoodID},
					OrderStates: &basetypes.Uint32SliceVal{
						Op: cruder.IN,
						Value: []uint32{
							uint32(ordertypes.OrderState_OrderStatePaid),
							uint32(ordertypes.OrderState_OrderStateInService),
							uint32(ordertypes.OrderState_OrderStateExpired),
							uint32(ordertypes.OrderState_OrderStateWaitPayment),
						},
					},
				},
			)
			if err != nil {
				return nil, err
			}
			purchaseCount, err := decimal.NewFromString(purchaseCountStr)
			if err != nil {
				return nil, err
			}

			userPurchaseLimit, err := decimal.NewFromString(appgood.UserPurchaseLimit)
			if err != nil {
				return nil, err
			}

			if userPurchaseLimit.Cmp(decimal.NewFromInt(0)) > 0 && purchaseCount.Add(units).Cmp(userPurchaseLimit) > 0 {
				return nil, fmt.Errorf("too many units")
			}
		}

		goodCoin, err := coininfocli.GetCoin(ctx, good.CoinTypeID)
		if err != nil {
			return nil, err
		}
		if goodCoin == nil {
			return nil, fmt.Errorf("invalid good coin")
		}
		ordergood.goodCoins[goodReq.GoodID] = goodCoin

		topMostGood, err := topmostgoodmwcli.GetTopMostGoodOnly(ctx, &topmostgoodpb.Conds{
			AppID:  &basetypes.StringVal{Op: cruder.EQ, Value: *h.AppID},
			GoodID: &basetypes.StringVal{Op: cruder.EQ, Value: goodReq.GoodID},
		})
		if err != nil {
			return nil, err
		}
		if topMostGood != nil {
			ordergood.topMostGoods[*h.AppID+goodReq.GoodID] = topMostGood
		}
	}

	return ordergood, nil
}
