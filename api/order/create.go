//nolint:dupl
package order

import (
	"context"

	appgoodmwcli "github.com/NpoolPlatform/good-middleware/pkg/client/appgood"
	"github.com/NpoolPlatform/libent-cruder/pkg/cruder"
	commonpb "github.com/NpoolPlatform/message/npool"
	appgoodpb "github.com/NpoolPlatform/message/npool/good/mgr/v1/appgood"
	ordermwpb "github.com/NpoolPlatform/message/npool/order/mw/v1/order"
	ordermwcli "github.com/NpoolPlatform/order-middleware/pkg/client/order"

	commontracer "github.com/NpoolPlatform/order-gateway/pkg/tracer"

	constant "github.com/NpoolPlatform/order-gateway/pkg/message/const"
	order1 "github.com/NpoolPlatform/order-gateway/pkg/order"

	"go.opentelemetry.io/otel"
	scodes "go.opentelemetry.io/otel/codes"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/NpoolPlatform/go-service-framework/pkg/logger"
	npool "github.com/NpoolPlatform/message/npool/order/gw/v1/order"
	ordermgrpb "github.com/NpoolPlatform/message/npool/order/mgr/v1/order"

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
	units, err := decimal.NewFromString(in.GetUnits())
	if err != nil {
		logger.Sugar().Errorw("CreateOrder", "Units", in.GetUnits())
		return nil, status.Error(codes.InvalidArgument, "Units is 0")
	}
	if units.Cmp(decimal.NewFromInt32(0)) <= 0 {
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
	for _, id := range in.GetCouponIDs() {
		if _, err := uuid.Parse(id); err != nil {
			logger.Sugar().Errorw("CreateOrder", "error", err)
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
		CouponIDs:      in.CouponIDs,
	})
	if err != nil {
		logger.Sugar().Errorw("CreateOrder", "error", err)
		return nil, status.Error(codes.Internal, err.Error())
	}

	return info, nil
}

//nolint:gocyclo
func (s *Server) CreateOrder(ctx context.Context, in *npool.CreateOrderRequest) (*npool.CreateOrderResponse, error) {
	ag, err := appgoodmwcli.GetGoodOnly(ctx, &appgoodpb.Conds{
		AppID: &commonpb.StringVal{
			Op:    cruder.EQ,
			Value: in.AppID,
		},
		GoodID: &commonpb.StringVal{
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
	offset := int32(0)
	limit := int32(1000) //nolint
	purchaseCount := decimal.NewFromInt(0)
	for {
		orderInfos, _, err := ordermwcli.GetOrders(ctx, &ordermwpb.Conds{
			AppID: &commonpb.StringVal{
				Op:    cruder.EQ,
				Value: in.AppID,
			},
			UserID: &commonpb.StringVal{
				Op:    cruder.EQ,
				Value: in.UserID,
			},
			GoodID: &commonpb.StringVal{
				Op:    cruder.EQ,
				Value: in.GoodID,
			},
		}, offset, limit)
		if err != nil {
			return &npool.CreateOrderResponse{}, status.Error(codes.Internal, "too many units")
		}
		offset += limit
		if len(orderInfos) == 0 {
			break
		}

		for _, val := range orderInfos {
			orderUnits, err := decimal.NewFromString(val.Units)
			if err != nil {
				logger.Sugar().Errorw("ValidateInit", "error", err)
				continue
			}
			switch val.OrderState {
			case ordermgrpb.OrderState_Paid:
				fallthrough //nolint
			case ordermgrpb.OrderState_InService:
				fallthrough //nolint
			case ordermgrpb.OrderState_Expired:
				fallthrough //nolint
			case ordermgrpb.OrderState_WaitPayment:
				purchaseCount = purchaseCount.Add(orderUnits)
			}
		}
	}

	userPurchaseLimit, err := decimal.NewFromString(ag.UserPurchaseLimit)
	if err != nil {
		logger.Sugar().Errorw("ValidateInit", "error", err)
		return &npool.CreateOrderResponse{}, status.Error(codes.Internal, err.Error())
	}

	if userPurchaseLimit.Cmp(decimal.NewFromInt(0)) > 0 && purchaseCount.Add(units).Cmp(userPurchaseLimit) > 0 {
		return &npool.CreateOrderResponse{}, status.Error(codes.Internal, "too many units")
	}

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
	case ordermgrpb.OrderType_Offline:
	case ordermgrpb.OrderType_Airdrop:
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
