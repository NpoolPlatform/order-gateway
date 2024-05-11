//nolint:dupl
package outofgas

import (
	"context"

	npool "github.com/NpoolPlatform/message/npool/order/gw/v1/powerrental/outofgas"
	outofgas1 "github.com/NpoolPlatform/order-gateway/pkg/powerrental/outofgas"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/NpoolPlatform/go-service-framework/pkg/logger"
)

func (s *Server) AdminDeleteCompensate(ctx context.Context, in *npool.AdminDeleteCompensateRequest) (*npool.AdminDeleteCompensateResponse, error) {
	handler, err := outofgas1.NewHandler(
		ctx,
		outofgas1.WithAppID(&in.TargetAppID, true),
		outofgas1.WithUserID(&in.TargetUserID, true),
		outofgas1.WithID(&in.ID, true),
		outofgas1.WithEntID(&in.EntID, true),
		outofgas1.WithOrderID(&in.OrderID, true),
	)
	if err != nil {
		logger.Sugar().Errorw(
			"AdminDeleteCompensate",
			"In", in,
			"Error", err,
		)
		return &npool.AdminDeleteCompensateResponse{}, status.Error(codes.InvalidArgument, err.Error())
	}

	info, err := handler.DeleteCompensate(ctx)
	if err != nil {
		logger.Sugar().Errorw(
			"AdminDeleteCompensate",
			"In", in,
			"Error", err,
		)
		return &npool.AdminDeleteCompensateResponse{}, status.Error(codes.Internal, err.Error())
	}

	return &npool.AdminDeleteCompensateResponse{
		Info: info,
	}, nil
}
