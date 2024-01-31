//nolint:dupl
package config

import (
	"context"

	"github.com/NpoolPlatform/go-service-framework/pkg/logger"

	npool "github.com/NpoolPlatform/message/npool/order/gw/v1/simulate/config"
	config1 "github.com/NpoolPlatform/order-gateway/pkg/simulate/config"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (s *Server) GetSimulateConfigs(ctx context.Context, in *npool.GetSimulateConfigsRequest) (*npool.GetSimulateConfigsResponse, error) {
	handler, err := config1.NewHandler(
		ctx,
		config1.WithAppID(&in.AppID, true),
		config1.WithOffset(in.GetOffset()),
		config1.WithLimit(in.GetLimit()),
	)
	if err != nil {
		logger.Sugar().Errorw(
			"GetSimulateConfigs",
			"In", in,
			"Error", err,
		)
		return &npool.GetSimulateConfigsResponse{}, status.Error(codes.InvalidArgument, err.Error())
	}

	infos, total, err := handler.GetSimulateConfigs(ctx)
	if err != nil {
		logger.Sugar().Errorw(
			"GetSimulateConfigs",
			"In", in,
			"Error", err,
		)
		return &npool.GetSimulateConfigsResponse{}, status.Error(codes.Internal, err.Error())
	}

	return &npool.GetSimulateConfigsResponse{
		Infos: infos,
		Total: total,
	}, nil
}

func (s *Server) GetAppSimulateConfigs(ctx context.Context, in *npool.GetAppSimulateConfigsRequest) (*npool.GetAppSimulateConfigsResponse, error) {
	handler, err := config1.NewHandler(
		ctx,
		config1.WithAppID(&in.TargetAppID, true),
		config1.WithOffset(in.GetOffset()),
		config1.WithLimit(in.GetLimit()),
	)
	if err != nil {
		logger.Sugar().Errorw(
			"GetAppSimulateConfigs",
			"In", in,
			"Error", err,
		)
		return &npool.GetAppSimulateConfigsResponse{}, status.Error(codes.InvalidArgument, err.Error())
	}

	infos, total, err := handler.GetSimulateConfigs(ctx)
	if err != nil {
		logger.Sugar().Errorw(
			"GetAppSimulateConfigs",
			"In", in,
			"Error", err,
		)
		return &npool.GetAppSimulateConfigsResponse{}, status.Error(codes.Internal, err.Error())
	}

	return &npool.GetAppSimulateConfigsResponse{
		Infos: infos,
		Total: total,
	}, nil
}
