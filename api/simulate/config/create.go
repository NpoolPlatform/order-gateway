package config

import (
	"context"

	npool "github.com/NpoolPlatform/message/npool/order/gw/v1/simulate/config"
	config1 "github.com/NpoolPlatform/order-gateway/pkg/simulate/config"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/NpoolPlatform/go-service-framework/pkg/logger"
)

func (s *Server) CreateSimulateConfig(ctx context.Context, in *npool.CreateSimulateConfigRequest) (*npool.CreateSimulateConfigResponse, error) {
	handler, err := config1.NewHandler(
		ctx,
		config1.WithAppID(&in.AppID, true),
		config1.WithUnits(&in.Units, true),
		config1.WithDuration(&in.Duration, true),
		config1.WithSendCouponMode(&in.SendCouponMode, true),
		config1.WithSendCouponProbability(&in.SendCouponProbability, true),
		config1.WithEnabled(&in.Enabled, false),
	)
	if err != nil {
		logger.Sugar().Errorw(
			"CreateSimulateConfig",
			"In", in,
			"Error", err,
		)
		return &npool.CreateSimulateConfigResponse{}, status.Error(codes.InvalidArgument, err.Error())
	}

	info, err := handler.CreateSimulateConfig(ctx)
	if err != nil {
		logger.Sugar().Errorw(
			"CreateSimulateConfig",
			"In", in,
			"Error", err,
		)
		return &npool.CreateSimulateConfigResponse{}, status.Error(codes.Internal, err.Error())
	}

	return &npool.CreateSimulateConfigResponse{
		Info: info,
	}, nil
}

func (s *Server) CreateAppSimulateConfig(ctx context.Context, in *npool.CreateAppSimulateConfigRequest) (*npool.CreateAppSimulateConfigResponse, error) {
	handler, err := config1.NewHandler(
		ctx,
		config1.WithAppID(&in.TargetAppID, true),
		config1.WithUnits(&in.Units, true),
		config1.WithSendCouponMode(&in.SendCouponMode, true),
		config1.WithSendCouponProbability(&in.SendCouponProbability, true),
	)
	if err != nil {
		logger.Sugar().Errorw(
			"CreateAppSimulateConfig",
			"In", in,
			"Error", err,
		)
		return &npool.CreateAppSimulateConfigResponse{}, status.Error(codes.InvalidArgument, err.Error())
	}

	info, err := handler.CreateSimulateConfig(ctx)
	if err != nil {
		logger.Sugar().Errorw(
			"CreateAppSimulateConfig",
			"In", in,
			"Error", err,
		)
		return &npool.CreateAppSimulateConfigResponse{}, status.Error(codes.Internal, err.Error())
	}

	return &npool.CreateAppSimulateConfigResponse{
		Info: info,
	}, nil
}
