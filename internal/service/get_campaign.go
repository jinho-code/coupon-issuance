package service

import (
	"context"
	pb "coupon_issuance/gen/coupon/v1"
	"coupon_issuance/internal/datastore"
	"errors"
)

func (s *CouponService) GetCampaign(ctx context.Context, req *pb.GetCampaignRequest) (*pb.GetCampaignResponse, error) {
	campaign, err := s.DataStore.GetCampaign(ctx, req.CampaignId)
	if err != nil {
		if errors.Is(err, datastore.ErrCampaignNotFound) {
			return &pb.GetCampaignResponse{
				Error: &pb.Error{
					Code:    "NOT_FOUND",
					Message: err.Error(),
				},
			}, nil
		}
		if errors.Is(err, datastore.ErrDataStoreUnavailable) {
			return &pb.GetCampaignResponse{
				Error: &pb.Error{
					Code:    "DATASTORE_UNAVAILABLE",
					Message: err.Error(),
				},
			}, nil
		}
		return &pb.GetCampaignResponse{
			Error: &pb.Error{
				Code:    "INTERNAL",
				Message: err.Error(),
			},
		}, nil
	}
	return &pb.GetCampaignResponse{
		Campaign: toProtoCampaign(campaign),
	}, nil
} 