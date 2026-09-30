package k8s

import (
	"fmt"
	"sort"
	"strings"

	"k8s.io/apimachinery/pkg/runtime/schema"
)

// KindInfo describes one browsable resource kind.
type KindInfo struct {
	// Kind is the URL-friendly plural name used by the API and UI routes.
	Kind string `json:"kind"`
	// DisplayName is the human-readable singular name.
	DisplayName string `json:"displayName"`
	Namespaced  bool   `json:"namespaced"`
	GVR         schema.GroupVersionResource
}

// registry maps the kinds exposed in the UI to their GroupVersionResource.
var registry = map[string]KindInfo{
	"pods":                   {Kind: "pods", DisplayName: "Pod", Namespaced: true, GVR: schema.GroupVersionResource{Version: "v1", Resource: "pods"}},
	"deployments":            {Kind: "deployments", DisplayName: "Deployment", Namespaced: true, GVR: schema.GroupVersionResource{Group: "apps", Version: "v1", Resource: "deployments"}},
	"statefulsets":           {Kind: "statefulsets", DisplayName: "StatefulSet", Namespaced: true, GVR: schema.GroupVersionResource{Group: "apps", Version: "v1", Resource: "statefulsets"}},
	"daemonsets":             {Kind: "daemonsets", DisplayName: "DaemonSet", Namespaced: true, GVR: schema.GroupVersionResource{Group: "apps", Version: "v1", Resource: "daemonsets"}},
	"services":               {Kind: "services", DisplayName: "Service", Namespaced: true, GVR: schema.GroupVersionResource{Version: "v1", Resource: "services"}},
	"configmaps":             {Kind: "configmaps", DisplayName: "ConfigMap", Namespaced: true, GVR: schema.GroupVersionResource{Version: "v1", Resource: "configmaps"}},
	"secrets":                {Kind: "secrets", DisplayName: "Secret", Namespaced: true, GVR: schema.GroupVersionResource{Version: "v1", Resource: "secrets"}},
	"ingresses":              {Kind: "ingresses", DisplayName: "Ingress", Namespaced: true, GVR: schema.GroupVersionResource{Group: "networking.k8s.io", Version: "v1", Resource: "ingresses"}},
	"persistentvolumeclaims": {Kind: "persistentvolumeclaims", DisplayName: "PersistentVolumeClaim", Namespaced: true, GVR: schema.GroupVersionResource{Version: "v1", Resource: "persistentvolumeclaims"}},
	"jobs":                   {Kind: "jobs", DisplayName: "Job", Namespaced: true, GVR: schema.GroupVersionResource{Group: "batch", Version: "v1", Resource: "jobs"}},
	"cronjobs":               {Kind: "cronjobs", DisplayName: "CronJob", Namespaced: true, GVR: schema.GroupVersionResource{Group: "batch", Version: "v1", Resource: "cronjobs"}},
	"nodes":                  {Kind: "nodes", DisplayName: "Node", Namespaced: false, GVR: schema.GroupVersionResource{Version: "v1", Resource: "nodes"}},
}

// LookupKind resolves a URL kind segment to its registry entry.
func LookupKind(kind string) (KindInfo, error) {
	normalized := strings.ToLower(strings.TrimSpace(kind))
	info, ok := registry[normalized]
	if !ok {
		// LLMs and humans naturally use Kubernetes' singular Kind names
		// ("Deployment", "pod") while the HTTP API uses plural resources.
		// Resolve both forms here so every caller gets the same deterministic
		// behavior instead of forcing an agent to spend retries correcting it.
		for _, candidate := range registry {
			if strings.EqualFold(normalized, candidate.DisplayName) {
				info, ok = candidate, true
				break
			}
		}
	}
	if !ok {
		return KindInfo{}, fmt.Errorf("unsupported resource kind %q", kind)
	}
	return info, nil
}

// SupportedKinds returns all registered kinds sorted by name.
func SupportedKinds() []KindInfo {
	kinds := make([]KindInfo, 0, len(registry))
	for _, info := range registry {
		kinds = append(kinds, info)
	}
	sort.Slice(kinds, func(i, j int) bool { return kinds[i].Kind < kinds[j].Kind })
	return kinds
}
