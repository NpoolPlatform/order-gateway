package fee

import (
	"context"

	ordergwcommon "github.com/NpoolPlatform/order-gateway/pkg/common"
	ordercommon "github.com/NpoolPlatform/order-gateway/pkg/order/common"
)

type Handler struct {
	ID    *uint32
	EntID *string
	ordercommon.OrderCheckHandler
	ordergwcommon.CoinCheckHandler
	ParentOrderID             *string
	DurationSeconds           *uint32
	Balances                  *paymentgwpb.PaymentBalance
	PaymentTransferCoinTypeID *string
	CouponIDs                 []string
	Paid                      *bool
	UserSetCanceled           *bool
	AdminSetCanceled          *bool
	Offset                    int32
	Limit                     int32
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

func WithID(id *uint32, must bool) func(context.Context, *Handler) error {
	return func(ctx context.Context, h *Handler) error {
		if id == nil {
			if must {
				return fmt.Errorf("invalid id")
			}
			return nil
		}
		h.ID = id
		return nil
	}
}

func WithEntID(id *string, must bool) func(context.Context, *Handler) error {
	return func(ctx context.Context, h *Handler) error {
		if id == nil {
			if must {
				return fmt.Errorf("invalid entid")
			}
			return nil
		}
		_, err := uuid.Parse(*id)
		if err != nil {
			return err
		}
		h.EntID = id
		return nil
	}
}

func WithAppID(id *string, must bool) func(context.Context, *Handler) error {
	return func(ctx context.Context, h *Handler) error {
		if id == nil {
			if must {
				return fmt.Errorf("invalid appid")
			}
			return nil
		}
		if err := h.CheckAppWithAppID(ctx, *id); err != nil {
			return err
		}
		h.AppID = id
		return nil
	}
}

func WithUserID(id *string, must bool) func(context.Context, *Handler) error {
	return func(ctx context.Context, h *Handler) error {
		if id == nil {
			if must {
				return fmt.Errorf("invalid userid")
			}
			return nil
		}
		if err := h.CheckUserWithUserID(ctx, *id); err != nil {
			return err
		}
		h.UserID = id
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
		if err := h.CheckGoodWithGoodID(ctx, *id); err != nil {
			return err
		}
		h.GoodID = id
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
		if err := h.CheckAppGoodWithAppGoodID(ctx, *id); err != nil {
			return err
		}
		h.AppGoodID = id
		return nil
	}
}

func WithOrderID(id *string, must bool) func(context.Context, *Handler) error {
	return func(ctx context.Context, h *Handler) error {
		if id == nil {
			if must {
				return fmt.Errorf("invalid orderid")
			}
			return nil
		}
		if err := h.CheckOrderWithOrderID(ctx, *id); err != nil {
			return err
		}
		h.OrderID = id
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
		if err := h.CheckOrderWithOrderID(ctx, *id); err != nil {
			return err
		}
		h.ParentOrderID = id
		return nil
	}
}
