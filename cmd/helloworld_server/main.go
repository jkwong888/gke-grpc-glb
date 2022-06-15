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
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"os"

	gcp "helloworld/pkg/gcp"
	http_health "helloworld/pkg/healthcheck"
	tenant "helloworld/pkg/tenant"
	pb "helloworld/proto/helloworld"

	"google.golang.org/grpc/codes"
	grpc_health "google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/peer"

	cmux "github.com/soheilhy/cmux"
)

const (
	port = ":50051"
	defaultVersion = "v1.0.0"
)

// server is used to implement helloworld.GreeterServer.
type grpcServer struct {
	pb.GreeterServer
	grpc_health.HealthServer

	serverTenantConfig tenant.TenantConfig
}

func (s *grpcServer) Check(context.Context, *grpc_health.HealthCheckRequest) (*grpc_health.HealthCheckResponse, error) {
	return &grpc_health.HealthCheckResponse{Status: grpc_health.HealthCheckResponse_SERVING}, nil
}

func (s *grpcServer) Watch(*grpc_health.HealthCheckRequest, grpc_health.Health_WatchServer) error {
	return status.Error(codes.Unimplemented, "unimplemented")
}

func getHelloReply(ctx context.Context, in *pb.HelloRequest, clientTargetTenantId string) (*pb.HelloReply, error) {
	host, _ := os.Hostname()
	zoneStr := gcp.GetMetaData(ctx, "instance/zone")
	nodeName := gcp.GetMetaData(ctx, "instance/hostname")
	region := gcp.GetMetaData(ctx, "instance/attributes/cluster-location")
	clusterName := gcp.GetMetaData(ctx, "instance/attributes/cluster-name")
	project := gcp.GetMetaData(ctx, "project/project-id")

	version, err := ioutil.ReadFile("version.txt")
	if err != nil {
		version = []byte(defaultVersion)
		log.Printf("<%v> unable to open version file version.txt, using default version %s", clientTargetTenantId, version)
	}

	result := &pb.HelloReply{
		Message:  "Hello " + in.GetName(),
		Version:  string(version),
		Hostname: host,
		TenantId: clientTargetTenantId,
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

func (s *grpcServer) validateTenantId(md metadata.MD) (string, error) {
	if md.Get("X-Tenant-Id") == nil {
		return "", status.Error(codes.InvalidArgument, "Missing X-Tenant-Id header")
	}

	clientTargetTenantId := md.Get("X-Tenant-Id")[0]
	/* check if this tenant is allowed */
	if !s.serverTenantConfig.CheckTenantId(clientTargetTenantId) {
		return "", status.Error(codes.InvalidArgument, "Wrong Tenant-Id for instance")
	}

	return clientTargetTenantId, nil
}

// SayHello implements helloworld.GreeterServer
func (s *grpcServer) SayHello(ctx context.Context, in *pb.HelloRequest) (*pb.HelloReply, error) {
	p, _ := peer.FromContext(ctx)
	frontendip := p.Addr.String()

	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return nil, status.Error(codes.InvalidArgument, "Unable to retrieve request metadata")
	}

	clientTargetTenantId, err := s.validateTenantId(md)
	if err != nil {
		return nil, err
	}

	log.Printf("[%v] <%v> Received request: %v", frontendip, clientTargetTenantId, in.GetName())

	return getHelloReply(ctx, in, clientTargetTenantId)

}

/* streaming hello ... client sends hellos to us with random intervals and we respond to each one as we receive it until 
   the client closes the connection */
func (s *grpcServer) StreamingHello(stream pb.Greeter_StreamingHelloServer) error {
	p, _ := peer.FromContext(stream.Context())
	frontendip := p.Addr.String()

	// get the stream header
	md, ok := metadata.FromIncomingContext(stream.Context())
	if !ok {
		return status.Error(codes.InvalidArgument, fmt.Sprintf("[%v] Unable to retrieve request metadata", frontendip))
	}

	clientTargetTenantId, err := s.validateTenantId(md)
	if err != nil {
		return err
	}

	log.Printf("[%v] <%v> Client opened request stream", frontendip, clientTargetTenantId)

	for {
		in, err := stream.Recv()

		if err == io.EOF { 
			// client closed the connection
			log.Printf("[%v] <%v> Client closed connection", frontendip, clientTargetTenantId)
			break
		}

		if err != nil {
			log.Printf("[%v] <%v> Error: %v", frontendip, clientTargetTenantId, err.Error())
			return err
		}
		
		log.Printf("[%v] <%v> Received request: %v", frontendip, clientTargetTenantId, in.GetName())
		reply, err := getHelloReply(stream.Context(), in, clientTargetTenantId)
		if err != nil {
			log.Printf("[%v] <%v> Error: %v", frontendip, clientTargetTenantId, err.Error())
			return err
		}

		if err := stream.Send(reply); err != nil {
			log.Printf("[%v] <%v> Error: %v", frontendip, clientTargetTenantId, err.Error())
			return err
		}
	}

	return nil
}

func main() {

	tlsCrt := flag.String("crt", "certs/tls.crt", "TLS certificate")
	tlsKey := flag.String("key", "certs/tls.key", "TLS private key")
	tlsB := flag.Bool("tls", true, "enable TLS")
	configDir := flag.String("config-dir", "config/", "config directory")
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

	/* get the tenant config */
	t := tenant.LoadTenantConfig(*configDir)
	tenantConfigJSON, _ := json.Marshal(t)
	log.Printf("Loaded Tenant Config: %v", string(tenantConfigJSON))

	/* register grpc services */
	g := &grpcServer{
		serverTenantConfig: *t,
	}

	pb.RegisterGreeterServer(s, g)
	grpc_health.RegisterHealthServer(s, g)

	/* register http services */
	h := &http.Server{}
	http.DefaultServeMux.Handle("/healthz", &http_health.HttpHealthCheckHandler{})

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
