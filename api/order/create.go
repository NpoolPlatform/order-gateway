//nolint:nolintlint,dupl
package order

import (
	"context"
	commontracer "github.com/NpoolPlatform/order-gateway/pkg/tracer"

	constant "github.com/NpoolPlatform/order-gateway/pkg/message/const"
	order1 "github.com/NpoolPlatform/order-gateway/pkg/order"

	"go.opentelemetry.io/otel"
	scodes "go.opentelemetry.io/otel/codes"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/NpoolPlatform/go-service-framework/pkg/logger"
	npool "github.com/NpoolPlatform/message/npool/order/gw/v1/order"
	ordermgrpb "github.com/NpoolPlatform/message/npool/order/mgr/v1/order/order"

	"github.com/shopspring/decimal"

	"github.com/google/uuid"
)

func createOrder(ctx context.Context, in *npool.CreateOrderRequest) (*npool.Order, error) { //nolint
	var err error

	_, span := otel.Tracer(constant.ServiceName).Start(ctx, "CreateOrder")
	defer span.End()

	defer func() {
		if err != nil {
			span.SetStatus(scodes.Error, err.Error())
			span.RecordError(err)
		}
	}()

	if _, err := uuid.Parse(in.GetAppID()); err != nil {
		logger.Sugar().Errorw("CreateOrder", "AppID", in.GetAppID(), "error", err)
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}
	if _, err := uuid.Parse(in.GetUserID()); err != nil {
		logger.Sugar().Errorw("CreateOrder", "UserID", in.GetUserID(), "error", err)
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}
	if _, err := uuid.Parse(in.GetGoodID()); err != nil {
		logger.Sugar().Errorw("CreateOrder", "GoodID", in.GetGoodID(), "error", err)
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}
	if in.GetUnits() <= 0 {
		logger.Sugar().Errorw("CreateOrder", "Units", in.GetUnits())
		return nil, status.Error(codes.InvalidArgument, "Units is 0")
	}
	if _, err := uuid.Parse(in.GetPaymentCoinID()); err != nil {
		logger.Sugar().Errorw("CreateOrder", "PaymentCoinID", in.GetPaymentCoinID(), "error", err)
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}
	if in.ParentOrderID != nil {
		if _, err := uuid.Parse(in.GetParentOrderID()); err != nil {
			logger.Sugar().Errorw("CreateOrder", "ParentOrderID", in.GetParentOrderID(), "error", err)
			return nil, status.Error(codes.InvalidArgument, err.Error())
		}
	}
	if in.PayWithBalanceAmount != nil {
		amount, err := decimal.NewFromString(in.GetPayWithBalanceAmount())
		if err != nil {
			logger.Sugar().Errorw("CreateOrder", "PayWithBalanceAmount", in.GetPayWithBalanceAmount(), "error", err)
			return nil, status.Error(codes.InvalidArgument, err.Error())
		}
		if amount.Cmp(decimal.NewFromInt(0)) < 0 {
			logger.Sugar().Errorw("CreateOrder", "PayWithBalanceAmount", in.GetPayWithBalanceAmount())
			return nil, status.Error(codes.InvalidArgument, "PayWithBalanceAmount less than 0")
		}
	}
	if in.FixAmountID != nil {
		if _, err := uuid.Parse(in.GetFixAmountID()); err != nil {
			logger.Sugar().Errorw("CreateOrder", "FixAmountID", in.GetFixAmountID(), "error", err)
			return nil, status.Error(codes.InvalidArgument, err.Error())
		}
	}
	if in.DiscountID != nil {
		if _, err := uuid.Parse(in.GetDiscountID()); err != nil {
			logger.Sugar().Errorw("CreateOrder", "DiscountID", in.GetDiscountID(), "error", err)
			return nil, status.Error(codes.InvalidArgument, err.Error())
		}
	}
	if in.SpecialOfferID != nil {
		if _, err := uuid.Parse(in.GetSpecialOfferID()); err != nil {
			logger.Sugar().Errorw("CreateOrder", "SpecialOfferID", in.GetSpecialOfferID(), "error", err)
			return nil, status.Error(codes.InvalidArgument, err.Error())
		}
	}

	span = commontracer.TraceInvoker(span, "order", "gateway", "CreateOrder")

	// Here we may create sub order according to good info, but we only return main order
	info, err := order1.CreateOrder(ctx, &order1.OrderCreate{
		AppID:          in.GetAppID(),
		UserID:         in.GetUserID(),
		GoodID:         in.GetGoodID(),
		PaymentCoinID:  in.GetPaymentCoinID(),
		Units:          in.GetUnits(),
		ParentOrderID:  in.ParentOrderID,
		BalanceAmount:  in.PayWithBalanceAmount,
		FixAmountID:    in.FixAmountID,
		DiscountID:     in.DiscountID,
		SpecialOfferID: in.SpecialOfferID,
		OrderType:      in.GetOrderType(),
	})
	if err != nil {
		logger.Sugar().Errorw("CreateOrder", "error", err)
		return nil, status.Error(codes.Internal, err.Error())
	}

	return info, nil
}

func (s *Server) CreateOrder(ctx context.Context, in *npool.CreateOrderRequest) (*npool.CreateOrderResponse, error) {
	in.OrderType = ordermgrpb.OrderType_Normal
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
	case ordermgrpb.OrderType_Offline:
	case ordermgrpb.OrderType_Airdrop:
	default:
		return &npool.CreateUserOrderResponse{}, status.Errorf(codes.InvalidArgument, "order type invalid")
	}

	ord, err := createOrder(ctx, &npool.CreateOrderRequest{
		AppID:                in.AppID,
		UserID:               in.TargetUserID,
		GoodID:               in.GoodID,
		PaymentCoinID:        in.PaymentCoinID,
		Units:                in.Units,
		ParentOrderID:        in.ParentOrderID,
		PayWithBalanceAmount: in.PayWithBalanceAmount,
		FixAmountID:          in.FixAmountID,
		DiscountID:           in.DiscountID,
		SpecialOfferID:       in.SpecialOfferID,
		OrderType:            in.OrderType,
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
	case ordermgrpb.OrderType_Offline:
	case ordermgrpb.OrderType_Airdrop:
	default:
		return &npool.CreateAppUserOrderResponse{}, status.Errorf(codes.InvalidArgument, "order type invalid")
	}

	ord, err := createOrder(ctx, &npool.CreateOrderRequest{
		AppID:                in.TargetAppID,
		UserID:               in.TargetUserID,
		GoodID:               in.GoodID,
		PaymentCoinID:        in.PaymentCoinID,
		Units:                in.Units,
		ParentOrderID:        in.ParentOrderID,
		PayWithBalanceAmount: in.PayWithBalanceAmount,
		FixAmountID:          in.FixAmountID,
		DiscountID:           in.DiscountID,
		SpecialOfferID:       in.SpecialOfferID,
		OrderType:            in.OrderType,
	})
	if err != nil {
		return &npool.CreateAppUserOrderResponse{}, err
	}
	return &npool.CreateAppUserOrderResponse{
		Info: ord,
	}, nil
}
