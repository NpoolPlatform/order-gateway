package config

import (
	"context"
	"fmt"

	appcli "github.com/NpoolPlatform/appuser-middleware/pkg/client/app"
	ordertypes "github.com/NpoolPlatform/message/npool/basetypes/order/v1"
	configmw "github.com/NpoolPlatform/message/npool/order/mw/v1/simulate/config"
	constant "github.com/NpoolPlatform/order-gateway/pkg/const"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

type Handler struct {
	ID                    *uint32
	EntID                 *string
	AppID                 *string
	Units                 *string
	Duration              *uint32
	SendCouponMode        *ordertypes.SendCouponMode
	SendCouponProbability *string
	EnabledProfitTx       *bool
	ProfitTxProbability   *string
	Enabled               *bool
	Reqs                  []*configmw.SimulateConfigReq
	Offset                int32
	Limit                 int32
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
		_, err := uuid.Parse(*id)
		if err != nil {
			return err
		}
		exist, err := appcli.ExistApp(ctx, *id)
		if err != nil {
			return err
		}
		if !exist {
			return fmt.Errorf("invalid app")
		}
		h.AppID = id
		return nil
	}
}

//nolint:dupl
func WithUnits(amount *string, must bool) func(context.Context, *Handler) error {
	return func(ctx context.Context, h *Handler) error {
		if amount == nil {
			if must {
				return fmt.Errorf("invalid units")
			}
			return nil
		}
		_amount, err := decimal.NewFromString(*amount)
		if err != nil {
			return err
		}
		if _amount.Cmp(decimal.NewFromInt32(0)) <= 0 {
			return fmt.Errorf("invalid units")
		}
		h.Units = amount
		return nil
	}
}

func WithDuration(duration *uint32, must bool) func(context.Context, *Handler) error {
	return func(ctx context.Context, h *Handler) error {
		if duration == nil {
			if must {
				return fmt.Errorf("invalid duration")
			}
			return nil
		}
		h.Duration = duration
		return nil
	}
}

//nolint:dupl
func WithSendCouponProbability(amount *string, must bool) func(context.Context, *Handler) error {
	return func(ctx context.Context, h *Handler) error {
		if amount == nil {
			if must {
				return fmt.Errorf("invalid sendcouponprobability")
			}
			return nil
		}
		_amount, err := decimal.NewFromString(*amount)
		if err != nil {
			return err
		}
		if _amount.Cmp(decimal.NewFromInt32(0)) <= 0 {
			return fmt.Errorf("invalid sendcouponprobability")
		}
		h.SendCouponProbability = amount
		return nil
	}
}

func WithSendCouponMode(value *ordertypes.SendCouponMode, must bool) func(context.Context, *Handler) error {
	return func(ctx context.Context, h *Handler) error {
		if value == nil {
			if must {
				return fmt.Errorf("invalid sendcouponmode")
			}
			return nil
		}
		switch *value {
		case ordertypes.SendCouponMode_WithoutCoupon:
		case ordertypes.SendCouponMode_FirstBenifit:
		case ordertypes.SendCouponMode_RandomBenifit:
		case ordertypes.SendCouponMode_FirstAndRandomBenifit:
		default:
			return fmt.Errorf("invalid sendcouponmode")
		}
		h.SendCouponMode = value
		return nil
	}
}

//nolint:dupl
func WithProfitTxProbability(amount *string, must bool) func(context.Context, *Handler) error {
	return func(ctx context.Context, h *Handler) error {
		if amount == nil {
			if must {
				return fmt.Errorf("invalid profittxprobability")
			}
			return nil
		}
		_amount, err := decimal.NewFromString(*amount)
		if err != nil {
			return err
		}
		if _amount.Cmp(decimal.NewFromInt32(0)) <= 0 {
			return fmt.Errorf("invalid profittxprobability")
		}
		h.ProfitTxProbability = amount
		return nil
	}
}

func WithEnabledProfitTx(enabled *bool, must bool) func(context.Context, *Handler) error {
	return func(ctx context.Context, h *Handler) error {
		if enabled == nil {
			if must {
				return fmt.Errorf("invalid enabledprofittx")
			}
			return nil
		}
		h.EnabledProfitTx = enabled
		return nil
	}
}

func WithEnabled(enabled *bool, must bool) func(context.Context, *Handler) error {
	return func(ctx context.Context, h *Handler) error {
		if enabled == nil {
			if must {
				return fmt.Errorf("invalid enabled")
			}
			return nil
		}
		h.Enabled = enabled
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
