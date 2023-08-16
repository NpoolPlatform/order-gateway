package order

import (
	"context"

	"github.com/NpoolPlatform/go-service-framework/pkg/logger"

	npool "github.com/NpoolPlatform/message/npool/order/gw/v1/order"
	order1 "github.com/NpoolPlatform/order-gateway/pkg/order"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (s *Server) GetOrders(ctx context.Context, in *npool.GetOrdersRequest) (*npool.GetOrdersResponse, error) {
	handler, err := order1.NewHandler(
		ctx,
		order1.WithAppID(&in.AppID),
		order1.WithUserID(&in.AppID, &in.UserID),
		order1.WithOffset(in.GetOffset()),
		order1.WithLimit(in.GetLimit()),
	)
	if err != nil {
		logger.Sugar().Errorw(
			"GetOrders",
			"In", in,
			"Error", err,
		)
		return &npool.GetOrdersResponse{}, status.Error(codes.InvalidArgument, err.Error())
	}

	infos, total, err := handler.GetOrders(ctx)
	if err != nil {
		logger.Sugar().Errorw(
			"GetOrders",
			"In", in,
			"Error", err,
		)
		return &npool.GetOrdersResponse{}, status.Error(codes.Internal, err.Error())
	}

	return &npool.GetOrdersResponse{
		Infos: infos,
		Total: total,
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
	handler, err := order1.NewHandler(
		ctx,
		order1.WithID(&in.ID),
	)
	if err != nil {
		logger.Sugar().Errorw(
			"GetOrder",
			"In", in,
			"Error", err,
		)
		return &npool.GetOrderResponse{}, status.Error(codes.InvalidArgument, err.Error())
	}

	info, err := handler.GetOrder(ctx)
	if err != nil {
		logger.Sugar().Errorw(
			"GetOrder",
			"In", in,
			"Error", err,
		)
		return &npool.GetOrderResponse{}, status.Error(codes.Internal, err.Error())
	}

	if info.AppID != in.GetAppID() || info.UserID != in.GetUserID() {
		logger.Sugar().Errorw("GetOrder", "Order", info, "error", "permission denied")
		return &npool.GetOrderResponse{}, status.Error(codes.PermissionDenied, "permission denied")
	}

	return &npool.GetOrderResponse{
		Info: info,
	}, nil
}

func (s *Server) GetAppOrders(ctx context.Context, in *npool.GetAppOrdersRequest) (*npool.GetAppOrdersResponse, error) {
	resp, err := s.GetOrders(ctx, &npool.GetOrdersRequest{
		AppID:  in.GetAppID(),
		Offset: in.Offset,
		Limit:  in.Limit,
	})
	if err != nil {
		return &npool.GetAppOrdersResponse{}, err
	}

	return &npool.GetAppOrdersResponse{
		Infos: resp.Infos,
		Total: resp.Total,
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
