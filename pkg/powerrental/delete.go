package powerrental

import (
	"context"

	wlog "github.com/NpoolPlatform/go-service-framework/pkg/wlog"
	npool "github.com/NpoolPlatform/message/npool/order/gw/v1/powerrental"
	powerrentalordermwcli "github.com/NpoolPlatform/order-middleware/pkg/client/powerrental"
)

func (h *Handler) DeletePowerRentalOrder(ctx context.Context) (*npool.PowerRentalOrder, error) {
	info, err := h.GetPowerRentalOrder(ctx)
	if err != nil {
		return nil, wlog.WrapError(err)
	}
	if info == nil {
		return nil, wlog.Errorf("invalid powerrentalorder")
	}
	if err := powerrentalordermwcli.DeletePowerRentalOrder(ctx, h.ID, h.EntID, h.OrderID); err != nil {
		return nil, err
	}
	return info, nil
}
