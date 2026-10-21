package inbox

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestSubmitRoutesVerifiedWorkOrderToInbox(t *testing.T) {
	tests := []struct {
		name      string
		form      ContactForm
		wantID    string
		wantCalls int
		wantErr   bool
	}{
		{
			name:   "verified work order reaches dispatch inbox",
			form:   ContactForm{WorkOrderID: "WO-104", CustomerEmail: "sam@example.com", DispatchStatus: "dispatched", TechnicianNote: "Call after inspection", PhotoReferences: []string{"photo-1.jpg"}, CaptchaToken: "captcha-token", WidgetRecordID: "widget-104"},
			wantID: "msg-104", wantCalls: 2,
		},
		{
			name:      "incomplete form does not call either capability",
			form:      ContactForm{CustomerEmail: "sam@example.com", CaptchaToken: "captcha-token"},
			wantCalls: 0, wantErr: true,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var emailBody emailRequest
			calls := 0
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
				calls++
				switch req.URL.Path {
				case "/v1/captcha/verify":
					var captchaBody captchaRequest
					if err := json.NewDecoder(req.Body).Decode(&captchaBody); err != nil || captchaBody.WidgetRecordID != "widget-104" || captchaBody.Token != "captcha-token" {
						t.Errorf("captcha payload = %+v, error = %v", captchaBody, err)
					}
					if req.Method != http.MethodPost {
						t.Errorf("captcha method = %s", req.Method)
					}
					w.Write([]byte(`{"ok":true,"data":{}}`))
				case "/v1/email/send":
					json.NewDecoder(req.Body).Decode(&emailBody)
					if got := req.Header.Get("Idempotency-Key"); got != "WO-104" {
						t.Errorf("idempotency key = %q", got)
					}
					w.Write([]byte(`{"ok":true,"data":{"message_id":"msg-104"}}`))
				default:
					t.Errorf("unexpected path %s", req.URL.Path)
				}
			}))
			defer server.Close()

			router := NewRouter(NewClientWith(server.URL, "test-key", server.Client()), "dispatch@example.com")
			messageID, err := router.Submit(context.Background(), test.form, "203.0.113.4")
			if (err != nil) != test.wantErr {
				t.Fatalf("error = %v, want error %t", err, test.wantErr)
			}
			if messageID != test.wantID || calls != test.wantCalls {
				t.Fatalf("message_id = %q, calls = %d", messageID, calls)
			}
			if !test.wantErr && (emailBody.To != "dispatch@example.com" || !strings.Contains(emailBody.Body, "dispatch_status: dispatched") || !strings.Contains(emailBody.Body, "work_order_photos: photo-1.jpg")) {
				t.Fatalf("email did not preserve work-order decision: %+v", emailBody)
			}
		})
	}
}
