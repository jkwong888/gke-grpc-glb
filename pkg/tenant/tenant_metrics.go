package tenant

import (
	"context"

	"github.com/prometheus/client_golang/prometheus"
	"google.golang.org/grpc"
)

type TenantMetricsInterceptor interface {
	TenantMetricsUnaryInterceptor(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error)
	TenantMetricsStreamInterceptor(req interface{}, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error
}

type TenantMetrics struct {
	requests prometheus.CounterVec
	connections prometheus.GaugeVec
}

type monitoredServerStream struct {
	grpc.ServerStream
	metrics *TenantMetrics
}

func NewTenantMetrics() *TenantMetrics {
	val := &TenantMetrics{}

	if err := val.init(); err != nil {
		return nil
	}

	return val
}

func (metrics *TenantMetrics) init() (error) {
	metrics.requests = *prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "requests",
		},
		[]string{"tenantId"},
	)

	if err := prometheus.Register(metrics.requests); err != nil {
		return err
	}

	metrics.connections = *prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "open_connections",
		},
		[]string{"tenantId"},
	)

	if err := prometheus.Register(metrics.connections); err != nil {
		return err
	}

	return nil
}

func (metrics *TenantMetrics) incRequests(tenantId string) {
	metrics.requests.WithLabelValues(tenantId).Inc()
}

func (metrics *TenantMetrics) incConnections(tenantId string) {
	metrics.connections.WithLabelValues(tenantId).Inc()
}

func (metrics *TenantMetrics) decConnections(tenantId string) {
	metrics.connections.WithLabelValues(tenantId).Dec()
}


func (metrics *TenantMetrics) TenantMetricsUnaryInterceptor(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
	tenantId, err := GetTenantId(ctx)

	if err != nil {
		return nil, err
	}

	metrics.incConnections(tenantId)
	metrics.incRequests(tenantId)

	resp, err := handler(ctx, req)

	metrics.decConnections(tenantId)

	return resp, err

}

func (metrics *TenantMetrics) TenantMetricsStreamInterceptor(req interface{}, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
	tenantId, err := GetTenantId(ss.Context())

	if err != nil {
		return err
	}

	metrics.incConnections(tenantId)

	// wrap the server stream so we can monitor individual requests inside the stream
	monitoredStream := &monitoredServerStream{
		ss, 
		metrics,
	}

	err = handler(req, monitoredStream)

	metrics.decConnections(tenantId)

	return err
}

func (stream *monitoredServerStream) SendMsg(m interface{}) error {
	return stream.ServerStream.SendMsg(m)
}

func (stream *monitoredServerStream) RecvMsg(m interface{}) error {
	tenantId, err := GetTenantId(stream.Context())
	if err != nil {
		return err
	}

	err = stream.ServerStream.RecvMsg(m)
	stream.metrics.incRequests(tenantId)

	return err
}

