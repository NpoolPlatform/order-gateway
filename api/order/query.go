package order

import (
	"context"

	"github.com/NpoolPlatform/go-service-framework/pkg/logger"

	npool "github.com/NpoolPlatform/message/npool/order/gw/v1/order"
	order1 "github.com/NpoolPlatform/order-gateway/pkg/order"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/google/uuid"
)

func (s *Server) GetOrders(ctx context.Context, in *npool.GetOrdersRequest) (*npool.GetOrdersResponse, error) {
	if _, err := uuid.Parse(in.GetAppID()); err != nil {
		logger.Sugar().Errorw("GetOrders", "AppID", in.GetAppID(), "error", err)
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	if _, err := uuid.Parse(in.GetUserID()); err != nil {
		logger.Sugar().Errorw("GetOrders", "UserID", in.GetUserID(), "error", err)
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	ords, n, err := order1.GetOrders(ctx, in.GetAppID(), in.GetUserID(), in.GetOffset(), in.GetLimit())
	if err != nil {
		logger.Sugar().Errorw("GetOrders", "error", err)
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &npool.GetOrdersResponse{
		Infos: ords,
		Total: n,
	}, nil
}

func (s *Server) GetUserOrders(ctx context.Context, in *npool.GetUserOrdersRequest) (*npool.GetUserOrdersResponse, error) {
	resp, err := s.GetOrders(ctx, &npool.GetOrdersRequest{
		AppID:  in.AppID,
		UserID: in.TargetUserID,
		Offset: in.Offset,
		Limit:  in.Limit,
	})
	if err != nil {
		return nil, err
	}

	return &npool.GetUserOrdersResponse{
		Infos: resp.Infos,
		Total: resp.Total,
	}, nil
}

func (s *Server) GetAppUserOrders(ctx context.Context, in *npool.GetAppUserOrdersRequest) (*npool.GetAppUserOrdersResponse, error) {
	resp, err := s.GetOrders(ctx, &npool.GetOrdersRequest{
		AppID:  in.TargetAppID,
		UserID: in.TargetUserID,
		Offset: in.Offset,
		Limit:  in.Limit,
	})
	if err != nil {
		return nil, err
	}

	return &npool.GetAppUserOrdersResponse{
		Infos: resp.Infos,
		Total: resp.Total,
	}, nil
}
