// Package webhook provides the mutating admission webhook logic for
// injecting the APSS sidecar into pods.
package webhook

import (
	"fmt"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"

	"github.com/invisible-tech/autopilot-security-sensor/internal/config"
)

// PatchOperation represents a JSON patch operation (RFC 6902).
type PatchOperation struct {
	Op    string      `json:"op"`
	Path  string      `json:"path"`
	Value interface{} `json:"value,omitempty"`
}

// ShouldSkipInjection returns true if the pod/namespace should not receive the sidecar.
func ShouldSkipInjection(cfg config.WebhookConfig, pod *corev1.Pod, namespace string) bool {
	for _, ns := range cfg.ExcludeNamespaces {
		if namespace == ns {
			return true
		}
	}
	for _, c := range pod.Spec.Containers {
		if c.Name == "apss-agent" {
			return true
		}
	}
	if pod.Annotations != nil {
		if val, ok := pod.Annotations["apss.invisible.tech/inject"]; ok && val == "false" {
			return true
		}
	}
	if pod.Spec.HostNetwork {
		return true
	}
	return false
}

// CreateSidecarPatches returns JSON patch operations to inject the APSS sidecar.
func CreateSidecarPatches(cfg config.WebhookConfig, pod *corev1.Pod) []PatchOperation {
	var patches []PatchOperation

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
			{Name: "POD_NAME", ValueFrom: &corev1.EnvVarSource{FieldRef: &corev1.ObjectFieldSelector{FieldPath: "metadata.name"}}},
			{Name: "POD_NAMESPACE", ValueFrom: &corev1.EnvVarSource{FieldRef: &corev1.ObjectFieldSelector{FieldPath: "metadata.namespace"}}},
			{Name: "NODE_NAME", ValueFrom: &corev1.EnvVarSource{FieldRef: &corev1.ObjectFieldSelector{FieldPath: "spec.nodeName"}}},
			{Name: "AGENT_ID", Value: fmt.Sprintf("%s-%s", pod.Name, pod.Namespace)},
			{Name: "CONTROLLER_ENDPOINT", Value: cfg.ControllerEndpoint},
		},
		SecurityContext: &corev1.SecurityContext{
			RunAsNonRoot:             boolPtr(true),
			ReadOnlyRootFilesystem:   boolPtr(true),
			AllowPrivilegeEscalation: boolPtr(false),
			Capabilities:             &corev1.Capabilities{Drop: []corev1.Capability{"ALL"}},
		},
		VolumeMounts: []corev1.VolumeMount{
			{Name: "apss-proc", MountPath: "/proc", ReadOnly: true},
		},
	}

	patches = append(patches, PatchOperation{Op: "add", Path: "/spec/containers/-", Value: sidecar})

	procVolume := corev1.Volume{
		Name: "apss-proc",
		VolumeSource: corev1.VolumeSource{
			EmptyDir: &corev1.EmptyDirVolumeSource{Medium: "Memory"},
		},
	}
	if len(pod.Spec.Volumes) == 0 {
		patches = append(patches, PatchOperation{Op: "add", Path: "/spec/volumes", Value: []corev1.Volume{procVolume}})
	} else {
		patches = append(patches, PatchOperation{Op: "add", Path: "/spec/volumes/-", Value: procVolume})
	}

	if pod.Spec.ShareProcessNamespace == nil || !*pod.Spec.ShareProcessNamespace {
		patches = append(patches, PatchOperation{Op: "add", Path: "/spec/shareProcessNamespace", Value: true})
	}

	if pod.Annotations == nil {
		patches = append(patches, PatchOperation{
			Op: "add", Path: "/metadata/annotations",
			Value: map[string]string{"apss.invisible.tech/injected": "true"},
		})
	} else {
		patches = append(patches, PatchOperation{
			Op: "add", Path: "/metadata/annotations/apss.invisible.tech~1injected", Value: "true",
		})
	}

	return patches
}

func boolPtr(b bool) *bool {
	return &b
}
