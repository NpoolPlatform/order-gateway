package fee

import (
	"context"
	"fmt"

	ordermwpb "github.com/NpoolPlatform/message/npool/order/mw/v1/order"
	ordermwcli "github.com/NpoolPlatform/order-middleware/pkg/client/order"
)

type baseCreateHandler struct {
	*Handler
	parentOrder *ordermwpb.Order
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
