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
	"errors"
	"flag"
	"log"
	"net"
	"net/http"
	"os"
	"encoding/json"

	gcp "helloworld/pkg/gcp"
	pb "helloworld/proto/helloworld"

	"google.golang.org/grpc/codes"
	health "google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/status"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/peer"

	cmux "github.com/soheilhy/cmux"
)

const (
	port = ":50051"
)

// server is used to implement helloworld.GreeterServer.
type grpcServer struct {
	pb.GreeterServer
	health.HealthServer
}

func (s *grpcServer) Check(context.Context, *health.HealthCheckRequest) (*health.HealthCheckResponse, error) {
	return &health.HealthCheckResponse{Status: health.HealthCheckResponse_SERVING}, nil
}

func (s *grpcServer) Watch(*health.HealthCheckRequest, health.Health_WatchServer) error {
	return status.Error(codes.Unimplemented, "unimplemented")
}

// SayHello implements helloworld.GreeterServer
func (s *grpcServer) SayHello(ctx context.Context, in *pb.HelloRequest) (*pb.HelloReply, error) {
	p, _ := peer.FromContext(ctx)
	frontendip := p.Addr.String()
	log.Printf("Received request from %v: %v", frontendip, in.GetName())

	host, _ := os.Hostname()
	zoneStr := gcp.GetMetaData(ctx, "instance/zone")
	nodeName := gcp.GetMetaData(ctx, "instance/hostname")
	region := gcp.GetMetaData(ctx, "instance/attributes/cluster-location")
	clusterName := gcp.GetMetaData(ctx, "instance/attributes/cluster-name")
	project := gcp.GetMetaData(ctx, "project/project-id")

	result := &pb.HelloReply{
		Message:  "Hello " + in.GetName(),
		Version:  "v2.0.0",
		Hostname: host,
	}

	if zoneStr != nil {
		result.Zone = *zoneStr
	}

	if nodeName != nil {
		result.Nodename = *nodeName
	}

	if region != nil {
		result.Region = *region
	}

	if clusterName != nil {
		result.Clustername = *clusterName
	}

	if project != nil {
		result.Project = *project
	}

	return result, nil
}

type httpHealthCheckHandler struct {}

func (h *httpHealthCheckHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Header().Set("Content-Type", "application/json")

	w.Header().Set("Content-Type", "application/json")
	resp := make(map[string]string)
	resp["message"] = "Status OK"
	jsonResp, err := json.Marshal(resp)
	if err != nil {
		log.Fatalf("Error happened in JSON marshal. Err: %s", err)
	}
	w.Write(jsonResp)
}

func main() {

	tlsCrt := flag.String("crt", "certs/tls.crt", "TLS certificate")
	tlsKey := flag.String("key", "certs/tls.key", "TLS private key")
	tlsB := flag.Bool("tls", true, "enable TLS")
	flag.Parse()

	lis, err := net.Listen("tcp", port)
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	log.Printf("Listening on port: %v", port)

	/* check if grpc needs to listen on TLS */
	tls := *tlsB
	grpcOptions := make([]grpc.ServerOption, 0)
	if tls {
		if _, err := os.Stat(*tlsCrt); errors.Is(err, os.ErrNotExist) {
			tls = false
			log.Printf("Could not find %v", *tlsCrt)
		}

		if _, err := os.Stat(*tlsKey); errors.Is(err, os.ErrNotExist) {
			tls = false
			log.Printf("Could not find %v", *tlsKey)
		}
	}

	if tls {
		log.Printf("TLS enabled using cert: %v, private key: %v", *tlsCrt, *tlsKey)
		creds, err := credentials.NewServerTLSFromFile(*tlsCrt, *tlsKey)
		if err != nil {
			log.Fatalf("Failed to setup TLS: %v", err)
		}

		grpcOptions = append(grpcOptions, grpc.Creds(creds))
	}

	s := grpc.NewServer(grpcOptions...)

	/* register grpc services */
	g := &grpcServer{}

	pb.RegisterGreeterServer(s, g)
	health.RegisterHealthServer(s, g)

	/* register http services */
	h := &http.Server{}
	http.DefaultServeMux.Handle("/healthz", &httpHealthCheckHandler{})

	m := cmux.New(lis)

	// if http1.1 match, send to the http handler 
	httpL := m.Match(cmux.HTTP1Fast())

	// otherwise assume grpc
	grpcL := m.Match(cmux.Any())

	go s.Serve(grpcL)
	go h.Serve(httpL)

	if err := m.Serve(); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}
