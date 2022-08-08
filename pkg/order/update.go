package order

import (
	"context"
	"fmt"

	npool "github.com/NpoolPlatform/message/npool/order/gw/v1/order"

	ordermwpb "github.com/NpoolPlatform/message/npool/order/mw/v1/order"
	ordermwcli "github.com/NpoolPlatform/order-middleware/pkg/client/order"
)

func UpdateOrder(ctx context.Context, in *ordermwpb.OrderReq) (*npool.Order, error) {
	ord, err := ordermwcli.GetOrder(ctx, in.GetID())
	if err != nil {
		return nil, err
	}
	if ord == nil {
		return nil, fmt.Errorf("invalid order")
	}

	if in.GetAppID() != ord.AppID || in.GetUserID() != ord.UserID {
		return nil, fmt.Errorf("permission denied")
	}

	ord, err = ordermwcli.UpdateOrder(ctx, in)
	if err != nil {
		return nil, err
	}

	return GetOrder(ctx, ord.ID)
}
