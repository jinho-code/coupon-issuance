package service

import (
	"context"
	"errors"
	"time"

	pb "coupon_issuance/gen/coupon/v1"
	"coupon_issuance/internal/model"

	"coupon_issuance/internal/datastore"

	"github.com/google/uuid"
)

func (s *CouponService) CreateCampaign(ctx context.Context, req *pb.CreateCampaignRequest) (*pb.CreateCampaignResponse, error) {
	if req.Name == "" || req.TotalCouponsCount <= 0 || req.StartTime == "" {
		return &pb.CreateCampaignResponse{
			Error: &pb.Error{
				Code:    "INVALID_ARGUMENT",
				Message: "name, total_coupons_count, start_time are required",
			},
		}, nil
	}
	startTime, err := time.Parse(time.RFC3339, req.StartTime)
	if err != nil {
		return &pb.CreateCampaignResponse{
			Error: &pb.Error{
				Code:    "INVALID_ARGUMENT",
				Message: "invalid start_time format",
			},
		}, nil
	}
	if startTime.Before(time.Now()) {
		return &pb.CreateCampaignResponse{
			Error: &pb.Error{
				Code:    "INVALID_ARGUMENT",
				Message: "start_time cannot be in the past",
			},
		}, nil
	}
	id := uuid.NewString()
	campaign := &model.Campaign{
		ID:                 id,
		Name:               req.Name,
		Description:        req.Description,
		TotalCouponsCount:  int(req.TotalCouponsCount),
		IssuedCouponsCount: 0,
		StartTime:          startTime,
		Status:             1, // Pending
		IssuedCodes:        []string{},
	}
	if err := s.DataStore.CreateCampaign(ctx, campaign); err != nil {
		if errors.Is(err, datastore.ErrDataStoreUnavailable) {
			return &pb.CreateCampaignResponse{
				Error: &pb.Error{
					Code:    "DATASTORE_UNAVAILABLE",
					Message: err.Error(),
				},
			}, nil
		}
		return &pb.CreateCampaignResponse{
			Error: &pb.Error{
				Code:    "INTERNAL",
				Message: err.Error(),
			},
		}, nil
	}
	return &pb.CreateCampaignResponse{
		Campaign: toProtoCampaign(campaign),
	}, nil
} 