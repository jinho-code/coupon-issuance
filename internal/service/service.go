package service

import (
	"coupon_issuance/internal/coupon_code_gen"
	"coupon_issuance/internal/datastore"
)

type CouponService struct {
	DataStore           *datastore.RedisDataStore
	CouponCodeGenerator coupon_code_gen.CouponCodeGenerator
}

func NewCouponService(ds *datastore.RedisDataStore, cg coupon_code_gen.CouponCodeGenerator) *CouponService {
	return &CouponService{DataStore: ds, CouponCodeGenerator: cg}
} 