package service

import (
	"context"
)

type OrderService struct {
	// GetOrders
	GetOrders func(ctx context.Context, req []interface{}) error
}

func (OrderService) Reference() string {
	return "orderService"
}
