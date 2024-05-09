//nolint:dupl
package feeorder

import (
	"context"

	types "github.com/NpoolPlatform/message/npool/basetypes/order/v1"
	npool "github.com/NpoolPlatform/message/npool/order/gw/v1/fee"
	feeorder1 "github.com/NpoolPlatform/order-gateway/pkg/fee"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/NpoolPlatform/go-service-framework/pkg/logger"
)

func (s *Server) AdminCreateFeeOrder(ctx context.Context, in *npool.AdminCreateFeeOrderRequest) (*npool.AdminCreateFeeOrderResponse, error) {
	switch in.OrderType {
	case types.OrderType_Offline:
	case types.OrderType_Airdrop:
	default:
		logger.Sugar().Errorw(
			"AdminCreateUserFeeOrder",
			"In", in,
		)
		return &npool.AdminCreateFeeOrderResponse{}, status.Error(codes.InvalidArgument, "invalid ordertype")
	}
	handler, err := feeorder1.NewHandler(
		ctx,
		feeorder1.WithAppID(&in.TargetAppID, true),
		feeorder1.WithUserID(&in.TargetUserID, true),
		feeorder1.WithAppGoodID(&in.AppGoodID, true),
		feeorder1.WithParentOrderID(&in.ParentOrderID, true),
		feeorder1.WithOrderType(&in.OrderType, true),
		feeorder1.WithCreateMethod(func() *types.OrderCreateMethod { e := types.OrderCreateMethod_OrderCreatedByAdmin; return &e }(), true),
		feeorder1.WithDurationSeconds(&in.DurationSeconds, true),
	)
	if err != nil {
		logger.Sugar().Errorw(
			"AdminCreateFeeOrder",
			"In", in,
			"Error", err,
		)
		return &npool.AdminCreateFeeOrderResponse{}, status.Error(codes.InvalidArgument, err.Error())
	}

	info, err := handler.CreateFeeOrder(ctx)
	if err != nil {
		logger.Sugar().Errorw(
			"AdminCreateFeeOrder",
			"In", in,
			"Error", err,
		)
		return &npool.AdminCreateFeeOrderResponse{}, status.Error(codes.Internal, err.Error())
	}

	return &npool.AdminCreateFeeOrderResponse{
		Info: info,
	}, nil
}

func (s *Server) AdminCreateFeeOrders(ctx context.Context, in *npool.AdminCreateFeeOrdersRequest) (*npool.AdminCreateFeeOrdersResponse, error) {
	switch in.OrderType {
	case types.OrderType_Offline:
	case types.OrderType_Airdrop:
	default:
		logger.Sugar().Errorw(
			"AdminCreateUserFeeOrders",
			"In", in,
		)
		return &npool.AdminCreateFeeOrdersResponse{}, status.Error(codes.InvalidArgument, "invalid ordertype")
	}
	handler, err := feeorder1.NewHandler(
		ctx,
		feeorder1.WithAppID(&in.TargetAppID, true),
		feeorder1.WithUserID(&in.TargetUserID, true),
		feeorder1.WithAppGoodIDs(in.AppGoodIDs, true),
		feeorder1.WithParentOrderID(&in.ParentOrderID, true),
		feeorder1.WithOrderType(&in.OrderType, true),
		feeorder1.WithCreateMethod(func() *types.OrderCreateMethod { e := types.OrderCreateMethod_OrderCreatedByAdmin; return &e }(), true),
		feeorder1.WithDurationSeconds(&in.DurationSeconds, true),
	)
	if err != nil {
		logger.Sugar().Errorw(
			"AdminCreateFeeOrders",
			"In", in,
			"Error", err,
		)
		return &npool.AdminCreateFeeOrdersResponse{}, status.Error(codes.InvalidArgument, err.Error())
	}

	infos, err := handler.CreateFeeOrders(ctx)
	if err != nil {
		logger.Sugar().Errorw(
			"AdminCreateFeeOrders",
			"In", in,
			"Error", err,
		)
		return &npool.AdminCreateFeeOrdersResponse{}, status.Error(codes.Internal, err.Error())
	}

	return &npool.AdminCreateFeeOrdersResponse{
		Infos: infos,
	}, nil
}
