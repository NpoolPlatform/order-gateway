package order

import (
	"context"
	"fmt"

	ordermgrpb "github.com/NpoolPlatform/message/npool/order/mgr/v1/order"
	ordermwpb "github.com/NpoolPlatform/message/npool/order/mw/v1/order"
	ordermwcli "github.com/NpoolPlatform/order-middleware/pkg/client/order"

	npool "github.com/NpoolPlatform/message/npool/order/gw/v1/order"
)

func CreateOrder(ctx context.Context, op *OrderCreate) (info *npool.Order, err error) {
	if err := op.ValidateInit(ctx); err != nil {
		return nil, err
	}

	if err := op.SetReduction(ctx); err != nil {
		return nil, err
	}

	if err := op.SetPrice(ctx); err != nil {
		return nil, err
	}

	if err := op.SetCurrency(ctx); err != nil {
		return nil, err
	}

	if err := op.SetPaymentAmount(ctx); err != nil {
		return nil, err
	}

	if err := op.PeekAddress(ctx); err != nil {
		return nil, err
	}

	if err := op.SetBalance(ctx); err != nil {
		_ = op.ReleaseAddress(ctx)
		return nil, err
	}

	if err := op.LockStock(ctx); err != nil {
		_ = op.ReleaseAddress(ctx)
		return nil, err
	}

	if err := op.LockBalance(ctx); err != nil {
		_ = op.ReleaseAddress(ctx)
		_ = op.ReleaseStock(ctx)
		return nil, err
	}

	ord, err := op.Create(ctx)
	if err != nil {
		_ = op.ReleaseAddress(ctx)
		_ = op.ReleaseStock(ctx)
		_ = op.ReleaseBalance(ctx)
		return nil, err
	}

	return ord, nil
}

//nolint:gocyclo
func UpdateOrder(ctx context.Context, in *ordermwpb.OrderReq, fromAdmin bool) (*npool.Order, error) {
	ord, err := ordermwcli.GetOrder(ctx, in.GetID())
	if err != nil {
		return nil, err
	}
	if ord == nil {
		return nil, fmt.Errorf("invalid order")
	}

	if in.GetAppID() != ord.AppID || in.GetUserID() != ord.UserID {
		return nil, fmt.Errorf("permission denied")
	}
	if in.GetCanceled() {
		switch ord.OrderType {
		case ordermgrpb.OrderType_Normal:
			switch ord.OrderState {
			case ordermgrpb.OrderState_WaitPayment:
				ord, err = ordermwcli.UpdateOrder(ctx, in)
				if err != nil {
					return nil, err
				}
				return GetOrder(ctx, ord.ID)
			case ordermgrpb.OrderState_Paid:
				fallthrough // nolint
			case ordermgrpb.OrderState_InService:
				if err := cancelNormalOrder(ctx, ord); err != nil {
					return nil, err
				}
			default:
				return nil, fmt.Errorf("order state uncancellable")
			}
		case ordermgrpb.OrderType_Offline:
			if !fromAdmin {
				return nil, fmt.Errorf("permission denied")
			}
			if ord.OrderState != ordermgrpb.OrderState_Paid {
				return nil, fmt.Errorf("order state not paid")
			}
			if err := cancelOfflineOrder(ctx, ord); err != nil {
				return nil, err
			}
		case ordermgrpb.OrderType_Airdrop:
			if !fromAdmin {
				return nil, fmt.Errorf("permission denied")
			}
			if ord.OrderState != ordermgrpb.OrderState_Paid {
				return nil, fmt.Errorf("order state not paid")
			}
			if err := cancelAirdropOrder(ctx, ord); err != nil {
				return nil, err
			}
		default:
			return nil, fmt.Errorf("order type uncancellable")
		}
	}
	return GetOrder(ctx, ord.ID)
}
