package common

import (
	"context"

	wlog "github.com/NpoolPlatform/go-service-framework/pkg/wlog"
	"github.com/NpoolPlatform/libent-cruder/pkg/cruder"
	basetypes "github.com/NpoolPlatform/message/npool/basetypes/v1"
	ordermwpb "github.com/NpoolPlatform/message/npool/order/mw/v1/order"
	ordercommon "github.com/NpoolPlatform/order-gateway/pkg/common"
	ordermwcli "github.com/NpoolPlatform/order-middleware/pkg/client/order"
)

type OrderCheckHandler struct {
	ordercommon.AppGoodCheckHandler
	OrderID *string
}

func (h *OrderCheckHandler) CheckOrderWithOrderID(ctx context.Context, orderID string) error {
	conds := &ordermwpb.Conds{
		EntID: &basetypes.StringVal{Op: cruder.EQ, Value: orderID},
		AppID: &basetypes.StringVal{Op: cruder.EQ, Value: *h.AppID},
	}
	if h.UserID != nil {
		conds.UserID = &basetypes.StringVal{Op: cruder.EQ, Value: *h.UserID}
	}
	if h.GoodID != nil {
		conds.GoodID = &basetypes.StringVal{Op: cruder.EQ, Value: *h.GoodID}
	}
	if h.AppGoodID != nil {
		conds.AppGoodID = &basetypes.StringVal{Op: cruder.EQ, Value: *h.AppGoodID}
	}
	exist, err := ordermwcli.ExistOrderConds(ctx, conds)
	if err != nil {
		return wlog.WrapError(err)
	}
	if !exist {
		return wlog.Errorf("invalid order")
	}
	return nil
}

func (h *OrderCheckHandler) CheckOrder(ctx context.Context) error {
	return h.CheckOrderWithOrderID(ctx, *h.OrderID)
}
