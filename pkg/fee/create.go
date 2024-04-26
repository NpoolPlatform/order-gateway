package fee

import (
	"context"

	npool "github.com/NpoolPlatform/message/npool/order/gw/v1/fee"
)

type createHandler struct {
	*Handler
}

func (h *Handler) CreateFeeOrder(ctx context.Context) (*npool.FeeOrder, error) {
	return nil, nil
}
