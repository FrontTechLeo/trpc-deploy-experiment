package main

import (
	"flag"

	"trpc.group/trpc-go/trpc-gateway/core/config"
	_ "trpc.group/trpc-go/trpc-gateway/core/loader/file"
	"trpc.group/trpc-go/trpc-gateway/core/service/fhttp"
	_ "trpc.group/trpc-go/trpc-gateway/core/service/protocol/trpc"
	"trpc.group/trpc-go/trpc-go/log"
	_ "trpc.group/trpc-go/trpc-naming-polarismesh"
)

func main() {
	flag.Parse()
	s := config.NewServer()
	fhttp.RegisterFastHTTPService(s)

	log.Info("tRPC-Gateway starting on port 8080, forwarding HTTP to tRPC with PolarisMesh...")
	if err := s.Serve(); err != nil {
		log.Fatalf("gateway server error: %v", err)
	}
}
