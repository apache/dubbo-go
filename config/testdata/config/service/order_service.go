package service

import (
	"context"
)

type OrderService struct {
	// GetOrders
	GetOrders func(ctx context.Context, req []interface{}) error
}

func (OrderService) Name() string {
	return "orderService"
}

func (OrderService) Reference() string {
	return "org.github.dubbo.OrderService"
}
