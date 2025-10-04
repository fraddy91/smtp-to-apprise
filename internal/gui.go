package internal

import (
	"database/sql"
	"embed"
	"html/template"
	"log"

	"github.com/gin-gonic/gin"
)

type Record struct {
	Email string `json:"email"`
	Key   string `json:"key"`
	Tags  string `json:"tags"`
}

//go:embed templates/*
var templatesFS embed.FS

func StartGUI(db *sql.DB, addr string) {
	r := gin.Default()
	tmpl, err := template.ParseFS(templatesFS, "templates/*")
	if err != nil {
		log.Fatalf("Failed to parse templates: %v", err)
	}
	r.SetHTMLTemplate(tmpl)

	r.GET("/", func(c *gin.Context) {
		rows, err := db.Query("SELECT email, apprise_key, tags FROM records")
		if err != nil {
			c.String(500, "DB error: %v", err)
			return
		}
		defer rows.Close()

		var recs []Record
		for rows.Next() {
			var r Record
			rows.Scan(&r.Email, &r.Key, &r.Tags)
			recs = append(recs, r)
		}

		c.HTML(200, "index.html", gin.H{
			"Records": recs,
			"Success": c.Query("success") == "1",
			"Error":   c.Query("error"),
		})
	})

	r.POST("/add", func(c *gin.Context) {
		_, err := db.Exec(
			"INSERT OR REPLACE INTO records (email, apprise_key, tags) VALUES (?, ?, ?)",
			c.PostForm("email"), c.PostForm("key"), c.PostForm("tags"),
		)
		if err != nil {
			c.Redirect(302, "/?error=DB+insert+failed")
			return
		}
		c.Redirect(302, "/?success=1")
	})

	r.POST("/delete", func(c *gin.Context) {
		_, err := db.Exec(
			"DELETE FROM records WHERE email = ? AND apprise_key = ?",
			c.PostForm("email"), c.PostForm("key"),
		)
		if err != nil {
			c.Redirect(302, "/?error=Delete+failed")
			return
		}
		c.Redirect(302, "/?success=1")
	})

	log.Printf("Admin GUI listening on %s", addr)
	log.Fatal(r.Run(addr))
}
