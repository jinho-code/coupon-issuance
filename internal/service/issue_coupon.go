package service

import (
	"context"
	pb "coupon_issuance/gen/coupon/v1"
	"coupon_issuance/internal/datastore"
	"errors"
	"time"
)

func validateIssueCouponRequest(s *CouponService, ctx context.Context, req *pb.IssueCouponRequest) error {
	campaign, err := s.DataStore.GetCampaign(ctx, req.CampaignId)
	if err != nil {
		if errors.Is(err, datastore.ErrCampaignNotFound) {
			return errors.New("campaign not found")
		}
		return err // DataStoreUnavailable 또는 다른 내부 에러
	}
	if (req.UserId == "") {
		return errors.New("user_id is required")
	}
	now := time.Now()
	if now.Before(campaign.StartTime) {
		return errors.New("campaign not started yet")
	}
	if campaign.IssuedCouponsCount >= campaign.TotalCouponsCount {
		return errors.New("all coupons issued")
	}
	return nil
}

func (s *CouponService) IssueCoupon(ctx context.Context, req *pb.IssueCouponRequest) (*pb.IssueCouponResponse, error) {
	if err := validateIssueCouponRequest(s, ctx, req); err != nil {
		code := "INVALID_REQUEST"
		if errors.Is(err, datastore.ErrDataStoreUnavailable) {
			code = "DATASTORE_UNAVAILABLE"
		}
		return &pb.IssueCouponResponse{
			Error: &pb.Error{
				Code:    code,
				Message: err.Error(),
			},
		}, nil
	}
	
	code, err := s.CouponCodeGenerator.Generate(ctx)
	if err != nil {
		if errors.Is(err, datastore.ErrDataStoreUnavailable) {
			return &pb.IssueCouponResponse{
				Error: &pb.Error{
					Code:    "DATASTORE_UNAVAILABLE",
					Message: err.Error(),
				},
			}, nil
		}
		return &pb.IssueCouponResponse{
			Error: &pb.Error{
				Code:    "INTERNAL",
				Message: "failed to generate coupon code",
			},
		}, nil
	}
	coupon, err := s.DataStore.IssueCoupon(ctx, req.CampaignId, req.UserId, code)
	if err != nil {
		if errors.Is(err, datastore.ErrDataStoreUnavailable) {
			return &pb.IssueCouponResponse{
				Error: &pb.Error{
					Code:    "DATASTORE_UNAVAILABLE",
					Message: err.Error(),
				},
			}, nil
		}
		return &pb.IssueCouponResponse{
			Error: &pb.Error{
				Code:    "INTERNAL",
				Message: err.Error(),
			},
		}, nil
	}

	return &pb.IssueCouponResponse{
		Coupon: &pb.Coupon{
			Code:       coupon.Code,
			CampaignId: coupon.CampaignID,
			IssuedAt:   coupon.IssuedAt.Format(time.RFC3339),
			UserId:     coupon.UserID,
		},
	}, nil
} 