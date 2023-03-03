package order

import (
	"context"
	"fmt"
	miningdetailcli "github.com/NpoolPlatform/ledger-middleware/pkg/client/mining/detail"
	miningdetailpb "github.com/NpoolPlatform/message/npool/ledger/mgr/v1/mining/detail"
	"time"

	"github.com/NpoolPlatform/libent-cruder/pkg/cruder"
	commonpb "github.com/NpoolPlatform/message/npool"

	"github.com/shopspring/decimal"

	appgoodmwcli "github.com/NpoolPlatform/good-middleware/pkg/client/appgood"
	appgoodmwpb "github.com/NpoolPlatform/message/npool/good/mgr/v1/appgood"

	goodmwcli "github.com/NpoolPlatform/good-middleware/pkg/client/good"
	archivementmwcli "github.com/NpoolPlatform/inspire-middleware/pkg/client/archivement"
	goodmgrpb "github.com/NpoolPlatform/message/npool/good/mgr/v1/good"
	goodmwpb "github.com/NpoolPlatform/message/npool/good/mw/v1/good"
	ordermgrpb "github.com/NpoolPlatform/message/npool/order/mgr/v1/order"
	paymentmgrpb "github.com/NpoolPlatform/message/npool/order/mgr/v1/payment"

	ordermwpb "github.com/NpoolPlatform/message/npool/order/mw/v1/order"
	ordermwcli "github.com/NpoolPlatform/order-middleware/pkg/client/order"

	archivementdetailcli "github.com/NpoolPlatform/inspire-middleware/pkg/client/archivement/detail"
	archivementdetailpb "github.com/NpoolPlatform/message/npool/inspire/mgr/v1/archivement/detail"

	ledgercli "github.com/NpoolPlatform/ledger-middleware/pkg/client/ledger/v2"

	ledgerdetailpb "github.com/NpoolPlatform/message/npool/ledger/mgr/v1/ledger/detail"
)

//nolint:gocyclo
func validateInit(ctx context.Context, ord *ordermwpb.Order) error {
	good, err := appgoodmwcli.GetGoodOnly(ctx, &appgoodmwpb.Conds{
		AppID: &commonpb.StringVal{
			Op:    cruder.EQ,
			Value: ord.AppID,
		},
		GoodID: &commonpb.StringVal{
			Op:    cruder.EQ,
			Value: ord.GoodID,
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
			Value: ord.GoodID,
		},
	}, 0, 1)
	if err != nil {
		return err
	}
	if total > 0 {
		return fmt.Errorf("app good have mining detail data uncancellable")
	}

	switch good.CancelMode {
	case appgoodmwpb.CancelMode_Uncancellable:
		return fmt.Errorf("app good uncancellable")
	case appgoodmwpb.CancelMode_CancellableBeforeStart:
		switch ord.OrderState {
		case ordermgrpb.OrderState_WaitPayment:
		case ordermgrpb.OrderState_Paid:
		default:
			return fmt.Errorf("order state is uncancellable")
		}

		if uint32(time.Now().Unix()) >= ord.Start-good.CancellableBeforeStart {
			return fmt.Errorf("cancellable time exceeded")
		}
	case appgoodmwpb.CancelMode_CancellableBeforeBenefit:
		switch ord.OrderState {
		case ordermgrpb.OrderState_WaitPayment:
		case ordermgrpb.OrderState_Paid:
		case ordermgrpb.OrderState_InService:
		default:
			return fmt.Errorf("order state is uncancellable")
		}

		if uint32(time.Now().Unix()) >= ord.Start-good.CancellableBeforeStart &&
			uint32(time.Now().Unix()) <= ord.Start+good.CancellableBeforeStart {
			return fmt.Errorf("app good uncancellable order start at > cancellable before start")
		}

		if good.BenefitState != goodmgrpb.BenefitState_BenefitWait {
			return fmt.Errorf("app good uncancellable benefit state not wait")
		}
	default:
		return fmt.Errorf("unknown CancelMode type %v", good.CancelMode)
	}
	return nil
}

func processStock(ctx context.Context, ord *ordermwpb.Order) error {
	units, err := decimal.NewFromString(ord.Units)
	if err != nil {
		return err
	}
	unitsStr := units.Neg().String()

	stockReq := &goodmwpb.GoodReq{
		ID: &ord.GoodID,
	}

	switch ord.OrderState {
	case ordermgrpb.OrderState_Paid:
		stockReq.WaitStart = &unitsStr
	case ordermgrpb.OrderState_InService:
		stockReq.InService = &unitsStr
	}

	_, err = goodmwcli.UpdateGood(ctx, stockReq)
	if err != nil {
		return err
	}
	return nil
}

func processOrderState(ctx context.Context, ord *ordermwpb.Order) error {
	cancle := true
	state := ordermgrpb.OrderState_Canceled
	paymentState := paymentmgrpb.PaymentState_Canceled
	_, err := ordermwcli.UpdateOrder(ctx, &ordermwpb.OrderReq{
		ID:           &ord.ID,
		State:        &state,
		PaymentState: &paymentState,
		PaymentID:    &ord.PaymentID,
		Canceled:     &cancle,
	})
	if err != nil {
		return err
	}

	return nil
}

//nolint:funlen
func processLedger(ctx context.Context, ord *ordermwpb.Order) error {
	offset := uint32(0)
	limit := uint32(1000) //nolint
	detailInfos := []*ledgerdetailpb.DetailReq{}
	in := ledgerdetailpb.IOType_Incoming
	out := ledgerdetailpb.IOType_Outcoming
	ioTypeCR := ledgerdetailpb.IOSubType_CommissionRevoke
	ioTypeOrder := ledgerdetailpb.IOSubType_OrderRevoke

	for {
		infos, _, err := archivementdetailcli.GetDetails(ctx, &archivementdetailpb.Conds{
			OrderID: &commonpb.StringVal{
				Op:    cruder.EQ,
				Value: ord.ID,
			},
		}, offset, limit)
		if err != nil {
			return err
		}
		offset += limit
		if len(detailInfos) == 0 {
			break
		}
		for _, val := range infos {
			_, total, err := ledgercli.GetDetails(ctx, &ledgerdetailpb.Conds{
				AppID: &commonpb.StringVal{
					Op:    cruder.EQ,
					Value: ord.AppID,
				},
				UserID: &commonpb.StringVal{
					Op:    cruder.EQ,
					Value: ord.UserID,
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
					Value: ord.ID,
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
				CoinTypeID: &val.CoinTypeID,
				IOType:     &out,
				IOSubType:  &ioTypeCR,
				Amount:     &val.Commission,
				IOExtra:    &inIoExtra,
			})
		}
	}

	paymentAmount, err := decimal.NewFromString(ord.PaymentAmount)
	if err != nil {
		return err
	}

	payWithBalanceAmount, err := decimal.NewFromString(ord.PayWithBalanceAmount)
	if err != nil {
		return err
	}
	amount := paymentAmount.Add(payWithBalanceAmount).String()

	inIoExtra := fmt.Sprintf(
		`{"AppID":"%v","UserID":"%v","OrderID":"%v","Amount":"%v","Date":"%v"}`,
		ord.AppID,
		ord.UserID,
		ord.ID,
		amount,
		time.Now(),
	)

	detailInfos = append(detailInfos, &ledgerdetailpb.DetailReq{
		AppID:      &ord.AppID,
		UserID:     &ord.UserID,
		CoinTypeID: &ord.PaymentCoinTypeID,
		IOType:     &in,
		IOSubType:  &ioTypeOrder,
		Amount:     &amount,
		IOExtra:    &inIoExtra,
	})

	err = ledgercli.BookKeeping(ctx, detailInfos)
	if err != nil {
		return err
	}
	return nil
}

func processArchivement(ctx context.Context, ord *ordermwpb.Order) error {
	err := archivementmwcli.Expropriate(ctx, ord.ID)
	if err != nil {
		return err
	}
	return nil
}

func cancelAirdropOrder(ctx context.Context, ord *ordermwpb.Order) error {
	err := validateInit(ctx, ord)
	if err != nil {
		return err
	}
	err = processStock(ctx, ord)
	if err != nil {
		return err
	}
	// TODO Distributed transactions should be used
	return processOrderState(ctx, ord)
}

func cancelOfflineOrder(ctx context.Context, ord *ordermwpb.Order) error {
	err := validateInit(ctx, ord)
	if err != nil {
		return err
	}
	// TODO Distributed transactions should be used

	err = processArchivement(ctx, ord)
	if err != nil {
		return err
	}

	err = processStock(ctx, ord)
	if err != nil {
		return err
	}

	return processOrderState(ctx, ord)
}

func cancelNormalOrder(ctx context.Context, ord *ordermwpb.Order) error {
	err := validateInit(ctx, ord)
	if err != nil {
		return err
	}

	err = processStock(ctx, ord)
	if err != nil {
		return err
	}

	err = processOrderState(ctx, ord)
	if err != nil {
		return err
	}

	err = processArchivement(ctx, ord)
	if err != nil {
		return err
	}

	return processLedger(ctx, ord)
}
