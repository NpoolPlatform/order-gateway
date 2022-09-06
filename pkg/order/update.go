package order

import (
	"context"
	"fmt"

	"github.com/NpoolPlatform/go-service-framework/pkg/logger"

	"github.com/NpoolPlatform/libent-cruder/pkg/cruder"

	"github.com/NpoolPlatform/cloud-hashing-order/pkg/db/ent/order"

	orderconst "github.com/NpoolPlatform/cloud-hashing-order/pkg/const"

	npool "github.com/NpoolPlatform/message/npool/order/gw/v1/order"
	ordermgrpb "github.com/NpoolPlatform/message/npool/order/mgr/v1/order/order"

	stockcli "github.com/NpoolPlatform/stock-manager/pkg/client"
	stockconst "github.com/NpoolPlatform/stock-manager/pkg/const"

	"google.golang.org/protobuf/types/known/structpb"

	ordermwpb "github.com/NpoolPlatform/message/npool/order/mw/v1/order"
	ordermwcli "github.com/NpoolPlatform/order-middleware/pkg/client/order"

	ordercli "github.com/NpoolPlatform/cloud-hashing-order/pkg/client"

	orderstatepb "github.com/NpoolPlatform/message/npool/order/mgr/v1/order/state"

	archivementmwcli "github.com/NpoolPlatform/archivement-middleware/pkg/client/archivement"
)

func UpdateOrder(ctx context.Context, in *ordermwpb.OrderReq, fromAdmin bool) (*npool.Order, error) {
	ord, err := ordermwcli.GetOrder(ctx, in.GetID())
	if err != nil {
		return nil, err
	}
	if ord == nil {
		return nil, fmt.Errorf("invalid order")
	}

	if !fromAdmin {
		if in.GetAppID() != ord.AppID || in.GetUserID() != ord.UserID {
			return nil, fmt.Errorf("permission denied")
		}

		ord, err = ordermwcli.UpdateOrder(ctx, in)
		if err != nil {
			return nil, err
		}

		return GetOrder(ctx, ord.ID)
	}

	if ord.OrderType.String() != orderconst.OrderTypeOffline && ord.OrderType != ordermgrpb.OrderType_Offline {
		return nil, fmt.Errorf("order type not offline")
	}
	if ord.State != orderstatepb.EState_Paid {
		return nil, fmt.Errorf("order state not paid")
	}

	// TODO Distributed transactions should be used
	stock, err := stockcli.GetStockOnly(ctx, cruder.NewFilterConds().
		WithCond(stockconst.StockFieldGoodID, cruder.EQ, structpb.NewStringValue(ord.GoodID)))
	if err != nil {
		return nil, err
	}
	if stock == nil {
		return nil, fmt.Errorf("invalid stock")
	}

	err = archivementmwcli.Delete(ctx, ord.ID)
	if err != nil {
		return nil, err
	}

	fields := cruder.NewFilterFields().WithField(stockconst.StockFieldInService, structpb.NewNumberValue(float64(-int(ord.Units))))
	_, err = stockcli.AddStockFields(ctx, stock.ID, fields)
	if err != nil {
		return nil, err
	}

	payment, err := ordercli.GetOrderPayment(ctx, ord.ID)
	if err != nil {
		logger.Sugar().Infow("processOrderPayments", "OrderID", order.ID, "error", err)
		return nil, err
	}
	if payment == nil {
		return nil, fmt.Errorf("invalid payment")
	}

	payment.State = orderconst.PaymentStateCanceled
	_, err = ordercli.UpdatePayment(ctx, payment)
	if err != nil {
		return nil, err
	}

	return GetOrder(ctx, ord.ID)
}
