package appconfig

import (
	"context"

	"github.com/NpoolPlatform/go-service-framework/pkg/logger"

	npool "github.com/NpoolPlatform/message/npool/order/gw/v1/app/config"
	config1 "github.com/NpoolPlatform/order-gateway/pkg/app/config"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (s *Server) AdminGetAppConfigs(ctx context.Context, in *npool.AdminGetAppConfigsRequest) (*npool.AdminGetAppConfigsResponse, error) {
	handler, err := config1.NewHandler(
		ctx,
		config1.WithAppID(in.TargetAppID, false),
		config1.WithOffset(in.Offset),
		config1.WithOffset(in.Limit),
	)
	if err != nil {
		logger.Sugar().Errorw(
			"AdminGetAppConfigs",
			"In", in,
			"Error", err,
		)
		return &npool.AdminGetAppConfigsResponse{}, status.Error(codes.InvalidArgument, err.Error())
	}

	infos, total, err := handler.GetAppConfigs(ctx)
	if err != nil {
		logger.Sugar().Errorw(
			"AdminGetAppConfigs",
			"In", in,
			"Error", err,
		)
		return &npool.AdminGetAppConfigsResponse{}, status.Error(codes.Internal, err.Error())
	}

	return &npool.AdminGetAppConfigsResponse{
		Infos: infos,
		Total: total,
	}, nil
}
