package fee

import (
	"context"

	wlog "github.com/NpoolPlatform/go-service-framework/pkg/wlog"
	npool "github.com/NpoolPlatform/message/npool/order/gw/v1/fee"
	feeordermwcli "github.com/NpoolPlatform/order-middleware/pkg/client/fee"
)

func (h *Handler) DeleteFeeOrder(ctx context.Context) (*npool.FeeOrder, error) {
	handler := &checkHandler{
		Handler: h,
	}
	if err := handler.checkFeeOrder(ctx); err != nil {
		return nil, wlog.WrapError(err)
	}
	info, err := h.GetFeeOrder(ctx)
	if err != nil {
		return nil, wlog.WrapError(err)
	}
	if info == nil {
		return nil, wlog.Errorf("invalid feeorder")
	}
	if err := feeordermwcli.DeleteFeeOrder(ctx, h.ID, h.EntID, h.OrderID); err != nil {
		return nil, err
	}
	return info, nil
}
