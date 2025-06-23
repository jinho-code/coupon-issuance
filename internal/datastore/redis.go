package datastore

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"coupon_issuance/internal/model"

	"github.com/redis/go-redis/v9"
)

var (
	ErrCampaignNotFound = errors.New("campaign not found")
	ErrDataStoreUnavailable = errors.New("datastore unavailable")
)

type RedisDataStore struct {
	client *redis.Client
}

func NewRedisDataStore(addr string) *RedisDataStore {
	client := redis.NewClient(&redis.Options{
		Addr: addr,
	})
	return &RedisDataStore{client: client}
}

func toCampaignKey(id string) string {
	return fmt.Sprintf("campaign:%s", id)
}

func toCouponIssuedCountKey(campaignID string) string {
	return fmt.Sprintf("campaign:%s:issued_count", campaignID)
}

func toCouponIssuedCodesKey(campaignID string) string {
	return fmt.Sprintf("campaign:%s:issued_codes", campaignID)
}

func (r *RedisDataStore) CreateCampaign(ctx context.Context, c *model.Campaign) error {
	key := toCampaignKey(c.ID)
	issuedKey := toCouponIssuedCountKey(c.ID)
	if err := r.client.Set(ctx, issuedKey, 0, 0).Err(); err != nil {
		return fmt.Errorf("%w: %v", ErrDataStoreUnavailable, err)
	}

	data, err := json.Marshal(c)
	if err != nil {
		return err
	}
	if err := r.client.Set(ctx, key, data, 0).Err(); err != nil {
		return fmt.Errorf("%w: %v", ErrDataStoreUnavailable, err)
	}
	return nil
}

func (r *RedisDataStore) GetCampaign(ctx context.Context, id string) (*model.Campaign, error) {
	key := toCampaignKey(id)
	data, err := r.client.Get(ctx, key).Bytes()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return nil, ErrCampaignNotFound
		}
		return nil, fmt.Errorf("%w: %v", ErrDataStoreUnavailable, err)
	}
	var c model.Campaign
	if err := json.Unmarshal(data, &c); err != nil {
		return nil, err
	}
	
	issuedKey := toCouponIssuedCountKey(id)
	issuedCount, err := r.client.Get(ctx, issuedKey).Int()
	if err != nil && !errors.Is(err, redis.Nil) {
		return nil, fmt.Errorf("%w: %v", ErrDataStoreUnavailable, err)
	}
	c.IssuedCouponsCount = issuedCount

	issuedCodesKey := toCouponIssuedCodesKey(id)
	issuedCodes, err := r.client.SMembers(ctx, issuedCodesKey).Result()
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrDataStoreUnavailable, err)
	}
	c.IssuedCodes = issuedCodes

	return &c, nil
}

// Lua script for atomic coupon issuance
const issueCouponScript = `
local campaign_key = KEYS[1]
local issued_key = KEYS[2]
local code_set = KEYS[3]
local coupon_key = KEYS[4]
local campaign_issued_codes_key = KEYS[5]
local total = tonumber(ARGV[1])
local code = ARGV[2]
local user_id = ARGV[3]
local now = ARGV[4]

local issued = tonumber(redis.call('GET', issued_key) or '0')
if issued >= total then
  return {err = 'all coupons issued'}
end
if redis.call('SISMEMBER', code_set, code) == 1 then
  return {err = 'coupon code already issued'}
end
issued = redis.call('INCR', issued_key)
if issued > total then
  redis.call('DECR', issued_key)
  return {err = 'all coupons issued'}
end
redis.call('SADD', code_set, code)
redis.call('SADD', campaign_issued_codes_key, code)
redis.call('SET', coupon_key, cjson.encode({code=code, campaign_id=campaign_key, issued_at=now, user_id=user_id}))
return issued
`

func (r *RedisDataStore) IssueCoupon(ctx context.Context, campaignID, userID, code string) (*model.Coupon, error) {
	campaign, err := r.GetCampaign(ctx, campaignID)
	if err != nil {
		return nil, err
	}
	
	issuedKey := toCouponIssuedCountKey(campaignID)
	codeSet := "coupon:codes"
	couponKey := fmt.Sprintf("coupon:%s", code)
	campaignKey := toCampaignKey(campaignID)
	campaignIssuedCodesKey := toCouponIssuedCodesKey(campaignID)

	now := time.Now().Format(time.RFC3339)
	res, err := r.client.Eval(ctx, issueCouponScript, []string{
		campaignKey, issuedKey, codeSet, couponKey, campaignIssuedCodesKey,
	}, campaign.TotalCouponsCount, code, userID, now).Result()
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrDataStoreUnavailable, err)
	}
	if _, ok := res.(int64); !ok {
		if resMap, ok := res.(map[interface{}]interface{}); ok {
			if errMsg, ok := resMap["err"].(string); ok {
				return nil, errors.New(errMsg)
			}
		}
		return nil, errors.New("failed to issue coupon")
	}
	coupon := &model.Coupon{
		Code:       code,
		CampaignID: campaignID,
		IssuedAt:   time.Now(),
		UserID:     userID,
	}
	return coupon, nil
}

func (r *RedisDataStore) CheckCouponCodeExists(ctx context.Context, code string) (bool, error) {
	codeSet := "coupon:codes"
	exists, err := r.client.SIsMember(ctx, codeSet, code).Result()
	if err != nil {
		return false, fmt.Errorf("%w: %v", ErrDataStoreUnavailable, err)
	}
	return exists, nil
} 