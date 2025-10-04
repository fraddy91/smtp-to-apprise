package internal

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/mail"
	"os"

	"github.com/emersion/go-sasl"
	"github.com/emersion/go-smtp"
)

type Session struct {
	bkd    *Backend
	authed bool
	to     string
}

func (b *Backend) NewSession(c *smtp.Conn) (smtp.Session, error) {
	return &Session{bkd: b}, nil
}

func (s *Session) AuthMechanisms() []string {
	return []string{"PLAIN"}
}

func (s *Session) Auth(mech string) (sasl.Server, error) {
	log.Printf("SMTP: client requested auth mechanism=%s", mech)
	if mech != "PLAIN" {
		return nil, smtp.ErrAuthUnsupported
	}
	return sasl.NewPlainServer(func(_, username, password string) error {
		if username == os.Getenv("ADMIN_USER") && password == os.Getenv("ADMIN_PASS") {
			s.authed = true
			return nil
		}
		return smtp.ErrAuthFailed
	}), nil
}

func (s *Session) Mail(from string, opts *smtp.MailOptions) error { return nil }

func (s *Session) Rcpt(to string, opts *smtp.RcptOptions) error {
	s.to = to
	return nil
}

func (s *Session) Data(r io.Reader) error {
	log.Printf("SMTP: data revceived, auth status is %t", s.authed)
	if !s.authed {
		log.Printf("unauthorized attempt to send mail")
		return smtp.ErrAuthRequired
	}

	msg, err := io.ReadAll(r)
	if err != nil {
		return err
	}

	rec, err := s.bkd.GetRecord(s.to)
	if err != nil {
		log.Printf("No mapping for %s: %v", s.to, err)
		return nil
	}

	return s.forwardToApprise(rec, msg)
}

func (s *Session) forwardToApprise(rec *Record, raw []byte) error {
	m, err := mail.ReadMessage(bytes.NewReader(raw))
	var body []byte
	if err == nil {
		body, _ = io.ReadAll(m.Body)
	} else {
		body = raw
	}

	payload := map[string]string{
		"title": m.Header.Get("Subject"),
		"body":  string(body),
		"tag":   rec.Tags,
	}
	data, _ := json.Marshal(payload)

	url := fmt.Sprintf("%s/%s", s.bkd.AppriseURL, rec.Key)
	resp, err := http.Post(url, "application/json", bytes.NewReader(data))
	if err != nil {
		log.Printf("Apprise error: %v", err)
		return nil
	}
	defer resp.Body.Close()

	log.Printf("Forwarded message for %s to Apprise key %s", rec.Email, rec.Key)
	return nil
}

func (s *Session) Reset()        {}
func (s *Session) Logout() error { return nil }
