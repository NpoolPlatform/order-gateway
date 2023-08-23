//nolint:dupl
package order

import (
	"context"

	appgoodmwcli "github.com/NpoolPlatform/good-middleware/pkg/client/app/good"
	appgoodpb "github.com/NpoolPlatform/message/npool/good/mw/v1/app/good"

	"github.com/NpoolPlatform/libent-cruder/pkg/cruder"

	ordertypes "github.com/NpoolPlatform/message/npool/basetypes/order/v1"
	basetypes "github.com/NpoolPlatform/message/npool/basetypes/v1"

	npool "github.com/NpoolPlatform/message/npool/order/gw/v1/order"
	ordermwpb "github.com/NpoolPlatform/message/npool/order/mw/v1/order"
	order1 "github.com/NpoolPlatform/order-gateway/pkg/order"
	ordermwcli "github.com/NpoolPlatform/order-middleware/pkg/client/order"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/NpoolPlatform/go-service-framework/pkg/logger"

	"github.com/shopspring/decimal"
)

func createOrder(ctx context.Context, in *npool.CreateOrderRequest) (*npool.Order, error) { //nolint
	handler, err := order1.NewHandler(
		ctx,
		order1.WithAppID(&in.AppID),
		order1.WithUserID(&in.AppID, &in.UserID),
		order1.WithGoodID(&in.GoodID),
		order1.WithUnits(in.Units),
		order1.WithPaymentCoinID(&in.PaymentCoinID),
		order1.WithParentOrderID(in.ParentOrderID),
		order1.WithOrderType(&in.OrderType),
		order1.WithPayWithBalanceAmount(in.GetPayWithBalanceAmount()),
		order1.WithFixAmountID(in.FixAmountID),
		order1.WithDiscountID(in.DiscountID),
		order1.WithSpecialOfferID(in.SpecialOfferID),
		order1.WithCouponIDs(in.CouponIDs),
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
	ag, err := appgoodmwcli.GetGoodOnly(ctx, &appgoodpb.Conds{
		AppID: &basetypes.StringVal{
			Op:    cruder.EQ,
			Value: in.AppID,
		},
		GoodID: &basetypes.StringVal{
			Op:    cruder.EQ,
			Value: in.GoodID,
		},
	})
	if err != nil {
		return &npool.CreateOrderResponse{}, status.Error(codes.Internal, err.Error())
	}
	if ag == nil {
		return &npool.CreateOrderResponse{}, status.Error(codes.Internal, "invalid app good")
	}
	units, err := decimal.NewFromString(in.Units)
	if err != nil {
		return &npool.CreateOrderResponse{}, status.Error(codes.Internal, err.Error())
	}
	if ag.PurchaseLimit > 0 && units.Cmp(decimal.NewFromInt32(ag.PurchaseLimit)) > 0 {
		return &npool.CreateOrderResponse{}, status.Error(codes.Internal, "too many units")
	}

	if !ag.EnablePurchase {
		return &npool.CreateOrderResponse{}, status.Error(codes.Internal, "app good is not enabled purchase")
	}

	purchaseCountStr, err := ordermwcli.SumOrderUnits(
		ctx,
		&ordermwpb.Conds{
			AppID:  &basetypes.StringVal{Op: cruder.EQ, Value: in.AppID},
			UserID: &basetypes.StringVal{Op: cruder.EQ, Value: in.UserID},
			GoodID: &basetypes.StringVal{Op: cruder.EQ, Value: in.GoodID},
			States: &basetypes.Uint32SliceVal{
				Op: cruder.IN,
				Value: []uint32{
					uint32(ordertypes.OrderState_OrderStatePaid),
					uint32(ordertypes.OrderState_OrderStateInService),
					uint32(ordertypes.OrderState_OrderStateExpired),
					uint32(ordertypes.OrderState_OrderStateWaitPayment),
				},
			},
		},
	)
	if err != nil {
		return nil, err
	}

	purchaseCount, err := decimal.NewFromString(purchaseCountStr)
	if err != nil {
		return nil, err
	}

	userPurchaseLimit, err := decimal.NewFromString(ag.UserPurchaseLimit)
	if err != nil {
		logger.Sugar().Errorw("ValidateInit", "error", err)
		return &npool.CreateOrderResponse{}, status.Error(codes.Internal, err.Error())
	}

	if userPurchaseLimit.Cmp(decimal.NewFromInt(0)) > 0 && purchaseCount.Add(units).Cmp(userPurchaseLimit) > 0 {
		return &npool.CreateOrderResponse{}, status.Error(codes.Internal, "too many units")
	}

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
		PaymentCoinID: in.PaymentCoinID,
		Units:         in.Units,
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
