package service

import (
	"time"

	pb "coupon_issuance/gen/coupon/v1"
	"coupon_issuance/internal/model"
)

func toProtoCampaign(c *model.Campaign) *pb.Campaign {
	status := pb.CampaignStatus_CAMPAIGN_STATUS_PENDING
	now := time.Now()
	if now.After(c.StartTime) && c.IssuedCouponsCount < c.TotalCouponsCount {
		status = pb.CampaignStatus_CAMPAIGN_STATUS_ACTIVE
	}
	if c.IssuedCouponsCount >= c.TotalCouponsCount {
		status = pb.CampaignStatus_CAMPAIGN_STATUS_ENDED
	}
	return &pb.Campaign{
		Id:                 c.ID,
		Name:               c.Name,
		Description:        c.Description,
		TotalCouponsCount:  int32(c.TotalCouponsCount),
		IssuedCouponsCount: int32(c.IssuedCouponsCount),
		StartTime:          c.StartTime.Format(time.RFC3339),
		Status:             status,
		IssuedCouponCodes:  c.IssuedCodes,
	}
} 