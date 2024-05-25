package fee

import (
	"context"

	wlog "github.com/NpoolPlatform/go-service-framework/pkg/wlog"
	appfeemwcli "github.com/NpoolPlatform/good-middleware/pkg/client/app/fee"
	appfeemwpb "github.com/NpoolPlatform/message/npool/good/mw/v1/app/fee"
	npool "github.com/NpoolPlatform/message/npool/order/gw/v1/fee"
	feeordermwpb "github.com/NpoolPlatform/message/npool/order/mw/v1/fee"
	paymentmwpb "github.com/NpoolPlatform/message/npool/order/mw/v1/payment"
	ordercommon "github.com/NpoolPlatform/order-gateway/pkg/order/common"
	ordermwsvcname "github.com/NpoolPlatform/order-middleware/pkg/servicename"

	dtmcli "github.com/NpoolPlatform/dtm-cluster/pkg/dtm"
	"github.com/dtm-labs/dtm/client/dtmcli/dtmimp"
)

type baseUpdateHandler struct {
	*Handler
	*ordercommon.DtmHandler
	feeOrder    *npool.FeeOrder
	feeOrderReq *feeordermwpb.FeeOrderReq
	appFee      *appfeemwpb.Fee
}

func (h *baseUpdateHandler) getFeeOrder(ctx context.Context) (err error) {
	h.feeOrder, err = h.GetFeeOrder(ctx)
	return wlog.WrapError(err)
}

func (h *baseUpdateHandler) getAppFee(ctx context.Context) (err error) {
	h.appFee, err = appfeemwcli.GetFee(ctx, h.feeOrder.AppGoodID)
	return wlog.WrapError(err)
}

func (h *baseUpdateHandler) validateCancelParam() error {
	if err := h.ValidateCancelParam(); err != nil {
		return wlog.WrapError(err)
	}
	if h.feeOrder.AdminSetCanceled || h.feeOrder.UserSetCanceled {
		return wlog.Errorf("permission denied")
	}
	return nil
}

func (h *baseUpdateHandler) constructFeeOrderReq() {
	req := &feeordermwpb.FeeOrderReq{
		ID:               &h.feeOrder.ID,
		EntID:            &h.feeOrder.EntID,
		OrderID:          &h.feeOrder.OrderID,
		PaymentType:      &h.PaymentType,
		LedgerLockID:     h.BalanceLockID,
		PaymentID:        h.PaymentID,
		UserSetPaid:      h.UserSetPaid,
		UserSetCanceled:  h.Handler.UserSetCanceled,
		AdminSetCanceled: h.Handler.AdminSetCanceled,
	}
	req.PaymentBalances = h.PaymentBalanceReqs
	if h.PaymentTransferReq != nil {
		req.PaymentTransfers = []*paymentmwpb.PaymentTransferReq{h.PaymentTransferReq}
	}
	h.OrderIDs = append(h.OrderIDs, *req.OrderID)
	h.feeOrderReq = req
}

func (h *baseUpdateHandler) withUpdateFeeOrder(dispose *dtmcli.SagaDispose) {
	dispose.Add(
		ordermwsvcname.ServiceDomain,
		"order.middleware.fee.v1.Middleware/UpdateFeeOrder",
		"",
		&feeordermwpb.UpdateFeeOrderRequest{
			Info: h.feeOrderReq,
		},
	)
}

func (h *baseUpdateHandler) formalizePayment() {
	h.feeOrderReq.PaymentType = &h.PaymentType
	h.feeOrderReq.PaymentBalances = h.PaymentBalanceReqs
	if h.PaymentTransferReq != nil {
		h.feeOrderReq.PaymentTransfers = []*paymentmwpb.PaymentTransferReq{h.PaymentTransferReq}
	}
	h.feeOrderReq.LedgerLockID = h.BalanceLockID
}

func (h *baseUpdateHandler) updateFeeOrder(ctx context.Context) error {
	sagaDispose := dtmcli.NewSagaDispose(dtmimp.TransOptions{
		WaitResult:     true,
		RequestTimeout: 10,
		TimeoutToFail:  10,
	})

	if len(h.CommissionLockIDs) > 0 {
		h.WithCreateOrderCommissionLocks(sagaDispose)
		h.WithLockCommissions(sagaDispose)
	}
	h.withUpdateFeeOrder(sagaDispose)
	return h.DtmDo(ctx, sagaDispose)
}
