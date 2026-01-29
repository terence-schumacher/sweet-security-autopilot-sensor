package webhook

import (
	"encoding/json"
	"fmt"

	"github.com/sirupsen/logrus"
	admissionv1 "k8s.io/api/admission/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/invisible-tech/autopilot-security-sensor/internal/config"
)

// ProcessAdmissionReview decodes the admission review request, applies webhook logic,
// and returns the response body (AdmissionReview with Response set).
func ProcessAdmissionReview(body []byte, cfg config.WebhookConfig, log *logrus.Logger) ([]byte, error) {
	var review admissionv1.AdmissionReview
	if err := json.Unmarshal(body, &review); err != nil {
		return nil, fmt.Errorf("decode admission review: %w", err)
	}
	if review.Request == nil {
		return nil, fmt.Errorf("admission review has no request")
	}

	response := processRequest(review.Request, cfg, log)
	review.Response = response
	review.Response.UID = review.Request.UID

	return json.Marshal(review)
}

func processRequest(req *admissionv1.AdmissionRequest, cfg config.WebhookConfig, log *logrus.Logger) *admissionv1.AdmissionResponse {
	if req.Kind.Kind != "Pod" {
		return &admissionv1.AdmissionResponse{Allowed: true}
	}

	var pod corev1.Pod
	if err := json.Unmarshal(req.Object.Raw, &pod); err != nil {
		log.WithError(err).Error("Failed to unmarshal pod")
		return &admissionv1.AdmissionResponse{
			Allowed: false,
			Result:  &metav1.Status{Message: fmt.Sprintf("Failed to unmarshal pod: %v", err)},
		}
	}

	log.WithFields(logrus.Fields{"pod": pod.Name, "namespace": req.Namespace}).Debug("Processing pod admission")

	if ShouldSkipInjection(cfg, &pod, req.Namespace) {
		log.WithFields(logrus.Fields{"pod": pod.Name, "namespace": req.Namespace}).Debug("Skipping sidecar injection")
		return &admissionv1.AdmissionResponse{Allowed: true}
	}

	patches := CreateSidecarPatches(cfg, &pod)
	patchBytes, err := json.Marshal(patches)
	if err != nil {
		log.WithError(err).Error("Failed to marshal patches")
		return &admissionv1.AdmissionResponse{
			Allowed: false,
			Result:  &metav1.Status{Message: fmt.Sprintf("Failed to marshal patches: %v", err)},
		}
	}

	log.WithFields(logrus.Fields{"pod": pod.Name, "namespace": req.Namespace, "patches": len(patches)}).Info("Injecting APSS sidecar")

	patchType := admissionv1.PatchTypeJSONPatch
	return &admissionv1.AdmissionResponse{
		Allowed:   true,
		Patch:     patchBytes,
		PatchType: &patchType,
	}
}
