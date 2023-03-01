//nolint:nolintlint,dupl
package order

import (
	"context"

	order1 "github.com/NpoolPlatform/order-gateway/pkg/order"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/NpoolPlatform/go-service-framework/pkg/logger"
	npool "github.com/NpoolPlatform/message/npool/order/gw/v1/order"
	ordermwpb "github.com/NpoolPlatform/message/npool/order/mw/v1/order"

	"github.com/google/uuid"
)

func (s *Server) UpdateOrder(ctx context.Context, in *npool.UpdateOrderRequest) (*npool.UpdateOrderResponse, error) {
	if _, err := uuid.Parse(in.GetAppID()); err != nil {
		logger.Sugar().Errorw("UpdateOrder", "AppID", in.GetAppID(), "error", err)
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	if _, err := uuid.Parse(in.GetUserID()); err != nil {
		logger.Sugar().Errorw("UpdateOrder", "UserID", in.GetUserID(), "error", err)
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	if _, err := uuid.Parse(in.GetID()); err != nil {
		logger.Sugar().Errorw("UpdateOrder", "ID", in.GetID(), "error", err)
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	if _, err := uuid.Parse(in.GetPaymentID()); err != nil {
		logger.Sugar().Errorw("UpdateOrder", "PaymentID", in.GetPaymentID(), "error", err)
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	if in.Canceled == nil {
		logger.Sugar().Errorw("UpdateOrder", "error", "nothing todo")
		return nil, status.Error(codes.InvalidArgument, "nothing todo")
	}

	ord, err := order1.UpdateOrder(ctx, &ordermwpb.OrderReq{
		AppID:     &in.AppID,
		UserID:    &in.UserID,
		ID:        &in.ID,
		PaymentID: &in.PaymentID,
		Canceled:  in.Canceled,
	}, false)
	if err != nil {
		logger.Sugar().Errorw("UpdateOrder", "error", err)
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &npool.UpdateOrderResponse{
		Info: ord,
	}, nil
}

func (s *Server) UpdateUserOrder(ctx context.Context, in *npool.UpdateUserOrderRequest) (*npool.UpdateUserOrderResponse, error) {
	if _, err := uuid.Parse(in.GetAppID()); err != nil {
		logger.Sugar().Errorw("UpdateUserOrder", "AppID", in.GetAppID(), "error", err)
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	if _, err := uuid.Parse(in.GetTargetUserID()); err != nil {
		logger.Sugar().Errorw("UpdateUserOrder", "UserID", in.GetTargetUserID(), "error", err)
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	if _, err := uuid.Parse(in.GetID()); err != nil {
		logger.Sugar().Errorw("UpdateUserOrder", "ID", in.GetID(), "error", err)
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	if _, err := uuid.Parse(in.GetPaymentID()); err != nil {
		logger.Sugar().Errorw("UpdateUserOrder", "PaymentID", in.GetPaymentID(), "error", err)
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	if in.Canceled == nil {
		logger.Sugar().Errorw("UpdateUserOrder", "error", "nothing todo")
		return nil, status.Error(codes.InvalidArgument, "nothing todo")
	}

	ord, err := order1.UpdateOrder(ctx, &ordermwpb.OrderReq{
		AppID:     &in.AppID,
		UserID:    &in.TargetUserID,
		ID:        &in.ID,
		PaymentID: &in.PaymentID,
		Canceled:  in.Canceled,
	}, true)
	if err != nil {
		logger.Sugar().Errorw("UpdateUserOrder", "error", err)
		return nil, status.Error(codes.Internal, "fail update order")
	}

	return &npool.UpdateUserOrderResponse{
		Info: ord,
	}, nil
}

func (s *Server) UpdateAppUserOrder(ctx context.Context, in *npool.UpdateAppUserOrderRequest) (*npool.UpdateAppUserOrderResponse, error) {
	if _, err := uuid.Parse(in.GetTargetUserID()); err != nil {
		logger.Sugar().Errorw("UpdateAppUserOrder", "AppID", in.GetTargetUserID(), "error", err)
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	if _, err := uuid.Parse(in.GetTargetUserID()); err != nil {
		logger.Sugar().Errorw("UpdateAppUserOrder", "UserID", in.GetTargetUserID(), "error", err)
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	if _, err := uuid.Parse(in.GetID()); err != nil {
		logger.Sugar().Errorw("UpdateAppUserOrder", "ID", in.GetID(), "error", err)
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	if _, err := uuid.Parse(in.GetPaymentID()); err != nil {
		logger.Sugar().Errorw("UpdateAppUserOrder", "PaymentID", in.GetPaymentID(), "error", err)
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	if in.Canceled == nil {
		logger.Sugar().Errorw("UpdateAppUserOrder", "error", "nothing todo")
		return nil, status.Error(codes.InvalidArgument, "nothing todo")
	}

	ord, err := order1.UpdateOrder(ctx, &ordermwpb.OrderReq{
		AppID:     &in.TargetAppID,
		UserID:    &in.TargetUserID,
		ID:        &in.ID,
		PaymentID: &in.PaymentID,
		Canceled:  in.Canceled,
	}, true)
	if err != nil {
		logger.Sugar().Errorw("UpdateAppUserOrder", "error", err)
		return nil, status.Error(codes.Internal, "fail update order")
	}

	return &npool.UpdateAppUserOrderResponse{
		Info: ord,
	}, nil
}
