package config

import (
	"context"

	config1 "github.com/NpoolPlatform/order-gateway/pkg/simulate/config"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/NpoolPlatform/go-service-framework/pkg/logger"
	npool "github.com/NpoolPlatform/message/npool/order/gw/v1/simulate/config"
)

func (s *Server) DeleteAppSimulateConfig(ctx context.Context, in *npool.DeleteAppSimulateConfigRequest) (*npool.DeleteAppSimulateConfigResponse, error) {
	handler, err := config1.NewHandler(
		ctx,
		config1.WithID(&in.ID, true),
		config1.WithEntID(&in.EntID, true),
		config1.WithAppID(&in.TargetAppID, true),
	)
	if err != nil {
		logger.Sugar().Errorw(
			"DeleteAppSimulateConfig",
			"In", in,
			"Error", err,
		)
		return &npool.DeleteAppSimulateConfigResponse{}, status.Error(codes.InvalidArgument, err.Error())
	}

	info, err := handler.DeleteSimulateConfig(ctx)
	if err != nil {
		logger.Sugar().Errorw(
			"DeleteAppSimulateConfig",
			"In", in,
			"Error", err,
		)
		return &npool.DeleteAppSimulateConfigResponse{}, status.Error(codes.Internal, err.Error())
	}

	return &npool.DeleteAppSimulateConfigResponse{
		Info: info,
	}, nil
}
