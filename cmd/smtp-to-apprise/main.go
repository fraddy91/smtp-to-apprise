package main

import (
	"log"
	"os"
	"path/filepath"
	"strconv"

	"github.com/emersion/go-smtp"
	"github.com/fraddy91/smtp-to-apprise/internal"
	_ "modernc.org/sqlite"
)

func main() {
	//Get the cur file dir
	path, err := filepath.Abs("./") //
	if err != nil {
		log.Println("error msg", err)
	}

	outPath := filepath.Join(path, "data")
	if _, err = os.Stat(outPath); os.IsNotExist(err) {
		var dirMod uint64
		if dirMod, err = strconv.ParseUint("0775", 8, 32); err == nil {
			err = os.Mkdir(outPath, os.FileMode(dirMod))
		}
	}
	if err != nil && !os.IsExist(err) {
		log.Println("error msg", err)
	}
	cfg := internal.LoadConfig()
	db := internal.InitDB("data/" + cfg.StoreFile)

	// Start SMTP
	be := &internal.Backend{Db: db, AppriseURL: cfg.AppriseURL}
	s := smtp.NewServer(be)
	s.Addr = ":" + cfg.ListenSMTP
	s.Domain = "localhost"
	s.AllowInsecureAuth = true
	go func() { log.Fatal(s.ListenAndServe()) }()

	// Start GUI if enabled
	if cfg.GuiEnabled {
		go internal.StartGUI(db, ":"+cfg.ListenHTTP)
	}

	// Block forever
	select {}
}
