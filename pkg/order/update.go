package order

import (
	"context"
	"fmt"
	"time"

	goodmgrpb "github.com/NpoolPlatform/message/npool/good/mgr/v1/good"

	miningdetailcli "github.com/NpoolPlatform/ledger-middleware/pkg/client/mining/detail"
	miningdetailpb "github.com/NpoolPlatform/message/npool/ledger/mgr/v1/mining/detail"

	"github.com/NpoolPlatform/libent-cruder/pkg/cruder"
	commonpb "github.com/NpoolPlatform/message/npool"

	"github.com/shopspring/decimal"

	appgoodmwcli "github.com/NpoolPlatform/good-middleware/pkg/client/appgood"
	appgoodmwpb "github.com/NpoolPlatform/message/npool/good/mgr/v1/appgood"

	goodmwcli "github.com/NpoolPlatform/good-middleware/pkg/client/good"
	goodmwpb "github.com/NpoolPlatform/message/npool/good/mw/v1/good"

	ordermwpb "github.com/NpoolPlatform/message/npool/order/mw/v1/order"
	ordermwcli "github.com/NpoolPlatform/order-middleware/pkg/client/order"

	ledgercli "github.com/NpoolPlatform/ledger-middleware/pkg/client/ledger/v2"

	ledgerdetailpb "github.com/NpoolPlatform/message/npool/ledger/mgr/v1/ledger/detail"

	achievementmwcli "github.com/NpoolPlatform/inspire-middleware/pkg/client/achievement"
	statementmwcli "github.com/NpoolPlatform/inspire-middleware/pkg/client/achievement/statement"
	ordertypes "github.com/NpoolPlatform/message/npool/basetypes/order/v1"
	basetypes "github.com/NpoolPlatform/message/npool/basetypes/v1"
	statementmwpb "github.com/NpoolPlatform/message/npool/inspire/mw/v1/achievement/statement"
	npool "github.com/NpoolPlatform/message/npool/order/gw/v1/order"
)

type updateHandler struct {
	*Handler
	ord *ordermwpb.Order
}

//nolint:gocyclo
func (h *updateHandler) validate(ctx context.Context) error {
	good, err := appgoodmwcli.GetGoodOnly(ctx, &appgoodmwpb.Conds{
		AppID: &commonpb.StringVal{
			Op:    cruder.EQ,
			Value: h.ord.AppID,
		},
		GoodID: &commonpb.StringVal{
			Op:    cruder.EQ,
			Value: h.ord.GoodID,
		},
	})
	if err != nil {
		return err
	}
	if good == nil {
		return fmt.Errorf("invalid good")
	}

	_, total, err := miningdetailcli.GetDetails(ctx, &miningdetailpb.Conds{
		GoodID: &commonpb.StringVal{
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

	if good.BenefitState != goodmgrpb.BenefitState_BenefitWait {
		return fmt.Errorf("app good uncancellable benefit state not wait")
	}

	switch good.CancelMode {
	case appgoodmwpb.CancelMode_Uncancellable:
		return fmt.Errorf("app good uncancellable")
	case appgoodmwpb.CancelMode_CancellableBeforeStart:
		switch h.ord.OrderState {
		case ordertypes.OrderState_OrderStateWaitPayment:
		case ordertypes.OrderState_OrderStatePaid:
		default:
			return fmt.Errorf("order state is uncancellable")
		}

		if uint32(time.Now().Unix()) >= h.ord.Start-good.CancellableBeforeStart {
			return fmt.Errorf("cancellable time exceeded")
		}
	case appgoodmwpb.CancelMode_CancellableBeforeBenefit:
		switch h.ord.OrderState {
		case ordertypes.OrderState_OrderStateWaitPayment:
		case ordertypes.OrderState_OrderStatePaid:
		case ordertypes.OrderState_OrderStateInService:
		default:
			return fmt.Errorf("order state is uncancellable")
		}

		if uint32(time.Now().Unix()) >= h.ord.Start-good.CancellableBeforeStart &&
			uint32(time.Now().Unix()) <= h.ord.Start+good.CancellableBeforeStart {
			return fmt.Errorf("app good uncancellable order start at > cancellable before start")
		}
	default:
		return fmt.Errorf("unknown CancelMode type %v", good.CancelMode)
	}
	return nil
}

func (h *updateHandler) processStock(ctx context.Context) error {
	units, err := decimal.NewFromString(h.ord.Units)
	if err != nil {
		return err
	}
	unitsStr := units.Neg().String()

	stockReq := &goodmwpb.GoodReq{
		ID: &h.ord.GoodID,
	}

	switch h.ord.OrderState {
	case ordertypes.OrderState_OrderStatePaid:
		stockReq.WaitStart = &unitsStr
	case ordertypes.OrderState_OrderStateInService:
		stockReq.InService = &unitsStr
	}

	_, err = goodmwcli.UpdateGood(ctx, stockReq)
	if err != nil {
		return err
	}
	return nil
}

func (h *updateHandler) processOrderState(ctx context.Context) error {
	cancle := true
	state := ordertypes.OrderState_OrderStateCanceled
	paymentState := ordertypes.PaymentState_PaymentStateCanceled
	_, err := ordermwcli.UpdateOrder(ctx, &ordermwpb.OrderReq{
		ID:           &h.ord.ID,
		State:        &state,
		PaymentState: &paymentState,
		PaymentID:    &h.ord.PaymentID,
		Canceled:     &cancle,
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
	detailInfos := []*ledgerdetailpb.DetailReq{}
	in := ledgerdetailpb.IOType_Incoming
	out := ledgerdetailpb.IOType_Outcoming
	ioTypeCR := ledgerdetailpb.IOSubType_CommissionRevoke
	ioTypeOrder := ledgerdetailpb.IOSubType_OrderRevoke

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

			_, total, err := ledgercli.GetDetails(ctx, &ledgerdetailpb.Conds{
				AppID: &commonpb.StringVal{
					Op:    cruder.EQ,
					Value: val.AppID,
				},
				UserID: &commonpb.StringVal{
					Op:    cruder.EQ,
					Value: val.UserID,
				},
				IOType: &commonpb.Int32Val{
					Op:    cruder.EQ,
					Value: int32(in),
				},
				IOSubType: &commonpb.Int32Val{
					Op:    cruder.EQ,
					Value: int32(ledgerdetailpb.IOSubType_Commission),
				},
				IOExtra: &commonpb.StringVal{
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

			detailInfos = append(detailInfos, &ledgerdetailpb.DetailReq{
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

	payWithBalanceAmount, err := decimal.NewFromString(h.ord.PayWithBalanceAmount)
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

		detailInfos = append(detailInfos, &ledgerdetailpb.DetailReq{
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
		err = ledgercli.BookKeeping(ctx, detailInfos)
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
	if h.Canceled == nil {
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
	if !*h.Canceled {
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
			ID:        h.ID,
			AppID:     h.AppID,
			UserID:    h.UserID,
			PaymentID: h.PaymentID,
			Canceled:  h.Canceled,
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
