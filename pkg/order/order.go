package order

import (
	"context"

	npool "github.com/NpoolPlatform/message/npool/order/gw/v1/order"
)

func CreateOrder(ctx context.Context, op *OrderCreate) (info *npool.Order, err error) {
	if err := op.ValidateInit(ctx); err != nil {
		return nil, err
	}

	if err := op.SetReduction(ctx); err != nil {
		return nil, err
	}

	if err := op.SetPrice(ctx); err != nil {
		return nil, err
	}

	if err := op.SetCurrency(ctx); err != nil {
		return nil, err
	}

	if err := op.SetPaymentAmount(ctx); err != nil {
		return nil, err
	}

	if err := op.ValidateBalance(ctx); err != nil {
		return nil, err
	}

	if err := op.PeekAddress(ctx); err != nil {
		return nil, err
	}

	if err := op.SetBalance(ctx); err != nil {
		_ = op.ReleaseAddress(ctx)
		return nil, err
	}

	if err := op.LockStock(ctx); err != nil {
		_ = op.ReleaseAddress(ctx)
		return nil, err
	}

	ord, err := op.Create(ctx)
	if err != nil {
		_ = op.ReleaseAddress(ctx)
		_ = op.ReleaseStock(ctx)
		return nil, err
	}

	return ord, nil
}
