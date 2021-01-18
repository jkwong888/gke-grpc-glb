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
	"flag"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"os"

	pb "helloworld/rpc/helloworld"

	"github.com/golang/protobuf/proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/peer"
)

const (
	port = ":50051"
)

// server is used to implement helloworld.GreeterServer.
type server struct {
	pb.GreeterServer
}

func getMetaData(ctx context.Context, path string) *string {
	metaDataURL := "http://metadata/computeMetadata/v1/"
	req, _ := http.NewRequest(
		"GET",
		metaDataURL+path,
		nil,
	)
	req.Header.Add("Metadata-Flavor", "Google")
	req = req.WithContext(ctx)
	code, body := makeRequest(req)

	if code == 200 {
		bodyStr := string(body)
		return &bodyStr
	}

	return nil
}

func makeRequest(r *http.Request) (int, []byte) {
	//transport := http.Transport{DisableKeepAlives: true}
	//octr := &ochttp.Transport{}
	//client := &http.Client{Transport: octr}
	client := &http.Client{}
	resp, err := client.Do(r)
	if err != nil {
		message := "Unable to call backend: " + err.Error()
		panic(message)
	}
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		message := "Unable to read response body: " + err.Error()
		panic(message)
	}

	return resp.StatusCode, body
}

// SayHello implements helloworld.GreeterServer
func (s *server) SayHello(ctx context.Context, in *pb.HelloRequest) (*pb.HelloReply, error) {
	p, _ := peer.FromContext(ctx)
	frontendip := p.Addr.String()
	log.Printf("Received request from %v: %v", frontendip, proto.MarshalTextString(in))

	host, _ := os.Hostname()
	zoneStr := getMetaData(ctx, "instance/zone")
	nodeName := getMetaData(ctx, "instance/hostname")
	region := getMetaData(ctx, "instance/attributes/cluster-location")
	clusterName := getMetaData(ctx, "instance/attributes/cluster-name")
	project := getMetaData(ctx, "project/project-id")

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

func main() {

	tls := flag.Bool("tls", false, "listen on TLS")

	flag.Parse()

	lis, err := net.Listen("tcp", port)
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	s := grpc.NewServer()
	if *tls {
		log.Printf("Listening on TLS")
		creds, err := credentials.NewServerTLSFromFile("certs/service.pem", "certs/service.key")
		if err != nil {
			log.Fatalf("Failed to setup TLS: %v", err)
		}

		s = grpc.NewServer(grpc.Creds(creds))
	}

	pb.RegisterGreeterServer(s, &server{})

	log.Printf("Listening on port: %v", port)

	if err := s.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}
