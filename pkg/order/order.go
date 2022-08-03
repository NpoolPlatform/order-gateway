package order

import (
	"context"
	"fmt"

	npool "github.com/NpoolPlatform/message/npool/order/gw/v1/order"
)

func CreateOrder(ctx context.Context, op *OrderCreate) (info *npool.Order, err error) {
	if err := op.ValidateInit(ctx); err != nil {
		return nil, err
	}

	if err := op.SetReduction(ctx); err != nil {
		return nil, err
	}

	if err := op.SetCurrency(ctx); err != nil {
		return nil, err
	}

	if err := op.SetPaymentAmount(ctx); err != nil {
		return nil, err
	}

	if err := op.PeekAddress(ctx); err != nil {
		return nil, err
	}

	return &npool.Order{}, fmt.Errorf("NOT IMPLEMENTED")
}
