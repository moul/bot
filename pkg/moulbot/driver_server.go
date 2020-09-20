package moulbot

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net"
	"net/http"
	"net/http/pprof"
	"os"
	"runtime/debug"
	"strings"

	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
	"github.com/gogo/gateway"
	grpc_middleware "github.com/grpc-ecosystem/go-grpc-middleware"
	grpc_auth "github.com/grpc-ecosystem/go-grpc-middleware/auth"
	grpc_zap "github.com/grpc-ecosystem/go-grpc-middleware/logging/zap"
	grpc_recovery "github.com/grpc-ecosystem/go-grpc-middleware/recovery"
	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	"github.com/oklog/run"
	"github.com/rs/cors"
	"github.com/soheilhy/cmux"
	chilogger "github.com/treastech/logger"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"moul.io/banner"
	"moul.io/bot/pkg/moulbotpb"
)

type serverDriver struct {
	listener net.Listener
}

func (svc *Service) StartServer() error {
	fmt.Fprintln(os.Stderr, banner.Inline("server"))
	svc.logger.Debug("starting server", zap.String("bind", svc.opts.ServerBind))

	// listeners
	var err error
	svc.server.listener, err = net.Listen("tcp", svc.opts.ServerBind)
	if err != nil {
		return err
	}
	smux := cmux.New(svc.server.listener)
	smux.HandleError(func(err error) bool {
		svc.logger.Warn("cmux error", zap.Error(err))
		return true
	})
	grpcListener := smux.MatchWithWriters(cmux.HTTP2MatchHeaderFieldPrefixSendSettings("content-type", "application/grpc"))
	httpListener := smux.Match(cmux.HTTP2(), cmux.HTTP1())

	// grpc server
	grpcServer := svc.grpcServer()
	var gr run.Group
	gr.Add(func() error {
		err := grpcServer.Serve(grpcListener)
		if err != cmux.ErrListenerClosed {
			return err
		}
		return nil
	}, func(error) {
		err := grpcListener.Close()
		if err != nil {
			svc.logger.Warn("close gRPC listener", zap.Error(err))
		}
	})

	// http server
	httpServer, err := svc.httpServer()
	if err != nil {
		return err
	}
	gr.Add(func() error {
		err := httpServer.Serve(httpListener)
		if err != cmux.ErrListenerClosed {
			return err
		}
		return nil
	}, func(error) {
		ctx, cancel := context.WithTimeout(svc.ctx, svc.opts.ServerShutdownTimeout)
		err := httpServer.Shutdown(ctx)
		if err != nil {
			svc.logger.Warn("shutdown HTTP server", zap.Error(err))
		}
		defer cancel()
		err = httpListener.Close()
		if err != nil {
			svc.logger.Warn("close HTTP listener", zap.Error(err))
		}
	})

	// cmux
	gr.Add(
		smux.Serve,
		func(err error) {
			svc.logger.Warn("cmux terminated", zap.Error(err))
		},
	)

	// context
	gr.Add(func() error {
		<-svc.ctx.Done()
		svc.logger.Warn("parent ctx done")
		return nil
	}, func(error) {})

	return gr.Run()
}

func (svc *Service) CloseServer(error) {
	svc.logger.Debug("closing server", zap.Bool("was-started", svc.server.listener != nil))
	if svc.server.listener != nil {
		svc.server.listener.Close()
	}
	svc.cancel()
}

func (svc *Service) ServerListenerAddr() string {
	return svc.server.listener.Addr().String()
}

func (svc *Service) httpServer() (*http.Server, error) {
	r := chi.NewRouter()
	r.Use(cors.New(cors.Options{
		AllowedOrigins:   strings.Split(svc.opts.ServerCORSAllowedOrigins, ","),
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token"},
		ExposedHeaders:   []string{"Link"},
		AllowCredentials: true,
		MaxAge:           300,
	}).Handler)
	r.Use(chilogger.Logger(svc.logger))
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(svc.opts.ServerRequestTimeout))
	r.Use(middleware.RealIP)
	r.Use(middleware.RequestID)

	runtimeMux := runtime.NewServeMux(
		runtime.WithMarshalerOption(runtime.MIMEWildcard, &gateway.JSONPb{
			EmitDefaults: false,
			Indent:       "  ",
			OrigName:     true,
		}),
		runtime.WithProtoErrorHandler(runtime.DefaultHTTPProtoErrorHandler),
	)
	var gwmux http.Handler = runtimeMux
	dialOpts := []grpc.DialOption{grpc.WithInsecure()}

	err := moulbotpb.RegisterWebAPIHandlerFromEndpoint(svc.ctx, runtimeMux, svc.ServerListenerAddr(), dialOpts)
	if err != nil {
		return nil, err
	}

	r.Mount("/", gwmux)
	if svc.opts.ServerWithPprof {
		r.HandleFunc("/debug/pprof/*", pprof.Index)
		r.HandleFunc("/debug/pprof/cmdline", pprof.Cmdline)
		r.HandleFunc("/debug/pprof/profile", pprof.Profile)
		r.HandleFunc("/debug/pprof/symbol", pprof.Symbol)
		r.HandleFunc("/debug/pprof/trace", pprof.Trace)
	}
	http.DefaultServeMux = http.NewServeMux() // disables default handlers registered by importing net/http/pprof for security reasons
	return &http.Server{
		Addr:    "osef",
		Handler: r,
	}, nil
}

func (svc *Service) AuthFuncOverride(ctx context.Context, path string) (context.Context, error) {
	// accept everything
	return ctx, nil
}

func (svc *Service) grpcServer() *grpc.Server {
	authFunc := func(context.Context) (context.Context, error) {
		return nil, errors.New("auth: dummy function, see AuthFuncOverride")
	}
	recoveryOpts := []grpc_recovery.Option{}
	if svc.logger.Check(zap.DebugLevel, "") != nil {
		recoveryOpts = append(recoveryOpts, grpc_recovery.WithRecoveryHandlerContext(func(ctx context.Context, p interface{}) error {
			log.Println("stacktrace from panic: \n" + string(debug.Stack()))
			return status.Errorf(codes.Internal, "recover: %s", p)
		}))
	}
	serverStreamOpts := []grpc.StreamServerInterceptor{grpc_recovery.StreamServerInterceptor(recoveryOpts...)}
	serverUnaryOpts := []grpc.UnaryServerInterceptor{grpc_recovery.UnaryServerInterceptor(recoveryOpts...)}
	serverStreamOpts = append(serverStreamOpts,
		grpc_auth.StreamServerInterceptor(authFunc),
		// grpc_ctxtags.StreamServerInterceptor(),
		grpc_zap.StreamServerInterceptor(svc.logger),
	)
	serverUnaryOpts = append(
		serverUnaryOpts,
		grpc_auth.UnaryServerInterceptor(authFunc),
		// grpc_ctxtags.UnaryServerInterceptor(),
		grpc_zap.UnaryServerInterceptor(svc.logger),
	)
	if svc.logger.Check(zap.DebugLevel, "") != nil {
		serverStreamOpts = append(serverStreamOpts, grpcServerStreamInterceptor())
		serverUnaryOpts = append(serverUnaryOpts, grpcServerUnaryInterceptor())
	}
	serverStreamOpts = append(serverStreamOpts, grpc_recovery.StreamServerInterceptor(recoveryOpts...))
	serverUnaryOpts = append(serverUnaryOpts, grpc_recovery.UnaryServerInterceptor(recoveryOpts...))
	grpcServer := grpc.NewServer(
		grpc.StreamInterceptor(grpc_middleware.ChainStreamServer(serverStreamOpts...)),
		grpc.UnaryInterceptor(grpc_middleware.ChainUnaryServer(serverUnaryOpts...)),
	)
	moulbotpb.RegisterWebAPIServer(grpcServer, svc)

	return grpcServer
}

func grpcServerStreamInterceptor() grpc.StreamServerInterceptor {
	return func(srv interface{}, stream grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		err := handler(srv, stream)
		if err != nil {
			log.Printf("%+v", err)
		}
		return err
	}
}

func grpcServerUnaryInterceptor() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		ret, err := handler(ctx, req)
		if err != nil {
			log.Printf("%+v", err)
		}
		return ret, err
	}
}
