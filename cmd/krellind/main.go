package main

import (
	"context"
	"flag"
	"os"
	"os/signal"
	"syscall"

	"krellin/internal/app"
	daemonpkg "krellin/internal/daemon"
	"krellin/internal/session"
)

func main() {
	sockFlag := flag.String("sock", "/tmp/krellin.sock", "daemon unix socket")
	flag.Parse()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	d := daemonpkg.New()
	d.SetFactory(func(ctx context.Context, repoRoot string) (*session.Session, error) {
		return app.BuildSession(ctx, repoRoot)
	})
	router := daemonpkg.NewRouter(d, daemonpkg.NewTransport())
	srv := daemonpkg.NewServerWithRouter(*sockFlag, router)
	_ = srv.Start(ctx)
	defer srv.Stop()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	<-sigCh
}
