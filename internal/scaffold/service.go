package scaffold

import (
	"embed"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"text/template"
)

//go:embed templates/service/**
var svcFS embed.FS

type Options struct {
	Name     string
	Module   string
	HTTPAddr string
	GRPCAddr string

	WithHTTP bool
	WithGRPC bool

	WithPostgres bool
	WithRedis    bool
	WithAuth     bool
}

func Service(opt Options) error {

	root := opt.Name
	paths := []string{
		filepath.Join(root, "cmd", opt.Name),
		filepath.Join(root, "internal", "app"),
		filepath.Join(root, "internal", "adapters"),
		filepath.Join(root, "internal", "infrastructure"),
		filepath.Join(root, "internal", "core", "domain"),
		filepath.Join(root, "internal", "core", "services"),
		filepath.Join(root, "internal", "core", "ports"),
	}
	if opt.WithHTTP {

		paths = append(paths,
			filepath.Join(root, "internal", "adapters", "http", "handlers"),
		)

	}
	if opt.WithGRPC {
		paths = append(paths,
			filepath.Join(root, "internal", "adapters", "grpc"),
			filepath.Join(root, "internal", "adapters", "grpc", "pb"),
		)
	}
	if opt.WithPostgres {
		paths = append(paths,

			filepath.Join(
				root, "sql"),
			filepath.Join(root, "sql", "migrations"),
			filepath.Join(root, "sql", "queries"),
			filepath.Join(root, "internal", "adapters", "postgres"),
			filepath.Join(root, "internal", "adapters", "postgres", "sqlc"),
		)
	}
	if opt.WithRedis {
		paths = append(paths,

			filepath.Join(root,
				"internal", "adapters", "redis"),
			filepath.Join(root, "internal", "adapters", "redis", "cache"),
		)
	}

	for _, p := range paths {
		if err := os.MkdirAll(p, 0755); err != nil {
			return err
		}

	}
	// render go.mod
	if err := render("templates/service/go.mod.tmpl",
		filepath.Join(root, "go.mod"), opt); err != nil {
		return err
	}

	// render .env.example
	if err := render("templates/service/env.example.tmpl",
		filepath.Join(root, ".env"), opt); err != nil {
		return err
	}
	// render Makefile
	if err := render("templates/service/Makefile.tmpl",
		filepath.Join(root, "Makefile"), opt); err != nil {
		return err
	}

	// render Dockerfile
	if err := render("templates/service/Dockerfile.tmpl",
		filepath.Join(root, "Dockerfile"), opt); err != nil {
		return err
	}

	// .gitignore
	if err := render("templates/service/gitignore.tmpl",
		filepath.Join(root, ".gitignore"), opt); err != nil {
		return err
	}
	// workflows
	if err := render("templates/service/.github/workflows/ci.yml.tmpl",
		filepath.Join(root, ".github", "workflows", "ci.yml"), opt); err != nil {
		return err
	}
	if err := render("templates/service/.github/workflows/release.yml.tmpl",
		filepath.Join(root, ".github", "workflows", "release.yml"), opt); err != nil {
		return err
	}
	// K8s: base + overlays
	if err := render("templates/service/k8s/base/kustomization.yaml.tmpl",
		filepath.Join(root, "k8s", "base", "kustomization.yaml"), opt); err != nil {
		return err
	}
	if err := render("templates/service/k8s/base/deployment.yaml.tmpl",
		filepath.Join(root, "k8s", "base", "deployment.yaml"), opt); err != nil {
		return err
	}
	if err := render("templates/service/k8s/base/service.yaml.tmpl",
		filepath.Join(root, "k8s", "base", "service.yaml"), opt); err != nil {
		return err
	}
	if err := render("templates/service/k8s/base/configmap.yaml.tmpl",
		filepath.Join(root, "k8s", "base", "configmap.yaml"), opt); err != nil {
		return err
	}
	if err := render("templates/service/k8s/base/secret.yaml.tmpl",
		filepath.Join(root, "k8s", "base", "secret.yaml"), opt); err != nil {
		return err
	}
	// If you want ingress by default, uncomment this render call and the resource in kustomization.
	// if err := render("templates/service/k8s/base/ingress.yaml.tmpl",
	// 	filepath.Join(root, "k8s", "base", "ingress.yaml"), opt); err != nil { return err }

	if err := render("templates/service/k8s/overlays/dev/kustomization.yaml.tmpl",
		filepath.Join(root, "k8s", "overlays", "dev", "kustomization.yaml"), opt); err != nil {
		return err
	}
	if err := render("templates/service/k8s/overlays/dev/patch-deploy.yaml.tmpl",
		filepath.Join(root, "k8s", "overlays", "dev", "patch-deploy.yaml"), opt); err != nil {
		return err
	}

	if err := render("templates/service/k8s/overlays/prod/kustomization.yaml.tmpl",
		filepath.Join(root, "k8s", "overlays", "prod", "kustomization.yaml"), opt); err != nil {
		return err
	}
	if err := render("templates/service/k8s/overlays/prod/patch-deploy.yaml.tmpl",
		filepath.Join(root, "k8s", "overlays", "prod", "patch-deploy.yaml"), opt); err != nil {
		return err
	}
	if err := render("templates/service/main.go.tmpl",
		filepath.Join(root, "cmd", "accounts", "main.go"), opt); err != nil {
		return err
	}
	if err := render("templates/service/app.go.tmpl",
		filepath.Join(root, "internal", "app", "app.go"), opt); err != nil {
		return err
	}
	if err := render("templates/service/observability.go.tmpl",
		filepath.Join(root, "internal", "app", "observability.go"), opt); err != nil {
		return err
	}
	if err := render("templates/service/run.go.tmpl",
		filepath.Join(root, "internal", "app", "run.go"), opt); err != nil {
		return err
	}

	if opt.WithPostgres {
		if err := render("templates/service/sqlc.yaml.tmpl",
			filepath.Join(root, "sqlc.yaml"), opt); err != nil {
			return err
		}

	}

	cmd := exec.Command("go", "mod", "tidy")
	cmd.Dir = root
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("go mod tidy: %w", err)
	}

	return nil
}
func render(src, dst string, data any) error {
	b, err := svcFS.ReadFile(src)
	if err != nil {
		return err
	}
	// If this is a GitHub Actions workflow, write it RAW to avoid
	// conflicts between Go templates {{...}} and GitHub's ${{ ... }}.
	if strings.Contains(src, "/.github/workflows/") {
		if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
			return err
		}
		return os.WriteFile(dst, b, 0o644)
	}

	tpl, err := template.New(filepath.Base(src)).
		Funcs(template.FuncMap{"join": strings.Join}).
		Parse(string(b))
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}
	f, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer f.Close()
	return tpl.Execute(f, data)
}
