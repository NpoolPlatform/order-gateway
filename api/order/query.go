package order

import (
	"context"

	npool "github.com/NpoolPlatform/message/npool/order/gw/v1/order"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (s *Server) GetOrders(ctx context.Context, in *npool.GetOrdersRequest) (*npool.GetOrdersResponse, error) {
	return nil, status.Error(codes.Internal, "NOT IMPLEMENTED")
}

func (s *Server) GetUserOrders(ctx context.Context, in *npool.GetUserOrdersRequest) (*npool.GetUserOrdersResponse, error) {
	return nil, status.Error(codes.Internal, "NOT IMPLEMENTED")
}

func (s *Server) GetAppUserOrders(ctx context.Context, in *npool.GetAppUserOrdersRequest) (*npool.GetAppUserOrdersResponse, error) {
	return nil, status.Error(codes.Internal, "NOT IMPLEMENTED")
}
