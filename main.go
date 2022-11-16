package main

import (
	"bytes"
	"context"
	"easyman/pb/webbff"
	"github.com/dan-and-dna/gin-grpc"
	gingrpcnetwork "github.com/dan-and-dna/gin-grpc-network"
	"github.com/dan-and-dna/gin-grpc-network/core"
	mgrpc "github.com/dan-and-dna/gin-grpc-network/modules/grpc"
	mhttp "github.com/dan-and-dna/gin-grpc-network/modules/http"
	"github.com/dan-and-dna/gin-grpc-network/modules/network"
	"github.com/dan-and-dna/gin-grpc-network/utils"
	"github.com/gin-gonic/gin"
	grpc_zap "github.com/grpc-ecosystem/go-grpc-middleware/logging/zap"
	"github.com/grpc-ecosystem/go-grpc-middleware/logging/zap/ctxzap"
	grpc_ctxtags "github.com/grpc-ecosystem/go-grpc-middleware/tags"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"io"
	"log"
	"net/http"
	"time"
)

var (
	zapLogger *zap.Logger
)

func init() {
	var err error
	zapLogger, err = zap.NewProduction()
	if err != nil {
		panic(err)
	}
	//grpc_zap.ReplaceGrpcLoggerV2(zapLogger)
}

type ZapOption struct {
}

func (option *ZapOption) Apply(ctx context.Context) context.Context {
	return ctxzap.ToContext(ctx, zapLogger)
}

func setHttp() {
	moduleCore := mhttp.ModuleLock()
	defer mhttp.ModuleUnlock()

	httpCore := moduleCore.(*core.HttpCore)

	httpCore.Enable = true
	httpCore.ListenPort = 3737
	httpCore.WriteTimeOut = 10
	httpCore.ReadTimeOut = 10
	httpCore.Path = "/:pkg/:service/:method"
	httpCore.PathToServiceName = func(c *gin.Context) string {
		pkg := c.Param("pkg")
		service := c.Param("service")
		method := c.Param("method")
		return utils.MakeKey(pkg, service, method)
	}
	// gin中间件
	httpCore.Middlewares = append(httpCore.Middlewares,
		gin.Recovery(),
		gin.Logger(),
	)

	// 跟gin的ctx无关，给每个handler使用
	httpCore.CtxOptions = append(httpCore.CtxOptions,
		&ZapOption{},
	)
	setupHttpClient()
}

func setGrpc() {
	moduleCore := mgrpc.ModuleLock()
	defer mgrpc.ModuleUnlock()

	grpcCore := moduleCore.(*core.GrpcCore)

	grpcCore.Enable = true
	grpcCore.ListenPort = 3730

	// grpc中间件
	grpcCore.Middlewares = append(grpcCore.Middlewares,
		grpc_ctxtags.UnaryServerInterceptor(),
		grpc_zap.UnaryServerInterceptor(zapLogger, grpc_zap.WithDurationField(grpc_zap.DefaultDurationToField)),
	)

	setupGrpcClient()
}

func setupHandlers() {
	// grpc 以pkg service method来区别请求
	network.HandleProto("webbff", "WebBFF", "Login", &webbff.WebBFF_ServiceDesc, gingrpc.Handler{
		Proto: &webbff.LoginReq{},
		HandleProto: func(ctx context.Context, req interface{}) (interface{}, error) {

			ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
			defer cancel()

			l := ctxzap.Extract(ctx)
			md, ok := metadata.FromIncomingContext(ctx)
			if !ok {
				return nil, status.Error(codes.Internal, "")
			}

			ct := md.Get("content-type")
			if len(ct) == 0 {
				return nil, status.Error(codes.NotFound, "content-type lost")
			}

			l.Info("get Content-Type", zap.String("Content-Type", ct[0]))

			reqProto := req.(*webbff.LoginReq)
			username := reqProto.GetName()
			password := reqProto.GetPassword()

			l.Info("user try to login", zap.String("name", username), zap.String("password", password))

			return &webbff.LoginResp{Token: "xxxxxxxx"}, nil
		},
	})
}

func setupGrpcClient() {
	go func() {
		serviceConn, err := grpc.Dial("127.0.0.1:3730", grpc.WithTransportCredentials(insecure.NewCredentials()))
		if err != nil {
			log.Printf("[webbff] connect to UserService failed: %s\n", err.Error())
		}

		serviceClient := webbff.NewWebBFFClient(serviceConn)

		for {
			time.Sleep(3 * time.Second)
			_, err := serviceClient.Login(context.TODO(), &webbff.LoginReq{
				Name:     "Dan",
				Password: "p1234567",
			})
			if err != nil {
				panic(err)
			}
		}
	}()
}

func setupHttpClient() {
	go func() {
		b := &bytes.Buffer{}

		for {
			time.Sleep(3 * time.Second)
			b.WriteString(`{"name": "Dan","password": "12345678"}`)
			resp, err := http.Post("http://127.0.0.1:3737/webbff/webbff/login", "application/json", b)
			if err != nil {
				panic(err)
			}
			buff, err := io.ReadAll(resp.Body)
			if err != nil {
				panic(err)
			}

			resp.Body.Close()
			log.Println(string(buff))
		}
	}()

}

func main() {
	defer zapLogger.Sync()
	setupHandlers()

	//setHttp()
	setGrpc()

	gingrpcnetwork.Poll()

}
