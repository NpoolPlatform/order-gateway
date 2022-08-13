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
		return &npool.GetOrdersResponse{}, status.Error(codes.InvalidArgument, err.Error())
	}

	if _, err := uuid.Parse(in.GetUserID()); err != nil {
		logger.Sugar().Errorw("GetOrders", "UserID", in.GetUserID(), "error", err)
		return &npool.GetOrdersResponse{}, status.Error(codes.InvalidArgument, err.Error())
	}

	ords, n, err := order1.GetOrders(ctx, in.GetAppID(), in.GetUserID(), in.GetOffset(), in.GetLimit())
	if err != nil {
		logger.Sugar().Errorw("GetOrders", "error", err)
		return &npool.GetOrdersResponse{}, status.Error(codes.Internal, err.Error())
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
		return &npool.GetUserOrdersResponse{}, err
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
		return &npool.GetAppUserOrdersResponse{}, err
	}

	return &npool.GetAppUserOrdersResponse{
		Infos: resp.Infos,
		Total: resp.Total,
	}, nil
}

func (s *Server) GetOrder(ctx context.Context, in *npool.GetOrderRequest) (*npool.GetOrderResponse, error) {
	if _, err := uuid.Parse(in.GetAppID()); err != nil {
		logger.Sugar().Errorw("GetOrder", "AppID", in.GetAppID(), "error", err)
		return &npool.GetOrderResponse{}, status.Error(codes.InvalidArgument, err.Error())
	}

	if _, err := uuid.Parse(in.GetUserID()); err != nil {
		logger.Sugar().Errorw("GetOrder", "UserID", in.GetUserID(), "error", err)
		return &npool.GetOrderResponse{}, status.Error(codes.InvalidArgument, err.Error())
	}

	if _, err := uuid.Parse(in.GetID()); err != nil {
		logger.Sugar().Errorw("GetOrder", "ID", in.GetID(), "error", err)
		return &npool.GetOrderResponse{}, status.Error(codes.InvalidArgument, err.Error())
	}

	ord, err := order1.GetOrder(ctx, in.GetID())
	if err != nil {
		logger.Sugar().Errorw("GetOrder", "error", err)
		return &npool.GetOrderResponse{}, status.Error(codes.Internal, err.Error())
	}

	if ord.AppID != in.GetAppID() || ord.UserID != in.GetUserID() {
		logger.Sugar().Errorw("GetOrder", "Order", ord, "error", "permission denied")
		return &npool.GetOrderResponse{}, status.Error(codes.PermissionDenied, "permission denied")
	}

	return &npool.GetOrderResponse{
		Info: ord,
	}, nil
}

func (s *Server) GetAppOrders(ctx context.Context, in *npool.GetAppOrdersRequest) (*npool.GetAppOrdersResponse, error) {
	if _, err := uuid.Parse(in.GetAppID()); err != nil {
		logger.Sugar().Errorw("GetAppOrders", "AppID", in.GetAppID(), "error", err)
		return &npool.GetAppOrdersResponse{}, status.Error(codes.InvalidArgument, err.Error())
	}

	ords, n, err := order1.GetAppOrders(ctx, in.GetAppID(), in.GetOffset(), in.GetLimit())
	if err != nil {
		logger.Sugar().Errorw("GetAppOrders", "error", err)
		return &npool.GetAppOrdersResponse{}, status.Error(codes.Internal, err.Error())
	}

	return &npool.GetAppOrdersResponse{
		Infos: ords,
		Total: n,
	}, nil
}

func (s *Server) GetNAppOrders(ctx context.Context, in *npool.GetNAppOrdersRequest) (*npool.GetNAppOrdersResponse, error) {
	resp, err := s.GetAppOrders(ctx, &npool.GetAppOrdersRequest{
		AppID:  in.TargetAppID,
		Offset: in.Offset,
		Limit:  in.Limit,
	})
	if err != nil {
		return &npool.GetNAppOrdersResponse{}, err
	}

	return &npool.GetNAppOrdersResponse{
		Infos: resp.Infos,
		Total: resp.Total,
	}, nil
}
