package appconfig

import (
	"context"
	"fmt"

	types "github.com/NpoolPlatform/message/npool/basetypes/order/v1"
	ordergwcommon "github.com/NpoolPlatform/order-gateway/pkg/common"
	constant "github.com/NpoolPlatform/order-gateway/pkg/const"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

type Handler struct {
	ID    *uint32
	EntID *string
	ordergwcommon.AppCheckHandler
	Units                                  *string
	Duration                               *uint32
	EnableSimulateOrder                    *bool
	SimulateOrderUnits                     *string
	SimulateOrderDurationSeconds           *uint32
	SimulateOrderCouponMode                *types.SimulateOrderCouponMode
	SimulateOrderCouponProbability         *string
	SimulateOrderCashableProfitProbability *string
	MaxUnpaidOrders                        *uint32
	Offset                                 int32
	Limit                                  int32
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

func WithEnableSimulateOrder(enabled *bool, must bool) func(context.Context, *Handler) error {
	return func(ctx context.Context, h *Handler) error {
		h.EnableSimulateOrder = enabled
		return nil
	}
}

func WithSimulateOrderUnits(amount *string, must bool) func(context.Context, *Handler) error {
	return func(ctx context.Context, h *Handler) error {
		if amount == nil {
			if must {
				return fmt.Errorf("invalid simulateorderunits")
			}
			return nil
		}
		_amount, err := decimal.NewFromString(*amount)
		if err != nil {
			return err
		}
		if _amount.Cmp(decimal.NewFromInt32(0)) <= 0 {
			return fmt.Errorf("invalid simulateorderunits")
		}
		h.SimulateOrderUnits = amount
		return nil
	}
}

func WithSimulateOrderDurationSeconds(duration *uint32, must bool) func(context.Context, *Handler) error {
	return func(ctx context.Context, h *Handler) error {
		if duration == nil {
			if must {
				return fmt.Errorf("invalid simulateorderdurationseconds")
			}
			return nil
		}
		if *duration <= 0 {
			return fmt.Errorf("invalid simulateorderdurationseconds")
		}
		h.SimulateOrderDurationSeconds = duration
		return nil
	}
}

func WithMaxUnpaidOrders(duration *uint32, must bool) func(context.Context, *Handler) error {
	return func(ctx context.Context, h *Handler) error {
		if duration == nil {
			if must {
				return fmt.Errorf("invalid maxunpaidorders")
			}
			return nil
		}
		if *duration <= 0 {
			return fmt.Errorf("invalid maxunpaidorders")
		}
		h.MaxUnpaidOrders = duration
		return nil
	}
}

//nolint:dupl
func WithSimulateOrderCouponProbability(amount *string, must bool) func(context.Context, *Handler) error {
	return func(ctx context.Context, h *Handler) error {
		if amount == nil {
			if must {
				return fmt.Errorf("invalid simulateordercouponprobability")
			}
			return nil
		}
		_amount, err := decimal.NewFromString(*amount)
		if err != nil {
			return err
		}
		if _amount.Cmp(decimal.NewFromInt(0)) < 0 {
			return fmt.Errorf("invalid simulateordercouponprobability")
		}
		if _amount.Cmp(decimal.NewFromInt(1)) > 0 {
			return fmt.Errorf("invalid simulateordercouponprobability")
		}
		h.SimulateOrderCouponProbability = amount
		return nil
	}
}

func WithSimulateOrderCouponMode(value *types.SimulateOrderCouponMode, must bool) func(context.Context, *Handler) error {
	return func(ctx context.Context, h *Handler) error {
		if value == nil {
			if must {
				return fmt.Errorf("invalid simulateordercouponmode")
			}
			return nil
		}
		switch *value {
		case types.SimulateOrderCouponMode_WithoutCoupon:
		case types.SimulateOrderCouponMode_FirstBenifit:
		case types.SimulateOrderCouponMode_RandomBenifit:
		case types.SimulateOrderCouponMode_FirstAndRandomBenifit:
		default:
			return fmt.Errorf("invalid simulateordercouponmode")
		}
		h.SimulateOrderCouponMode = value
		return nil
	}
}

//nolint:dupl
func WithSimulateOrderCashableProfitProbability(amount *string, must bool) func(context.Context, *Handler) error {
	return func(ctx context.Context, h *Handler) error {
		if amount == nil {
			if must {
				return fmt.Errorf("invalid simulateordercashableprofitprobability")
			}
			return nil
		}
		_amount, err := decimal.NewFromString(*amount)
		if err != nil {
			return err
		}
		if _amount.Cmp(decimal.NewFromInt(0)) < 0 {
			return fmt.Errorf("invalid simulateordercashableprofitprobability")
		}
		if _amount.Cmp(decimal.NewFromInt(1)) > 0 {
			return fmt.Errorf("invalid simulateordercashableprofitprobability")
		}
		h.SimulateOrderCashableProfitProbability = amount
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
