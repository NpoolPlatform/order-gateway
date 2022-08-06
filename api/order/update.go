//nolint:nolintlint,dupl
package order

import (
	"context"

	order1 "github.com/NpoolPlatform/order-gateway/pkg/order"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/NpoolPlatform/go-service-framework/pkg/logger"
	npool "github.com/NpoolPlatform/message/npool/order/gw/v1/order"
	ordermwpb "github.com/NpoolPlatform/message/npool/order/mw/v1/order"

	"github.com/google/uuid"
)

func (s *Server) UpdateOrder(ctx context.Context, in *npool.UpdateOrderRequest) (*npool.UpdateOrderResponse, error) {
	// TODO: who create, who update

	if _, err := uuid.Parse(in.GetID()); err != nil {
		logger.Sugar().Errorw("UpdateOrder", "ID", in.GetID(), "error", err)
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	if _, err := uuid.Parse(in.GetPaymentID()); err != nil {
		logger.Sugar().Errorw("UpdateOrder", "PaymentID", in.GetPaymentID(), "error", err)
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	if in.Canceled == nil {
		logger.Sugar().Errorw("UpdateOrder", "error", "nothing todo")
		return nil, status.Error(codes.InvalidArgument, "nothing todo")
	}

	ord, err := order1.UpdateOrder(ctx, &ordermwpb.OrderReq{
		ID:        &in.ID,
		PaymentID: &in.PaymentID,
		Canceled:  in.Canceled,
	})
	if err != nil {
		logger.Sugar().Errorw("UpdateOrder", "error", err)
		return nil, status.Error(codes.Internal, "fail update order")
	}

	return &npool.UpdateOrderResponse{
		Info: ord,
	}, nil
}
