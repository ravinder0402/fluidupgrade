package main

import (
	"context"
	"flag"
	"io/fs"
	"log"
	"mime"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	grpc_auth "github.com/grpc-ecosystem/go-grpc-middleware/auth"
	gwruntime "github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/coredgeio/compass/pkg/auth"
	"github.com/coredgeio/compass/pkg/infra/configdb"
	inframanager "github.com/coredgeio/compass/pkg/infra/manager"
	infraowner "github.com/coredgeio/compass/pkg/infra/owner"

	api "github.com/coredgeio/workflow-manager/api/workflow"
	"github.com/coredgeio/workflow-manager/api/workflow/swagger"
	"github.com/coredgeio/workflow-manager/pkg/config"
	"github.com/coredgeio/workflow-manager/pkg/manager/modules"
	"github.com/coredgeio/workflow-manager/pkg/manager/workflow"
	"github.com/coredgeio/workflow-manager/pkg/runtime"
	"github.com/coredgeio/workflow-manager/pkg/server"
	"github.com/coredgeio/workflow-manager/pkg/server/websocket"
)

const (
	// Internal GRPC port to host the grpc server
	GRPC_PORT = ":8090"

	// Port Over which registry API will be supported
	// for UI portal
	API_PORT = ":8080"

	// port for hosting websocket server
	WEBSOCKET_PORT = ":9080"
)

// parseFlags will create and parse the CLI flags
// and return the path to be used elsewhere
func parseFlags() (string, error) {
	var configPath string

	// Set up a CLI flag called "-config" to allow users
	// to supply the configuration file
	flag.StringVar(&configPath, "config", "/opt/config.yml", "path to config file")

	// Actually parse the flags
	flag.Parse()

	// Return the configuration path
	return configPath, nil
}

func getOpenAPIHandler() http.Handler {
	err := mime.AddExtensionType(".svg", "image/svg+xml")
	if err != nil {
		log.Fatalf("couldn't add mime extension: %s", err)
	}
	// Use subdirectory in embedded files
	subFS, err := fs.Sub(swagger.OpenAPI, "OpenAPI")
	if err != nil {
		panic("couldn't create sub filesystem: " + err.Error())
	}
	return http.FileServer(http.FS(subFS))
}

func main() {
	cfgPath, err := parseFlags()
	if err != nil {
		log.Fatal(err)
	}

	err = config.ParseConfig(cfgPath)
	if err != nil {
		log.Fatal(err)
	}

	// following section can be enabled for using mongodb client
	err = configdb.InitializeDatabaseConnection(config.GetMongodbHost(),
		config.GetMongodbPort(), runtime.WorkflowEngineDatabaseName)
	if err != nil {
		log.Println("Unable to initialize mongo database connection...")
		log.Println(err)
		log.Fatalln("Exiting...")
	}

	err = configdb.InitializeMetricsDatabaseConnection(config.GetMetricsdbHost(),
		config.GetMetricsdbPort(), "compass-metrics")
	if err != nil {
		log.Println("Unable to initialize metrics database connection...")
		log.Println(err)
		log.Fatalln("Exiting...")
	}

	var opts []grpc.ServerOption
	// following code can be enabled for authentication services
	opts = append(opts, grpc.StreamInterceptor(grpc_auth.StreamServerInterceptor(auth.ProcessUserInfoInContext)))
	opts = append(opts, grpc.UnaryInterceptor(grpc_auth.UnaryServerInterceptor(auth.ProcessUserInfoInContext)))

	grpcServer := grpc.NewServer(opts...)
	api.RegisterBaseImageApiServer(grpcServer, server.NewBaseImageApiServer())
	api.RegisterModuleApiServer(grpcServer, server.NewModuleApiServer())
	api.RegisterWorkflowTemplateApiServer(grpcServer, server.NewWorkflowTemplateApiServer())
	api.RegisterWorkflowApiServer(grpcServer, server.NewWorkflowApiServer())

	lis, err := net.Listen("tcp", GRPC_PORT)
	if err != nil {
		log.Fatalf("failed to listen: %v for grpc server", err)
	}

	go func() {
		log.Println("serving grpc server...")
		log.Fatal(grpcServer.Serve(lis))
	}()

	// Create a client connection to just started grpc server
	conn, err := grpc.DialContext(
		context.Background(),
		GRPC_PORT,
		grpc.WithBlock(),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		log.Fatal("Failed to dial server:", err)
	}

	gwmux := gwruntime.NewServeMux(gwruntime.WithIncomingHeaderMatcher(func(key string) (string, bool) {
		// enable this section while using orbiter-auth module
		if key == auth.UserInfoHeader {
			return auth.UserInfoContext, true
		}
		return key, false
	}))

	// Register Base Image API server
	err = api.RegisterBaseImageApiHandler(context.Background(), gwmux, conn)
	if err != nil {
		log.Fatal("Failed to register Base Image api handler with gateway:", err)
	}

	// Register Module API server
	err = api.RegisterModuleApiHandler(context.Background(), gwmux, conn)
	if err != nil {
		log.Fatal("Failed to register Module api handler with gateway:", err)
	}

	// Register Workflow Template API server
	err = api.RegisterWorkflowTemplateApiHandler(context.Background(), gwmux, conn)
	if err != nil {
		log.Fatal("Failed to register workflow template api handler with gateway:", err)
	}

	// Register Workflow API server
	err = api.RegisterWorkflowApiHandler(context.Background(), gwmux, conn)
	if err != nil {
		log.Fatal("Failed to register workflow api handler with gateway:", err)
	}

	oa := getOpenAPIHandler()
	gwServer := &http.Server{
		Addr: API_PORT,
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if strings.HasPrefix(r.URL.Path, "/api/") {
				gwmux.ServeHTTP(w, r)
				return
			}
			oa.ServeHTTP(w, r)
		}),
	}

	regClientConfig := &modules.RegClientConfig{
		Realm:  config.GetInternalAuthRealm(),
		Domain: config.GetInternalAuthDomain(),
		User:   config.GetInternalAuthUser(),
		Scheme: config.GetRegistryScheme(),
		Host:   config.GetRegistryHost() + ":" + config.GetRegistryPort(),
	}
	// start managers that requires to be running in workflow manager
	_ = modules.CreateModuleBuilder(regClientConfig, config.GetWorkflowRegistryName())
	_ = workflow.CreateWorkflowExecutor(config.GetWorkflowServiceAccount(), config.GetWorkflowRegistryName())

	go func() {
		log.Println("starting api server")
		log.Println("Serving gRPC-Gateway on http://0.0.0.0" + API_PORT)
		log.Fatalln(gwServer.ListenAndServe())
	}()

	// create websocket mux
	wsMux := websocket.NewWebSocketServer()

	// register build logs server for websocket
	websocket.CreateModuleBuildLogsServer(wsMux)

	// register workflow logs server for websocket
	websocket.CreateWorkflowLogsServer(wsMux)

	wsServer := &http.Server{
		Addr:    WEBSOCKET_PORT,
		Handler: wsMux,
	}

	go func() {
		log.Println("Serving websocket on server *" + WEBSOCKET_PORT)
		log.Fatalln(wsServer.ListenAndServe())
	}()

	defer func() {
		infraowner.HandleTerminate()
		inframanager.StopManagers()
		// pause for few seconds for cleanup to happen
		time.Sleep(10 * time.Second)
		log.Println("process terminated")
	}()

	// catch sigterm for graceful cleanup
	sigchan := make(chan os.Signal, 2)
	signal.Notify(sigchan, syscall.SIGTERM)
	sig := <-sigchan
	log.Println("got signal:", sig)
}
