//nolint:dupl
package feeorder

import (
	"context"

	npool "github.com/NpoolPlatform/message/npool/order/gw/v1/fee"
	feeorder1 "github.com/NpoolPlatform/order-gateway/pkg/fee"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/NpoolPlatform/go-service-framework/pkg/logger"
)

func (s *Server) AdminUpdateFeeOrder(ctx context.Context, in *npool.AdminUpdateFeeOrderRequest) (*npool.AdminUpdateFeeOrderResponse, error) {
	handler, err := feeorder1.NewHandler(
		ctx,
		feeorder1.WithID(&in.ID, true),
		feeorder1.WithEntID(&in.EntID, true),
		feeorder1.WithAppID(&in.TargetAppID, true),
		feeorder1.WithUserID(&in.TargetUserID, true),
		feeorder1.WithOrderID(&in.OrderID, true),
		feeorder1.WithAdminSetCanceled(in.Canceled, false),
	)
	if err != nil {
		logger.Sugar().Errorw(
			"AdminUpdateFeeOrder",
			"In", in,
			"Error", err,
		)
		return &npool.AdminUpdateFeeOrderResponse{}, status.Error(codes.InvalidArgument, err.Error())
	}

	info, err := handler.UpdateFeeOrder(ctx)
	if err != nil {
		logger.Sugar().Errorw(
			"AdminUpdateFeeOrder",
			"In", in,
			"Error", err,
		)
		return &npool.AdminUpdateFeeOrderResponse{}, status.Error(codes.Internal, err.Error())
	}

	return &npool.AdminUpdateFeeOrderResponse{
		Info: info,
	}, nil
}
