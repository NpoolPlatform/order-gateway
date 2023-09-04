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
	goodmwcli "github.com/NpoolPlatform/good-middleware/pkg/client/good"
	appgoodmwpb "github.com/NpoolPlatform/message/npool/good/mw/v1/app/good"

	npool "github.com/NpoolPlatform/message/npool/order/gw/v1/order"
	ordermwpb "github.com/NpoolPlatform/message/npool/order/mw/v1/order"
	ordermwcli "github.com/NpoolPlatform/order-middleware/pkg/client/order"

	miningdetailcli "github.com/NpoolPlatform/ledger-middleware/pkg/client/good/ledger/statement"

	ledgerstatementcli "github.com/NpoolPlatform/ledger-middleware/pkg/client/ledger/statement"
	miningdetailpb "github.com/NpoolPlatform/message/npool/ledger/mw/v2/good/ledger/statement"
	ledgerstatementpb "github.com/NpoolPlatform/message/npool/ledger/mw/v2/ledger/statement"

	statementmwcli "github.com/NpoolPlatform/inspire-middleware/pkg/client/achievement/statement"
	statementmwpb "github.com/NpoolPlatform/message/npool/inspire/mw/v1/achievement/statement"

	"github.com/shopspring/decimal"
)

type updateHandler struct {
	*Handler
	ord *ordermwpb.Order
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

	statements, _, err := miningdetailcli.GetGoodStatements(ctx, &miningdetailpb.Conds{
		GoodID: &basetypes.StringVal{
			Op:    cruder.EQ,
			Value: h.ord.GoodID,
		},
	}, 0, 1)
	if err != nil {
		return err
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

func (h *updateHandler) processCancel(ctx context.Context) error {
	_, err := ordermwcli.UpdateOrder(ctx, &ordermwpb.OrderReq{
		ID:               &h.ord.ID,
		AppID:            h.AppID,
		UserID:           h.UserID,
		UserSetCanceled:  h.UserSetCanceled,
		AdminSetCanceled: h.AdminSetCanceled,
	})
	if err != nil {
		return err
	}

	return nil
}

func (h *updateHandler) processLedger(ctx context.Context) error {
	offset := int32(0)
	limit := int32(1000) //nolint
	detailInfos := []*ledgerstatementpb.StatementReq{}
	in := ledgertypes.IOType_Incoming
	out := ledgertypes.IOType_Outcoming
	ioTypeCR := ledgertypes.IOSubType_CommissionRevoke

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

	if len(detailInfos) > 0 {
		_, err := ledgerstatementcli.CreateStatements(ctx, detailInfos)
		if err != nil {
			return err
		}
	}

	return nil
}

//nolint:funlen,gocyclo
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
	if h.FromAdmin && h.UserSetCanceled != nil {
		return nil, fmt.Errorf("permission denied")
	}
	if !h.FromAdmin && h.AdminSetCanceled != nil {
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
			if h.FromAdmin {
				return nil, fmt.Errorf("permission denied")
			}
			fallthrough //nolint
		case ordertypes.OrderState_OrderStatePaid:
			fallthrough //nolint
		case ordertypes.OrderState_OrderStateInService:
			if err := handler.validate(ctx); err != nil {
				return nil, err
			}
			if err := handler.processLedger(ctx); err != nil {
				return nil, err
			}
			if err := handler.processCancel(ctx); err != nil {
				return nil, err
			}
		}
	case ordertypes.OrderType_Offline:
		fallthrough //nolint
	case ordertypes.OrderType_Airdrop:
		if !h.FromAdmin {
			return nil, fmt.Errorf("permission denied")
		}
		if err := handler.validate(ctx); err != nil {
			return nil, err
		}
		if err := handler.processCancel(ctx); err != nil {
			return nil, err
		}
	default:
		return nil, fmt.Errorf("order type uncancellable")
	}

	return handler.GetOrder(ctx)
}
