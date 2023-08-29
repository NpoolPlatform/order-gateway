//nolint:dupl
package order

import (
	"context"

	ordertypes "github.com/NpoolPlatform/message/npool/basetypes/order/v1"

	npool "github.com/NpoolPlatform/message/npool/order/gw/v1/order"
	order1 "github.com/NpoolPlatform/order-gateway/pkg/order"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/NpoolPlatform/go-service-framework/pkg/logger"
)

func createOrder(ctx context.Context, in *npool.CreateOrderRequest) (*npool.Order, error) { //nolint
	handler, err := order1.NewHandler(
		ctx,
		order1.WithAppID(&in.AppID, true),
		order1.WithUserID(&in.AppID, &in.UserID, true),
		order1.WithGoodID(&in.GoodID, true),
		order1.WithUnits(in.Units, true),
		order1.WithPaymentCoinID(&in.PaymentCoinID, true),
		order1.WithParentOrderID(in.ParentOrderID, false),
		order1.WithOrderType(&in.OrderType, true),
		order1.WithBalanceAmount(in.GetPayWithBalanceAmount(), false),
		order1.WithCouponIDs(in.CouponIDs, false),
	)
	if err != nil {
		logger.Sugar().Errorw(
			"createOrder",
			"In", in,
			"Error", err,
		)
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	info, err := handler.CreateOrder(ctx)
	if err != nil {
		logger.Sugar().Errorw(
			"createOrder",
			"In", in,
			"Error", err,
		)
		return nil, status.Error(codes.Internal, err.Error())
	}

	return info, nil
}

func (s *Server) CreateOrder(ctx context.Context, in *npool.CreateOrderRequest) (*npool.CreateOrderResponse, error) {
	in.OrderType = ordertypes.OrderType_Normal
	ord, err := createOrder(ctx, in)
	if err != nil {
		return &npool.CreateOrderResponse{}, err
	}
	return &npool.CreateOrderResponse{
		Info: ord,
	}, nil
}

func (s *Server) CreateUserOrder(ctx context.Context, in *npool.CreateUserOrderRequest) (*npool.CreateUserOrderResponse, error) {
	switch in.OrderType {
	case ordertypes.OrderType_Offline:
	case ordertypes.OrderType_Airdrop:
	default:
		return &npool.CreateUserOrderResponse{}, status.Errorf(codes.InvalidArgument, "order type invalid")
	}

	ord, err := createOrder(ctx, &npool.CreateOrderRequest{
		AppID:         in.AppID,
		UserID:        in.TargetUserID,
		GoodID:        in.GoodID,
		Units:         in.Units,
		PaymentCoinID: in.PaymentCoinID,
		ParentOrderID: in.ParentOrderID,
		OrderType:     in.OrderType,
	})
	if err != nil {
		return &npool.CreateUserOrderResponse{}, err
	}
	return &npool.CreateUserOrderResponse{
		Info: ord,
	}, nil
}

func (s *Server) CreateAppUserOrder(ctx context.Context, in *npool.CreateAppUserOrderRequest) (*npool.CreateAppUserOrderResponse, error) {
	switch in.OrderType {
	case ordertypes.OrderType_Offline:
	case ordertypes.OrderType_Airdrop:
	default:
		return &npool.CreateAppUserOrderResponse{}, status.Errorf(codes.InvalidArgument, "order type invalid")
	}

	ord, err := createOrder(ctx, &npool.CreateOrderRequest{
		AppID:         in.TargetAppID,
		UserID:        in.TargetUserID,
		GoodID:        in.GoodID,
		PaymentCoinID: in.PaymentCoinID,
		Units:         in.Units,
		ParentOrderID: in.ParentOrderID,
		OrderType:     in.OrderType,
	})
	if err != nil {
		return &npool.CreateAppUserOrderResponse{}, err
	}
	return &npool.CreateAppUserOrderResponse{
		Info: ord,
	}, nil
}
