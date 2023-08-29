//nolint:nolintlint,dupl
package order

import (
	"context"

	order1 "github.com/NpoolPlatform/order-gateway/pkg/order"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/NpoolPlatform/go-service-framework/pkg/logger"
	npool "github.com/NpoolPlatform/message/npool/order/gw/v1/order"
)

func (s *Server) UpdateOrder(ctx context.Context, in *npool.UpdateOrderRequest) (*npool.UpdateOrderResponse, error) {
	handler, err := order1.NewHandler(
		ctx,
		order1.WithID(&in.ID, true),
		order1.WithAppID(&in.AppID, true),
		order1.WithUserID(&in.AppID, &in.UserID, true),
		order1.WithPaymentID(&in.PaymentID, true),
		order1.WithCanceled(in.Canceled, true),
		order1.WithFromAdmin(false),
	)
	if err != nil {
		logger.Sugar().Errorw(
			"UpdateOrder",
			"In", in,
			"Error", err,
		)
		return &npool.UpdateOrderResponse{}, status.Error(codes.InvalidArgument, err.Error())
	}

	info, err := handler.UpdateOrder(ctx)
	if err != nil {
		logger.Sugar().Errorw(
			"UpdateOrder",
			"In", in,
			"Error", err,
		)
		return &npool.UpdateOrderResponse{}, status.Error(codes.Internal, err.Error())
	}

	return &npool.UpdateOrderResponse{
		Info: info,
	}, nil
}

func (s *Server) UpdateUserOrder(ctx context.Context, in *npool.UpdateUserOrderRequest) (*npool.UpdateUserOrderResponse, error) {
	handler, err := order1.NewHandler(
		ctx,
		order1.WithID(&in.ID, true),
		order1.WithAppID(&in.AppID, true),
		order1.WithUserID(&in.AppID, &in.TargetUserID, true),
		order1.WithPaymentID(&in.PaymentID, true),
		order1.WithCanceled(in.Canceled, true),
		order1.WithFromAdmin(false),
	)
	if err != nil {
		logger.Sugar().Errorw(
			"UpdateUserOrder",
			"In", in,
			"Error", err,
		)
		return &npool.UpdateUserOrderResponse{}, status.Error(codes.InvalidArgument, err.Error())
	}

	info, err := handler.UpdateOrder(ctx)
	if err != nil {
		logger.Sugar().Errorw(
			"UpdateUserOrder",
			"In", in,
			"Error", err,
		)
		return &npool.UpdateUserOrderResponse{}, status.Error(codes.Internal, err.Error())
	}

	return &npool.UpdateUserOrderResponse{
		Info: info,
	}, nil
}

func (s *Server) UpdateAppUserOrder(ctx context.Context, in *npool.UpdateAppUserOrderRequest) (*npool.UpdateAppUserOrderResponse, error) {
	handler, err := order1.NewHandler(
		ctx,
		order1.WithID(&in.ID, true),
		order1.WithAppID(&in.TargetAppID, true),
		order1.WithUserID(&in.TargetAppID, &in.TargetUserID, true),
		order1.WithPaymentID(&in.PaymentID, true),
		order1.WithCanceled(in.Canceled, true),
		order1.WithFromAdmin(false),
	)
	if err != nil {
		logger.Sugar().Errorw(
			"UpdateAppUserOrder",
			"In", in,
			"Error", err,
		)
		return &npool.UpdateAppUserOrderResponse{}, status.Error(codes.InvalidArgument, err.Error())
	}

	info, err := handler.UpdateOrder(ctx)
	if err != nil {
		logger.Sugar().Errorw(
			"UpdateAppUserOrder",
			"In", in,
			"Error", err,
		)
		return &npool.UpdateAppUserOrderResponse{}, status.Error(codes.Internal, err.Error())
	}

	return &npool.UpdateAppUserOrderResponse{
		Info: info,
	}, nil
}
