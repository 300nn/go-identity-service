package grpcapi

import (
	"context"
	"time"

	"github.com/300nn/go-identity-service/internal/metrics"

	"google.golang.org/grpc"
	"google.golang.org/grpc/status"
)

type MetricsInterceptor struct {
	metrics *metrics.Metrics
}

func NewMetricsInterceptor(metrics *metrics.Metrics) *MetricsInterceptor {
	return &MetricsInterceptor{
		metrics: metrics,
	}
}

func (i *MetricsInterceptor) Unary() grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req any,
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (any, error) {
		start := time.Now()

		resp, err := handler(ctx, req)

		code := status.Code(err).String()
		duration := time.Since(start).Seconds()

		i.metrics.GRPCRequestsTotal.WithLabelValues(info.FullMethod, code).Inc()
		i.metrics.GRPCRequestDuration.WithLabelValues(info.FullMethod, code).Observe(duration)

		return resp, err
	}
}
