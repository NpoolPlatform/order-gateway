//nolint:dupl
package powerrental

import (
	"context"

	npool "github.com/NpoolPlatform/message/npool/order/gw/v1/powerrental"
	powerrental1 "github.com/NpoolPlatform/order-gateway/pkg/powerrental"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/NpoolPlatform/go-service-framework/pkg/logger"
)

func (s *Server) AdminUpdatePowerRentalOrder(ctx context.Context, in *npool.AdminUpdatePowerRentalOrderRequest) (*npool.AdminUpdatePowerRentalOrderResponse, error) {
	handler, err := powerrental1.NewHandler(
		ctx,
		powerrental1.WithID(&in.ID, true),
		powerrental1.WithEntID(&in.EntID, true),
		powerrental1.WithAppID(&in.TargetAppID, true),
		powerrental1.WithUserID(&in.TargetUserID, true),
		powerrental1.WithOrderID(&in.OrderID, true),
		powerrental1.WithAdminSetCanceled(in.Canceled, false),
	)
	if err != nil {
		logger.Sugar().Errorw(
			"AdminUpdatePowerRentalOrder",
			"In", in,
			"Error", err,
		)
		return &npool.AdminUpdatePowerRentalOrderResponse{}, status.Error(codes.InvalidArgument, err.Error())
	}

	info, err := handler.UpdatePowerRentalOrder(ctx)
	if err != nil {
		logger.Sugar().Errorw(
			"AdminUpdatePowerRentalOrder",
			"In", in,
			"Error", err,
		)
		return &npool.AdminUpdatePowerRentalOrderResponse{}, status.Error(codes.Internal, err.Error())
	}

	return &npool.AdminUpdatePowerRentalOrderResponse{
		Info: info,
	}, nil
}
