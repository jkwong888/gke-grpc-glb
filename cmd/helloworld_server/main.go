/*
 *
 * Copyright 2015 gRPC authors.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 *
 */

//go:generate protoc -I ../helloworld --go_out=plugins=grpc:../helloworld ../helloworld/helloworld.proto

// Package main implements a server for Greeter service.
package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"log"
	"net"
	"net/http"
	"os"

	http_health "helloworld/pkg/healthcheck"
	tenant "helloworld/pkg/tenant"
	pb "helloworld/proto/helloworld"
	helloServer "helloworld/pkg/helloServer"

	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	grpc_health "google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/status"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"

	grpc_middleware "github.com/grpc-ecosystem/go-grpc-middleware"
	grpc_zap "github.com/grpc-ecosystem/go-grpc-middleware/logging/zap"
	grpc_ctxtags "github.com/grpc-ecosystem/go-grpc-middleware/tags"
	grpc_prometheus "github.com/grpc-ecosystem/go-grpc-prometheus"
	grpc_recovery "github.com/grpc-ecosystem/go-grpc-middleware/recovery"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	cmux "github.com/soheilhy/cmux"
)

const (
	port = ":50051"
)

// server is used to implement helloworld.GreeterServer.
type grpcServer struct {
	helloServer.HelloServer
	grpc_health.HealthServer
}

func (s *grpcServer) Check(context.Context, *grpc_health.HealthCheckRequest) (*grpc_health.HealthCheckResponse, error) {
	return &grpc_health.HealthCheckResponse{Status: grpc_health.HealthCheckResponse_SERVING}, nil
}

func (s *grpcServer) Watch(*grpc_health.HealthCheckRequest, grpc_health.Health_WatchServer) error {
	return status.Error(codes.Unimplemented, "unimplemented")
}


func main() {
	opts := []grpc_zap.Option{}

	zapLogger, err := zap.NewProduction()
	if err != nil {
		log.Fatalf("can't initialize zap logger: %v", err)
	}
	grpc_zap.ReplaceGrpcLoggerV2(zapLogger)
	defer zapLogger.Sync()

	tlsCrt := flag.String("crt", "certs/tls.crt", "TLS certificate")
	tlsKey := flag.String("key", "certs/tls.key", "TLS private key")
	tlsB := flag.Bool("tls", true, "enable TLS")
	configDir := flag.String("config-dir", "config/", "config directory")
	flag.Parse()

	lis, err := net.Listen("tcp", port)
	if err != nil {
		zapLogger.Fatal("failed to listen", 
			zap.String("error", err.Error()), 
			zap.String("address", port),
		)
	}

	zapLogger.Info("Listening on address", zap.String("address", port))

	/* check if grpc needs to listen on TLS */
	tls := *tlsB
	grpcOptions := make([]grpc.ServerOption, 0)
	if tls {
		if _, err := os.Stat(*tlsCrt); errors.Is(err, os.ErrNotExist) {
			tls = false
			zapLogger.Info("Could not find cert", zap.String("cert", *tlsCrt))
		}

		if _, err := os.Stat(*tlsKey); errors.Is(err, os.ErrNotExist) {
			tls = false
			zapLogger.Info("Could not find key", zap.String("key",*tlsKey))
		}
	}

	if tls {
		zapLogger.Info("TLS enabled", 
			zap.String("cert", *tlsCrt), 
			zap.String("key", *tlsKey),
		)
		creds, err := credentials.NewServerTLSFromFile(*tlsCrt, *tlsKey)
		if err != nil {
			zapLogger.Fatal("Failed to setup TLS",
				zap.String("cert", *tlsCrt), 
				zap.String("key", *tlsKey),
				zap.Error(err))
		}

		grpcOptions = append(grpcOptions, grpc.Creds(creds))
	}

	// initialize tenant metrics
	tenantMetrics := tenant.NewTenantMetrics()

	// add interceptors
	grpcOptions = append (grpcOptions, 
		grpc_middleware.WithUnaryServerChain(
			tenantMetrics.TenantMetricsUnaryInterceptor,
			grpc_prometheus.UnaryServerInterceptor,
			grpc_ctxtags.UnaryServerInterceptor(grpc_ctxtags.WithFieldExtractor(grpc_ctxtags.CodeGenRequestFieldExtractor)),
			grpc_zap.UnaryServerInterceptor(zapLogger, opts...),
			grpc_recovery.UnaryServerInterceptor(),
		),
		grpc_middleware.WithStreamServerChain(
			tenantMetrics.TenantMetricsStreamInterceptor,
			grpc_prometheus.StreamServerInterceptor,
			grpc_ctxtags.StreamServerInterceptor(grpc_ctxtags.WithFieldExtractor(grpc_ctxtags.CodeGenRequestFieldExtractor)),
			grpc_zap.StreamServerInterceptor(zapLogger, opts...),
			grpc_recovery.StreamServerInterceptor(),
		),
	)

	s := grpc.NewServer(grpcOptions...)

	/* get the tenant config */
	t, err := tenant.LoadTenantConfig(*configDir)
	if err != nil {
		zapLogger.Warn("Error loading tenant config", zap.Error(err))
	}
	tenantConfigJSON, _ := json.Marshal(t)
	zapLogger.Info("Loaded Tenant Config", 
		zap.String("tenantConfigJson", string(tenantConfigJSON)),
	)

	/* register grpc services */
	g := &grpcServer{
		HelloServer: *helloServer.NewHelloServer(*t),
	}

	pb.RegisterGreeterServer(s, g)
	grpc_health.RegisterHealthServer(s, g)

	/* reset all prometheus to zero */
	grpc_prometheus.Register(s)

	/* register http services */
	h := &http.Server{}
	http.DefaultServeMux.Handle("/healthz", &http_health.HttpHealthCheckHandler{})

	// Register Prometheus metrics handler.    
	http.Handle("/metrics", promhttp.Handler())

	m := cmux.New(lis)

	// if http1.1 match, send to the http handler 
	httpL := m.Match(cmux.HTTP1Fast())

	// otherwise assume grpc
	grpcL := m.Match(cmux.Any())

	go s.Serve(grpcL)
	go h.Serve(httpL)

	if err := m.Serve(); err != nil {
		zapLogger.Fatal("failed to serve", 
			zap.Error(err),
		)
	}
}
