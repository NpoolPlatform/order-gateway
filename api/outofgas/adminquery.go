//nolint:dupl
package outofgas

import (
	"context"

	npool "github.com/NpoolPlatform/message/npool/order/gw/v1/outofgas"
	outofgas1 "github.com/NpoolPlatform/order-gateway/pkg/outofgas"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/NpoolPlatform/go-service-framework/pkg/logger"
)

func (s *Server) AdminGetOutOfGases(ctx context.Context, in *npool.AdminGetOutOfGasesRequest) (*npool.AdminGetOutOfGasesResponse, error) {
	handler, err := outofgas1.NewHandler(
		ctx,
		outofgas1.WithAppID(in.TargetAppID, false),
		outofgas1.WithGoodID(in.GoodID, false),
		outofgas1.WithOffset(in.Offset),
		outofgas1.WithLimit(in.Limit),
	)
	if err != nil {
		logger.Sugar().Errorw(
			"AdminGetOutOfGases",
			"In", in,
			"Error", err,
		)
		return &npool.AdminGetOutOfGasesResponse{}, status.Error(codes.InvalidArgument, err.Error())
	}

	infos, total, err := handler.GetOutOfGases(ctx)
	if err != nil {
		logger.Sugar().Errorw(
			"AdminGetOutOfGases",
			"In", in,
			"Error", err,
		)
		return &npool.AdminGetOutOfGasesResponse{}, status.Error(codes.Internal, err.Error())
	}

	return &npool.AdminGetOutOfGasesResponse{
		Infos: infos,
		Total: total,
	}, nil
}
