package powerrental

import (
	"context"
	"time"

	timedef "github.com/NpoolPlatform/go-service-framework/pkg/const/time"
	wlog "github.com/NpoolPlatform/go-service-framework/pkg/wlog"
	apppowerrentalmwcli "github.com/NpoolPlatform/good-middleware/pkg/client/app/powerrental"
	goodledgerstatementcli "github.com/NpoolPlatform/ledger-middleware/pkg/client/good/ledger/statement"
	ledgerstatementmwcli "github.com/NpoolPlatform/ledger-middleware/pkg/client/ledger/statement"
	ledgermwsvcname "github.com/NpoolPlatform/ledger-middleware/pkg/servicename"
	"github.com/NpoolPlatform/libent-cruder/pkg/cruder"
	goodtypes "github.com/NpoolPlatform/message/npool/basetypes/good/v1"
	ledgertypes "github.com/NpoolPlatform/message/npool/basetypes/ledger/v1"
	types "github.com/NpoolPlatform/message/npool/basetypes/order/v1"
	basetypes "github.com/NpoolPlatform/message/npool/basetypes/v1"
	apppowerrentalmwpb "github.com/NpoolPlatform/message/npool/good/mw/v1/app/powerrental"
	goodledgerstatementpb "github.com/NpoolPlatform/message/npool/ledger/mw/v2/good/ledger/statement"
	ledgermwpb "github.com/NpoolPlatform/message/npool/ledger/mw/v2/ledger"
	ledgerstatementmwpb "github.com/NpoolPlatform/message/npool/ledger/mw/v2/ledger/statement"
	npool "github.com/NpoolPlatform/message/npool/order/gw/v1/powerrental"
	orderlockmwpb "github.com/NpoolPlatform/message/npool/order/mw/v1/order/lock"
	paymentmwpb "github.com/NpoolPlatform/message/npool/order/mw/v1/payment"
	powerrentalordermwpb "github.com/NpoolPlatform/message/npool/order/mw/v1/powerrental"
	constant "github.com/NpoolPlatform/order-gateway/pkg/const"
	ordercommon "github.com/NpoolPlatform/order-gateway/pkg/order/common"
	ordermwsvcname "github.com/NpoolPlatform/order-middleware/pkg/servicename"

	dtmcli "github.com/NpoolPlatform/dtm-cluster/pkg/dtm"
	"github.com/dtm-labs/dtm/client/dtmcli/dtmimp"

	"github.com/google/uuid"
)

type baseUpdateHandler struct {
	*dtmHandler
	*ordercommon.OrderOpHandler
	powerRentalOrder           *npool.PowerRentalOrder
	powerRentalOrderReq        *powerrentalordermwpb.PowerRentalOrderReq
	appPowerRental             *apppowerrentalmwpb.PowerRental
	commissionLedgerStatements []*ledgerstatementmwpb.Statement
	commissionLockIDs          map[string]string
	goodBenefitedAt            uint32
}

func (h *baseUpdateHandler) getPowerRentalOrder(ctx context.Context) (err error) {
	h.powerRentalOrder, err = h.GetPowerRentalOrder(ctx)
	return wlog.WrapError(err)
}

func (h *baseUpdateHandler) paymentUpdatable() error {
	switch h.powerRentalOrder.OrderState {
	case types.OrderState_OrderStateCreated:
	case types.OrderState_OrderStateWaitPayment:
	default:
		return wlog.Errorf("permission denied")
	}
	return nil
}

func (h *baseUpdateHandler) validateCancelParam() error {
	if h.UserSetCanceled != nil && !*h.UserSetCanceled {
		return wlog.Errorf("permission denied")
	}
	if h.AdminSetCanceled != nil && !*h.AdminSetCanceled {
		return wlog.Errorf("permission denied")
	}
	if h.powerRentalOrder.AdminSetCanceled || h.powerRentalOrder.UserSetCanceled {
		return wlog.Errorf("permission denied")
	}
	return nil
}

func (h *baseUpdateHandler) userCancelable() error {
	switch h.powerRentalOrder.OrderType {
	case types.OrderType_Normal:
		switch h.powerRentalOrder.OrderState {
		case types.OrderState_OrderStateWaitPayment:
			if h.AdminSetCanceled != nil {
				return wlog.Errorf("permission denied")
			}
		case types.OrderState_OrderStatePaid:
		case types.OrderState_OrderStateInService:
		default:
			return wlog.Errorf("permission denied")
		}
	case types.OrderType_Offline:
		fallthrough //nolint
	case types.OrderType_Airdrop:
		if h.UserSetCanceled != nil {
			return wlog.Errorf("permission denied")
		}
		switch h.powerRentalOrder.OrderState {
		case types.OrderState_OrderStatePaid:
		case types.OrderState_OrderStateInService:
		default:
			return wlog.Errorf("permission denied")
		}
	default:
		return wlog.Errorf("permission denied")
	}
	return nil
}

func (h *baseUpdateHandler) getGoodBenefitTime(ctx context.Context) error {
	statements, _, err := goodledgerstatementcli.GetGoodStatements(ctx, &goodledgerstatementpb.Conds{
		GoodID: &basetypes.StringVal{Op: cruder.EQ, Value: h.powerRentalOrder.GoodID},
	}, 0, 1)
	if err != nil {
		return wlog.WrapError(err)
	}
	if len(statements) > 0 {
		h.goodBenefitedAt = statements[0].BenefitDate
	}
	return nil
}

func (h *baseUpdateHandler) getAppPowerRental(ctx context.Context) (err error) {
	h.appPowerRental, err = apppowerrentalmwcli.GetPowerRental(ctx, h.powerRentalOrder.AppGoodID)
	return wlog.WrapError(err)
}

func (h *baseUpdateHandler) goodCancelable() error {
	switch h.appPowerRental.CancelMode {
	case goodtypes.CancelMode_Uncancellable:
		return wlog.Errorf("permission denied")
	case goodtypes.CancelMode_CancellableBeforeStart:
		switch h.powerRentalOrder.OrderState {
		case types.OrderState_OrderStatePaid:
		default:
			return wlog.Errorf("permission denied")
		}
	case goodtypes.CancelMode_CancellableBeforeBenefit:
		switch h.powerRentalOrder.OrderState {
		case types.OrderState_OrderStatePaid:
		case types.OrderState_OrderStateInService:
			if h.goodBenefitedAt == 0 {
				return nil
			}
			checkBenefitStartAt := h.goodBenefitedAt + timedef.SecondsPerDay - h.appPowerRental.CancelableBeforeStartSeconds
			checkBenefitEndAt := h.goodBenefitedAt + timedef.SecondsPerDay + h.appPowerRental.CancelableBeforeStartSeconds
			now := uint32(time.Now().Unix())
			if checkBenefitStartAt <= now && now <= checkBenefitEndAt {
				return wlog.Errorf("permission denied")
			}
		default:
			return wlog.Errorf("permission denied")
		}
	default:
		return wlog.Errorf("invalid cancelmode")
	}
	return nil
}

func (h *baseUpdateHandler) getCommissions(ctx context.Context) error {
	offset := int32(0)
	limit := constant.DefaultRowLimit
	for {
		infos, _, err := ledgerstatementmwcli.GetStatements(ctx, &ledgerstatementmwpb.Conds{
			AppID:     &basetypes.StringVal{Op: cruder.EQ, Value: h.powerRentalOrder.AppID},
			IOType:    &basetypes.Uint32Val{Op: cruder.EQ, Value: uint32(ledgertypes.IOType_Incoming)},
			IOSubType: &basetypes.Uint32Val{Op: cruder.EQ, Value: uint32(ledgertypes.IOSubType_Commission)},
			IOExtra:   &basetypes.StringVal{Op: cruder.LIKE, Value: h.powerRentalOrder.OrderID},
		}, offset, limit)
		if err != nil {
			return wlog.WrapError(err)
		}
		if len(infos) == 0 {
			return nil
		}
		h.commissionLedgerStatements = append(h.commissionLedgerStatements, infos...)
	}
}

func (h *baseUpdateHandler) prepareCommissionLockIDs() {
	for _, statement := range h.commissionLedgerStatements {
		if _, ok := h.commissionLockIDs[statement.UserID]; ok {
			continue
		}
		h.commissionLockIDs[statement.UserID] = uuid.NewString()
	}
}

func (h *baseUpdateHandler) constructPowerRentalOrderReq() {
	req := &powerrentalordermwpb.PowerRentalOrderReq{
		ID:               &h.powerRentalOrder.ID,
		EntID:            &h.powerRentalOrder.EntID,
		OrderID:          &h.powerRentalOrder.OrderID,
		PaymentType:      &h.PaymentType,
		LedgerLockID:     h.BalanceLockID,
		PaymentID:        h.PaymentID,
		UserSetPaid:      h.UserSetPaid,
		UserSetCanceled:  h.UserSetCanceled,
		AdminSetCanceled: h.AdminSetCanceled,
	}
	req.PaymentBalances = h.PaymentBalanceReqs
	if h.PaymentTransferReq != nil {
		req.PaymentTransfers = []*paymentmwpb.PaymentTransferReq{h.PaymentTransferReq}
	}
	h.OrderIDs = append(h.OrderIDs, *req.OrderID)
	h.powerRentalOrderReq = req
}

func (h *baseUpdateHandler) withCreateOrderCommissionLocks(dispose *dtmcli.SagaDispose) {
	reqs := func() (_reqs []*orderlockmwpb.OrderLockReq) {
		for _, statement := range h.commissionLedgerStatements {
			_reqs = append(_reqs, &orderlockmwpb.OrderLockReq{
				EntID:    func() *string { s := h.commissionLockIDs[statement.EntID]; return &s }(),
				UserID:   &statement.UserID,
				OrderID:  h.OrderID,
				LockType: types.OrderLockType_LockCommission.Enum(),
			})
		}
		return
	}()
	dispose.Add(
		ordermwsvcname.ServiceDomain,
		"order.middleware.order1.orderlock.v1.Middleware/CreateOrderLocks",
		"order.middleware.order1.orderlock.v1.Middleware/DeleteOrderLocks",
		&orderlockmwpb.CreateOrderLocksRequest{
			Infos: reqs,
		},
	)
}

func (h *baseUpdateHandler) withLockCommissions(dispose *dtmcli.SagaDispose) {
	balances := map[string][]*ledgermwpb.LockBalancesRequest_XBalance{}
	for _, statement := range h.commissionLedgerStatements {
		balances[statement.UserID] = append(balances[statement.UserID], &ledgermwpb.LockBalancesRequest_XBalance{
			CoinTypeID: statement.CoinTypeID,
			Amount:     statement.Amount,
		})
	}
	for userID, userBalances := range balances {
		dispose.Add(
			ledgermwsvcname.ServiceDomain,
			"ledger.middleware.ledger.v2.Middleware/LockBalances",
			"ledger.middleware.ledger.v2.Middleware/UnlockBalances",
			&ledgermwpb.LockBalancesRequest{
				AppID:    h.powerRentalOrder.AppID,
				UserID:   userID,
				LockID:   h.commissionLockIDs[userID],
				Rollback: true,
				Balances: userBalances,
			},
		)
	}
}

func (h *baseUpdateHandler) withUpdatePowerRentalOrder(dispose *dtmcli.SagaDispose) {
	dispose.Add(
		ordermwsvcname.ServiceDomain,
		"order.middleware.powerrental.v1.Middleware/UpdatePowerRentalOrder",
		"",
		&powerrentalordermwpb.UpdatePowerRentalOrderRequest{
			Info: h.powerRentalOrderReq,
		},
	)
}

func (h *baseUpdateHandler) formalizePayment() {
	h.powerRentalOrderReq.PaymentType = &h.PaymentType
	h.powerRentalOrderReq.PaymentBalances = h.PaymentBalanceReqs
	if h.PaymentTransferReq != nil {
		h.powerRentalOrderReq.PaymentTransfers = []*paymentmwpb.PaymentTransferReq{h.PaymentTransferReq}
	}
	h.powerRentalOrderReq.LedgerLockID = h.BalanceLockID
}

func (h *baseUpdateHandler) updatePowerRentalOrder(ctx context.Context) error {
	sagaDispose := dtmcli.NewSagaDispose(dtmimp.TransOptions{
		WaitResult:     true,
		RequestTimeout: 10,
		TimeoutToFail:  10,
	})

	if len(h.commissionLockIDs) > 0 {
		h.withCreateOrderCommissionLocks(sagaDispose)
		h.withLockCommissions(sagaDispose)
	}
	h.withUpdatePowerRentalOrder(sagaDispose)
	return h.dtmDo(ctx, sagaDispose)
}
