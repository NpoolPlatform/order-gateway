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

func (s *Server) CreateOrder(ctx context.Context, in *npool.CreateOrderRequest) (*npool.CreateOrderResponse, error) {
	orderType := ordertypes.OrderType_Normal
	handler, err := order1.NewHandler(
		ctx,
		order1.WithAppID(&in.AppID, true),
		order1.WithUserID(&in.UserID, true),
		order1.WithAppGoodID(&in.AppGoodID, true),
		order1.WithUnits(in.Units, true),
		order1.WithDuration(in.Duration, true),
		order1.WithPaymentCoinID(&in.PaymentCoinID, true),
		order1.WithParentOrderID(in.ParentOrderID, false),
		order1.WithOrderType(&orderType, true),
		order1.WithBalanceAmount(in.PayWithBalanceAmount, false),
		order1.WithCouponIDs(in.CouponIDs, false),
		order1.WithInvestmentType(&in.InvestmentType, true),
	)
	if err != nil {
		logger.Sugar().Errorw(
			"CreateOrder",
			"In", in,
			"Error", err,
		)
		return &npool.CreateOrderResponse{}, status.Error(codes.InvalidArgument, err.Error())
	}

	info, err := handler.CreateOrder(ctx)
	if err != nil {
		logger.Sugar().Errorw(
			"CreateOrder",
			"In", in,
			"Error", err,
		)
		return &npool.CreateOrderResponse{}, status.Error(codes.Internal, err.Error())
	}

	return &npool.CreateOrderResponse{
		Info: info,
	}, nil
}

func (s *Server) CreateUserOrder(ctx context.Context, in *npool.CreateUserOrderRequest) (*npool.CreateUserOrderResponse, error) {
	switch in.OrderType {
	case ordertypes.OrderType_Offline:
	case ordertypes.OrderType_Airdrop:
	default:
		return &npool.CreateUserOrderResponse{}, status.Errorf(codes.InvalidArgument, "order type invalid")
	}

	handler, err := order1.NewHandler(
		ctx,
		order1.WithAppID(&in.AppID, true),
		order1.WithUserID(&in.TargetUserID, true),
		order1.WithAppGoodID(&in.AppGoodID, true),
		order1.WithUnits(in.Units, true),
		order1.WithDuration(in.Duration, true),
		order1.WithParentOrderID(in.ParentOrderID, false),
		order1.WithOrderType(&in.OrderType, true),
		order1.WithInvestmentType(&in.InvestmentType, true),
	)
	if err != nil {
		logger.Sugar().Errorw(
			"CreateUserOrder",
			"In", in,
			"Error", err,
		)
		return &npool.CreateUserOrderResponse{}, status.Error(codes.InvalidArgument, err.Error())
	}

	info, err := handler.CreateOrder(ctx)
	if err != nil {
		logger.Sugar().Errorw(
			"CreateUserOrder",
			"In", in,
			"Error", err,
		)
		return &npool.CreateUserOrderResponse{}, status.Error(codes.Internal, err.Error())
	}

	return &npool.CreateUserOrderResponse{
		Info: info,
	}, nil
}

func (s *Server) CreateAppUserOrder(ctx context.Context, in *npool.CreateAppUserOrderRequest) (*npool.CreateAppUserOrderResponse, error) {
	switch in.OrderType {
	case ordertypes.OrderType_Offline:
	case ordertypes.OrderType_Airdrop:
	default:
		return &npool.CreateAppUserOrderResponse{}, status.Errorf(codes.InvalidArgument, "order type invalid")
	}

	handler, err := order1.NewHandler(
		ctx,
		order1.WithAppID(&in.TargetAppID, true),
		order1.WithUserID(&in.TargetUserID, true),
		order1.WithAppGoodID(&in.AppGoodID, true),
		order1.WithUnits(in.Units, true),
		order1.WithDuration(in.Duration, true),
		order1.WithParentOrderID(in.ParentOrderID, false),
		order1.WithOrderType(&in.OrderType, true),
		order1.WithInvestmentType(&in.InvestmentType, true),
	)
	if err != nil {
		logger.Sugar().Errorw(
			"CreateAppUserOrder",
			"In", in,
			"Error", err,
		)
		return &npool.CreateAppUserOrderResponse{}, status.Error(codes.InvalidArgument, err.Error())
	}

	info, err := handler.CreateOrder(ctx)
	if err != nil {
		logger.Sugar().Errorw(
			"CreateAppUserOrder",
			"In", in,
			"Error", err,
		)
		return &npool.CreateAppUserOrderResponse{}, status.Error(codes.Internal, err.Error())
	}

	return &npool.CreateAppUserOrderResponse{
		Info: info,
	}, nil
}

func (s *Server) CreateOrders(ctx context.Context, in *npool.CreateOrdersRequest) (*npool.CreateOrdersResponse, error) {
	orderType := ordertypes.OrderType_Normal
	handler, err := order1.NewHandler(
		ctx,
		order1.WithAppID(&in.AppID, true),
		order1.WithUserID(&in.UserID, true),
		order1.WithPaymentCoinID(&in.PaymentCoinID, true),
		order1.WithOrderType(&orderType, true),
		order1.WithBalanceAmount(in.PayWithBalanceAmount, false),
		order1.WithCouponIDs(in.CouponIDs, false),
		order1.WithInvestmentType(&in.InvestmentType, true),
		order1.WithOrders(in.Orders, true),
	)
	if err != nil {
		logger.Sugar().Errorw(
			"CreateOrders",
			"In", in,
			"Error", err,
		)
		return &npool.CreateOrdersResponse{}, status.Error(codes.InvalidArgument, err.Error())
	}

	infos, err := handler.CreateOrders(ctx)
	if err != nil {
		logger.Sugar().Errorw(
			"CreateOrders",
			"In", in,
			"Error", err,
		)
		return &npool.CreateOrdersResponse{}, status.Error(codes.Internal, err.Error())
	}

	return &npool.CreateOrdersResponse{
		Infos: infos,
	}, nil
}
