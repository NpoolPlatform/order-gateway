package order

import (
	"context"

	"github.com/NpoolPlatform/go-service-framework/pkg/logger"

	npool "github.com/NpoolPlatform/message/npool/order/gw/v1/order"
	order1 "github.com/NpoolPlatform/order-gateway/pkg/order"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (s *Server) GetOrders(ctx context.Context, in *npool.GetOrdersRequest) (*npool.GetOrdersResponse, error) {
	handler, err := order1.NewHandler(
		ctx,
		order1.WithAppID(&in.AppID, true),
		order1.WithUserID(in.TargetUserID, false),
		order1.WithOffset(in.GetOffset()),
		order1.WithLimit(in.GetLimit()),
	)
	if err != nil {
		logger.Sugar().Errorw(
			"GetOrders",
			"In", in,
			"Error", err,
		)
		return &npool.GetOrdersResponse{}, status.Error(codes.InvalidArgument, err.Error())
	}

	infos, total, err := handler.GetOrders(ctx)
	if err != nil {
		logger.Sugar().Errorw(
			"GetOrders",
			"In", in,
			"Error", err,
		)
		return &npool.GetOrdersResponse{}, status.Error(codes.Internal, err.Error())
	}

	return &npool.GetOrdersResponse{
		Infos: infos,
		Total: total,
	}, nil
}

func (s *Server) GetMyOrders(ctx context.Context, in *npool.GetMyOrdersRequest) (*npool.GetMyOrdersResponse, error) {
	handler, err := order1.NewHandler(
		ctx,
		order1.WithAppID(&in.AppID, true),
		order1.WithUserID(&in.UserID, true),
		order1.WithOffset(in.GetOffset()),
		order1.WithLimit(in.GetLimit()),
	)
	if err != nil {
		logger.Sugar().Errorw(
			"GetMyOrders",
			"In", in,
			"Error", err,
		)
		return &npool.GetMyOrdersResponse{}, status.Error(codes.InvalidArgument, err.Error())
	}

	infos, total, err := handler.GetOrders(ctx)
	if err != nil {
		logger.Sugar().Errorw(
			"GetMyOrders",
			"In", in,
			"Error", err,
		)
		return &npool.GetMyOrdersResponse{}, status.Error(codes.Internal, err.Error())
	}

	return &npool.GetMyOrdersResponse{
		Infos: infos,
		Total: total,
	}, nil
}
