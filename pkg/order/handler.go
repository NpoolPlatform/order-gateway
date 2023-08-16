package order

import (
	"context"
	"fmt"

	appmwcli "github.com/NpoolPlatform/appuser-middleware/pkg/client/app"
	appusermwcli "github.com/NpoolPlatform/appuser-middleware/pkg/client/user"
	ordertypes "github.com/NpoolPlatform/message/npool/basetypes/order/v1"
	constant "github.com/NpoolPlatform/order-gateway/pkg/const"
	"github.com/shopspring/decimal"

	"github.com/google/uuid"
)

type Handler struct {
	ID                   *string
	AppID                *string
	UserID               *string
	GoodID               *string
	Units                string
	PaymentCoinID        *string
	ParentOrderID        *string
	PayWithBalanceAmount *string
	FixAmountID          *string
	DiscountID           *string
	SpecialOfferID       *string
	OrderType            *ordertypes.OrderType
	CouponIDs            []string
	Canceled             *bool
	FromAdmin            bool
	PaymentID            *string
	Offset               int32
	Limit                int32
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

func WithID(id *string) func(context.Context, *Handler) error {
	return func(ctx context.Context, h *Handler) error {
		if id == nil {
			return nil
		}
		if _, err := uuid.Parse(*id); err != nil {
			return err
		}
		h.ID = id
		return nil
	}
}

func WithAppID(appID *string) func(context.Context, *Handler) error {
	return func(ctx context.Context, h *Handler) error {
		if appID == nil {
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

func WithUserID(appID, userID *string) func(context.Context, *Handler) error {
	return func(ctx context.Context, h *Handler) error {
		if appID == nil || userID == nil {
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

func WithGoodID(id *string) func(context.Context, *Handler) error {
	return func(ctx context.Context, h *Handler) error {
		if id == nil {
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

func WithPaymentCoinID(id *string) func(context.Context, *Handler) error {
	return func(ctx context.Context, h *Handler) error {
		if id == nil {
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

func WithParentOrderID(id *string) func(context.Context, *Handler) error {
	return func(ctx context.Context, h *Handler) error {
		if id == nil {
			return nil
		}
		_, err := uuid.Parse(*id)
		if err != nil {
			return err
		}
		h.ParentOrderID = id
		return nil
	}
}

func WithFixAmountID(id *string) func(context.Context, *Handler) error {
	return func(ctx context.Context, h *Handler) error {
		if id == nil {
			return nil
		}
		_, err := uuid.Parse(*id)
		if err != nil {
			return err
		}
		h.FixAmountID = id
		return nil
	}
}

func WithDiscountID(id *string) func(context.Context, *Handler) error {
	return func(ctx context.Context, h *Handler) error {
		if id == nil {
			return nil
		}
		_, err := uuid.Parse(*id)
		if err != nil {
			return err
		}
		h.DiscountID = id
		return nil
	}
}

func WithSpecialOfferID(id *string) func(context.Context, *Handler) error {
	return func(ctx context.Context, h *Handler) error {
		if id == nil {
			return nil
		}
		_, err := uuid.Parse(*id)
		if err != nil {
			return err
		}
		h.SpecialOfferID = id
		return nil
	}
}

func WithUnits(amount string) func(context.Context, *Handler) error {
	return func(ctx context.Context, h *Handler) error {
		_, err := decimal.NewFromString(amount)
		if err != nil {
			return err
		}
		h.Units = amount
		return nil
	}
}

func WithPayWithBalanceAmount(amount string) func(context.Context, *Handler) error {
	return func(ctx context.Context, h *Handler) error {
		_, err := decimal.NewFromString(amount)
		if err != nil {
			return err
		}
		h.PayWithBalanceAmount = &amount
		return nil
	}
}

func WithCouponIDs(couponIDs []string) func(context.Context, *Handler) error {
	return func(ctx context.Context, h *Handler) error {
		if len(couponIDs) == 0 {
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

func WithCanceled(value *bool) func(context.Context, *Handler) error {
	return func(ctx context.Context, h *Handler) error {
		if value == nil {
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

func WithPaymentID(id *string) func(context.Context, *Handler) error {
	return func(ctx context.Context, h *Handler) error {
		if id == nil {
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

func WithOrderType(orderType *ordertypes.OrderType) func(context.Context, *Handler) error {
	return func(ctx context.Context, h *Handler) error {
		if orderType == nil {
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
