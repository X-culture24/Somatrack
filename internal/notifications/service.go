package notifications

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/smtp"
	"os"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/hibiken/asynq"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/stmaryskabete/lms/internal/shared/httpx"
)

const (
	TaskSendSMS   = "notification:sms"
	TaskSendEmail = "notification:email"
)

type Queue struct {
	client *asynq.Client
	pool   *pgxpool.Pool
}

func NewQueue(pool *pgxpool.Pool, redisURL string) (*Queue, func(), error) {
	opt, err := asynq.ParseRedisURI(redisURL)
	if err != nil {
		return nil, nil, err
	}
	c := asynq.NewClient(opt)
	return &Queue{client: c, pool: pool}, func() { c.Close() }, nil
}

func (q *Queue) EnqueueSMS(ctx context.Context, to, message string, meta map[string]string) (uuid.UUID, error) {
	id := uuid.New()
	payload, _ := json.Marshal(map[string]interface{}{
		"id":      id.String(),
		"to":      to,
		"message": message,
		"meta":    meta,
	})
	_, err := q.client.EnqueueContext(ctx, asynq.NewTask(TaskSendSMS, payload,
		asynq.MaxRetry(5), asynq.Queue("notifications"), asynq.Timeout(30*time.Second)))
	if err != nil {
		return uuid.Nil, err
	}
	_, _ = q.pool.Exec(ctx, `
		INSERT INTO notification_logs (id, channel, recipient, content, status, meta, created_at)
		VALUES ($1, 'sms', $2, $3, 'queued', $4::jsonb, NOW())`, id, to, message, meta)
	return id, nil
}

func (q *Queue) EnqueueEmail(ctx context.Context, to []string, subject, htmlBody, textBody string, attachments map[string][]byte) (uuid.UUID, error) {
	id := uuid.New()
	recipient := strings.Join(to, ",")
	meta := map[string]interface{}{
		"attachments_count": len(attachments),
		"recipients":        len(to),
	}
	payload, _ := json.Marshal(map[string]interface{}{
		"id":        id.String(),
		"to":        to,
		"subject":   subject,
		"html_body": htmlBody,
		"text_body": textBody,
	})
	_, err := q.client.EnqueueContext(ctx, asynq.NewTask(TaskSendEmail, payload,
		asynq.MaxRetry(5), asynq.Queue("notifications"), asynq.Timeout(60*time.Second)))
	if err != nil {
		return uuid.Nil, err
	}
	_, _ = q.pool.Exec(ctx, `
		INSERT INTO notification_logs (id, channel, recipient, content, status, meta, subject, created_at)
		VALUES ($1, 'email', $2, $3, 'queued', $4::jsonb, $5, NOW())`, id, recipient, textBody, meta, subject)
	return id, nil
}

type SMSSender struct {
	Backend   string
	APIKey    string
	APISender string
	APIURL    string
}

func (s *SMSSender) Send(ctx context.Context, to, message string) error {
	if s.Backend == "console" || s.Backend == "" {
		fmt.Fprintf(os.Stderr, "[sms:console] TO=%s MSG=%s\n", to, message)
		return nil
	}
	// africastalking / twilio placeholder hooks
	log.Printf("[sms:%s] SEND %s: %s", s.Backend, to, message)
	return nil
}

type EmailSender struct {
	Backend      string
	SMTPHost     string
	SMTPPort     int
	SMTPUser     string
	SMTPPassword string
	FromEmail    string
	FromName     string
}

func (e *EmailSender) Send(ctx context.Context, to []string, subject, htmlBody, textBody string) error {
	if e.Backend == "console" || e.Backend == "" {
		fmt.Fprintf(os.Stderr, "[email:console] TO=%s SUBJECT=%s\nTEXT=%s\n",
			strings.Join(to, ","), subject, textBody)
		return nil
	}
	if e.Backend == "smtp" {
		return e.sendSMTP(to, subject, htmlBody, textBody)
	}
	log.Printf("[email:%s] SEND %s: %s", e.Backend, strings.Join(to, ","), subject)
	return nil
}

func (e *EmailSender) sendSMTP(to []string, subject, htmlBody, textBody string) error {
	var auth smtp.Auth
	if e.SMTPUser != "" {
		auth = smtp.PlainAuth("", e.SMTPUser, e.SMTPPassword, e.SMTPHost)
	}
	addr := fmt.Sprintf("%s:%d", e.SMTPHost, e.SMTPPort)
	msg := "From: " + e.FromEmail + "\r\n" +
		"To: " + strings.Join(to, ", ") + "\r\n" +
		"Subject: " + subject + "\r\n" +
		"MIME-Version: 1.0\r\n" +
		"Content-Type: multipart/alternative; boundary=stmarys-boundary\r\n\r\n" +
		"--stmarys-boundary\r\n" +
		"Content-Type: text/plain; charset=UTF-8\r\n\r\n" +
		textBody + "\r\n\r\n" +
		"--stmarys-boundary\r\n" +
		"Content-Type: text/html; charset=UTF-8\r\n\r\n" +
		htmlBody + "\r\n\r\n" +
		"--stmarys-boundary--"
	return smtp.SendMail(addr, auth, e.FromEmail, to, []byte(msg))
}

type Service struct {
	queue       *Queue
	sms         *SMSSender
	email       *EmailSender
	useAsync    bool
	pool        *pgxpool.Pool
	fromEmail   string
}

func NewService(queue *Queue, sms *SMSSender, email *EmailSender, useAsync bool, pool *pgxpool.Pool, fromEmail string) *Service {
	return &Service{queue: queue, sms: sms, email: email, useAsync: useAsync, pool: pool, fromEmail: fromEmail}
}

func (s *Service) RegisterInternalRoutes(r chi.Router) {
	r.Post("/sms/enqueue", s.handleEnqueueSMS)
	r.Post("/email/enqueue", s.handleEnqueueEmail)
	r.Post("/notifications/guardian", s.handleGuardianNotification)
}

func (s *Service) handleEnqueueSMS(w http.ResponseWriter, r *http.Request) {
	var body struct {
		To      string            `json:"to" validate:"required"`
		Message string            `json:"message" validate:"required"`
		Meta    map[string]string `json:"meta"`
	}
	if err := httpx.BindAndValidate(r, &body); err != nil {
		httpx.ErrBadRequest(w, err.Error())
		return
	}
	if s.useAsync && s.queue != nil {
		id, err := s.queue.EnqueueSMS(r.Context(), body.To, body.Message, body.Meta)
		if err != nil {
			_, err = s.pool.Exec(r.Context(), `
				INSERT INTO notification_logs (id, channel, recipient, content, status, error, meta, created_at)
				VALUES ($1,'sms',$2,$3,'failed',$4,$5::jsonb,NOW())`,
				uuid.New(), body.To, body.Message, err.Error(), body.Meta)
			httpx.Error(w, http.StatusInternalServerError, err)
			return
		}
		httpx.JSON(w, http.StatusOK, map[string]interface{}{"status": "queued", "id": id})
		return
	}
	if err := s.sms.Send(r.Context(), body.To, body.Message); err != nil {
		httpx.Error(w, http.StatusInternalServerError, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]string{"status": "sent"})
}

func (s *Service) handleEnqueueEmail(w http.ResponseWriter, r *http.Request) {
	var body struct {
		To          []string          `json:"to" validate:"required,min=1"`
		Subject     string            `json:"subject" validate:"required"`
		HTMLBody    string            `json:"html_body"`
		TextBody    string            `json:"text_body"`
		Attachments map[string][]byte `json:"attachments"`
	}
	if err := httpx.BindAndValidate(r, &body); err != nil {
		httpx.ErrBadRequest(w, err.Error())
		return
	}
	if body.TextBody == "" {
		body.TextBody = body.HTMLBody
	}
	if s.useAsync && s.queue != nil {
		id, err := s.queue.EnqueueEmail(r.Context(), body.To, body.Subject, body.HTMLBody, body.TextBody, body.Attachments)
		if err != nil {
			httpx.Error(w, http.StatusInternalServerError, err)
			return
		}
		httpx.JSON(w, http.StatusOK, map[string]interface{}{"status": "queued", "id": id})
		return
	}
	if err := s.email.Send(r.Context(), body.To, body.Subject, body.HTMLBody, body.TextBody); err != nil {
		httpx.Error(w, http.StatusInternalServerError, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]string{"status": "sent"})
}

func (s *Service) handleGuardianNotification(w http.ResponseWriter, r *http.Request) {
	var body struct {
		GuardianID   uuid.UUID         `json:"guardian_id" validate:"required,uuid"`
		StudentID    uuid.UUID         `json:"student_id" validate:"required,uuid"`
		TemplateKey  string            `json:"template_key" validate:"required"`
		TemplateData map[string]string `json:"template_data"`
		SMS          bool              `json:"sms"`
		Email        bool              `json:"email"`
	}
	if err := httpx.BindAndValidate(r, &body); err != nil {
		httpx.ErrBadRequest(w, err.Error())
		return
	}
	var phone, emailAddr string
	err := s.pool.QueryRow(r.Context(), `
		SELECT phone, email FROM guardians WHERE id = $1`, body.GuardianID).Scan(&phone, &emailAddr)
	if err != nil {
		httpx.ErrNotFound(w, "guardian not found")
		return
	}
	msg := renderTemplate(body.TemplateKey, body.TemplateData)
	if body.SMS && phone != "" {
		if s.useAsync && s.queue != nil {
			_, _ = s.queue.EnqueueSMS(r.Context(), phone, msg, map[string]string{
				"guardian_id":   body.GuardianID.String(),
				"student_id":    body.StudentID.String(),
				"template_key":  body.TemplateKey,
			})
		} else {
			_ = s.sms.Send(r.Context(), phone, msg)
		}
	}
	if body.Email && emailAddr != "" {
		subj := subjectFor(body.TemplateKey)
		if s.useAsync && s.queue != nil {
			_, _ = s.queue.EnqueueEmail(r.Context(), []string{emailAddr}, subj, "", msg, nil)
		} else {
			_ = s.email.Send(r.Context(), []string{emailAddr}, subj, "", msg)
		}
	}
	httpx.JSON(w, http.StatusOK, map[string]string{"status": "dispatched"})
}

func renderTemplate(key string, data map[string]string) string {
	out := ""
	switch key {
	case "payment_received":
		out = fmt.Sprintf("ACK St.Mary's: Payment of KES %s received for %s %s. Balance: KES %s.",
			data["amount"], data["student_first"], data["student_last"], data["balance"])
	case "fee_reminder":
		out = fmt.Sprintf("ACK St.Mary's: Reminder — %s %s has an outstanding balance of KES %s.",
			data["student_first"], data["student_last"], data["balance"])
	case "attendance_alert":
		out = fmt.Sprintf("ACK St.Mary's: Attendance alert for %s %s — %s at %s.",
			data["student_first"], data["student_last"], data["status"], data["time"])
	default:
		parts := make([]string, 0, len(data))
		for k, v := range data {
			parts = append(parts, k+": "+v)
		}
		out = "ACK St.Mary's — " + strings.Join(parts, " | ")
	}
	return out
}

func subjectFor(key string) string {
	switch key {
	case "payment_received":
		return "Payment Received — ACK St. Mary's School"
	case "fee_reminder":
		return "Fee Balance Reminder — ACK St. Mary's School"
	case "attendance_alert":
		return "Attendance Update — ACK St. Mary's School"
	}
	return "Notice from ACK St. Mary's School"
}

func WorkerHandler(sms *SMSSender, email *EmailSender, pool *pgxpool.Pool) asynq.Handler {
	mux := asynq.NewServeMux()
	mux.HandleFunc(TaskSendSMS, func(ctx context.Context, t *asynq.Task) error {
		var p struct {
			ID      string
			To      string
			Message string
		}
		_ = json.Unmarshal(t.Payload(), &p)
		err := sms.Send(ctx, p.To, p.Message)
		if p.ID != "" {
			status := "delivered"
			errStr := ""
			if err != nil {
				status = "failed"
				errStr = err.Error()
			}
			_, _ = pool.Exec(ctx, `
				UPDATE notification_logs SET status=$1, error=$2, delivered_at=NOW() WHERE id=$3`,
				status, errStr, p.ID)
		}
		return err
	})
	mux.HandleFunc(TaskSendEmail, func(ctx context.Context, t *asynq.Task) error {
		var p struct {
			ID        string
			To        []string
			Subject   string
			HTMLBody  string
			TextBody  string
		}
		_ = json.Unmarshal(t.Payload(), &p)
		if p.TextBody == "" {
			p.TextBody = p.HTMLBody
		}
		err := email.Send(ctx, p.To, p.Subject, p.HTMLBody, p.TextBody)
		if p.ID != "" {
			status := "delivered"
			errStr := ""
			if err != nil {
				status = "failed"
				errStr = err.Error()
			}
			_, _ = pool.Exec(ctx, `
				UPDATE notification_logs SET status=$1, error=$2, delivered_at=NOW() WHERE id=$3`,
				status, errStr, p.ID)
		}
		return err
	})
	return mux
}
