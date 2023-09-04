package order

import (
	"context"
	"fmt"
	"time"

	dtmcli "github.com/NpoolPlatform/dtm-cluster/pkg/dtm"
	"github.com/NpoolPlatform/libent-cruder/pkg/cruder"
	ordermwsvcname "github.com/NpoolPlatform/order-middleware/pkg/servicename"
	"github.com/dtm-labs/dtm/client/dtmcli/dtmimp"

	goodtypes "github.com/NpoolPlatform/message/npool/basetypes/good/v1"
	ledgertypes "github.com/NpoolPlatform/message/npool/basetypes/ledger/v1"
	ordertypes "github.com/NpoolPlatform/message/npool/basetypes/order/v1"
	basetypes "github.com/NpoolPlatform/message/npool/basetypes/v1"

	appgoodmwcli "github.com/NpoolPlatform/good-middleware/pkg/client/app/good"
	goodmwcli "github.com/NpoolPlatform/good-middleware/pkg/client/good"
	appgoodmwpb "github.com/NpoolPlatform/message/npool/good/mw/v1/app/good"

	npool "github.com/NpoolPlatform/message/npool/order/gw/v1/order"
	ordermwpb "github.com/NpoolPlatform/message/npool/order/mw/v1/order"
	ordermwcli "github.com/NpoolPlatform/order-middleware/pkg/client/order"

	goodledgerstatementcli "github.com/NpoolPlatform/ledger-middleware/pkg/client/good/ledger/statement"

	ledgerstatementcli "github.com/NpoolPlatform/ledger-middleware/pkg/client/ledger/statement"
	goodledgerstatementpb "github.com/NpoolPlatform/message/npool/ledger/mw/v2/good/ledger/statement"
	ledgermwpb "github.com/NpoolPlatform/message/npool/ledger/mw/v2/ledger"
	ledgerstatementpb "github.com/NpoolPlatform/message/npool/ledger/mw/v2/ledger/statement"

	statementmwcli "github.com/NpoolPlatform/inspire-middleware/pkg/client/achievement/statement"
	statementmwpb "github.com/NpoolPlatform/message/npool/inspire/mw/v1/achievement/statement"

	"github.com/shopspring/decimal"
)

type updateHandler struct {
	*Handler
	ord                   *ordermwpb.Order
	achievementStatements []*statementmwpb.Statement
}

//nolint:gocyclo
func (h *updateHandler) cancelable(ctx context.Context) error {
	appgood, err := appgoodmwcli.GetGoodOnly(ctx, &appgoodmwpb.Conds{
		AppID: &basetypes.StringVal{
			Op:    cruder.EQ,
			Value: h.ord.AppID,
		},
		GoodID: &basetypes.StringVal{
			Op:    cruder.EQ,
			Value: h.ord.GoodID,
		},
	})
	if err != nil {
		return err
	}
	if appgood == nil {
		return fmt.Errorf("invalid appgood")
	}

	good, err := goodmwcli.GetGood(ctx, h.ord.GoodID)
	if err != nil {
		return err
	}
	if good == nil {
		return fmt.Errorf("invalid good")
	}

	statements, _, err := goodledgerstatementcli.GetGoodStatements(ctx, &goodledgerstatementpb.Conds{
		GoodID: &basetypes.StringVal{
			Op:    cruder.EQ,
			Value: h.ord.GoodID,
		},
	}, 0, 1)
	if err != nil {
		return err
	}

	switch h.ord.OrderState {
	case ordertypes.OrderState_OrderStateWaitPayment:
	case ordertypes.OrderState_OrderStateCheckPayment:
		if len(statements) > 0 {
			return fmt.Errorf("had statements can not cancel")
		}
		return nil
	}

	if good.RewardState != goodtypes.BenefitState_BenefitWait {
		return fmt.Errorf("app good uncancellable benefit state not wait")
	}

	switch appgood.CancelMode {
	case goodtypes.CancelMode_Uncancellable:
		return fmt.Errorf("app good uncancellable")
	case goodtypes.CancelMode_CancellableBeforeStart:
		switch h.ord.OrderState {
		case ordertypes.OrderState_OrderStatePaid:
		case ordertypes.OrderState_OrderStateInService:
			return fmt.Errorf("order state is uncancellable")
		default:
			return fmt.Errorf("order state is uncancellable")
		}
	case goodtypes.CancelMode_CancellableBeforeBenefit:
		switch h.ord.OrderState {
		case ordertypes.OrderState_OrderStatePaid:
		case ordertypes.OrderState_OrderStateInService:
			if len(statements) > 0 {
				lastBenefitDate := statements[0].BenefitDate
				const secondsPerDay = 24 * 60 * 60
				checkBenefitStartAt := lastBenefitDate + secondsPerDay - appgood.CancellableBeforeStart
				checkBenefitEndAt := lastBenefitDate + secondsPerDay + appgood.CancellableBeforeStart
				now := uint32(time.Now().Unix())
				if checkBenefitStartAt <= now && now <= checkBenefitEndAt {
					return fmt.Errorf("invalid cancel in during time")
				}
			}
		default:
			return fmt.Errorf("order state is uncancellable")
		}
	default:
		return fmt.Errorf("unknown CancelMode type %v", appgood.CancelMode)
	}

	return nil
}

func (h *updateHandler) processStatements(ctx context.Context) error {
	offset := int32(0)
	limit := int32(1000) //nolint
	in := ledgertypes.IOType_Incoming
	for {
		infos, _, err := statementmwcli.GetStatements(ctx, &statementmwpb.Conds{
			OrderID: &basetypes.StringVal{Op: cruder.EQ, Value: h.ord.ID},
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
			_, total, err := ledgerstatementcli.GetStatements(ctx, &ledgerstatementpb.Conds{
				AppID: &basetypes.StringVal{
					Op:    cruder.EQ,
					Value: val.AppID,
				},
				UserID: &basetypes.StringVal{
					Op:    cruder.EQ,
					Value: val.UserID,
				},
				IOType: &basetypes.Uint32Val{
					Op:    cruder.EQ,
					Value: uint32(in),
				},
				IOSubType: &basetypes.Uint32Val{
					Op:    cruder.EQ,
					Value: uint32(ledgertypes.IOSubType_Commission),
				},
				IOExtra: &basetypes.StringVal{
					Op:    cruder.LIKE,
					Value: h.ord.ID,
				},
			}, 0, 1)
			if err != nil {
				return err
			}
			if total == 0 {
				return fmt.Errorf("commission ledger detail is not exist")
			}
			h.achievementStatements = append(h.achievementStatements, val)
		}
	}

	return nil
}

func (h *updateHandler) withLockCommission(dispose *dtmcli.SagaDispose) {
	for _, statement := range h.achievementStatements {
		req := &ledgermwpb.LedgerReq{
			AppID:      &statement.AppID,
			UserID:     &statement.UserID,
			CoinTypeID: &statement.CoinTypeID,
			Spendable:  &statement.Commission,
		}
		dispose.Add(
			ordermwsvcname.ServiceDomain,
			"ledger.middleware.ledger.v2.Middleware/SubBalance",
			"ledger.middleware.ledger.v2.Middleware/AddBalance",
			&ledgermwpb.AddBalanceRequest{
				Info: req,
			},
		)
	}
}

func (h *updateHandler) withProcessCancel(dispose *dtmcli.SagaDispose) {
	req := &ordermwpb.OrderReq{
		ID:               &h.ord.ID,
		UserSetCanceled:  h.UserSetCanceled,
		AdminSetCanceled: h.AdminSetCanceled,
	}
	dispose.Add(
		ordermwsvcname.ServiceDomain,
		"order.middleware.order1.v1.Middleware/SubBalance",
		"order.middleware.order1.v1.Middleware/AddBalance",
		&ordermwpb.UpdateOrderRequest{
			Info: req,
		},
	)
}

//nolint:gocyclo
func (h *Handler) UpdateOrder(ctx context.Context) (*npool.Order, error) {
	if h.UserSetCanceled == nil && h.AdminSetCanceled == nil {
		return nil, fmt.Errorf("nothing todo")
	}

	ord, err := ordermwcli.GetOrder(ctx, *h.ID)
	if err != nil {
		return nil, err
	}
	if ord == nil {
		return nil, fmt.Errorf("invalid order")
	}

	if *h.AppID != ord.AppID || *h.UserID != ord.UserID {
		return nil, fmt.Errorf("permission denied")
	}
	if h.UserSetCanceled != nil && !*h.UserSetCanceled {
		return h.GetOrder(ctx)
	}
	if h.AdminSetCanceled != nil && !*h.AdminSetCanceled {
		return h.GetOrder(ctx)
	}

	handler := &updateHandler{
		Handler: h,
		ord:     ord,
	}

	switch ord.OrderType {
	case ordertypes.OrderType_Normal:
		switch ord.OrderState {
		case ordertypes.OrderState_OrderStateWaitPayment:
			fallthrough //nolint
		case ordertypes.OrderState_OrderStateCheckPayment:
			if h.AdminSetCanceled != nil {
				return nil, fmt.Errorf("permission denied")
			}
		case ordertypes.OrderState_OrderStatePaid:
		case ordertypes.OrderState_OrderStateInService:
		}
	case ordertypes.OrderType_Offline:
		fallthrough //nolint
	case ordertypes.OrderType_Airdrop:
		if h.AdminSetCanceled == nil {
			return nil, fmt.Errorf("permission denied")
		}
	default:
		return nil, fmt.Errorf("order type uncancellable")
	}

	if err := handler.cancelable(ctx); err != nil {
		return nil, err
	}

	if err := handler.processStatements(ctx); err != nil {
		return nil, err
	}

	sagaDispose := dtmcli.NewSagaDispose(dtmimp.TransOptions{
		WaitResult:     true,
		RequestTimeout: handler.RequestTimeoutSeconds,
	})

	handler.withLockCommission(sagaDispose)
	handler.withProcessCancel(sagaDispose)

	if err := dtmcli.WithSaga(ctx, sagaDispose); err != nil {
		return nil, err
	}

	return handler.GetOrder(ctx)
}
