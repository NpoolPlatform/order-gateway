//nolint:dupl
package appconfig

import (
	"context"

	npool "github.com/NpoolPlatform/message/npool/order/gw/v1/app/config"
	config1 "github.com/NpoolPlatform/order-gateway/pkg/app/config"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/NpoolPlatform/go-service-framework/pkg/logger"
)

func (s *Server) AdminCreateAppConfig(ctx context.Context, in *npool.AdminCreateAppConfigRequest) (*npool.AdminCreateAppConfigResponse, error) {
	handler, err := config1.NewHandler(
		ctx,
		config1.WithAppID(&in.TargetAppID, true),
		config1.WithEnableSimulateOrder(in.EnableSimulateOrder, false),
		config1.WithSimulateOrderCouponMode(in.SimulateOrderCouponMode, false),
		config1.WithSimulateOrderCouponProbability(in.SimulateOrderCouponProbability, false),
		config1.WithSimulateOrderCashableProfitProbability(in.SimulateOrderCashableProfitProbability, false),
		config1.WithMaxUnpaidOrders(in.MaxUnpaidOrders, false),
		config1.WithMaxTypedCouponsPerOrder(in.MaxTypedCouponsPerOrder, false),
	)
	if err != nil {
		logger.Sugar().Errorw(
			"AdminCreateAppConfig",
			"In", in,
			"Error", err,
		)
		return &npool.AdminCreateAppConfigResponse{}, status.Error(codes.InvalidArgument, err.Error())
	}

	info, err := handler.CreateAppConfig(ctx)
	if err != nil {
		logger.Sugar().Errorw(
			"AdminCreateAppConfig",
			"In", in,
			"Error", err,
		)
		return &npool.AdminCreateAppConfigResponse{}, status.Error(codes.Internal, err.Error())
	}

	return &npool.AdminCreateAppConfigResponse{
		Info: info,
	}, nil
}
