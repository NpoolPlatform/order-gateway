package fee

import (
	"context"

	wlog "github.com/NpoolPlatform/go-service-framework/pkg/wlog"
	npool "github.com/NpoolPlatform/message/npool/order/gw/v1/fee"
	feeordermwpb "github.com/NpoolPlatform/message/npool/order/mw/v1/fee"
	paymentmwpb "github.com/NpoolPlatform/message/npool/order/mw/v1/payment"
	ordercommon "github.com/NpoolPlatform/order-gateway/pkg/order/common"
	feeordermwcli "github.com/NpoolPlatform/order-middleware/pkg/client/fee"
)

type baseUpdateHandler struct {
	*Handler
	*ordercommon.OrderOpHandler
	feeOrder    *npool.FeeOrder
	feeOrderReq *feeordermwpb.FeeOrderReq
}

func (h *baseUpdateHandler) getFeeOrder(ctx context.Context) (err error) {
	h.feeOrder, err = h.GetFeeOrder(ctx)
	return wlog.WrapError(err)
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
		UserSetCanceled:  h.UserSetCanceled,
		AdminSetCanceled: h.AdminSetCanceled,
	}
	req.PaymentBalances = h.PaymentBalanceReqs
	if h.PaymentTransferReq != nil {
		req.PaymentTransfers = []*paymentmwpb.PaymentTransferReq{h.PaymentTransferReq}
	}
	h.OrderIDs = append(h.OrderIDs, *req.OrderID)
	h.feeOrderReq = req
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
	return feeordermwcli.UpdateFeeOrder(ctx, h.feeOrderReq)
}
