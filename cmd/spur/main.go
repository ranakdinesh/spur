package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/ranakdinesh/spur/internal/scaffold"
)

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(2)
	}
	switch os.Args[1] {
	case "new":
		newService(os.Args[2:])
	default:
		usage()
		os.Exit(2)
	}
}

func usage() {
	fmt.Println(`spur - scaffolding CLI

Usage:
  spur new service <name> [flags]

Flags:
  --module         Go module path for the new service (required)
  --httpaddr       HTTP listen address (default ":8080")
  --with-grpc      Include gRPC server skeleton
  --with-db        Include Postgres (pgxkit) wiring
  --with-redis     Include Redis (rediskit) wiring
  --with-auth      Include Auth (authclient) wiring
`)
}

func newService(args []string) {
	fs := flag.NewFlagSet("new service", flag.ExitOnError)
	withGRPC := fs.Bool("with-grpc", false, "")
	withDB := fs.Bool("with-db", false, "")
	withRedis := fs.Bool("with-redis", false, "")
	withAuth := fs.Bool("with-auth", false, "")
	httpAddr := fs.String("httpaddr", ":8080", "")
	module := fs.String("module", "", "module path (required)")
	_ = fs.Parse(args)

	if fs.NArg() < 1 {
		fmt.Println("missing <name>")
		os.Exit(2)
	}
	name := fs.Arg(0)
	if *module == "" {
		fmt.Println("--module is required (e.g., github.com/you/" + name + ")")
		os.Exit(2)
	}

	err := scaffold.Service(scaffold.Options{
		Name:      name,
		Module:    *module,
		HTTPAddr:  *httpAddr,
		WithGRPC:  *withGRPC,
		WithDB:    *withDB,
		WithRedis: *withRedis,
		WithAuth:  *withAuth,
	})
	if err != nil {
		fmt.Println("error:", err)
		os.Exit(1)
	}
	fmt.Println("✅ created service:", name)
}
