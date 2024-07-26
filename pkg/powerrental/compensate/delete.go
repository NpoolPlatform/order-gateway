package compensate

import (
	"context"

	wlog "github.com/NpoolPlatform/go-service-framework/pkg/wlog"
	compensategwpb "github.com/NpoolPlatform/message/npool/order/gw/v1/compensate"
	compensate1 "github.com/NpoolPlatform/order-gateway/pkg/compensate"
	powerrentalcompensatemwcli "github.com/NpoolPlatform/order-middleware/pkg/client/powerrental/compensate"
)

func (h *Handler) DeleteCompensate(ctx context.Context) (*compensategwpb.Compensate, error) {
	handler := &checkHandler{
		Handler: h,
	}
	if err := handler.checkCompensate(ctx); err != nil {
		return nil, wlog.WrapError(err)
	}
	h1, err := compensate1.NewHandler(
		ctx,
		compensate1.WithEntID(h.EntID, true),
		compensate1.WithOrderID(h.OrderID, true),
	)
	if err != nil {
		return nil, wlog.WrapError(err)
	}
	info, err := h1.GetCompensate(ctx)
	if err != nil {
		return nil, wlog.WrapError(err)
	}
	if info == nil {
		return nil, wlog.Errorf("invalid compensate")
	}
	if err := powerrentalcompensatemwcli.DeleteCompensate(ctx, &info.ID, &info.EntID); err != nil {
		return nil, wlog.WrapError(err)
	}
	return info, nil
}
