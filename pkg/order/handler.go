package order

import (
	"context"
	"fmt"

	appmwcli "github.com/NpoolPlatform/appuser-middleware/pkg/client/app"
	appusermwcli "github.com/NpoolPlatform/appuser-middleware/pkg/client/user"
	ordertypes "github.com/NpoolPlatform/message/npool/basetypes/order/v1"
	npool "github.com/NpoolPlatform/message/npool/order/gw/v1/order"
	constant "github.com/NpoolPlatform/order-gateway/pkg/const"
	ordermwcli "github.com/NpoolPlatform/order-middleware/pkg/client/order"
	"github.com/shopspring/decimal"

	"github.com/google/uuid"
)

type Handler struct {
	ID                    *string
	AppID                 *string
	UserID                *string
	GoodID                *string
	Units                 string
	PaymentCoinID         *string
	ParentOrderID         *string
	BalanceAmount         *string
	OrderType             *ordertypes.OrderType
	CouponIDs             []string
	InvestmentType        *ordertypes.InvestmentType
	Canceled              *bool
	FromAdmin             bool
	PaymentID             *string
	Offset                int32
	Limit                 int32
	Goods                 []*npool.CreateOrdersRequest_Good
	IDs                   []string
	RequestTimeoutSeconds int64
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
		if _, err := uuid.Parse(*appID); err != nil {
			return err
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

func WithUserID(appID, userID *string, must bool) func(context.Context, *Handler) error {
	return func(ctx context.Context, h *Handler) error {
		if appID == nil || userID == nil {
			if must {
				return fmt.Errorf("invalid userid")
			}
			return nil
		}
		_, err := uuid.Parse(*userID)
		if err != nil {
			return err
		}
		exist, err := appusermwcli.ExistUser(ctx, *appID, *userID)
		if err != nil {
			return err
		}
		if !exist {
			return fmt.Errorf("invalid user")
		}

		h.UserID = userID
		return nil
	}
}

func WithGoodID(id *string, must bool) func(context.Context, *Handler) error {
	return func(ctx context.Context, h *Handler) error {
		if id == nil {
			if must {
				return fmt.Errorf("invalid goodid")
			}
			return nil
		}
		_, err := uuid.Parse(*id)
		if err != nil {
			return err
		}
		h.GoodID = id
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

func WithBalanceAmount(amount string, must bool) func(context.Context, *Handler) error {
	return func(ctx context.Context, h *Handler) error {
		_amount, err := decimal.NewFromString(amount)
		if err != nil {
			return err
		}
		if _amount.Cmp(decimal.NewFromInt32(0)) <= 0 {
			return fmt.Errorf("units is 0")
		}
		h.BalanceAmount = &amount
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

func WithCanceled(value *bool, must bool) func(context.Context, *Handler) error {
	return func(ctx context.Context, h *Handler) error {
		if value == nil {
			if must {
				return fmt.Errorf("invalid canceled")
			}
			return nil
		}
		h.Canceled = value
		return nil
	}
}

func WithFromAdmin(value bool) func(context.Context, *Handler) error {
	return func(ctx context.Context, h *Handler) error {
		h.FromAdmin = value
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

func WithGoods(goods []*npool.CreateOrdersRequest_Good, must bool) func(context.Context, *Handler) error {
	return func(ctx context.Context, h *Handler) error {
		h.Goods = goods
		return nil
	}
}
