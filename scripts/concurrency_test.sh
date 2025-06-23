#!/bin/bash
set -e # Exit immediately if a command exits with a non-zero status.

# --- Test Configuration ---
PORTS=(8080 8081 8082 8083 8084)
TOTAL_COUPONS=100
REQUESTS=200
BASE_URL="http://localhost"
# -------------------------

# --- Cleanup function to kill all server processes ---
cleanup() {
    echo "Cleaning up server processes..."
    if [ ${#PIDS[@]} -ne 0 ]; then
        kill ${PIDS[@]}
    fi
}
# 'trap' ensures that cleanup is called on script exit (normal or error)
trap cleanup EXIT

# --- Step 1: Start servers in the background ---
echo "Step 1: Starting ${#PORTS[@]} server instances in the background..."
PIDS=()
for PORT in "${PORTS[@]}"; do
    # Run the server and store its PID
    PORT=$PORT go run cmd/server/main.go &
    PIDS+=($!)
done

# Give servers a moment to start up
echo "Waiting for servers to be ready... (PIDS: ${PIDS[@]})"
sleep 3

# --- Step 2: Create a new campaign ---
echo "Step 2: Creating a new campaign..."
CREATE_CAMPAIGN_URL_PATH="/coupon_issuance.v1.CouponService/CreateCampaign"
if [[ "$(uname)" == "Darwin" ]]; then
    START_TIME=$(date -v+1S -u +"%Y-%m-%dT%H:%M:%SZ")
else
    START_TIME=$(date -d '+1 second' -u +"%Y-%m-%dT%H:%M:%SZ")
fi
CREATE_RESPONSE=$(curl -s -X POST -H "Content-Type: application/json" \
  -d "{\"name\": \"Concurrency Test Campaign\", \"total_coupons_count\": $TOTAL_COUPONS, \"start_time\": \"$START_TIME\"}" \
  $BASE_URL:${PORTS[0]}$CREATE_CAMPAIGN_URL_PATH)
CAMPAIGN_ID=$(echo $CREATE_RESPONSE | jq -r '.campaign.id')
if [ -z "$CAMPAIGN_ID" ] || [ "$CAMPAIGN_ID" == "null" ]; then
  echo "❌ Error: Failed to create campaign. Response:"
  echo $CREATE_RESPONSE
  exit 1
fi
echo "Campaign created successfully. ID: $CAMPAIGN_ID"
sleep 2

# --- Step 3: Run concurrency test ---
echo "Step 3: Starting concurrency test with $REQUESTS requests..."
ISSUE_COUPON_URL_PATH="/coupon_issuance.v1.CouponService/IssueCoupon"
rm -f results.txt
for i in $(seq 1 $REQUESTS); do
  PORT=${PORTS[$((RANDOM % ${#PORTS[@]}))]}
  (
    curl -s --max-time 5 -X POST -H "Content-Type: application/json" \
      -d "{\"campaign_id\": \"$CAMPAIGN_ID\", \"user_id\": \"user_$i\"}" \
      $BASE_URL:$PORT$ISSUE_COUPON_URL_PATH
    echo
  ) >> results.txt &
  wait
done

echo "All requests sent."

# --- Step 4: Analyze results ---
echo "Step 4: Analyzing results..."
SUCCESS_COUNT=$(grep -c '"coupon":{' results.txt || true)
echo "Total coupons issued (Success responses): $SUCCESS_COUNT / $TOTAL_COUPONS"
ISSUED_CODES=$(grep '"coupon":{' results.txt | jq -r '.coupon.code' || true)
ERROR_COUNT=$(grep -c '"error":' results.txt || true)
echo "Total errors: $ERROR_COUNT"
if [[ -z "$ISSUED_CODES" ]]; then
    TOTAL_ISSUED_CODES_COUNT=0
    UNIQUE_CODES_COUNT=0
else
    TOTAL_ISSUED_CODES_COUNT=$(echo "$ISSUED_CODES" | wc -l | xargs)
    UNIQUE_CODES_COUNT=$(echo "$ISSUED_CODES" | sort | uniq | wc -l | xargs)
fi
echo "Total coupon codes parsed: $TOTAL_ISSUED_CODES_COUNT | Unique coupon codes: $UNIQUE_CODES_COUNT"

if [ "$TOTAL_ISSUED_CODES_COUNT" -eq "$UNIQUE_CODES_COUNT" ] && [ "$SUCCESS_COUNT" -eq "$TOTAL_COUPONS" ]; then
    echo "✅ Test Passed: No duplicate codes and correct number of coupons issued."
else
    echo "❌ Test Failed: Duplicate codes found or incorrect number of coupons issued."
fi 

if [ "$ERROR_COUNT" -ne "$((REQUESTS - TOTAL_COUPONS))" ]; then
    echo "❌ Test Failed: Error count is not equal to the number of requests. Expected: $((REQUESTS - TOTAL_COUPONS)), Actual: $ERROR_COUNT"
    exit 1
else 
    echo "✅ Test Passed: Error count is equal to the number of requests."
fi 