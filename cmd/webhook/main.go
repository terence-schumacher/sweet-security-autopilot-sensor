package main

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/sirupsen/logrus"
	admissionv1 "k8s.io/api/admission/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
)

var (
	log            = logrus.New()
	runtimeScheme  = runtime.NewScheme()
	codecs         = serializer.NewCodecFactory(runtimeScheme)
	deserializer   = codecs.UniversalDeserializer()
)

// WebhookConfig holds configuration for the webhook
type WebhookConfig struct {
	SidecarImage       string
	ControllerEndpoint string
	ExcludeNamespaces  []string
	ExcludeLabels      map[string]string
}

// patchOperation represents a JSON patch operation
type patchOperation struct {
	Op    string      `json:"op"`
	Path  string      `json:"path"`
	Value interface{} `json:"value,omitempty"`
}

func main() {
	log.SetFormatter(&logrus.JSONFormatter{})
	log.SetLevel(logrus.InfoLevel)

	cfg := &WebhookConfig{
		SidecarImage:       getEnv("SIDECAR_IMAGE", "gcr.io/invisible-sre-sandbox/apss-agent:latest"),
		ControllerEndpoint: getEnv("CONTROLLER_ENDPOINT", "apss-controller.apss-system.svc.cluster.local:8080"),
		ExcludeNamespaces: strings.Split(getEnv("EXCLUDE_NAMESPACES", "kube-system,kube-public,apss-system"), ","),
	}

	// Setup HTTP server
	mux := http.NewServeMux()
	mux.HandleFunc("/mutate", func(w http.ResponseWriter, r *http.Request) {
		handleMutate(w, r, cfg)
	})
	mux.HandleFunc("/health", handleHealth)

	// Load TLS certificates
	certFile := getEnv("TLS_CERT_FILE", "/etc/webhook/certs/tls.crt")
	keyFile := getEnv("TLS_KEY_FILE", "/etc/webhook/certs/tls.key")

	cert, err := tls.LoadX509KeyPair(certFile, keyFile)
	if err != nil {
		log.WithError(err).Fatal("Failed to load TLS certificates")
	}

	server := &http.Server{
		Addr:    ":8443",
		Handler: mux,
		TLSConfig: &tls.Config{
			Certificates: []tls.Certificate{cert},
		},
	}

	// Graceful shutdown
	go func() {
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
		<-sigChan
		log.Info("Shutting down webhook server")
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		server.Shutdown(ctx)
	}()

	log.Info("Starting APSS webhook server on :8443")
	if err := server.ListenAndServeTLS("", ""); err != http.ErrServerClosed {
		log.WithError(err).Fatal("Server failed")
	}
}

// handleHealth handles health check requests
func handleHealth(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
}

// handleMutate handles admission review requests
func handleMutate(w http.ResponseWriter, r *http.Request, cfg *WebhookConfig) {
	// Read request body
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Failed to read request body", http.StatusBadRequest)
		return
	}

	// Decode admission review
	var admissionReview admissionv1.AdmissionReview
	if _, _, err := deserializer.Decode(body, nil, &admissionReview); err != nil {
		log.WithError(err).Error("Failed to decode admission review")
		http.Error(w, "Failed to decode request", http.StatusBadRequest)
		return
	}

	// Process the admission request
	response := processAdmission(admissionReview.Request, cfg)

	// Build response
	admissionReview.Response = response
	admissionReview.Response.UID = admissionReview.Request.UID

	respBytes, err := json.Marshal(admissionReview)
	if err != nil {
		log.WithError(err).Error("Failed to marshal response")
		http.Error(w, "Failed to marshal response", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(respBytes)
}

// processAdmission processes an admission request and returns a response
func processAdmission(request *admissionv1.AdmissionRequest, cfg *WebhookConfig) *admissionv1.AdmissionResponse {
	// Only handle Pod creation
	if request.Kind.Kind != "Pod" {
		return &admissionv1.AdmissionResponse{Allowed: true}
	}

	// Decode the pod
	var pod corev1.Pod
	if err := json.Unmarshal(request.Object.Raw, &pod); err != nil {
		log.WithError(err).Error("Failed to unmarshal pod")
		return &admissionv1.AdmissionResponse{
			Allowed: false,
			Result: &metav1.Status{
				Message: fmt.Sprintf("Failed to unmarshal pod: %v", err),
			},
		}
	}

	log.WithFields(logrus.Fields{
		"pod":       pod.Name,
		"namespace": request.Namespace,
	}).Debug("Processing pod admission")

	// Check if we should skip injection
	if shouldSkipInjection(&pod, request.Namespace, cfg) {
		log.WithFields(logrus.Fields{
			"pod":       pod.Name,
			"namespace": request.Namespace,
		}).Debug("Skipping sidecar injection")
		return &admissionv1.AdmissionResponse{Allowed: true}
	}

	// Generate patches to inject sidecar
	patches := createSidecarPatches(&pod, cfg)

	patchBytes, err := json.Marshal(patches)
	if err != nil {
		log.WithError(err).Error("Failed to marshal patches")
		return &admissionv1.AdmissionResponse{
			Allowed: false,
			Result: &metav1.Status{
				Message: fmt.Sprintf("Failed to marshal patches: %v", err),
			},
		}
	}

	log.WithFields(logrus.Fields{
		"pod":       pod.Name,
		"namespace": request.Namespace,
		"patches":   len(patches),
	}).Info("Injecting APSS sidecar")

	patchType := admissionv1.PatchTypeJSONPatch
	return &admissionv1.AdmissionResponse{
		Allowed:   true,
		Patch:     patchBytes,
		PatchType: &patchType,
	}
}

// shouldSkipInjection determines if sidecar injection should be skipped
func shouldSkipInjection(pod *corev1.Pod, namespace string, cfg *WebhookConfig) bool {
	// Skip excluded namespaces
	for _, ns := range cfg.ExcludeNamespaces {
		if namespace == ns {
			return true
		}
	}

	// Skip if already injected
	for _, c := range pod.Spec.Containers {
		if c.Name == "apss-agent" {
			return true
		}
	}

	// Skip if explicitly disabled via annotation
	if pod.Annotations != nil {
		if val, ok := pod.Annotations["apss.invisible.tech/inject"]; ok && val == "false" {
			return true
		}
	}

	// Skip pods with hostNetwork (we can't monitor those effectively anyway)
	if pod.Spec.HostNetwork {
		return true
	}

	return false
}

// createSidecarPatches creates JSON patches to inject the sidecar
func createSidecarPatches(pod *corev1.Pod, cfg *WebhookConfig) []patchOperation {
	var patches []patchOperation

	// Create sidecar container
	sidecar := corev1.Container{
		Name:  "apss-agent",
		Image: cfg.SidecarImage,
		Resources: corev1.ResourceRequirements{
			Requests: corev1.ResourceList{
				corev1.ResourceCPU:    resource.MustParse("10m"),
				corev1.ResourceMemory: resource.MustParse("32Mi"),
			},
			Limits: corev1.ResourceList{
				corev1.ResourceCPU:    resource.MustParse("100m"),
				corev1.ResourceMemory: resource.MustParse("128Mi"),
			},
		},
		Env: []corev1.EnvVar{
			{
				Name: "POD_NAME",
				ValueFrom: &corev1.EnvVarSource{
					FieldRef: &corev1.ObjectFieldSelector{
						FieldPath: "metadata.name",
					},
				},
			},
			{
				Name: "POD_NAMESPACE",
				ValueFrom: &corev1.EnvVarSource{
					FieldRef: &corev1.ObjectFieldSelector{
						FieldPath: "metadata.namespace",
					},
				},
			},
			{
				Name: "NODE_NAME",
				ValueFrom: &corev1.EnvVarSource{
					FieldRef: &corev1.ObjectFieldSelector{
						FieldPath: "spec.nodeName",
					},
				},
			},
			{
				Name:  "AGENT_ID",
				Value: fmt.Sprintf("%s-%s", pod.Name, pod.Namespace),
			},
			{
				Name:  "CONTROLLER_ENDPOINT",
				Value: cfg.ControllerEndpoint,
			},
		},
		SecurityContext: &corev1.SecurityContext{
			// Autopilot-compatible security context
			RunAsNonRoot:             boolPtr(true),
			ReadOnlyRootFilesystem:   boolPtr(true),
			AllowPrivilegeEscalation: boolPtr(false),
			Capabilities: &corev1.Capabilities{
				Drop: []corev1.Capability{"ALL"},
			},
		},
		VolumeMounts: []corev1.VolumeMount{
			{
				Name:      "apss-proc",
				MountPath: "/proc",
				ReadOnly:  true,
			},
		},
	}

	// Patch to add sidecar container
	patches = append(patches, patchOperation{
		Op:    "add",
		Path:  "/spec/containers/-",
		Value: sidecar,
	})

	// Note: On Autopilot, we can't mount /proc from host, but with shareProcessNamespace
	// we can access /proc from within the pod namespace. The volume mount is for
	// compatibility but the actual /proc access works via shareProcessNamespace.
	// We'll use a tmpfs for any temporary files needed
	procVolume := corev1.Volume{
		Name: "apss-proc",
		VolumeSource: corev1.VolumeSource{
			EmptyDir: &corev1.EmptyDirVolumeSource{
				Medium: "Memory",
			},
		},
	}

	if len(pod.Spec.Volumes) == 0 {
		patches = append(patches, patchOperation{
			Op:    "add",
			Path:  "/spec/volumes",
			Value: []corev1.Volume{procVolume},
		})
	} else {
		patches = append(patches, patchOperation{
			Op:    "add",
			Path:  "/spec/volumes/-",
			Value: procVolume,
		})
	}

	// Enable shareProcessNamespace if not already set
	// This allows the sidecar to see processes in other containers
	if pod.Spec.ShareProcessNamespace == nil || !*pod.Spec.ShareProcessNamespace {
		patches = append(patches, patchOperation{
			Op:    "add",
			Path:  "/spec/shareProcessNamespace",
			Value: true,
		})
	}

	// Add annotation to mark pod as injected
	if pod.Annotations == nil {
		patches = append(patches, patchOperation{
			Op:    "add",
			Path:  "/metadata/annotations",
			Value: map[string]string{
				"apss.invisible.tech/injected": "true",
			},
		})
	} else {
		patches = append(patches, patchOperation{
			Op:    "add",
			Path:  "/metadata/annotations/apss.invisible.tech~1injected",
			Value: "true",
		})
	}

	return patches
}

// Helper functions
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func boolPtr(b bool) *bool {
	return &b
}

