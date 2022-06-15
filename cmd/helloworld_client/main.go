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

// Package main implements a client for Greeter service.
package main

import (
	"context"
	"crypto/tls"
	"flag"
	"io"
	"log"
	"math/rand"
	"time"

	pb "helloworld/proto/helloworld"

	"github.com/google/uuid"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	"google.golang.org/protobuf/encoding/protojson"
)

const (
	defaultAddress = "localhost:50051"
	defaultName    = "world"
)

func main() {
	connecttls := flag.Bool("tls", true, "connect over TLS")
	verifytls := flag.Bool("verifytls", true, "verify TLS")
	address := flag.String("addr", defaultAddress, "address to connect to, default localhost:50051")
	name := flag.String("name", defaultName, "name, default is world")
	tenantId := flag.String("tenant", "", "tenantId to connect to, default will generated one")
	useStream := flag.Bool ("stream", false, "use streaming rpc, default false to use unary rpc")
	streamCount := flag.Int("stream-count", -1, "for streaming rpc, send this many requests, -1 for infinite")
	streamIntervalMSecs := flag.Int("stream-interval-msecs", -1, "for streaming rpc, wait this number of milliseconds between requests, -1 for random")

	flag.Parse()

	if *tenantId == "" {
		defaultTenantId := uuid.New().String()
		tenantId = &defaultTenantId
	}

	log.Printf("Connecting to %v as %s with TenantID: %v", *address, *name, *tenantId)
	// Set up a connection to the server.

	grpcOptions := make([]grpc.DialOption, 0)

	if *connecttls {
		log.Printf("Connecting over TLS ...")
		config := &tls.Config{
			InsecureSkipVerify: !*verifytls,
		}
		grpcOptions = append(grpcOptions, grpc.WithTransportCredentials(credentials.NewTLS(config)))
	} else {
		grpcOptions = append(grpcOptions, grpc.WithTransportCredentials(insecure.NewCredentials()))
	}

	conn, err := grpc.Dial(*address, grpcOptions...)

	//conn, err := grpc.Dial(address, grpc.WithInsecure())
	if err != nil {
		log.Fatalf("did not connect: %v", err)
	}
	defer conn.Close()
	c := pb.NewGreeterClient(conn)

	var ctx context.Context
	var cancel context.CancelFunc

	if !*useStream {
		// Contact the server and print out its response, but the server should respond in 5 seconds.
		ctx, cancel = context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
	} else {
		// streaming RPCs just last forever
		ctx = context.Background()
	}

	/* set up the tenant id in the metadata */
	ctx = metadata.AppendToOutgoingContext(ctx, "X-Tenant-Id", *tenantId)
	req := &pb.HelloRequest{Name: *name}

	// unary RPC call and exit
	if !*useStream {
		r, err := c.SayHello(ctx, req)
		if err != nil {
			log.Fatalf("could not greet: %v", err)
		}
		log.Printf("Response: %v", protojson.Format(r))
		return
	}

	streamInterval := *streamIntervalMSecs
	total := *streamCount

	// start streaming RPC
	stream, err := c.StreamingHello(ctx)
	if err != nil {
		log.Fatalf("could not start streaming RPC: %v", err.Error())
	}

	log.Printf("stream: %v, total: %v", stream, total)

	i := 0
	// send reqs and receive replies -- if streamCount was not set (-1) this is an infinite loop
	for  {

		if err := stream.Send(req); err != nil {
			log.Printf("Error sending request: %v", err.Error())
		}

		// receive the response
		r := &pb.HelloReply{}
		err := stream.RecvMsg(r)
		if err == io.EOF {
			log.Printf("EOF received")
			break
		}
		
		if err != nil {
			log.Printf("Error receiving reply: %v", err.Error())
			break
		}

		log.Printf("Response: %v", protojson.Format(r))

		i = i + 1
		if  total != -1 && i >= total {
			// not infinite loop and we've reached target number of calls
			break
		}

		if *streamIntervalMSecs == -1 {
			streamInterval = rand.Intn(3000)
		}
		log.Printf("Sleeping for %v ms ...", streamInterval)
		time.Sleep(time.Duration(streamInterval) * time.Millisecond)

	}

	stream.CloseSend()


		
}
