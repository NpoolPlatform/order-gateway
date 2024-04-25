package common

import (
	"context"
	"fmt"

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
	exist, err := ordermwcli.ExistOrderConds(ctx, &ordermwpb.Conds{
		EntID:     &basetypes.StringVal{Op: cruder.EQ, Value: orderID},
		AppID:     &basetypes.StringVal{Op: cruder.EQ, Value: *h.AppID},
		UserID:    &basetypes.StringVal{Op: cruder.EQ, Value: *h.UserID},
		GoodID:    &basetypes.StringVal{Op: cruder.EQ, Value: *h.GoodID},
		AppGoodID: &basetypes.StringVal{Op: cruder.EQ, Value: *h.AppGoodID},
	})
	if err != nil {
		return err
	}
	if !exist {
		return fmt.Errorf("invalid order")
	}
	return nil
}

func (h *OrderCheckHandler) CheckOrder(ctx context.Context) error {
	return h.CheckOrderWithOrderID(ctx, *h.OrderID)
}
