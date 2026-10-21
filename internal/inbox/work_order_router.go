package inbox

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"strings"
)

type ContactForm struct {
	WorkOrderID     string   `json:"work_order_id"`
	CustomerEmail   string   `json:"customer_email"`
	DispatchStatus  string   `json:"dispatch_status"`
	TechnicianNote  string   `json:"technician_follow_up"`
	PhotoReferences []string `json:"photo_references"`
	CaptchaToken    string   `json:"captcha_token"`
	WidgetRecordID  string   `json:"widget_record_id"`
}

type Router struct {
	client *Client
	inbox  string
}

func NewRouter(client *Client, inbox string) *Router {
	return &Router{client: client, inbox: inbox}
}

func (r *Router) Submit(ctx context.Context, form ContactForm, clientIP string) (string, error) {
	if form.WorkOrderID == "" || form.CustomerEmail == "" || form.CaptchaToken == "" || form.WidgetRecordID == "" {
		return "", &APIError{Code: "invalid_form", Message: "work_order_id, customer_email, captcha_token, and widget_record_id are required", Status: http.StatusBadRequest}
	}
	if err := r.client.VerifyCaptcha(ctx, form.WidgetRecordID, form.CaptchaToken, clientIP); err != nil {
		return "", err
	}
	return r.client.SendInboxEmail(ctx, r.inbox, "Work order "+form.WorkOrderID+" needs review", renderWorkOrder(form), form.WorkOrderID)
}

func renderWorkOrder(form ContactForm) string {
	photos := "none"
	if len(form.PhotoReferences) > 0 {
		photos = strings.Join(form.PhotoReferences, ", ")
	}
	return fmt.Sprintf("work_order_id: %s\ncustomer_email: %s\ndispatch_status: %s\ntechnician_follow_up: %s\nwork_order_photos: %s\n", form.WorkOrderID, form.CustomerEmail, form.DispatchStatus, form.TechnicianNote, photos)
}

func ClientIP(req *http.Request) string {
	host, _, err := net.SplitHostPort(req.RemoteAddr)
	if err != nil {
		return req.RemoteAddr
	}
	return host
}
