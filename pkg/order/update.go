package order

import (
	"context"
	"fmt"
	"time"

	usermwcli "github.com/NpoolPlatform/appuser-middleware/pkg/client/user"
	appgoodmwcli "github.com/NpoolPlatform/good-middleware/pkg/client/app/good"
	goodmwcli "github.com/NpoolPlatform/good-middleware/pkg/client/good"
	statementmwcli "github.com/NpoolPlatform/inspire-middleware/pkg/client/achievement/statement"
	goodledgerstatementcli "github.com/NpoolPlatform/ledger-middleware/pkg/client/good/ledger/statement"
	ledgerstatementcli "github.com/NpoolPlatform/ledger-middleware/pkg/client/ledger/statement"
	ledgermwsvcname "github.com/NpoolPlatform/ledger-middleware/pkg/servicename"
	"github.com/NpoolPlatform/libent-cruder/pkg/cruder"
	goodtypes "github.com/NpoolPlatform/message/npool/basetypes/good/v1"
	ledgertypes "github.com/NpoolPlatform/message/npool/basetypes/ledger/v1"
	ordertypes "github.com/NpoolPlatform/message/npool/basetypes/order/v1"
	basetypes "github.com/NpoolPlatform/message/npool/basetypes/v1"
	appgoodmwpb "github.com/NpoolPlatform/message/npool/good/mw/v1/app/good"
	statementmwpb "github.com/NpoolPlatform/message/npool/inspire/mw/v1/achievement/statement"
	goodledgerstatementpb "github.com/NpoolPlatform/message/npool/ledger/mw/v2/good/ledger/statement"
	ledgermwpb "github.com/NpoolPlatform/message/npool/ledger/mw/v2/ledger"
	ledgerstatementpb "github.com/NpoolPlatform/message/npool/ledger/mw/v2/ledger/statement"
	npool "github.com/NpoolPlatform/message/npool/order/gw/v1/order"
	ordermwpb "github.com/NpoolPlatform/message/npool/order/mw/v1/order"
	orderlockmwpb "github.com/NpoolPlatform/message/npool/order/mw/v1/order/orderlock"
	ordermwcli "github.com/NpoolPlatform/order-middleware/pkg/client/order"
	ordermwsvcname "github.com/NpoolPlatform/order-middleware/pkg/servicename"

	dtmcli "github.com/NpoolPlatform/dtm-cluster/pkg/dtm"
	"github.com/dtm-labs/dtm/client/dtmcli/dtmimp"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

type updateHandler struct {
	*dtmHandler
	order                 *ordermwpb.Order
	appGood               *appgoodmwpb.Good
	achievementStatements []*statementmwpb.Statement
	commissionLockIDs     map[string]string
}

func (h *updateHandler) checkCancelParam() error {
	if h.UserSetCanceled == nil && h.AdminSetCanceled == nil {
		return fmt.Errorf("nothing todo")
	}
	if h.UserSetCanceled != nil && !*h.UserSetCanceled {
		return fmt.Errorf("nothing todo")
	}
	if h.AdminSetCanceled != nil && !*h.AdminSetCanceled {
		return fmt.Errorf("nothing todo")
	}
	if h.order.AdminSetCanceled || h.order.UserSetCanceled {
		return fmt.Errorf("permission denied")
	}
	return nil
}

func (h *updateHandler) checkUser(ctx context.Context) error {
	user, err := usermwcli.GetUser(ctx, *h.AppID, *h.UserID)
	if err != nil {
		return err
	}
	if user == nil {
		return fmt.Errorf("invalid user")
	}
	return nil
}

func (h *updateHandler) checkOrder(ctx context.Context) error {
	order, err := ordermwcli.GetOrderOnly(ctx, &ordermwpb.Conds{
		ID:    &basetypes.Uint32Val{Op: cruder.EQ, Value: *h.ID},
		EntID: &basetypes.StringVal{Op: cruder.EQ, Value: *h.EntID},
	})
	if err != nil {
		return err
	}
	if order == nil {
		return fmt.Errorf("invalid order")
	}
	if *h.AppID != order.AppID || *h.UserID != order.UserID {
		return fmt.Errorf("permission denied")
	}
	if order.PaymentType == ordertypes.PaymentType_PayWithParentOrder {
		return fmt.Errorf("permission denied")
	}
	h.order = order
	return nil
}

func (h *updateHandler) checkOrderType() error {
	switch h.order.OrderType {
	case ordertypes.OrderType_Normal:
		switch h.order.OrderState {
		case ordertypes.OrderState_OrderStateWaitPayment:
			if h.AdminSetCanceled != nil {
				return fmt.Errorf("permission denied")
			}
		case ordertypes.OrderState_OrderStatePaid:
		case ordertypes.OrderState_OrderStateInService:
		default:
			return fmt.Errorf("orderstate uncancellable")
		}
	case ordertypes.OrderType_Offline:
		fallthrough //nolint
	case ordertypes.OrderType_Airdrop:
		if h.AdminSetCanceled == nil {
			return fmt.Errorf("permission denied")
		}
		switch h.order.OrderState {
		case ordertypes.OrderState_OrderStatePaid:
		case ordertypes.OrderState_OrderStateInService:
		default:
			return fmt.Errorf("orderstate uncancellable")
		}
	default:
		return fmt.Errorf("order type uncancellable")
	}
	return nil
}

func (h *updateHandler) getAppGood(ctx context.Context) error {
	good, err := appgoodmwcli.GetGood(ctx, h.order.AppGoodID)
	if err != nil {
		return err
	}
	if good == nil {
		return fmt.Errorf("invalid appgood")
	}
	if good.AppID != *h.AppID || good.GoodID != h.order.GoodID {
		return fmt.Errorf("invalid appgood")
	}
	h.appGood = good
	return nil
}

func (h *updateHandler) checkGood(ctx context.Context) error {
	good, err := goodmwcli.GetGood(ctx, h.order.GoodID)
	if err != nil {
		return err
	}
	if good == nil {
		return fmt.Errorf("invalid good")
	}
	switch good.RewardState {
	case goodtypes.BenefitState_BenefitWait:
	default:
		return fmt.Errorf("permission denied")
	}
	return nil
}

func (h *updateHandler) checkCancelable(ctx context.Context) error {
	switch h.order.OrderState {
	case ordertypes.OrderState_OrderStateWaitPayment:
		return nil
	default:
	}

	goodStatements, _, err := goodledgerstatementcli.GetGoodStatements(ctx, &goodledgerstatementpb.Conds{
		GoodID: &basetypes.StringVal{Op: cruder.EQ, Value: h.order.GoodID},
	}, 0, 1)
	if err != nil {
		return err
	}

	switch h.appGood.CancelMode {
	case goodtypes.CancelMode_Uncancellable:
		return fmt.Errorf("permission denied")
	case goodtypes.CancelMode_CancellableBeforeStart:
		switch h.order.OrderState {
		case ordertypes.OrderState_OrderStatePaid:
		default:
			return fmt.Errorf("permission denied")
		}
	case goodtypes.CancelMode_CancellableBeforeBenefit:
		switch h.order.OrderState {
		case ordertypes.OrderState_OrderStatePaid:
		case ordertypes.OrderState_OrderStateInService:
			if len(goodStatements) == 0 {
				return nil
			}
			lastBenefitDate := goodStatements[0].BenefitDate
			const secondsPerDay = 24 * 60 * 60
			checkBenefitStartAt := lastBenefitDate + secondsPerDay - h.appGood.CancellableBeforeStart
			checkBenefitEndAt := lastBenefitDate + secondsPerDay + h.appGood.CancellableBeforeStart
			now := uint32(time.Now().Unix())
			if checkBenefitStartAt <= now && now <= checkBenefitEndAt {
				return fmt.Errorf("permission denied")
			}
		default:
			return fmt.Errorf("permission denied")
		}
	default:
		return fmt.Errorf("invalid cancelmode %v", h.appGood.CancelMode)
	}

	return nil
}

func (h *updateHandler) getCommission(ctx context.Context) error {
	offset := int32(0)
	limit := int32(1000) //nolint
	in := ledgertypes.IOType_Incoming
	for {
		infos, _, err := statementmwcli.GetStatements(ctx, &statementmwpb.Conds{
			OrderID: &basetypes.StringVal{Op: cruder.EQ, Value: h.order.EntID},
		}, offset, limit)
		if err != nil {
			return err
		}
		if len(infos) == 0 {
			break
		}

		offset += limit

		for _, val := range infos {
			commission, err := decimal.NewFromString(val.Commission)
			if err != nil {
				return err
			}
			if commission.Cmp(decimal.NewFromInt(0)) == 0 {
				continue
			}
			exist, err := ledgerstatementcli.ExistStatementConds(ctx, &ledgerstatementpb.Conds{
				AppID:     &basetypes.StringVal{Op: cruder.EQ, Value: val.AppID},
				UserID:    &basetypes.StringVal{Op: cruder.EQ, Value: val.UserID},
				IOType:    &basetypes.Uint32Val{Op: cruder.EQ, Value: uint32(in)},
				IOSubType: &basetypes.Uint32Val{Op: cruder.EQ, Value: uint32(ledgertypes.IOSubType_Commission)},
				IOExtra:   &basetypes.StringVal{Op: cruder.LIKE, Value: h.order.EntID},
			})
			if err != nil {
				return err
			}
			if !exist {
				return fmt.Errorf("invalid commission statement")
			}
			h.achievementStatements = append(h.achievementStatements, val)
			h.commissionLockIDs[val.ID] = uuid.NewString()
		}
	}

	return nil
}

func (h *updateHandler) withCreateCommissionLockIDs(dispose *dtmcli.SagaDispose) {
	if len(h.achievementStatements) == 0 {
		return
	}
	reqs := []*orderlockmwpb.OrderLockReq{}
	for _, statement := range h.achievementStatements {
		lockID := h.commissionLockIDs[statement.ID]
		req := &orderlockmwpb.OrderLockReq{
			EntID:    &lockID,
			AppID:    &statement.AppID,
			UserID:   &statement.UserID,
			OrderID:  h.EntID,
			LockType: ordertypes.OrderLockType_LockCommission.Enum(),
		}
		reqs = append(reqs, req)
	}
	dispose.Add(
		ordermwsvcname.ServiceDomain,
		"order.middleware.order1.orderlock.v1.Middleware/CreateOrderLocks",
		"order.middleware.order1.orderlock.v1.Middleware/DeleteOrderLocks",
		&orderlockmwpb.CreateOrderLocksRequest{
			Infos: reqs,
		},
	)
}

func (h *updateHandler) withLockCommission(dispose *dtmcli.SagaDispose) {
	for _, statement := range h.achievementStatements {
		dispose.Add(
			ledgermwsvcname.ServiceDomain,
			"ledger.middleware.ledger.v2.Middleware/LockBalance",
			"ledger.middleware.ledger.v2.Middleware/UnlockBalance",
			&ledgermwpb.LockBalanceRequest{
				AppID:      statement.AppID,
				UserID:     statement.UserID,
				CoinTypeID: statement.PaymentCoinTypeID,
				Amount:     statement.Commission,
				LockID:     h.commissionLockIDs[statement.ID],
				Rollback:   true,
			},
		)
	}
}

func (h *updateHandler) withProcessCancel(dispose *dtmcli.SagaDispose) {
	req := &ordermwpb.OrderReq{
		ID:               &h.order.ID,
		UserSetCanceled:  h.UserSetCanceled,
		AdminSetCanceled: h.AdminSetCanceled,
	}
	dispose.Add(
		ordermwsvcname.ServiceDomain,
		"order.middleware.order1.v1.Middleware/UpdateOrder",
		"",
		&ordermwpb.UpdateOrderRequest{
			Info: req,
		},
	)
}

func (h *Handler) UpdateOrder(ctx context.Context) (*npool.Order, error) {
	handler := &updateHandler{
		dtmHandler: &dtmHandler{
			Handler: h,
		},
		commissionLockIDs: map[string]string{},
	}
	if err := handler.checkUser(ctx); err != nil {
		return nil, err
	}
	if err := handler.checkOrder(ctx); err != nil {
		return nil, err
	}
	if err := handler.checkCancelParam(); err != nil {
		return nil, err
	}
	if err := handler.checkOrderType(); err != nil {
		return nil, err
	}
	if err := handler.getAppGood(ctx); err != nil {
		return nil, err
	}
	if err := handler.checkGood(ctx); err != nil {
		return nil, err
	}
	if err := handler.checkCancelable(ctx); err != nil {
		return nil, err
	}
	if err := handler.getCommission(ctx); err != nil {
		return nil, err
	}

	const timeoutSeconds = 10
	sagaDispose := dtmcli.NewSagaDispose(dtmimp.TransOptions{
		WaitResult:     true,
		RequestTimeout: timeoutSeconds,
		TimeoutToFail:  timeoutSeconds,
	})

	handler.withCreateCommissionLockIDs(sagaDispose)
	handler.withLockCommission(sagaDispose)
	handler.withProcessCancel(sagaDispose)

	if err := handler.dtmDo(ctx, sagaDispose); err != nil {
		return nil, err
	}

	return handler.GetOrder(ctx)
}
