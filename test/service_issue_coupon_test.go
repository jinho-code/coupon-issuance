package test

import (
	"context"
	"testing"
	"time"

	pb "coupon_issuance/gen/coupon/v1"
	"coupon_issuance/internal/coupon_code_gen"
	"coupon_issuance/internal/datastore"
	"coupon_issuance/internal/service"
)

func TestService_IssueCoupon(t *testing.T) {
	ds := datastore.NewRedisDataStore("localhost:6379")
	cg := coupon_code_gen.NewRandomCouponCodeGenerator(ds)
	svc := service.NewCouponService(ds, cg)
	ctx := context.Background()

	startTime := time.Now().Add(2 * time.Second).UTC().Format(time.RFC3339)
	createReq := &pb.CreateCampaignRequest{
		Name:              "test-campaign",
		Description:       "test",
		TotalCouponsCount: 2,
		StartTime:         startTime,
	}
	createResp, err := svc.CreateCampaign(ctx, createReq)
	if err != nil || createResp.Error != nil {
		t.Fatalf("failed to create campaign: %v, %v", err, createResp.Error)
	}
	campaignID := createResp.Campaign.Id

	// 1. Issue before start time (should fail)
	issueReq := &pb.IssueCouponRequest{CampaignId: campaignID, UserId: "user1"}
	resp, err := svc.IssueCoupon(ctx, issueReq)
	if err != nil || resp.Error == nil || resp.Error.Message != "campaign not started yet" {
		t.Errorf("expected campaign not started error, got: %v, %v", err, resp.Error)
	}

	t.Log("waiting for campaign to start...")
	time.Sleep(3 * time.Second)

	// 2. Issue after start time (should succeed)
	resp, err = svc.IssueCoupon(ctx, issueReq)
	if err != nil || resp.Error != nil {
		t.Fatalf("expected success, got: %v, %v", err, resp.Error)
	}
	if resp.Coupon == nil || resp.Coupon.UserId != "user1" {
		t.Errorf("invalid coupon response: %+v", resp.Coupon)
	}

	// 3. Issue second coupon (should succeed)
	issueReq2 := &pb.IssueCouponRequest{CampaignId: campaignID, UserId: "user2"}
	resp2, err := svc.IssueCoupon(ctx, issueReq2)
	if err != nil || resp2.Error != nil {
		t.Fatalf("expected success, got: %v, %v", err, resp2.Error)
	}

	// 4. Issue third coupon (should fail: all coupons issued)
	issueReq3 := &pb.IssueCouponRequest{CampaignId: campaignID, UserId: "user3"}
	resp3, err := svc.IssueCoupon(ctx, issueReq3)
	if err != nil || resp3.Error == nil || resp3.Error.Message != "all coupons issued" {
		t.Errorf("expected all coupons issued error, got: %v, %v", err, resp3.Error)
	}

	// Clean up: remove issued codes from Redis global set
	for _, c := range []string{resp.Coupon.Code, resp2.Coupon.Code} {
		if err := ds.DeleteCouponCode(ctx, c); err != nil {
			t.Logf("cleanup failed for code %s: %v", c, err)
		}
	}
} 