//nolint:dupl
package config

import (
	"context"

	config1 "github.com/NpoolPlatform/order-gateway/pkg/simulate/config"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/NpoolPlatform/go-service-framework/pkg/logger"
	npool "github.com/NpoolPlatform/message/npool/order/gw/v1/simulate/config"
)

func (s *Server) UpdateSimulateConfig(ctx context.Context, in *npool.UpdateSimulateConfigRequest) (*npool.UpdateSimulateConfigResponse, error) {
	handler, err := config1.NewHandler(
		ctx,
		config1.WithID(&in.ID, true),
		config1.WithEntID(&in.EntID, true),
		config1.WithAppID(&in.AppID, true),
		config1.WithUnits(in.Units, false),
		config1.WithDuration(in.Duration, false),
		config1.WithSendCouponMode(in.SendCouponMode, false),
		config1.WithSendCouponProbability(in.SendCouponProbability, false),
		config1.WithEnabled(in.Enabled, false),
	)
	if err != nil {
		logger.Sugar().Errorw(
			"UpdateSimulateConfig",
			"In", in,
			"Error", err,
		)
		return &npool.UpdateSimulateConfigResponse{}, status.Error(codes.InvalidArgument, err.Error())
	}

	info, err := handler.UpdateSimulateConfig(ctx)
	if err != nil {
		logger.Sugar().Errorw(
			"UpdateSimulateConfig",
			"In", in,
			"Error", err,
		)
		return &npool.UpdateSimulateConfigResponse{}, status.Error(codes.Internal, err.Error())
	}

	return &npool.UpdateSimulateConfigResponse{
		Info: info,
	}, nil
}

func (s *Server) UpdateAppSimulateConfig(ctx context.Context, in *npool.UpdateAppSimulateConfigRequest) (*npool.UpdateAppSimulateConfigResponse, error) {
	handler, err := config1.NewHandler(
		ctx,
		config1.WithID(&in.ID, true),
		config1.WithEntID(&in.EntID, true),
		config1.WithAppID(&in.TargetAppID, true),
		config1.WithUnits(in.Units, false),
		config1.WithDuration(in.Duration, false),
		config1.WithSendCouponMode(in.SendCouponMode, false),
		config1.WithSendCouponProbability(in.SendCouponProbability, false),
		config1.WithEnabled(in.Enabled, false),
	)
	if err != nil {
		logger.Sugar().Errorw(
			"UpdateAppSimulateConfig",
			"In", in,
			"Error", err,
		)
		return &npool.UpdateAppSimulateConfigResponse{}, status.Error(codes.InvalidArgument, err.Error())
	}

	info, err := handler.UpdateSimulateConfig(ctx)
	if err != nil {
		logger.Sugar().Errorw(
			"UpdateAppSimulateConfig",
			"In", in,
			"Error", err,
		)
		return &npool.UpdateAppSimulateConfigResponse{}, status.Error(codes.Internal, err.Error())
	}

	return &npool.UpdateAppSimulateConfigResponse{
		Info: info,
	}, nil
}
