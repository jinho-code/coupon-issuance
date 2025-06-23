package test

import (
	"context"
	"coupon_issuance/internal/coupon_code_gen"
	"coupon_issuance/internal/datastore"
	"testing"
)

func TestRandomCodeGenerator_UniquenessAndFormat(t *testing.T) {
	ds := datastore.NewRedisDataStore("localhost:6379")
	ctx := context.Background()
	gen := coupon_code_gen.NewRandomCouponCodeGenerator(ds)

	codes := make(map[string]struct{})
	for i := 0; i < 1000; i++ {
		code, err := gen.Generate(ctx)
		if err != nil {
			t.Fatalf("failed to generate code: %v", err)
		}
		if len([]rune(code)) != 10 {
			t.Errorf("code length is not 10: %s", code)
		}
		if _, exists := codes[code]; exists {
			t.Errorf("duplicate code generated: %s", code)
		}
		codes[code] = struct{}{}
	}

	// Clean up test codes from Redis
	for code := range codes {
		if err := ds.DeleteCouponCode(ctx, code); err != nil {
			t.Logf("cleanup failed for code %s: %v", code, err)
		}
	}
} 