package order

import (
	"context"
	"fmt"
	"time"

	"github.com/NpoolPlatform/libent-cruder/pkg/cruder"

	goodtypes "github.com/NpoolPlatform/message/npool/basetypes/good/v1"
	ledgertypes "github.com/NpoolPlatform/message/npool/basetypes/ledger/v1"
	ordertypes "github.com/NpoolPlatform/message/npool/basetypes/order/v1"
	basetypes "github.com/NpoolPlatform/message/npool/basetypes/v1"

	appgoodmwcli "github.com/NpoolPlatform/good-middleware/pkg/client/app/good"
	appgoodstockmwcli "github.com/NpoolPlatform/good-middleware/pkg/client/app/good/stock"
	goodmwcli "github.com/NpoolPlatform/good-middleware/pkg/client/good"
	appgoodmwpb "github.com/NpoolPlatform/message/npool/good/mw/v1/app/good"
	appgoodstockmwpb "github.com/NpoolPlatform/message/npool/good/mw/v1/app/good/stock"

	npool "github.com/NpoolPlatform/message/npool/order/gw/v1/order"
	ordermwpb "github.com/NpoolPlatform/message/npool/order/mw/v1/order"
	ordermwcli "github.com/NpoolPlatform/order-middleware/pkg/client/order"

	miningdetailcli "github.com/NpoolPlatform/ledger-middleware/pkg/client/good/ledger/statement"
	ledgerstatementcli "github.com/NpoolPlatform/ledger-middleware/pkg/client/ledger/statement"
	miningdetailpb "github.com/NpoolPlatform/message/npool/ledger/mw/v2/good/ledger/statement"
	ledgerstatementpb "github.com/NpoolPlatform/message/npool/ledger/mw/v2/ledger/statement"

	achievementmwcli "github.com/NpoolPlatform/inspire-middleware/pkg/client/achievement"
	statementmwcli "github.com/NpoolPlatform/inspire-middleware/pkg/client/achievement/statement"
	statementmwpb "github.com/NpoolPlatform/message/npool/inspire/mw/v1/achievement/statement"

	"github.com/shopspring/decimal"
)

type updateHandler struct {
	*Handler
	ord *ordermwpb.Order
}

//nolint:gocyclo
func (h *updateHandler) validate(ctx context.Context) error {
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

	_, total, err := miningdetailcli.GetGoodStatements(ctx, &miningdetailpb.Conds{
		GoodID: &basetypes.StringVal{
			Op:    cruder.EQ,
			Value: h.ord.GoodID,
		},
	}, 0, 1)
	if err != nil {
		return err
	}
	if total > 0 {
		return fmt.Errorf("app good have mining detail data uncancellable")
	}

	if good.RewardState != goodtypes.BenefitState_BenefitWait {
		return fmt.Errorf("app good uncancellable benefit state not wait")
	}

	switch appgood.CancelMode {
	case goodtypes.CancelMode_Uncancellable:
		return fmt.Errorf("app good uncancellable")
	case goodtypes.CancelMode_CancellableBeforeStart:
		switch h.ord.OrderState {
		case ordertypes.OrderState_OrderStateWaitPayment:
		case ordertypes.OrderState_OrderStatePaid:
		default:
			return fmt.Errorf("order state is uncancellable")
		}

		if uint32(time.Now().Unix()) >= h.ord.StartAt-appgood.CancellableBeforeStart {
			return fmt.Errorf("cancellable time exceeded")
		}
	case goodtypes.CancelMode_CancellableBeforeBenefit:
		switch h.ord.OrderState {
		case ordertypes.OrderState_OrderStateWaitPayment:
		case ordertypes.OrderState_OrderStatePaid:
		case ordertypes.OrderState_OrderStateInService:
		default:
			return fmt.Errorf("order state is uncancellable")
		}

		if uint32(time.Now().Unix()) >= h.ord.StartAt-appgood.CancellableBeforeStart &&
			uint32(time.Now().Unix()) <= h.ord.StartAt+appgood.CancellableBeforeStart {
			return fmt.Errorf("app good uncancellable order start at > cancellable before start")
		}
	default:
		return fmt.Errorf("unknown CancelMode type %v", appgood.CancelMode)
	}
	return nil
}

func (h *updateHandler) processStock(ctx context.Context) error {
	stockReq := &appgoodstockmwpb.StockReq{
		ID: &h.ord.GoodID,
	}

	switch h.ord.OrderState {
	case ordertypes.OrderState_OrderStatePaid:
		stockReq.WaitStart = &h.Units
	case ordertypes.OrderState_OrderStateInService:
		stockReq.InService = &h.Units
	}

	_, err := appgoodstockmwcli.SubStock(ctx, stockReq)
	if err != nil {
		return err
	}
	return nil
}

func (h *updateHandler) processOrderState(ctx context.Context) error {
	state := ordertypes.OrderState_OrderStateCanceled
	paymentState := ordertypes.PaymentState_PaymentStateCanceled
	_, err := ordermwcli.UpdateOrder(ctx, &ordermwpb.OrderReq{
		ID:               &h.ord.ID,
		OrderState:       &state,
		PaymentState:     &paymentState,
		UserSetCanceled:  h.UserSetCanceled,
		AdminSetCanceled: h.AdminSetCanceled,
	})
	if err != nil {
		return err
	}

	return nil
}

//nolint:funlen
func (h *updateHandler) processLedger(ctx context.Context) error {
	offset := int32(0)
	limit := int32(1000) //nolint
	detailInfos := []*ledgerstatementpb.StatementReq{}
	in := ledgertypes.IOType_Incoming
	out := ledgertypes.IOType_Outcoming
	ioTypeCR := ledgertypes.IOSubType_CommissionRevoke
	ioTypeOrder := ledgertypes.IOSubType_OrderRevoke

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

			inIoExtra := fmt.Sprintf(
				`{"AppID":"%v","UserID":"%v","ArchivementDetailID":"%v","Amount":"%v","Date":"%v"}`,
				val.AppID,
				val.UserID,
				val.ID,
				val.Commission,
				time.Now(),
			)

			detailInfos = append(detailInfos, &ledgerstatementpb.StatementReq{
				AppID:      &val.AppID,
				UserID:     &val.UserID,
				CoinTypeID: &val.PaymentCoinTypeID,
				IOType:     &out,
				IOSubType:  &ioTypeCR,
				Amount:     &val.Commission,
				IOExtra:    &inIoExtra,
			})
		}
	}

	paymentAmount, err := decimal.NewFromString(h.ord.PaymentAmount)
	if err != nil {
		return err
	}

	payWithBalanceAmount, err := decimal.NewFromString(h.ord.BalanceAmount)
	if err != nil {
		return err
	}

	if paymentAmount.Add(payWithBalanceAmount).Cmp(decimal.NewFromInt(0)) != 0 {
		amount := paymentAmount.Add(payWithBalanceAmount).String()
		inIoExtra := fmt.Sprintf(
			`{"AppID":"%v","UserID":"%v","OrderID":"%v","Amount":"%v","Date":"%v"}`,
			h.ord.AppID,
			h.ord.UserID,
			h.ord.ID,
			amount,
			time.Now(),
		)

		detailInfos = append(detailInfos, &ledgerstatementpb.StatementReq{
			AppID:      &h.ord.AppID,
			UserID:     &h.ord.UserID,
			CoinTypeID: &h.ord.PaymentCoinTypeID,
			IOType:     &in,
			IOSubType:  &ioTypeOrder,
			Amount:     &amount,
			IOExtra:    &inIoExtra,
		})
	}

	if len(detailInfos) > 0 {
		_, err = ledgerstatementcli.CreateStatements(ctx, detailInfos)
		if err != nil {
			return err
		}
	}

	return nil
}

func (h *updateHandler) processArchivement(ctx context.Context) error {
	err := achievementmwcli.ExpropriateAchievement(ctx, h.ord.ID)
	if err != nil {
		return err
	}
	return nil
}

func (h *updateHandler) cancelAirdropOrder(ctx context.Context) error {
	err := h.validate(ctx)
	if err != nil {
		return err
	}
	err = h.processStock(ctx)
	if err != nil {
		return err
	}
	// TODO Distributed transactions should be used
	return h.processOrderState(ctx)
}

func (h *updateHandler) cancelOfflineOrder(ctx context.Context) error {
	err := h.validate(ctx)
	if err != nil {
		return err
	}
	// TODO Distributed transactions should be used

	err = h.processStock(ctx)
	if err != nil {
		return err
	}
	err = h.processOrderState(ctx)
	if err != nil {
		return err
	}

	return h.processArchivement(ctx)
}

func (h *updateHandler) cancelNormalOrder(ctx context.Context) error {
	err := h.validate(ctx)
	if err != nil {
		return err
	}

	err = h.processStock(ctx)
	if err != nil {
		return err
	}

	err = h.processOrderState(ctx)
	if err != nil {
		return err
	}

	err = h.processLedger(ctx)
	if err != nil {
		return err
	}

	return h.processArchivement(ctx)
}

//nolint:gocyclo
func (h *Handler) UpdateOrder(ctx context.Context) (*npool.Order, error) {
	if h.UserSetCanceled == nil && h.AdminSetCanceled == nil {
		return nil, fmt.Errorf("nothing todo")
	}
	if h.ID == nil || *h.ID == "" {
		return nil, fmt.Errorf("id invalid")
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

	switch ord.OrderState {
	case ordertypes.OrderState_OrderStateWaitPayment:
		if h.FromAdmin && ord.OrderType == ordertypes.OrderType_Normal {
			return nil, fmt.Errorf("permission denied")
		}
		_, err = ordermwcli.UpdateOrder(ctx, &ordermwpb.OrderReq{
			ID:               h.ID,
			AppID:            h.AppID,
			UserID:           h.UserID,
			UserSetCanceled:  h.UserSetCanceled,
			AdminSetCanceled: h.AdminSetCanceled,
		})
		if err != nil {
			return nil, err
		}
		return handler.GetOrder(ctx)
	case ordertypes.OrderState_OrderStatePaid:
	case ordertypes.OrderState_OrderStateInService:
	default:
		return nil, fmt.Errorf("order state uncancellable")
	}

	switch ord.OrderType {
	case ordertypes.OrderType_Normal:
		if err := handler.cancelNormalOrder(ctx); err != nil {
			return nil, err
		}
	case ordertypes.OrderType_Offline:
		if !h.FromAdmin {
			return nil, fmt.Errorf("permission denied")
		}
		if err := handler.cancelOfflineOrder(ctx); err != nil {
			return nil, err
		}
	case ordertypes.OrderType_Airdrop:
		if !h.FromAdmin {
			return nil, fmt.Errorf("permission denied")
		}
		if err := handler.cancelAirdropOrder(ctx); err != nil {
			return nil, err
		}
	default:
		return nil, fmt.Errorf("order type uncancellable")
	}

	return handler.GetOrder(ctx)
}
