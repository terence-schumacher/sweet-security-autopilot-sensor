package webhook

import (
	"encoding/json"
	"testing"

	"github.com/sirupsen/logrus"
	admissionv1 "k8s.io/api/admission/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"

	"github.com/invisible-tech/autopilot-security-sensor/internal/config"
)

func TestProcessAdmissionReview_NonPod(t *testing.T) {
	log := logrus.New()
	cfg := config.DefaultWebhookConfig()
	review := admissionv1.AdmissionReview{
		Request: &admissionv1.AdmissionRequest{
			UID:  "req-1",
			Kind: metav1.GroupVersionKind{Kind: "Deployment"},
			Object: runtime.RawExtension{
				Raw: []byte(`{}`),
			},
		},
	}
	body, _ := json.Marshal(review)
	respBody, err := ProcessAdmissionReview(body, cfg, log)
	if err != nil {
		t.Fatalf("ProcessAdmissionReview: %v", err)
	}
	var resp admissionv1.AdmissionReview
	if err := json.Unmarshal(respBody, &resp); err != nil {
		t.Fatalf("Unmarshal response: %v", err)
	}
	if resp.Response == nil {
		t.Fatal("response nil")
	}
	if !resp.Response.Allowed {
		t.Error("expected Allowed=true for non-Pod")
	}
	if resp.Response.UID != "req-1" {
		t.Errorf("response UID = %q", resp.Response.UID)
	}
}

func TestProcessAdmissionReview_Pod_Inject(t *testing.T) {
	log := logrus.New()
	cfg := config.DefaultWebhookConfig()
	pod := corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{Name: "test-pod", Namespace: "app"},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{{Name: "app", Image: "app:latest"}},
		},
	}
	podRaw, _ := json.Marshal(pod)
	review := admissionv1.AdmissionReview{
		Request: &admissionv1.AdmissionRequest{
			UID:       "req-2",
			Kind:      metav1.GroupVersionKind{Kind: "Pod"},
			Namespace: "app",
			Object:    runtime.RawExtension{Raw: podRaw},
		},
	}
	body, _ := json.Marshal(review)
	respBody, err := ProcessAdmissionReview(body, cfg, log)
	if err != nil {
		t.Fatalf("ProcessAdmissionReview: %v", err)
	}
	var resp admissionv1.AdmissionReview
	if err := json.Unmarshal(respBody, &resp); err != nil {
		t.Fatalf("Unmarshal response: %v", err)
	}
	if !resp.Response.Allowed {
		t.Errorf("expected Allowed=true, Result=%v", resp.Response.Result)
	}
	if len(resp.Response.Patch) == 0 {
		t.Error("expected non-empty Patch")
	}
}

func TestProcessAdmissionReview_NoRequest(t *testing.T) {
	log := logrus.New()
	cfg := config.DefaultWebhookConfig()
	review := admissionv1.AdmissionReview{}
	body, _ := json.Marshal(review)
	_, err := ProcessAdmissionReview(body, cfg, log)
	if err == nil {
		t.Error("expected error when Request is nil")
	}
}

func TestProcessAdmissionReview_InvalidJSON(t *testing.T) {
	log := logrus.New()
	cfg := config.DefaultWebhookConfig()
	_, err := ProcessAdmissionReview([]byte("not json"), cfg, log)
	if err == nil {
		t.Error("expected error for invalid JSON")
	}
}

func TestProcessAdmissionReview_Pod_InvalidPodJSON(t *testing.T) {
	log := logrus.New()
	cfg := config.DefaultWebhookConfig()
	review := admissionv1.AdmissionReview{
		Request: &admissionv1.AdmissionRequest{
			UID:       "req-1",
			Kind:      metav1.GroupVersionKind{Kind: "Pod"},
			Namespace: "default",
			Object:    runtime.RawExtension{Raw: []byte("not a pod")},
		},
	}
	body, _ := json.Marshal(review)
	respBody, err := ProcessAdmissionReview(body, cfg, log)
	if err != nil {
		t.Fatalf("ProcessAdmissionReview: %v", err)
	}
	var resp admissionv1.AdmissionReview
	if err := json.Unmarshal(respBody, &resp); err != nil {
		t.Fatalf("Unmarshal response: %v", err)
	}
	if resp.Response.Allowed {
		t.Error("expected Allowed=false when pod JSON invalid")
	}
	if resp.Response.Result == nil || resp.Response.Result.Message == "" {
		t.Error("expected Result with Message")
	}
}
