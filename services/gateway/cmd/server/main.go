package main

import (
	"flag"
	"os"

	"trpc.group/trpc-go/trpc-gateway/core/config"
	_ "trpc.group/trpc-go/trpc-gateway/core/loader/file"
	"trpc.group/trpc-go/trpc-gateway/core/service/fhttp"
	_ "trpc.group/trpc-go/trpc-gateway/core/service/protocol/trpc"
	"trpc.group/trpc-go/trpc-go/log"
	_ "trpc.group/trpc-go/trpc-naming-polarismesh"
)

func expandConfigFiles() {
	if os.Getenv("POLARIS_ADDR") == "" {
		os.Setenv("POLARIS_ADDR", "192.168.137.240:8091")
	}
	candidates := []string{
		"./conf/trpc_go.yaml",
		"/app/conf/trpc_go.yaml",
		"./conf/router.yaml",
		"/app/conf/router.yaml",
		"./conf/polaris.yaml",
		"/app/conf/polaris.yaml",
		"/app/polaris.yaml",
	}
	for i := 1; i < len(os.Args); i++ {
		if (os.Args[i] == "-conf" || os.Args[i] == "--conf" || os.Args[i] == "-router" || os.Args[i] == "--router") && i+1 < len(os.Args) {
			candidates = append(candidates, os.Args[i+1])
		}
	}
	for _, p := range candidates {
		if data, err := os.ReadFile(p); err == nil {
			expanded := os.ExpandEnv(string(data))
			_ = os.WriteFile(p, []byte(expanded), 0644)
		}
	}
}

func main() {
	expandConfigFiles()
	flag.Parse()
	s := config.NewServer()
	fhttp.RegisterFastHTTPService(s)

	log.Info("tRPC-Gateway starting on port 8080, forwarding HTTP to tRPC with PolarisMesh...")
	if err := s.Serve(); err != nil {
		log.Fatalf("gateway server error: %v", err)
	}
}
