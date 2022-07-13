package helloserver

import (
	"context"
	"io"
	"io/ioutil"
	"os"

	gcp "helloworld/pkg/gcp"
	tenant "helloworld/pkg/tenant"
	pb "helloworld/proto/helloworld"

	"github.com/grpc-ecosystem/go-grpc-middleware/logging/zap/ctxzap"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/peer"
	"google.golang.org/grpc/status"
)

const (
	defaultVersion = "v1.0.0"
)

// server is used to implement helloworld.GreeterServer.
type HelloServer struct {
	pb.GreeterServer

	ServerTenantConfig tenant.TenantConfig
}

func NewHelloServer(tenantConfig tenant.TenantConfig) *HelloServer {
	s := &HelloServer{
		ServerTenantConfig: tenantConfig,
	}

	return s
}

func getHelloReply(ctx context.Context, in *pb.HelloRequest, clientTargetTenantId string) (*pb.HelloReply, error) {
	logger := ctxzap.Extract(ctx)

	host, _ := os.Hostname()
	zoneStr := gcp.GetMetaData(ctx, "instance/zone")
	nodeName := gcp.GetMetaData(ctx, "instance/hostname")
	region := gcp.GetMetaData(ctx, "instance/attributes/cluster-location")
	clusterName := gcp.GetMetaData(ctx, "instance/attributes/cluster-name")
	project := gcp.GetMetaData(ctx, "project/project-id")

	version, err := ioutil.ReadFile("version.txt")
	if err != nil {
		version = []byte(defaultVersion)
		logger.Warn("Unable to open version file version.txt, using default version", 
			zap.String("tenantId", clientTargetTenantId), 
			zap.String("version", string(version)),
		)
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

func (s *HelloServer) validateTenantId(tenantId string) (error) {
	if !s.ServerTenantConfig.CheckTenantId(tenantId) {
		return status.Error(codes.InvalidArgument, "Wrong Tenant-Id for instance")
	}

	return nil
}

// SayHello implements helloworld.GreeterServer
func (s *HelloServer) SayHello(ctx context.Context, in *pb.HelloRequest) (*pb.HelloReply, error) {
	p, _ := peer.FromContext(ctx)
	frontendip := p.Addr.String()

	clientTargetTenantId, err := tenant.GetTenantId(ctx)
	if err != nil {
		return nil, err
	}

	err = s.validateTenantId(clientTargetTenantId)
	if err != nil {
		return nil, err
	}

	logger := ctxzap.Extract(ctx)
	logger.Info("Received request", 
		zap.String("clientIp", frontendip), 
		zap.String("tenantId", clientTargetTenantId), 
		zap.String("name", in.GetName()))

	return getHelloReply(ctx, in, clientTargetTenantId)
}

/* streaming hello ... client sends hellos to us with random intervals and we respond to each one as we receive it until 
   the client closes the connection */
func (s *HelloServer) StreamingHello(stream pb.Greeter_StreamingHelloServer) error {
	p, _ := peer.FromContext(stream.Context())
	frontendip := p.Addr.String()

	clientTargetTenantId, err := tenant.GetTenantId(stream.Context())
	if err != nil {
		return err
	}

	err = s.validateTenantId(clientTargetTenantId)
	if err != nil {
		return err
	}

	logger := ctxzap.Extract(stream.Context())
	logger.Info("Client opened request stream", 
		zap.String("clientIp", frontendip), 
		zap.String("tenantId", clientTargetTenantId),
	)

	for {
		in, err := stream.Recv()

		if err == io.EOF { 
			// client closed the connection
			logger.Info("Client closed connection", 
				zap.String("clientIp", frontendip), 
				zap.String("tenantId", clientTargetTenantId),
			)
			break
		}

		if err != nil {
			logger.Error("Error receiving reply", 
				zap.String("clientIp", frontendip), 
				zap.String("tenantId", clientTargetTenantId),
				zap.Error(err),
			)

			return err
		}

		logger.Info("Received request", 
			zap.String("clientIp", frontendip), 
			zap.String("tenantId", clientTargetTenantId), 
			zap.String("name", in.GetName()))

	
		reply, err := getHelloReply(stream.Context(), in, clientTargetTenantId)
		if err != nil {
			logger.Error("Error processing reply", 
				zap.String("clientIp", frontendip), 
				zap.String("tenantId", clientTargetTenantId),
				zap.Error(err),
			)

			return err
		}

		if err := stream.Send(reply); err != nil {
			logger.Error("Error sending reply", 
				zap.String("clientIp", frontendip), 
				zap.String("tenantId", clientTargetTenantId),
				zap.Error(err),
			)

			return err
		}
	}

	return nil
}
