package internal

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime"
	"mime/multipart"
	"net/http"
	"net/mail"
	"os"
	"strings"
	"time"

	"github.com/fraddy91/smtp-to-apprise/logger"

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
	logger.Debugf("SMTP: client requested auth mechanism=%s", mech)
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
	logger.Debugf("SMTP: data received, auth status is %t", s.authed)
	if !s.authed {
		logger.Warnf("unauthorized attempt to send mail")
		return smtp.ErrAuthRequired
	}

	msg, err := io.ReadAll(r)
	if err != nil {
		return err
	}

	rec, err := s.bkd.GetRecords(s.to)
	if err != nil {
		logger.Errorf("No mapping for %s: %v", s.to, err)
		return fmt.Errorf("no mapping for recipient %s: %w", s.to, err)
	}

	if err := s.forwardToApprise(rec, msg); err != nil {
		logger.Errorf("forwardToApprise error: %v", err)
		return err
	}
	return nil
}

func (s *Session) forwardToApprise(records []*Record, raw []byte) error {
	parts, m, err := extractParts(raw)
	if err != nil {
		logger.Errorf("Parse error: %v", err)
		return err
	}

	var lastErr error
	for _, rec := range records {
		body, ok := parts[rec.MimeType]
		if !ok {
			logger.Errorf("No %s part for %s/%s", rec.MimeType, rec.Email, rec.Key)
			lastErr = fmt.Errorf("missing part for %s", rec.MimeType)
			continue
		}

		format := "html"
		if rec.MimeType == "text/plain" {
			format = "text"
		}
		payload := map[string]string{
			"title":  m.Header.Get("Subject"),
			"body":   body,
			"tag":    rec.Tags,
			"format": format,
		}
		data, err := json.Marshal(payload)
		if err != nil {
			logger.Errorf("JSON marshal error: %v", err)
			lastErr = err
			continue
		}

		url := fmt.Sprintf("%s/%s", s.bkd.AppriseURL, rec.Key)
		s.bkd.Dispatcher.Enqueue(url, data, rec)
	}
	return lastErr
}

func postWithRetry(url string, data []byte, maxAttempts int) error {
	var err error
	backoff := time.Second
	for attempt := 1; attempt <= maxAttempts; attempt++ {
		resp, e := http.Post(url, "application/json", bytes.NewReader(data))
		if e == nil && resp.StatusCode >= 200 && resp.StatusCode < 300 {
			resp.Body.Close()
			return nil
		}
		if e != nil {
			err = e
		} else {
			err = fmt.Errorf("apprise returned %s", resp.Status)
			resp.Body.Close()
		}
		logger.Warnf("Apprise send failed (attempt %d/%d): %v", attempt, maxAttempts, err)
		time.Sleep(backoff)
		if backoff < 30*time.Second {
			backoff *= 2
		}
	}
	return fmt.Errorf("all retries failed: %w", err)
}

func extractParts(raw []byte) (map[string]string, *mail.Message, error) {
	m, err := mail.ReadMessage(bytes.NewReader(raw))
	if err != nil {
		return nil, nil, err
	}

	parts := make(map[string]string)
	mediaType, params, err := mime.ParseMediaType(m.Header.Get("Content-Type"))
	if err != nil {
		// fallback: treat whole body as plain
		body, _ := io.ReadAll(m.Body)
		parts["text/plain"] = string(body)
		return parts, m, nil
	}

	if strings.HasPrefix(mediaType, "multipart/") {
		mr := multipart.NewReader(m.Body, params["boundary"])
		for {
			p, err := mr.NextPart()
			if err == io.EOF {
				break
			}
			if err != nil {
				break
			}
			ctype := p.Header.Get("Content-Type")
			b, _ := io.ReadAll(p)
			if strings.HasPrefix(ctype, "text/plain") {
				parts["text/plain"] = string(b)
			} else if strings.HasPrefix(ctype, "text/html") {
				parts["text/html"] = string(b)
			}
		}
	} else {
		body, _ := io.ReadAll(m.Body)
		parts[mediaType] = string(body)
	}

	// Always keep raw
	parts["multipart"] = string(raw)
	return parts, m, nil
}

func (s *Session) Reset()        {}
func (s *Session) Logout() error { return nil }
