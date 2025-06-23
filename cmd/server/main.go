package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"connectrpc.com/connect"
	"github.com/getsentry/sentry-go"
	sentrylogrus "github.com/getsentry/sentry-go/logrus"
	"github.com/sirupsen/logrus"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"

	couponv1 "coupon_issuance/gen/coupon/v1"
	couponv1connect "coupon_issuance/gen/coupon/v1/couponv1connect"

	"coupon_issuance/internal/coupon_code_gen"
	"coupon_issuance/internal/datastore"
	"coupon_issuance/internal/service"
)

type CouponServiceHandler struct {
	svc *service.CouponService
}

func (h *CouponServiceHandler) CreateCampaign(ctx context.Context, req *connect.Request[couponv1.CreateCampaignRequest]) (*connect.Response[couponv1.CreateCampaignResponse], error) {
	resp, err := h.svc.CreateCampaign(ctx, req.Msg)
	return connect.NewResponse(resp), err
}

func (h *CouponServiceHandler) GetCampaign(ctx context.Context, req *connect.Request[couponv1.GetCampaignRequest]) (*connect.Response[couponv1.GetCampaignResponse], error) {
	resp, err := h.svc.GetCampaign(ctx, req.Msg)
	return connect.NewResponse(resp), err
}

func (h *CouponServiceHandler) IssueCoupon(ctx context.Context, req *connect.Request[couponv1.IssueCouponRequest]) (*connect.Response[couponv1.IssueCouponResponse], error) {
	resp, err := h.svc.IssueCoupon(ctx, req.Msg)
	return connect.NewResponse(resp), err
}

func setupLog() {
	logrus.SetFormatter(&logrus.JSONFormatter{})
	logrus.SetOutput(os.Stdout)
	logrus.SetLevel(logrus.InfoLevel)

	sentryDsn := os.Getenv("SENTRY_DSN")
	if sentryDsn != "" {
		hook, err := sentrylogrus.New([]logrus.Level{logrus.ErrorLevel, logrus.FatalLevel, logrus.PanicLevel}, sentry.ClientOptions{
			Dsn:              sentryDsn,
			TracesSampleRate: 1.0,
		})

		if err != nil {
			log.Fatalf("Sentry hook initialization failed: %v", err)
		}
		logrus.AddHook(hook)
		logrus.RegisterExitHandler(func() { sentry.Flush(5 * time.Second) })
	} else {
		logrus.Info("SENTRY_DSN is not set. Sentry hook will not be added.")
	}
}

func setupServer() *http.ServeMux {
	redisAddr := os.Getenv("REDIS_ADDR")
	if redisAddr == "" {
		redisAddr = "localhost:6379"
	}
	ds := datastore.NewRedisDataStore(redisAddr)
	codeGen := coupon_code_gen.NewRandomCouponCodeGenerator(ds)
	svc := service.NewCouponService(ds, codeGen)
	handler := &CouponServiceHandler{svc: svc}

	mux := http.NewServeMux()
	path, hnd := couponv1connect.NewCouponServiceHandler(handler)
	mux.Handle(path, hnd)

	return mux
}

func main() {
	setupLog()

	mux := setupServer()

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	addr := fmt.Sprintf(":%s", port)

	redisAddr := os.Getenv("REDIS_ADDR")
	if redisAddr == "" {
		redisAddr = "localhost:6379"
	}

	logrus.Infof("Server listening on %s", addr)
	http.ListenAndServe(
		addr,
		h2c.NewHandler(mux, &http2.Server{}),
	)
}