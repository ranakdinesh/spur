package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/ranakdinesh/spur/internal/scaffold"
)

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(2)

	}
	switch os.Args[1] {
	case "new":
		args := os.Args[2:]

		if len(args) > 0 && args[0] == "service" {
			args = os.Args[1:]
		}
		newService(args)

	default:
		usage()
		os.Exit(2)
	}
}

func usage() {
	fmt.Println(`spur - Scaffolding CLI
Usage:
	spur new service <name> [flags]
	spur new <name> [flags] # also supported

Flags:
	--module            Go module path for the new service (required)
	--with-http         Add a HTTP server to the new service
	--httpAddr          HTTP listen address (default ":8080")
	--with-grpc         Inculde gRPC server skeleton
	--grpcAddr          gRPC listen address (default ":50051")
	--with-postgres     Include Postgres (pgxkit) wiring
	--with-redis        Include Redis (rediskit) wiring
	--with-kafka        Include Kafka (kafkit) wiring
	--with-auth         Include Auth (authkit) wiring

Examples:
		spur new service accounts --module github.com/you/accounts --with-db --with-auth
  		spur new accounts --module github.com/you/accounts
`)
}

func newService(argv []string) {
	fs := flag.NewFlagSet("new service", flag.ExitOnError)
	withHTTP := fs.Bool("with-http", false, "")
	withGRPC := fs.Bool("with-grpc", false, "")
	grpcAddr := fs.String("grpcaddr", ":50051", "")
	withDB := fs.Bool("with-postgres", false, "")
	withRedis := fs.Bool("with-redis", false, "")
	withAuth := fs.Bool("with-auth", false, "")
	httpAddr := fs.String("httpaddr", ":8080", "")
	module := fs.String("module", "", "module path (required)")

	// Accept flags in ANY position by pulling out the first non-flag as <name>
	name := ""
	rest := make([]string, 0, len(argv))
	for _, a := range argv {
		if name == "" && !strings.HasPrefix(a, "-") {
			name = a
			continue
		}
		rest = append(rest, a)
	}
	_ = fs.Parse(rest)

	if name == "" {
		fmt.Println("missing <name>")
		os.Exit(2)
	}
	if *module == "" {
		fmt.Println("--module is required (e.g., github.com/you/" + name + ")")
		os.Exit(2)
	}

	if err := scaffold.Service(scaffold.Options{
		Name:         name,
		Module:       *module,
		WithHTTP:     *withHTTP,
		HTTPAddr:     *httpAddr,
		WithGRPC:     *withGRPC,
		GRPCAddr:     *grpcAddr,
		WithPostgres: *withDB,
		WithRedis:    *withRedis,
		WithAuth:     *withAuth,
	}); err != nil {
		fmt.Println("error:", err)
		os.Exit(1)
	}

	fmt.Println("✅ created service:", name)
}
