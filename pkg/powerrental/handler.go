//nolint:dupl
package powerrental

import (
	"context"

	wlog "github.com/NpoolPlatform/go-service-framework/pkg/wlog"
	types "github.com/NpoolPlatform/message/npool/basetypes/order/v1"
	paymentgwpb "github.com/NpoolPlatform/message/npool/order/gw/v1/payment"
	powerrentalpb "github.com/NpoolPlatform/message/npool/order/gw/v1/powerrental"
	paymentmwpb "github.com/NpoolPlatform/message/npool/order/mw/v1/payment"
	ordergwcommon "github.com/NpoolPlatform/order-gateway/pkg/common"
	constant "github.com/NpoolPlatform/order-gateway/pkg/const"
	ordercommon "github.com/NpoolPlatform/order-gateway/pkg/order/common"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

type Handler struct {
	ID    *uint32
	EntID *string
	ordercommon.OrderCheckHandler
	ordergwcommon.CoinCheckHandler
	ordergwcommon.AllocatedCouponCheckHandler
	ordergwcommon.AppGoodCheckHandler
	ParentOrderID             *string
	DurationSeconds           *uint32
	Units                     *decimal.Decimal
	AppSpotUnits              *decimal.Decimal
	Balances                  []*paymentmwpb.PaymentBalanceReq
	PaymentTransferCoinTypeID *string
	CouponIDs                 []string
	UserSetPaid               *bool
	UserSetCanceled           *bool
	AdminSetCanceled          *bool
	FeeAppGoodIDs             []string
	FeeDurationSeconds        *uint32
	FeeAutoDeduction          *bool
	OrderIDs                  []string
	CreateMethod              *types.OrderCreateMethod
	Simulate                  *bool
	AppGoodStockID            *string
	InvestmentType            *types.InvestmentType
	OrderType                 *types.OrderType
	OrderBenefitAccounts      []*powerrentalpb.OrderBenefitAccountReq
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
				return wlog.Errorf("invalid id")
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
				return wlog.Errorf("invalid entid")
			}
			return nil
		}
		_, err := uuid.Parse(*id)
		if err != nil {
			return wlog.WrapError(err)
		}
		h.EntID = id
		return nil
	}
}

func WithAppID(id *string, must bool) func(context.Context, *Handler) error {
	return func(ctx context.Context, h *Handler) error {
		if id == nil {
			if must {
				return wlog.Errorf("invalid appid")
			}
			return nil
		}
		if err := h.OrderCheckHandler.CheckAppWithAppID(ctx, *id); err != nil {
			return wlog.WrapError(err)
		}
		h.OrderCheckHandler.AppID = id
		h.AppGoodCheckHandler.AppID = id
		h.AllocatedCouponCheckHandler.AppID = id
		return nil
	}
}

func WithUserID(id *string, must bool) func(context.Context, *Handler) error {
	return func(ctx context.Context, h *Handler) error {
		if id == nil {
			if must {
				return wlog.Errorf("invalid userid")
			}
			return nil
		}
		if err := h.OrderCheckHandler.CheckUserWithUserID(ctx, *id); err != nil {
			return wlog.WrapError(err)
		}
		h.OrderCheckHandler.UserID = id
		h.AppGoodCheckHandler.UserID = id
		h.AllocatedCouponCheckHandler.UserID = id
		return nil
	}
}

func WithGoodID(id *string, must bool) func(context.Context, *Handler) error {
	return func(ctx context.Context, h *Handler) error {
		if id == nil {
			if must {
				return wlog.Errorf("invalid goodid")
			}
			return nil
		}
		if err := h.OrderCheckHandler.CheckGoodWithGoodID(ctx, *id); err != nil {
			return wlog.WrapError(err)
		}
		h.OrderCheckHandler.GoodID = id
		h.AppGoodCheckHandler.GoodID = id
		return nil
	}
}

func WithAppGoodID(id *string, must bool) func(context.Context, *Handler) error {
	return func(ctx context.Context, h *Handler) error {
		if id == nil {
			if must {
				return wlog.Errorf("invalid appgoodid")
			}
			return nil
		}
		if err := h.OrderCheckHandler.CheckAppGoodWithAppGoodID(ctx, *id); err != nil {
			return wlog.WrapError(err)
		}
		h.AppGoodID = id
		return nil
	}
}

func WithOrderID(id *string, must bool) func(context.Context, *Handler) error {
	return func(ctx context.Context, h *Handler) error {
		if id == nil {
			if must {
				return wlog.Errorf("invalid orderid")
			}
			return nil
		}
		if err := h.CheckOrderWithOrderID(ctx, *id); err != nil {
			return wlog.WrapError(err)
		}
		h.OrderID = id
		return nil
	}
}

func WithParentOrderID(id *string, must bool) func(context.Context, *Handler) error {
	return func(ctx context.Context, h *Handler) error {
		if id == nil {
			if must {
				return wlog.Errorf("invalid parentorderid")
			}
			return nil
		}
		if err := h.CheckOrderWithOrderID(ctx, *id); err != nil {
			return wlog.WrapError(err)
		}
		h.ParentOrderID = id
		return nil
	}
}

func WithDurationSeconds(u *uint32, must bool) func(context.Context, *Handler) error {
	return func(ctx context.Context, h *Handler) error {
		if u == nil {
			if must {
				return wlog.Errorf("invalid durationseconds")
			}
			return nil
		}
		if *u <= 0 {
			return wlog.Errorf("invalid durationseconds")
		}
		h.DurationSeconds = u
		return nil
	}
}

func WithUnits(s *string, must bool) func(context.Context, *Handler) error {
	return func(ctx context.Context, h *Handler) error {
		if s == nil {
			if must {
				return wlog.Errorf("invalid units")
			}
			return nil
		}
		units, err := decimal.NewFromString(*s)
		if err != nil {
			return wlog.WrapError(err)
		}
		h.Units = &units
		return nil
	}
}

func WithAppSpotUnits(s *string, must bool) func(context.Context, *Handler) error {
	return func(ctx context.Context, h *Handler) error {
		if s == nil {
			if must {
				return wlog.Errorf("invalid appspotunits")
			}
			return nil
		}
		units, err := decimal.NewFromString(*s)
		if err != nil {
			return wlog.WrapError(err)
		}
		h.AppSpotUnits = &units
		return nil
	}
}

func WithPaymentBalances(bs []*paymentgwpb.PaymentBalance, must bool) func(context.Context, *Handler) error {
	return func(ctx context.Context, h *Handler) error {
		for _, balance := range bs {
			if err := h.CheckCoinWithCoinTypeID(ctx, balance.CoinTypeID); err != nil {
				return wlog.WrapError(err)
			}
			// Fill coin_usd_currency later
			h.Balances = append(h.Balances, &paymentmwpb.PaymentBalanceReq{
				CoinTypeID: &balance.CoinTypeID,
				Amount:     &balance.Amount,
			})
		}
		return nil
	}
}

func WithPaymentTransferCoinTypeID(s *string, must bool) func(context.Context, *Handler) error {
	return func(ctx context.Context, h *Handler) error {
		if s == nil {
			if must {
				return wlog.Errorf("invalid paymenttransfercointypeid")
			}
			return nil
		}
		if err := h.CheckCoinWithCoinTypeID(ctx, *s); err != nil {
			return wlog.WrapError(err)
		}
		h.PaymentTransferCoinTypeID = s
		return nil
	}
}

func WithCouponIDs(ss []string, must bool) func(context.Context, *Handler) error {
	return func(ctx context.Context, h *Handler) error {
		for _, couponID := range ss {
			if err := h.CheckAllocatedCouponWithAllocatedCouponID(ctx, couponID); err != nil {
				return wlog.WrapError(err)
			}
		}
		h.CouponIDs = ss
		return nil
	}
}

func WithUserSetPaid(b *bool, must bool) func(context.Context, *Handler) error {
	return func(ctx context.Context, h *Handler) error {
		h.UserSetPaid = b
		return nil
	}
}

func WithUserSetCanceled(b *bool, must bool) func(context.Context, *Handler) error {
	return func(ctx context.Context, h *Handler) error {
		h.UserSetCanceled = b
		return nil
	}
}

func WithAdminSetCanceled(b *bool, must bool) func(context.Context, *Handler) error {
	return func(ctx context.Context, h *Handler) error {
		h.AdminSetCanceled = b
		return nil
	}
}

func WithFeeAppGoodIDs(ss []string, must bool) func(context.Context, *Handler) error {
	return func(ctx context.Context, h *Handler) error {
		for _, appGoodID := range ss {
			if err := h.OrderCheckHandler.CheckAppGoodWithAppGoodID(ctx, appGoodID); err != nil {
				return wlog.WrapError(err)
			}
			h.FeeAppGoodIDs = append(h.FeeAppGoodIDs, appGoodID)
		}
		return nil
	}
}

func WithFeeDurationSeconds(u *uint32, must bool) func(context.Context, *Handler) error {
	return func(ctx context.Context, h *Handler) error {
		if u == nil {
			if must {
				return wlog.Errorf("invalid feedurationseconds")
			}
			return nil
		}
		if *u <= 0 {
			return wlog.Errorf("invalid feedurationseconds")
		}
		h.FeeDurationSeconds = u
		return nil
	}
}

func WithFeeAutoDeduction(b *bool, must bool) func(context.Context, *Handler) error {
	return func(ctx context.Context, h *Handler) error {
		h.FeeAutoDeduction = b
		return nil
	}
}

func WithCreateMethod(e *types.OrderCreateMethod, must bool) func(context.Context, *Handler) error {
	return func(ctx context.Context, h *Handler) error {
		if e == nil {
			if must {
				return wlog.Errorf("invalid createmethod")
			}
			return nil
		}
		switch *e {
		case types.OrderCreateMethod_OrderCreatedByPurchase:
		case types.OrderCreateMethod_OrderCreatedByAdmin:
		case types.OrderCreateMethod_OrderCreatedByRenew:
		default:
			return wlog.Errorf("invalid createmethod")
		}
		h.CreateMethod = e
		return nil
	}
}

func WithOrderType(orderType *types.OrderType, must bool) func(context.Context, *Handler) error {
	return func(ctx context.Context, h *Handler) error {
		if orderType == nil {
			if must {
				return wlog.Errorf("invalid ordertype")
			}
			return nil
		}
		switch *orderType {
		case types.OrderType_Airdrop:
		case types.OrderType_Offline:
		case types.OrderType_Normal:
		default:
			return wlog.Errorf("invalid ordertype")
		}
		h.OrderType = orderType
		return nil
	}
}

func WithSimulate(b *bool, must bool) func(context.Context, *Handler) error {
	return func(ctx context.Context, h *Handler) error {
		h.Simulate = b
		return nil
	}
}

func WithAppGoodStockID(s *string, must bool) func(context.Context, *Handler) error {
	return func(ctx context.Context, h *Handler) error {
		if s == nil {
			if must {
				return wlog.Errorf("invalid appgoodstockid")
			}
			return nil
		}
		if _, err := uuid.Parse(*s); err != nil {
			return wlog.WrapError(err)
		}
		h.AppGoodStockID = s
		return nil
	}
}

func WithInvestmentType(_type *types.InvestmentType, must bool) func(context.Context, *Handler) error {
	return func(ctx context.Context, h *Handler) error {
		if _type == nil {
			if must {
				return wlog.Errorf("invalid investmenttype")
			}
			return nil
		}
		switch *_type {
		case types.InvestmentType_FullPayment:
		case types.InvestmentType_UnionMining:
		default:
			return wlog.Errorf("invalid investmenttype")
		}
		h.InvestmentType = _type
		return nil
	}
}

func WithOrderBenefitReqs(reqs []*powerrentalpb.OrderBenefitAccountReq, must bool) func(context.Context, *Handler) error {
	return func(ctx context.Context, h *Handler) error {
		for _, req := range reqs {
			if req.AccountID == nil && (req.CoinTypeID == nil || req.Address == nil) {
				return wlog.Errorf("invalid orderbenefitaccountreqs")
			}
		}
		h.OrderBenefitAccounts = reqs
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
