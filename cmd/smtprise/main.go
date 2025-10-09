package main

import (
	"context"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"syscall"

	"github.com/emersion/go-smtp"
	"github.com/fraddy91/smtprise/internal"
	"github.com/fraddy91/smtprise/logger"
	_ "modernc.org/sqlite"
)

func main() {
	logger.Infof("Starting smtprise")

	// Prepare data directory
	dataDir, err := ensureDataDir("data", "0775")
	if err != nil {
		logger.Errorf("Failed to prepare data directory: %v", err)
	}

	// Load config + init DB
	cfg := internal.LoadConfig()
	db := internal.InitDB(filepath.Join(dataDir, cfg.StoreFile))

	// Backend with dispatcher
	dispatcherSize := 50
	be := &internal.Backend{
		Db:         db,
		AppriseURL: cfg.AppriseURL,
		Dispatcher: internal.NewDispatcher(dispatcherSize),
	}

	// Start SMTP server
	s := smtp.NewServer(be)
	s.Addr = ":" + cfg.ListenSMTP
	s.Domain = "localhost"
	s.AllowInsecureAuth = true

	go func() {
		logger.Infof("SMTP server listening on %s", s.Addr)
		if err := s.ListenAndServe(); err != nil {
			logger.Errorf("SMTP server error: %v", err)
		}
	}()

	// Start GUI if enabled
	if cfg.GuiEnabled {
		go func() {
			addr := ":" + cfg.ListenHTTP
			logger.Infof("Starting GUI on %s", addr)
			internal.StartGUI(*be, addr)
		}()
	}

	// Graceful shutdown
	waitForShutdown(be)
}

// ensureDataDir creates the data directory if missing.
func ensureDataDir(name, perm string) (string, error) {
	path, err := filepath.Abs("./")
	if err != nil {
		return "", err
	}
	outPath := filepath.Join(path, name)
	if _, err = os.Stat(outPath); os.IsNotExist(err) {
		if dirMod, perr := strconv.ParseUint(perm, 8, 32); perr == nil {
			if err = os.Mkdir(outPath, os.FileMode(dirMod)); err != nil {
				return "", err
			}
		}
	}
	return outPath, nil
}

// waitForShutdown blocks until SIGINT/SIGTERM and then drains dispatcher.
func waitForShutdown(be *internal.Backend) {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	<-ctx.Done()
	logger.Infof("Shutting down...")

	be.Db.Close()

	be.Dispatcher.Close()

	logger.Infof("Shutdown complete")
}
