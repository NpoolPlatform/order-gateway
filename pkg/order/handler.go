package order

import (
	"context"
	"fmt"

	appmwcli "github.com/NpoolPlatform/appuser-middleware/pkg/client/app"
	appgoodmwcli "github.com/NpoolPlatform/good-middleware/pkg/client/app/good"
	cruder "github.com/NpoolPlatform/libent-cruder/pkg/cruder"
	ordertypes "github.com/NpoolPlatform/message/npool/basetypes/order/v1"
	basetypes "github.com/NpoolPlatform/message/npool/basetypes/v1"
	npool "github.com/NpoolPlatform/message/npool/order/gw/v1/order"
	ordermwpb "github.com/NpoolPlatform/message/npool/order/mw/v1/order"
	constant "github.com/NpoolPlatform/order-gateway/pkg/const"
	ordermwcli "github.com/NpoolPlatform/order-middleware/pkg/client/order"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

type Handler struct {
	ID               *string
	AppID            *string
	UserID           *string
	AppGoodID        *string
	Units            string
	PaymentCoinID    *string
	ParentOrderID    *string
	BalanceAmount    *string
	OrderType        *ordertypes.OrderType
	CouponIDs        []string
	InvestmentType   *ordertypes.InvestmentType
	UserSetCanceled  *bool
	AdminSetCanceled *bool
	PaymentID        *string
	Offset           int32
	Limit            int32
	Orders           []*npool.CreateOrdersRequest_OrderReq
	IDs              []string
}

func NewHandler(ctx context.Context, options ...func(context.Context, *Handler) error) (*Handler, error) {
	handler := &Handler{}
	for _, opt := range options {
		if err := opt(ctx, handler); err != nil {
			return nil, err
		}
	}
	return handler, nil
}

func WithID(id *string, must bool) func(context.Context, *Handler) error {
	return func(ctx context.Context, h *Handler) error {
		if id == nil {
			if must {
				return fmt.Errorf("invalid id")
			}
			return nil
		}
		if _, err := uuid.Parse(*id); err != nil {
			return err
		}
		h.ID = id
		return nil
	}
}

func WithAppID(appID *string, must bool) func(context.Context, *Handler) error {
	return func(ctx context.Context, h *Handler) error {
		if appID == nil {
			if must {
				return fmt.Errorf("invalid appid")
			}
			return nil
		}
		exist, err := appmwcli.ExistApp(ctx, *appID)
		if err != nil {
			return err
		}
		if !exist {
			return fmt.Errorf("invalid app")
		}
		h.AppID = appID
		return nil
	}
}

func WithUserID(userID *string, must bool) func(context.Context, *Handler) error {
	return func(ctx context.Context, h *Handler) error {
		if userID == nil {
			if must {
				return fmt.Errorf("invalid userid")
			}
			return nil
		}
		_, err := uuid.Parse(*userID)
		if err != nil {
			return err
		}
		h.UserID = userID
		return nil
	}
}

func WithAppGoodID(id *string, must bool) func(context.Context, *Handler) error {
	return func(ctx context.Context, h *Handler) error {
		if id == nil {
			if must {
				return fmt.Errorf("invalid appgoodid")
			}
			return nil
		}
		exist, err := appgoodmwcli.ExistGood(ctx, *id)
		if err != nil {
			return err
		}
		if !exist {
			return fmt.Errorf("invalid appgood")
		}
		h.AppGoodID = id
		return nil
	}
}

func WithPaymentCoinID(id *string, must bool) func(context.Context, *Handler) error {
	return func(ctx context.Context, h *Handler) error {
		if id == nil {
			if must {
				return fmt.Errorf("invalid paymentcoinid")
			}
			return nil
		}
		_, err := uuid.Parse(*id)
		if err != nil {
			return err
		}
		h.PaymentCoinID = id
		return nil
	}
}

func WithParentOrderID(id *string, must bool) func(context.Context, *Handler) error {
	return func(ctx context.Context, h *Handler) error {
		if id == nil {
			if must {
				return fmt.Errorf("invalid parentorderid")
			}
			return nil
		}
		_, err := uuid.Parse(*id)
		if err != nil {
			return err
		}
		exist, err := ordermwcli.ExistOrder(ctx, *h.ParentOrderID)
		if err != nil {
			return err
		}
		if !exist {
			return fmt.Errorf("invalid parentorder")
		}
		h.ParentOrderID = id
		return nil
	}
}

func WithUnits(amount string, must bool) func(context.Context, *Handler) error {
	return func(ctx context.Context, h *Handler) error {
		_amount, err := decimal.NewFromString(amount)
		if err != nil {
			return err
		}
		if _amount.Cmp(decimal.NewFromInt32(0)) <= 0 {
			return fmt.Errorf("units is 0")
		}
		h.Units = amount
		return nil
	}
}

func WithBalanceAmount(amount *string, must bool) func(context.Context, *Handler) error {
	return func(ctx context.Context, h *Handler) error {
		if amount == nil {
			if must {
				return fmt.Errorf("invalid parentorderid")
			}
			return nil
		}
		_amount, err := decimal.NewFromString(*amount)
		if err != nil {
			return err
		}
		if _amount.Cmp(decimal.NewFromInt32(0)) <= 0 {
			return fmt.Errorf("units is 0")
		}
		h.BalanceAmount = amount
		return nil
	}
}

func WithCouponIDs(couponIDs []string, must bool) func(context.Context, *Handler) error {
	return func(ctx context.Context, h *Handler) error {
		if len(couponIDs) == 0 {
			if must {
				return fmt.Errorf("invalid couponids")
			}
			return nil
		}
		for _, id := range couponIDs {
			if _, err := uuid.Parse(id); err != nil {
				return err
			}
		}
		exist, err := ordermwcli.ExistOrderConds(ctx, &ordermwpb.Conds{
			CouponIDs: &basetypes.StringSliceVal{Op: cruder.IN, Value: couponIDs},
		})
		if err != nil {
			return err
		}
		if exist {
			return fmt.Errorf("invalid couponids")
		}
		h.CouponIDs = couponIDs
		return nil
	}
}

func WithInvestmentType(investmentType *ordertypes.InvestmentType, must bool) func(context.Context, *Handler) error {
	return func(ctx context.Context, h *Handler) error {
		if investmentType == nil {
			if must {
				return fmt.Errorf("invalid investmenttype")
			}
			return nil
		}
		switch *investmentType {
		case ordertypes.InvestmentType_FullPayment:
		case ordertypes.InvestmentType_UnionMining:
		default:
			return fmt.Errorf("invalid investmenttype")
		}
		h.InvestmentType = investmentType
		return nil
	}
}

func WithUserSetCanceled(value *bool, must bool) func(context.Context, *Handler) error {
	return func(ctx context.Context, h *Handler) error {
		if value == nil {
			if must {
				return fmt.Errorf("invalid canceled")
			}
			return nil
		}
		h.UserSetCanceled = value
		return nil
	}
}

func WithAdminSetCanceled(value *bool, must bool) func(context.Context, *Handler) error {
	return func(ctx context.Context, h *Handler) error {
		if value == nil {
			if must {
				return fmt.Errorf("invalid canceled")
			}
			return nil
		}
		h.AdminSetCanceled = value
		return nil
	}
}

func WithPaymentID(id *string, must bool) func(context.Context, *Handler) error {
	return func(ctx context.Context, h *Handler) error {
		if id == nil {
			if must {
				return fmt.Errorf("invalid paymentid")
			}
			return nil
		}
		_, err := uuid.Parse(*id)
		if err != nil {
			return err
		}
		h.PaymentID = id
		return nil
	}
}

func WithOrderType(orderType *ordertypes.OrderType, must bool) func(context.Context, *Handler) error {
	return func(ctx context.Context, h *Handler) error {
		if orderType == nil {
			if must {
				return fmt.Errorf("invalid ordertype")
			}
			return nil
		}
		switch *orderType {
		case ordertypes.OrderType_Airdrop:
		case ordertypes.OrderType_Offline:
		case ordertypes.OrderType_Normal:
		default:
			return fmt.Errorf("invalid order type")
		}
		h.OrderType = orderType
		return nil
	}
}

func WithOrders(orders []*npool.CreateOrdersRequest_OrderReq, must bool) func(context.Context, *Handler) error {
	return func(ctx context.Context, h *Handler) error {
		h.Orders = orders
		return nil
	}
}

func WithOffset(offset int32) func(context.Context, *Handler) error {
	return func(ctx context.Context, h *Handler) error {
		h.Offset = offset
		return nil
	}
}

func WithLimit(limit int32) func(context.Context, *Handler) error {
	return func(ctx context.Context, h *Handler) error {
		if limit == 0 {
			limit = constant.DefaultRowLimit
		}
		h.Limit = limit
		return nil
	}
}
