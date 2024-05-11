package powerrental

import (
	"context"

	wlog "github.com/NpoolPlatform/go-service-framework/pkg/wlog"
	npool "github.com/NpoolPlatform/message/npool/order/gw/v1/powerrental"
	paymentmwpb "github.com/NpoolPlatform/message/npool/order/mw/v1/payment"
	powerrentalordermwpb "github.com/NpoolPlatform/message/npool/order/mw/v1/powerrental"
	ordercommon "github.com/NpoolPlatform/order-gateway/pkg/order/common"
	powerrentalordermwcli "github.com/NpoolPlatform/order-middleware/pkg/client/powerrental"
)

type baseUpdateHandler struct {
	*Handler
	*ordercommon.OrderOpHandler
	powerRentalOrder    *npool.PowerRentalOrder
	powerRentalOrderReq *powerrentalordermwpb.PowerRentalOrderReq
}

func (h *baseUpdateHandler) getPowerRentalOrder(ctx context.Context) (err error) {
	h.powerRentalOrder, err = h.GetPowerRentalOrder(ctx)
	return wlog.WrapError(err)
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

func (h *baseUpdateHandler) formalizePayment() {
	h.powerRentalOrderReq.PaymentType = &h.PaymentType
	h.powerRentalOrderReq.PaymentBalances = h.PaymentBalanceReqs
	if h.PaymentTransferReq != nil {
		h.powerRentalOrderReq.PaymentTransfers = []*paymentmwpb.PaymentTransferReq{h.PaymentTransferReq}
	}
	h.powerRentalOrderReq.LedgerLockID = h.BalanceLockID
}

func (h *baseUpdateHandler) updatePowerRentalOrder(ctx context.Context) error {
	return powerrentalordermwcli.UpdatePowerRentalOrder(ctx, h.powerRentalOrderReq)
}
