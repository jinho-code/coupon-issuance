package model

import "time"

type Campaign struct {
    ID                string
    Name              string
    Description       string
    TotalCouponsCount  int
    IssuedCouponsCount int
    StartTime         time.Time
    Status            int // enum 매핑
    IssuedCodes       []string
}

type Coupon struct {
    Code       string
    CampaignID string
    IssuedAt   time.Time
    UserID     string
} 