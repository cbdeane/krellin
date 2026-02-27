package main

import (
	"context"
	"flag"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	clientui "krellin/internal/client"
	"krellin/internal/protocol"
	clientpkg "krellin/pkg/client"
)

type noopHandler struct{}

func (n *noopHandler) Handle(ctx context.Context, action protocol.Action) error {
	return nil
}

func main() {
	repoFlag := flag.String("repo", "", "repo root (defaults to cwd)")
	sockFlag := flag.String("sock", "/tmp/krellin.sock", "daemon unix socket")
	flag.Parse()

	repo := *repoFlag
	if repo == "" {
		cwd, _ := os.Getwd()
		repo = cwd
	}
	repo, _ = filepath.Abs(repo)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	cli := clientpkg.NewSocketClient(*sockFlag, "", repo)
	if err := clientui.EnsureDaemon(*sockFlag); err != nil {
		// best-effort; continue and let connection fail if needed
	}
	tui := clientui.NewTUI(cli, os.Stdout, os.Stdin, "", "user")

	go func() {
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
		<-sigCh
		cancel()
	}()

	_ = tui.Run(ctx)
}
