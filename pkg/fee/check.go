package fee

import (
	"context"

	wlog "github.com/NpoolPlatform/go-service-framework/pkg/wlog"
	cruder "github.com/NpoolPlatform/libent-cruder/pkg/cruder"
	basetypes "github.com/NpoolPlatform/message/npool/basetypes/v1"
	feeordermwpb "github.com/NpoolPlatform/message/npool/order/mw/v1/fee"
	feeordermwcli "github.com/NpoolPlatform/order-middleware/pkg/client/fee"
)

type checkHandler struct {
	*Handler
}

func (h *checkHandler) checkFeeOrder(ctx context.Context) error {
	exist, err := feeordermwcli.ExistFeeOrderConds(ctx, &feeordermwpb.Conds{
		ID:      &basetypes.Uint32Val{Op: cruder.EQ, Value: *h.ID},
		EntID:   &basetypes.StringVal{Op: cruder.EQ, Value: *h.EntID},
		OrderID: &basetypes.StringVal{Op: cruder.EQ, Value: *h.OrderID},
		AppID:   &basetypes.StringVal{Op: cruder.EQ, Value: *h.OrderCheckHandler.AppID},
		UserID:  &basetypes.StringVal{Op: cruder.EQ, Value: *h.OrderCheckHandler.UserID},
	})
	if err != nil {
		return err
	}
	if !exist {
		return wlog.Errorf("invalid feeorder")
	}
	return nil
}
