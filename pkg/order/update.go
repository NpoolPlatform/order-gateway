package order

import (
	"context"
	"fmt"

	archivementmwcli "github.com/NpoolPlatform/archivement-middleware/pkg/client/archivement"
	goodscli "github.com/NpoolPlatform/good-middleware/pkg/client/good"
	goodmwpb "github.com/NpoolPlatform/message/npool/good/mw/v1/good"
	ordermgrpb "github.com/NpoolPlatform/message/npool/order/mgr/v1/order"

	npool "github.com/NpoolPlatform/message/npool/order/gw/v1/order"
	ordermwpb "github.com/NpoolPlatform/message/npool/order/mw/v1/order"
	ordermwcli "github.com/NpoolPlatform/order-middleware/pkg/client/order"
)

func cancelOrder(ctx context.Context, ord *ordermwpb.Order) error {
	switch ord.OrderType.String() {
	case ordermgrpb.OrderType_Offline.String():
	default:
		return fmt.Errorf("permission denied")
	}

	if ord.OrderState != ordermgrpb.OrderState_Paid {
		return fmt.Errorf("order state not paid")
	}

	good, err := goodscli.GetGood(ctx, ord.GetGoodID())
	if err != nil {
		return err
	}
	if good == nil {
		return fmt.Errorf("invalid good")
	}
	// TODO Distributed transactions should be used

	err = archivementmwcli.Delete(ctx, ord.ID)
	if err != nil {
		return err
	}
	units := -int32(ord.Units)
	_, err = goodscli.UpdateGood(ctx, &goodmwpb.GoodReq{
		ID:        &good.ID,
		InService: &units,
	})
	if err != nil {
		return err
	}

	return nil
}

func UpdateOrder(ctx context.Context, in *ordermwpb.OrderReq, fromAdmin bool) (*npool.Order, error) {
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

	if !fromAdmin {
		ord, err = ordermwcli.UpdateOrder(ctx, in)
		if err != nil {
			return nil, err
		}
		return GetOrder(ctx, ord.ID)
	}

	if in.GetCanceled() {
		if err := cancelOrder(ctx, ord); err != nil {
			return nil, err
		}
		return GetOrder(ctx, ord.ID)
	}

	return GetOrder(ctx, ord.ID)
}
