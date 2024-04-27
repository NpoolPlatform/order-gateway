package fee

import (
	"context"
	"fmt"

	goodcoinmwcli "github.com/NpoolPlatform/good-middleware/pkg/client/good/coin"
	cruder "github.com/NpoolPlatform/libent-cruder/pkg/cruder"
	basetypes "github.com/NpoolPlatform/message/npool/basetypes/v1"
	appgoodmwpb "github.com/NpoolPlatform/message/npool/good/mw/v1/app/good"
	goodcoinmwpb "github.com/NpoolPlatform/message/npool/good/mw/v1/good/coin"
	ordermwpb "github.com/NpoolPlatform/message/npool/order/mw/v1/order"
	constant "github.com/NpoolPlatform/order-gateway/pkg/const"
	ordercommon "github.com/NpoolPlatform/order-gateway/pkg/order/common"
	ordermwcli "github.com/NpoolPlatform/order-middleware/pkg/client/order"
)

type baseCreateHandler struct {
	*Handler
	*ordercommon.OrderCreateHandler
	parentOrder     *ordermwpb.Order
	parentAppGood   *appgoodmwpb.Good
	parentGoodCoins []*goodcoinmwpb.GoodCoin
}

func (h *baseCreateHandler) getParentOrder(ctx context.Context) error {
	info, err := ordermwcli.GetOrder(ctx, *h.ParentOrderID)
	if err != nil {
		return err
	}
	if info == nil {
		return fmt.Errorf("invalid parentorder")
	}
	h.parentOrder = info
	return nil
}

func (h *baseCreateHandler) getAppGoods(ctx context.Context) error {
	h.OrderCreateHandler.AppGoodIDs = append(h.OrderCreateHandler.AppGoodIDs, h.parentOrder.AppGoodID)
	if err := h.GetAppGoods(ctx); err != nil {
		return err
	}
	for appGoodID, appGood := range h.AppGoods {
		if appGoodID == h.parentOrder.AppGoodID {
			h.parentAppGood = appGood
			break
		}
	}
	if h.parentAppGood == nil {
		return fmt.Errorf("invalid parentappgood")
	}
	return nil
}

func (h *baseCreateHandler) getParentGoodCoins(ctx context.Context) error {
	offset := int32(0)
	limit := int32(constant.DefaultRowLimit)

	for {
		goodCoins, _, err := goodcoinmwcli.GetGoodCoins(ctx, &goodcoinmwpb.Conds{
			GoodID: &basetypes.StringVal{Op: cruder.EQ, Value: h.parentAppGood.GoodID},
		}, offset, limit)
		if err != nil {
			return err
		}
		if len(goodCoins) == 0 {
			return nil
		}
		h.parentGoodCoins = append(h.parentGoodCoins, goodCoins...)
		offset += limit
	}
}

func (h *baseCreateHandler) validateRequiredAppGoods() error {
	requireds, ok := h.RequiredAppGoods[h.parentAppGood.EntID]
	if !ok {
		return fmt.Errorf("invalid requiredappgood")
	}
	for _, required := range requireds {
		if !required.Must {
			continue
		}
		if _, ok := h.AppGoods[required.RequiredAppGoodID]; !ok {
			return fmt.Errorf("miss requiredappgood")
		}
	}
	for appGoodID, _ := range h.AppGoods {
		if appGoodID == h.parentAppGood.EntID {
			continue
		}
		if _, ok := requireds[appGoodID]; !ok {
			return fmt.Errorf("invalid requiredappgood")
		}
	}
	return nil
}
