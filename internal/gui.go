package internal

import (
	"embed"
	"html/template"
	"net/http"

	"github.com/fraddy91/smtp-to-apprise/logger"
	"github.com/gin-gonic/gin"
)

type Record struct {
	Email    string `json:"email"`
	Key      string `json:"key"`
	Tags     string `json:"tags"`
	MimeType string `json:"mime_type"`
}

type UpdatePayload struct {
	Email    string `json:"email"`
	MimeType string `json:"mime_type"`
	Field    string `json:"field"`
	Value    string `json:"value"`
}

//go:embed templates/*
var templatesFS embed.FS

//go:embed assets/favicon.ico
var favicon []byte

func StartGUI(be Backend, addr string) {
	// TODO: Add setting for debug log level
	gin.SetMode(gin.ReleaseMode)
	r := gin.Default()
	// TODO: Add trusted proxies settings, e.g. r.SetTrustedProxies([]string{"127.0.0.1"})
	tmpl, err := template.ParseFS(templatesFS, "templates/*")
	if err != nil {
		logger.Errorf("Failed to parse templates: %v", err)
	}
	r.SetHTMLTemplate(tmpl)

	// Serve favicon directly from embedded bytes
	r.GET("/favicon.ico", func(c *gin.Context) {
		c.Data(http.StatusOK, "image/x-icon", favicon)
	})

	r.GET("/", func(c *gin.Context) {
		recs, err := be.GetAllRecords()
		if err != nil {
			c.String(500, "DB error: %v", err)
			return
		}
		c.HTML(200, "index.html", gin.H{
			"Records": recs,
			"Success": c.Query("success") == "1",
			"Error":   c.Query("error"),
		})
	})

	r.POST("/add", func(c *gin.Context) {
		var rec Record
		rec.Email = c.PostForm("email")
		rec.Key = c.PostForm("key")
		rec.Tags = c.PostForm("tags")
		rec.MimeType = c.PostForm("mime_type")
		err := be.AddRecord(&rec)
		if err != nil {
			c.Redirect(302, "/?error=DB+insert+failed")
			return
		}
		c.Redirect(302, "/?success=1")
	})

	r.POST("/update", func(c *gin.Context) {
		var p UpdatePayload
		if err := c.BindJSON(&p); err != nil {
			c.Status(http.StatusBadRequest)
			return
		}

		err := be.UpdateRecord(p.Field, p.Value, p.Email, p.MimeType)
		if err != nil {
			logger.Errorf("Update error: %v", err)
			c.Status(http.StatusInternalServerError)
			return
		}
		c.Status(http.StatusOK)
	})

	r.POST("/delete", func(c *gin.Context) {
		var rec Record
		rec.Email = c.PostForm("email")
		rec.Key = c.PostForm("key")
		rec.MimeType = c.PostForm("mime_type")
		err := be.DeleteRecord(&rec)
		if err != nil {
			c.Redirect(302, "/?error=Delete+failed")
			return
		}
		c.Redirect(302, "/?success=1")
	})

	logger.Infof("Admin GUI listening on %s", addr)
	if err := r.Run(addr); err != nil {
		logger.Errorf("Failed to run server: %v", err)
	}
}
