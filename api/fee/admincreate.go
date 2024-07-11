//nolint:dupl
package feeorder

import (
	"context"

	types "github.com/NpoolPlatform/message/npool/basetypes/order/v1"
	npool "github.com/NpoolPlatform/message/npool/order/gw/v1/fee"
	ordercommon "github.com/NpoolPlatform/order-gateway/api/order/common"
	feeorder1 "github.com/NpoolPlatform/order-gateway/pkg/fee"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/NpoolPlatform/go-service-framework/pkg/logger"
)

func (s *Server) AdminCreateFeeOrder(ctx context.Context, in *npool.AdminCreateFeeOrderRequest) (*npool.AdminCreateFeeOrderResponse, error) {
	if err := ordercommon.ValidateAdminCreateOrderType(in.GetOrderType()); err != nil {
		logger.Sugar().Errorw(
			"AdminCreateUserFeeOrder",
			"In", in,
		)
		return &npool.AdminCreateFeeOrderResponse{}, status.Error(codes.InvalidArgument, err.Error())
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
