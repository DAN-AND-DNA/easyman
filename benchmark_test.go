package main

import (
	"context"
	"easyman/pb/webbff"
	gingrpc "github.com/dan-and-dna/gin-grpc"
	"github.com/dan-and-dna/gin-grpc-network/core"
	"github.com/dan-and-dna/gin-grpc-network/modules/grpc"
	"github.com/dan-and-dna/gin-grpc-network/modules/network"
	"testing"
)

func setup() {
	// grpc 以pkg service method来区别请求
	network.HandleProto("webbff", "WebBFF", "Login", &webbff.WebBFF_ServiceDesc, gingrpc.Handler{
		Proto: &webbff.LoginReq{},
		HandleProto: func(ctx context.Context, req interface{}) (interface{}, error) {
			return nil, nil
		},
	})
}

func BenchmarkGrpc(b *testing.B) {
	setup()
	moduleCore := grpc.ModuleLock()

	grpcCore := moduleCore.(*core.GrpcCore)
	grpcCore.Enable = true
	grpcCore.Middlewares = nil
	grpc.ModuleBenchmark(b, "/webbff.WebBFF/Login", &webbff.LoginReq{}, &webbff.LoginResp{})
}
