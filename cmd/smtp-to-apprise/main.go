package main

import (
	"os"
	"path/filepath"
	"strconv"

	"github.com/emersion/go-smtp"
	"github.com/fraddy91/smtp-to-apprise/internal"
	"github.com/fraddy91/smtp-to-apprise/logger"
	_ "modernc.org/sqlite"
)

func main() {
	logger.Infof("Starting smtp-to-apprise")
	//Get the cur file dir
	path, err := filepath.Abs("./") //
	if err != nil {
		logger.Errorf("error msg", err)
	}

	// DB init
	outPath := filepath.Join(path, "data")
	if _, err = os.Stat(outPath); os.IsNotExist(err) {
		var dirMod uint64
		if dirMod, err = strconv.ParseUint("0775", 8, 32); err == nil {
			err = os.Mkdir(outPath, os.FileMode(dirMod))
		}
	}
	if err != nil && !os.IsExist(err) {
		logger.Errorf("error msg", err)
	}
	cfg := internal.LoadConfig()
	db := internal.InitDB("data/" + cfg.StoreFile)

	// Start SMTP
	logger.Infof("Using Apprise backend: %s", cfg.AppriseURL)
	be := &internal.Backend{Db: db, AppriseURL: cfg.AppriseURL, Dispatcher: internal.NewDispatcher(50)}
	s := smtp.NewServer(be)
	s.Addr = ":" + cfg.ListenSMTP
	s.Domain = "localhost"
	s.AllowInsecureAuth = true
	go func() {
		if err := s.ListenAndServe(); err != nil {
			logger.Errorf("SMTP server error: %v", err)
		}
	}()

	// Start GUI if enabled
	if cfg.GuiEnabled {
		go internal.StartGUI(*be, ":"+cfg.ListenHTTP)
	}

	// Block forever
	select {}
}
