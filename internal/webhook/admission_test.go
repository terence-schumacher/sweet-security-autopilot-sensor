package webhook

import (
	"testing"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/invisible-tech/autopilot-security-sensor/internal/config"
)

func TestShouldSkipInjection_ExcludedNamespace(t *testing.T) {
	cfg := config.WebhookConfig{ExcludeNamespaces: []string{"kube-system", "default"}}
	pod := &corev1.Pod{ObjectMeta: metav1.ObjectMeta{Name: "test"}}
	if !ShouldSkipInjection(cfg, pod, "kube-system") {
		t.Error("expected skip for kube-system")
	}
	if !ShouldSkipInjection(cfg, pod, "default") {
		t.Error("expected skip for default")
	}
	if ShouldSkipInjection(cfg, pod, "app-ns") {
		t.Error("expected no skip for app-ns")
	}
}

func TestShouldSkipInjection_AlreadyInjected(t *testing.T) {
	cfg := config.WebhookConfig{ExcludeNamespaces: []string{}}
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{Name: "test"},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{Name: "app"},
				{Name: "apss-agent"},
			},
		},
	}
	if !ShouldSkipInjection(cfg, pod, "default") {
		t.Error("expected skip when apss-agent already present")
	}
}

func TestShouldSkipInjection_AnnotationFalse(t *testing.T) {
	cfg := config.WebhookConfig{ExcludeNamespaces: []string{}}
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:        "test",
			Annotations: map[string]string{"apss.invisible.tech/inject": "false"},
		},
	}
	if !ShouldSkipInjection(cfg, pod, "default") {
		t.Error("expected skip when annotation inject=false")
	}
}

func TestShouldSkipInjection_HostNetwork(t *testing.T) {
	cfg := config.WebhookConfig{ExcludeNamespaces: []string{}}
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{Name: "test"},
		Spec:       corev1.PodSpec{HostNetwork: true},
	}
	if !ShouldSkipInjection(cfg, pod, "default") {
		t.Error("expected skip for hostNetwork")
	}
}

func TestCreateSidecarPatches(t *testing.T) {
	cfg := config.WebhookConfig{
		SidecarImage:       "apss-agent:test",
		ControllerEndpoint: "controller:8080",
	}
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{Name: "my-pod", Namespace: "default"},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{{Name: "app", Image: "app:latest"}},
		},
	}
	patches := CreateSidecarPatches(cfg, pod)
	if len(patches) < 4 {
		t.Errorf("expected at least 4 patches (container, volume, shareProcessNamespace, annotation), got %d", len(patches))
	}
	// First patch: add sidecar container
	if patches[0].Op != "add" || patches[0].Path != "/spec/containers/-" {
		t.Errorf("first patch: op=%q path=%q", patches[0].Op, patches[0].Path)
	}
	// Sidecar container value
	sidecar, ok := patches[0].Value.(corev1.Container)
	if !ok {
		t.Fatalf("first patch value is not Container: %T", patches[0].Value)
	}
	if sidecar.Name != "apss-agent" || sidecar.Image != "apss-agent:test" {
		t.Errorf("sidecar: Name=%q Image=%q", sidecar.Name, sidecar.Image)
	}
}

func TestCreateSidecarPatches_PodWithVolumes(t *testing.T) {
	cfg := config.WebhookConfig{SidecarImage: "agent:test", ControllerEndpoint: "ctrl:8080"}
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{Name: "p", Namespace: "ns"},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{{Name: "app"}},
			Volumes:    []corev1.Volume{{Name: "data"}},
		},
	}
	patches := CreateSidecarPatches(cfg, pod)
	// Should add volume with path /spec/volumes/- (append)
	foundVolumePatch := false
	for _, p := range patches {
		if p.Path == "/spec/volumes/-" && p.Op == "add" {
			foundVolumePatch = true
			break
		}
	}
	if !foundVolumePatch {
		t.Error("expected patch for /spec/volumes/- when pod already has volumes")
	}
}
