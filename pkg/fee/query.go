package fee

import (
	"context"

	npool "github.com/NpoolPlatform/message/npool/order/gw/v1/fee"
)

type queryHandler struct {
	*Handler
}

func (h *Handler) GetFeeOrder(ctx context.Context) (*npool.FeeOrder, error) {
	return nil, nil
}

func (h *Handler) GetFeeOrders(ctx context.Context) ([]*npool.FeeOrder, uint32, error) {
	return nil, 0, nil
}
