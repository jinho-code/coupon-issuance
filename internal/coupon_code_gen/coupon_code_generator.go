package coupon_code_gen

import (
	"context"
)

type CouponCodeGenerator interface {
	Generate(ctx context.Context) (string, error)
} 