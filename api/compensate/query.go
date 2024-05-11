//nolint:dupl
package compensate

import (
	"context"

	npool "github.com/NpoolPlatform/message/npool/order/gw/v1/compensate"
	compensate1 "github.com/NpoolPlatform/order-gateway/pkg/compensate"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/NpoolPlatform/go-service-framework/pkg/logger"
)

func (s *Server) GetCompensates(ctx context.Context, in *npool.GetCompensatesRequest) (*npool.GetCompensatesResponse, error) {
	handler, err := compensate1.NewHandler(
		ctx,
		compensate1.WithAppID(&in.AppID, true),
		compensate1.WithUserID(in.TargetUserID, false),
		compensate1.WithAppGoodID(in.AppGoodID, false),
		compensate1.WithOffset(in.Offset),
		compensate1.WithLimit(in.Limit),
	)
	if err != nil {
		logger.Sugar().Errorw(
			"GetCompensates",
			"In", in,
			"Error", err,
		)
		return &npool.GetCompensatesResponse{}, status.Error(codes.InvalidArgument, err.Error())
	}

	infos, total, err := handler.GetCompensates(ctx)
	if err != nil {
		logger.Sugar().Errorw(
			"GetCompensates",
			"In", in,
			"Error", err,
		)
		return &npool.GetCompensatesResponse{}, status.Error(codes.Internal, err.Error())
	}

	return &npool.GetCompensatesResponse{
		Infos: infos,
		Total: total,
	}, nil
}

func (s *Server) GetMyCompensates(ctx context.Context, in *npool.GetMyCompensatesRequest) (*npool.GetMyCompensatesResponse, error) {
	handler, err := compensate1.NewHandler(
		ctx,
		compensate1.WithAppID(&in.AppID, true),
		compensate1.WithUserID(&in.UserID, true),
		compensate1.WithAppGoodID(in.OrderID, false),
		compensate1.WithOffset(in.Offset),
		compensate1.WithLimit(in.Limit),
	)
	if err != nil {
		logger.Sugar().Errorw(
			"GetMyCompensates",
			"In", in,
			"Error", err,
		)
		return &npool.GetMyCompensatesResponse{}, status.Error(codes.InvalidArgument, err.Error())
	}

	infos, total, err := handler.GetCompensates(ctx)
	if err != nil {
		logger.Sugar().Errorw(
			"GetMyCompensates",
			"In", in,
			"Error", err,
		)
		return &npool.GetMyCompensatesResponse{}, status.Error(codes.Internal, err.Error())
	}

	return &npool.GetMyCompensatesResponse{
		Infos: infos,
		Total: total,
	}, nil
}
