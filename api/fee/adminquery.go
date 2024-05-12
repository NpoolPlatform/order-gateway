package feeorder

import (
	"context"

	npool "github.com/NpoolPlatform/message/npool/order/gw/v1/fee"
	feeorder1 "github.com/NpoolPlatform/order-gateway/pkg/fee"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/NpoolPlatform/go-service-framework/pkg/logger"
)

func (s *Server) AdminGetFeeOrders(ctx context.Context, in *npool.AdminGetFeeOrdersRequest) (*npool.AdminGetFeeOrdersResponse, error) {
	handler, err := feeorder1.NewHandler(
		ctx,
		feeorder1.WithAppID(in.TargetAppID, false),
		feeorder1.WithGoodID(in.GoodID, false),
		feeorder1.WithOffset(in.Offset),
		feeorder1.WithLimit(in.Limit),
	)
	if err != nil {
		logger.Sugar().Errorw(
			"AdminGetFeeOrders",
			"In", in,
			"Error", err,
		)
		return &npool.AdminGetFeeOrdersResponse{}, status.Error(codes.InvalidArgument, err.Error())
	}

	infos, total, err := handler.GetFeeOrders(ctx)
	if err != nil {
		logger.Sugar().Errorw(
			"AdminGetFeeOrders",
			"In", in,
			"Error", err,
		)
		return &npool.AdminGetFeeOrdersResponse{}, status.Error(codes.Internal, err.Error())
	}

	return &npool.AdminGetFeeOrdersResponse{
		Infos: infos,
		Total: total,
	}, nil
}
