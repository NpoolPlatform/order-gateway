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

func (s *Server) CreateFeeOrders(ctx context.Context, in *npool.CreateFeeOrdersRequest) (*npool.CreateFeeOrdersResponse, error) {
	handler, err := feeorder1.NewHandler(
		ctx,
		feeorder1.WithAppID(&in.AppID, true),
		feeorder1.WithUserID(&in.UserID, true),
		feeorder1.WithAppGoodIDs(in.AppGoodIDs, true),
		feeorder1.WithParentOrderID(&in.ParentOrderID, true),
		feeorder1.WithOrderType(func() *types.OrderType { e := types.OrderType_Normal; return &e }(), true),
		feeorder1.WithCreateMethod(func() *types.OrderCreateMethod { e := types.OrderCreateMethod_OrderCreatedByPurchase; return &e }(), true),
		feeorder1.WithDurationSeconds(&in.DurationSeconds, true),
		feeorder1.WithPaymentBalances(in.Balances, true),
		feeorder1.WithPaymentTransferCoinTypeID(in.PaymentTransferCoinTypeID, false),
		feeorder1.WithCouponIDs(in.CouponIDs, true),
	)
	if err != nil {
		logger.Sugar().Errorw(
			"CreateFeeOrders",
			"In", in,
			"Error", err,
		)
		return &npool.CreateFeeOrdersResponse{}, status.Error(codes.InvalidArgument, err.Error())
	}

	infos, err := handler.CreateFeeOrders(ctx)
	if err != nil {
		logger.Sugar().Errorw(
			"CreateFeeOrders",
			"In", in,
			"Error", err,
		)
		return &npool.CreateFeeOrdersResponse{}, status.Error(codes.Internal, err.Error())
	}

	return &npool.CreateFeeOrdersResponse{
		Infos: infos,
	}, nil
}

func (s *Server) CreateUserFeeOrders(ctx context.Context, in *npool.CreateUserFeeOrdersRequest) (*npool.CreateUserFeeOrdersResponse, error) {
	if err := ordercommon.ValidateAdminCreateOrderType(in.GetOrderType()); err != nil {
		logger.Sugar().Errorw(
			"CreateUserFeeOrders",
			"In", in,
		)
		return &npool.CreateUserFeeOrdersResponse{}, status.Error(codes.InvalidArgument, err.Error())
	}
	handler, err := feeorder1.NewHandler(
		ctx,
		feeorder1.WithAppID(&in.AppID, true),
		feeorder1.WithUserID(&in.TargetUserID, true),
		feeorder1.WithAppGoodIDs(in.AppGoodIDs, true),
		feeorder1.WithParentOrderID(&in.ParentOrderID, true),
		feeorder1.WithOrderType(&in.OrderType, true),
		feeorder1.WithCreateMethod(func() *types.OrderCreateMethod { e := types.OrderCreateMethod_OrderCreatedByAdmin; return &e }(), true),
		feeorder1.WithDurationSeconds(&in.DurationSeconds, true),
	)
	if err != nil {
		logger.Sugar().Errorw(
			"CreateUserFeeOrders",
			"In", in,
			"Error", err,
		)
		return &npool.CreateUserFeeOrdersResponse{}, status.Error(codes.InvalidArgument, err.Error())
	}

	infos, err := handler.CreateFeeOrders(ctx)
	if err != nil {
		logger.Sugar().Errorw(
			"CreateUserFeeOrders",
			"In", in,
			"Error", err,
		)
		return &npool.CreateUserFeeOrdersResponse{}, status.Error(codes.Internal, err.Error())
	}

	return &npool.CreateUserFeeOrdersResponse{
		Infos: infos,
	}, nil
}
