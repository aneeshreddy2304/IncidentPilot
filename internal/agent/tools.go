package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"time"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"sigs.k8s.io/yaml"

	"incidentpilot.local/incidentpilot/internal/k8s"
	"incidentpilot.local/incidentpilot/internal/llm"
)

// The agent's cluster tools are strictly read-only: list, get, logs, events,
// overview, and (when Prometheus is configured) metrics. In assisted mode one
// more tool — propose_change — is added, but it too performs NO cluster write:
// it only records a proposal in the agent's own database for a human to
// approve. The privileged apply happens in the server, never here.
func queryTools(metricsEnabled, assisted bool) []llm.Tool {
	kinds := make([]string, 0)
	for _, k := range k8s.SupportedKinds() {
		kinds = append(kinds, k.Kind)
	}
	kindList := strings.Join(kinds, ", ")

	tools := []llm.Tool{
		{
			Name:        "get_cluster_overview",
			Description: "Get cluster-wide stats: node/pod/namespace/deployment counts, pod phases, and recent warning events. Use this first for broad questions.",
			Properties:  map[string]interface{}{},
		},
		{
			Name:        "list_resources",
			Description: "List resources of a kind with status columns. Supported kinds: " + kindList + ".",
			Properties: map[string]interface{}{
				"kind":      map[string]interface{}{"type": "string", "description": "resource kind (plural), e.g. pods"},
				"namespace": map[string]interface{}{"type": "string", "description": "namespace filter; empty = all namespaces"},
			},
			Required: []string{"kind"},
		},
		{
			Name:        "get_resource",
			Description: "Get one resource's full manifest as YAML, including status.",
			Properties: map[string]interface{}{
				"kind":      map[string]interface{}{"type": "string", "description": "resource kind (plural), e.g. deployments"},
				"namespace": map[string]interface{}{"type": "string", "description": "namespace (empty for cluster-scoped kinds)"},
				"name":      map[string]interface{}{"type": "string", "description": "resource name"},
			},
			Required: []string{"kind", "name"},
		},
		{
			Name:        "get_pod_logs",
			Description: "Fetch recent logs of a pod container. Use sinceSeconds to bound the time window (e.g. 600 = last 10 minutes).",
			Properties: map[string]interface{}{
				"namespace":    map[string]interface{}{"type": "string", "description": "pod namespace"},
				"pod":          map[string]interface{}{"type": "string", "description": "pod name"},
				"container":    map[string]interface{}{"type": "string", "description": "container name; empty = first container"},
				"tailLines":    map[string]interface{}{"type": "integer", "description": "number of trailing lines (default 200, max 1000)"},
				"sinceSeconds": map[string]interface{}{"type": "integer", "description": "only logs newer than this many seconds"},
				"previous":     map[string]interface{}{"type": "boolean", "description": "logs of the previous (crashed) container instance"},
			},
			Required: []string{"namespace", "pod"},
		},
		{
			Name:        "get_events",
			Description: "List Kubernetes events, newest first. Filter by namespace and/or type (Normal, Warning).",
			Properties: map[string]interface{}{
				"namespace": map[string]interface{}{"type": "string", "description": "namespace filter; empty = all"},
				"type":      map[string]interface{}{"type": "string", "description": "event type filter: Normal or Warning"},
			},
		},
	}

	if assisted {
		tools = append(tools, llm.Tool{
			Name:        "propose_container_image_change",
			Description: "Propose a corrected container image for a Deployment, StatefulSet, or DaemonSet. Use this for ImagePullBackOff, ErrImagePull, invalid image tags, or wrong registries. The server reads the live workload and constructs the full reviewable YAML diff; this does NOT apply the change.",
			Properties: map[string]interface{}{
				"kind":      map[string]interface{}{"type": "string", "description": "workload kind: deployments, statefulsets, or daemonsets (singular names are also accepted)"},
				"namespace": map[string]interface{}{"type": "string", "description": "namespace"},
				"name":      map[string]interface{}{"type": "string", "description": "workload name"},
				"container": map[string]interface{}{"type": "string", "description": "container name; may be empty only when the workload has exactly one container"},
				"image":     map[string]interface{}{"type": "string", "description": "complete corrected image reference, including tag"},
				"rationale": map[string]interface{}{"type": "string", "description": "brief evidence-backed explanation"},
			},
			Required: []string{"kind", "namespace", "name", "image", "rationale"},
		}, llm.Tool{
			Name: "propose_change",
			Description: "Propose a fix to ONE resource for the human to review and approve. " +
				"This does NOT apply anything — it queues a proposed change that the user must approve; the server applies it only after approval. " +
				"Use this when the user asks you to fix/change/update something. First call get_resource to read the current manifest, then supply the full modified manifest as proposedYaml. Only propose changes you are confident are correct and minimal.",
			Properties: map[string]interface{}{
				"kind":         map[string]interface{}{"type": "string", "description": "resource kind (plural), e.g. deployments"},
				"namespace":    map[string]interface{}{"type": "string", "description": "namespace (empty for cluster-scoped kinds)"},
				"name":         map[string]interface{}{"type": "string", "description": "resource name"},
				"proposedYaml": map[string]interface{}{"type": "string", "description": "the FULL modified manifest as YAML (not a patch). Its kind/name/namespace must match the target."},
				"rationale":    map[string]interface{}{"type": "string", "description": "a one or two sentence explanation of what this change does and why"},
			},
			Required: []string{"kind", "name", "proposedYaml", "rationale"},
		})
	}

	if metricsEnabled {
		tools = append(tools,
			llm.Tool{
				Name:        "get_resource_usage",
				Description: "Get actual CPU/memory usage: top pods by memory and CPU, node usage, and CPU-throttled containers. Use this for performance questions, sizing checks, and to distinguish memory leaks from undersized limits.",
				Properties: map[string]interface{}{
					"namespace": map[string]interface{}{"type": "string", "description": "namespace filter; empty = whole cluster"},
				},
			},
			llm.Tool{
				Name:        "query_metrics",
				Description: "Run a PromQL instant query against Prometheus. Use rate()/avg_over_time()/increase() with a time window for trends (e.g. rate(pod_cpu_usage_seconds_total{namespace=\"app\"}[5m])). Available metrics include pod_memory_working_set_bytes, pod_cpu_usage_seconds_total, node_memory_working_set_bytes, node_cpu_usage_seconds_total, container_cpu_cfs_throttled_periods_total, machine_memory_bytes.",
				Properties: map[string]interface{}{
					"query": map[string]interface{}{"type": "string", "description": "the PromQL expression"},
				},
				Required: []string{"query"},
			},
		)
	}
	return tools
}

// runProposeContainerImageChange creates a minimal image-only proposal. The
// model never has to reproduce a large manifest: trusted code reads the live
// object, changes exactly one container image, validates the result, and
// stores the same full before/after YAML used by the approval UI.
func runProposeContainerImageChange(ctx context.Context, client *k8s.Client, store *Store, call llm.ToolCall) (string, *Proposal, error) {
	var args struct {
		Kind      string `json:"kind"`
		Namespace string `json:"namespace"`
		Name      string `json:"name"`
		Container string `json:"container"`
		Image     string `json:"image"`
		Rationale string `json:"rationale"`
	}
	if err := json.Unmarshal(call.Input, &args); err != nil {
		return "", nil, fmt.Errorf("invalid tool arguments: %w", err)
	}
	if store == nil {
		return "", nil, fmt.Errorf("proposals are unavailable (no persistent store)")
	}
	if strings.TrimSpace(args.Image) == "" {
		return "", nil, fmt.Errorf("image is required")
	}
	info, err := k8s.LookupKind(args.Kind)
	if err != nil {
		return "", nil, err
	}
	switch info.Kind {
	case "deployments", "statefulsets", "daemonsets":
	default:
		return "", nil, fmt.Errorf("image-change proposals support deployments, statefulsets, and daemonsets; got %q", info.Kind)
	}

	current, err := client.GetResource(ctx, info.Kind, args.Namespace, args.Name)
	if err != nil {
		return "", nil, fmt.Errorf("reading current %s %s: %w", info.Kind, args.Name, err)
	}
	containers, found, err := unstructured.NestedSlice(current.Object, "spec", "template", "spec", "containers")
	if err != nil || !found || len(containers) == 0 {
		return "", nil, fmt.Errorf("workload has no editable containers")
	}
	containerName := args.Container
	if containerName == "" {
		if len(containers) != 1 {
			return "", nil, fmt.Errorf("container is required because the workload has %d containers", len(containers))
		}
		containerName, _, _ = unstructured.NestedString(containers[0].(map[string]interface{}), "name")
	}
	changed := false
	oldImage := ""
	for i := range containers {
		container, ok := containers[i].(map[string]interface{})
		if !ok {
			continue
		}
		name, _, _ := unstructured.NestedString(container, "name")
		if name != containerName {
			continue
		}
		oldImage, _, _ = unstructured.NestedString(container, "image")
		if canonicalImageReference(oldImage) == canonicalImageReference(args.Image) {
			return "", nil, fmt.Errorf("proposed image %q resolves to the same repository and tag as current image %q; supply a genuinely corrected tag or repository", args.Image, oldImage)
		}
		if err := publicDockerHubImageValidator(ctx, args.Image); err != nil {
			return "", nil, err
		}
		container["image"] = args.Image
		containers[i] = container
		changed = true
		break
	}
	if !changed {
		return "", nil, fmt.Errorf("container %q was not found", containerName)
	}
	if strings.TrimSpace(args.Rationale) == "" {
		args.Rationale = fmt.Sprintf("Change container %s from %s to the registry-verified image %s.", containerName, oldImage, args.Image)
	}
	if err := unstructured.SetNestedSlice(current.Object, containers, "spec", "template", "spec", "containers"); err != nil {
		return "", nil, fmt.Errorf("building image proposal: %w", err)
	}
	proposed, err := yaml.Marshal(current.Object)
	if err != nil {
		return "", nil, fmt.Errorf("rendering proposed manifest: %w", err)
	}
	if err := k8s.ValidateManifestTarget(info.Kind, args.Namespace, args.Name, string(proposed)); err != nil {
		return "", nil, err
	}
	p, err := store.SaveProposal(Proposal{
		Kind: info.Kind, Namespace: args.Namespace, Name: args.Name,
		Rationale: args.Rationale, CurrentYAML: current.YAML, ProposedYAML: string(proposed),
	})
	if err != nil {
		return "", nil, err
	}
	result := fmt.Sprintf("Proposed changing container %s from %s to %s on %s %s/%s (proposal %s). It is pending human approval and has NOT been applied.", containerName, oldImage, args.Image, info.Kind, args.Namespace, args.Name, p.ID)
	return result, &p, nil
}

// canonicalImageReference normalizes Docker Hub's optional registry/library
// prefixes. It deliberately does not guess tags: it only detects proposals
// that look different as strings but resolve to the same image reference.
func canonicalImageReference(image string) string {
	ref := strings.TrimSpace(strings.ToLower(image))
	for _, prefix := range []string{"docker.io/", "index.docker.io/", "registry-1.docker.io/"} {
		ref = strings.TrimPrefix(ref, prefix)
	}
	ref = strings.TrimPrefix(ref, "library/")
	return ref
}

func validatePublicDockerHubImage(ctx context.Context, image string) error {
	client := &http.Client{Timeout: 8 * time.Second}
	return validatePublicDockerHubImageAt(ctx, client, "https://hub.docker.com/v2/repositories", image)
}

var publicDockerHubImageValidator = validatePublicDockerHubImage

// validatePublicDockerHubImageAt verifies public Docker Hub tags. Private and
// non-Docker-Hub registries are left to the human approval step because they
// may require credentials the read-only agent intentionally does not hold.
func validatePublicDockerHubImageAt(ctx context.Context, client *http.Client, baseURL, image string) error {
	ref := strings.TrimSpace(image)
	if strings.Contains(ref, "@") {
		return nil // immutable digest; registry-specific verification may require auth
	}
	for _, prefix := range []string{"docker.io/", "index.docker.io/", "registry-1.docker.io/"} {
		ref = strings.TrimPrefix(ref, prefix)
	}
	if slash := strings.IndexByte(ref, '/'); slash >= 0 {
		first := ref[:slash]
		if strings.Contains(first, ".") || strings.Contains(first, ":") || first == "localhost" {
			return nil // a different/private registry
		}
	}

	lastSlash := strings.LastIndexByte(ref, '/')
	lastColon := strings.LastIndexByte(ref, ':')
	tag := "latest"
	repository := ref
	if lastColon > lastSlash {
		tag = ref[lastColon+1:]
		repository = ref[:lastColon]
	}
	if !strings.Contains(repository, "/") {
		repository = "library/" + repository
	}
	if repository == "" || tag == "" {
		return fmt.Errorf("invalid image reference %q", image)
	}
	segments := strings.Split(repository, "/")
	for i := range segments {
		segments[i] = url.PathEscape(segments[i])
	}
	checkURL := strings.TrimRight(baseURL, "/") + "/" + strings.Join(segments, "/") + "/tags/" + url.PathEscape(tag)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, checkURL, nil)
	if err != nil {
		return fmt.Errorf("building registry validation request: %w", err)
	}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("could not verify proposed public Docker Hub image %q: %w", image, err)
	}
	defer resp.Body.Close()
	switch resp.StatusCode {
	case http.StatusOK:
		return nil
	case http.StatusNotFound:
		suggestions := dockerHubTagSuggestions(ctx, client, baseURL, segments, tag)
		if len(suggestions) > 0 {
			return fmt.Errorf("proposed public Docker Hub image %q does not exist; verified nearby tags are: %s. Retry with the intended verified tag", image, strings.Join(suggestions, ", "))
		}
		return fmt.Errorf("proposed public Docker Hub image %q does not exist; verify the repository and tag and retry", image)
	default:
		return fmt.Errorf("could not verify proposed public Docker Hub image %q: registry returned HTTP %d", image, resp.StatusCode)
	}
}

func dockerHubTagSuggestions(ctx context.Context, client *http.Client, baseURL string, repositorySegments []string, badTag string) []string {
	searchURL := strings.TrimRight(baseURL, "/") + "/" + strings.Join(repositorySegments, "/") + "/tags?page_size=20&name=" + url.QueryEscape(badTag)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, searchURL, nil)
	if err != nil {
		return nil
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil
	}
	var payload struct {
		Results []struct {
			Name string `json:"name"`
		} `json:"results"`
	}
	if err := json.NewDecoder(io.LimitReader(resp.Body, 1<<20)).Decode(&payload); err != nil {
		return nil
	}
	tags := make([]string, 0, len(payload.Results))
	for _, result := range payload.Results {
		if result.Name != "" {
			tags = append(tags, result.Name)
		}
	}
	sort.SliceStable(tags, func(i, j int) bool {
		return editDistance(badTag, tags[i]) < editDistance(badTag, tags[j])
	})
	if len(tags) > 5 {
		tags = tags[:5]
	}
	return tags
}

func editDistance(a, b string) int {
	prev := make([]int, len(b)+1)
	for j := range prev {
		prev[j] = j
	}
	for i := 1; i <= len(a); i++ {
		cur := make([]int, len(b)+1)
		cur[0] = i
		for j := 1; j <= len(b); j++ {
			cost := 0
			if a[i-1] != b[j-1] {
				cost = 1
			}
			cur[j] = min(cur[j-1]+1, prev[j]+1, prev[j-1]+cost)
		}
		prev = cur
	}
	return prev[len(b)]
}

const (
	maxToolLogLines  = 1000
	maxToolResultLen = 30000 // characters per tool result fed back to the model
)

// runTool executes one read-only cluster/metrics tool call. propose_change is
// NOT handled here — the query loop runs it separately (see runProposeChange)
// so it can surface the created proposal as an inline approval card. prom may
// be nil when metrics are disabled.
func runTool(ctx context.Context, client *k8s.Client, prom *promClient, store *Store, call llm.ToolCall) (string, error) {
	var args struct {
		Kind         string `json:"kind"`
		Namespace    string `json:"namespace"`
		Name         string `json:"name"`
		Pod          string `json:"pod"`
		Container    string `json:"container"`
		TailLines    int64  `json:"tailLines"`
		SinceSeconds int64  `json:"sinceSeconds"`
		Previous     bool   `json:"previous"`
		Type         string `json:"type"`
		Query        string `json:"query"`
	}
	if len(call.Input) > 0 {
		if err := json.Unmarshal(call.Input, &args); err != nil {
			return "", fmt.Errorf("invalid tool arguments: %w", err)
		}
	}

	switch call.Name {
	case "get_cluster_overview":
		overview, err := client.GetOverview(ctx)
		if err != nil {
			return "", err
		}
		return marshalJSON(overview)

	case "list_resources":
		items, err := client.ListResources(ctx, args.Kind, args.Namespace)
		if err != nil {
			return "", err
		}
		return marshalJSON(items)

	case "get_resource":
		detail, err := client.GetResource(ctx, args.Kind, args.Namespace, args.Name)
		if err != nil {
			return "", err
		}
		return capString(detail.YAML), nil

	case "get_pod_logs":
		tail := args.TailLines
		if tail <= 0 {
			tail = 200
		}
		if tail > maxToolLogLines {
			tail = maxToolLogLines
		}
		stream, err := client.StreamLogs(ctx, args.Namespace, args.Pod, k8s.LogOptions{
			Container:    args.Container,
			TailLines:    tail,
			SinceSeconds: args.SinceSeconds,
			Previous:     args.Previous,
		})
		if err != nil {
			return "", err
		}
		defer stream.Close()
		data, err := io.ReadAll(io.LimitReader(stream, 1<<20))
		if err != nil {
			return "", fmt.Errorf("reading logs: %w", err)
		}
		if len(data) == 0 {
			return "(no log output in the requested window)", nil
		}
		return capString(string(data)), nil

	case "get_events":
		events, err := client.ListEvents(ctx, args.Namespace, args.Type)
		if err != nil {
			return "", err
		}
		// Events are newest first. Keep the evidence compact enough that local
		// models retain room for a subsequent remediation proposal call.
		if len(events) > 10 {
			events = events[:10]
		}
		return marshalJSON(events)

	case "get_resource_usage":
		if prom == nil {
			return "", fmt.Errorf("metrics are not configured (set a Prometheus URL in Settings)")
		}
		report, err := resourceUsageReport(ctx, prom, args.Namespace)
		if err != nil {
			return "", err
		}
		return capString(report), nil

	case "query_metrics":
		if prom == nil {
			return "", fmt.Errorf("metrics are not configured (set a Prometheus URL in Settings)")
		}
		if args.Query == "" {
			return "", fmt.Errorf("query is required")
		}
		samples, err := prom.Query(ctx, args.Query)
		if err != nil {
			return "", err
		}
		return capString(renderSamples(samples, 50)), nil

	default:
		return "", fmt.Errorf("unknown tool %q", call.Name)
	}
}

// runProposeChange records a remediation proposal for human approval from a
// propose_change tool call. It makes NO change to the cluster — it snapshots
// the current manifest (read-only), validates the proposed one, and stores a
// pending proposal. It returns the tool result string for the model AND the
// created proposal so the query loop can surface an inline approval card.
func runProposeChange(ctx context.Context, client *k8s.Client, store *Store, call llm.ToolCall) (string, *Proposal, error) {
	var args struct {
		Kind         string `json:"kind"`
		Namespace    string `json:"namespace"`
		Name         string `json:"name"`
		ProposedYAML string `json:"proposedYaml"`
		Rationale    string `json:"rationale"`
	}
	if len(call.Input) > 0 {
		if err := json.Unmarshal(call.Input, &args); err != nil {
			return "", nil, fmt.Errorf("invalid tool arguments: %w", err)
		}
	}
	if store == nil {
		return "", nil, fmt.Errorf("proposals are unavailable (no persistent store)")
	}
	if args.ProposedYAML == "" {
		return "", nil, fmt.Errorf("proposedYaml is required")
	}
	kindInfo, err := k8s.LookupKind(args.Kind)
	if err != nil {
		return "", nil, err
	}
	// Store the canonical plural resource name even when the model used a
	// Kubernetes singular Kind (for example "Deployment").
	args.Kind = kindInfo.Kind
	// Validate the proposed manifest parses and its identity matches the
	// target. The server re-validates this at apply time (authoritative
	// guard in k8s.UpdateResource); this is early feedback for the model.
	if err := k8s.ValidateManifestTarget(args.Kind, args.Namespace, args.Name, args.ProposedYAML); err != nil {
		return "", nil, err
	}
	// Snapshot the current manifest for the approval diff (read-only).
	current, err := client.GetResource(ctx, args.Kind, args.Namespace, args.Name)
	if err != nil {
		return "", nil, fmt.Errorf("reading current %s %s: %w", args.Kind, args.Name, err)
	}

	p, err := store.SaveProposal(Proposal{
		Kind: args.Kind, Namespace: args.Namespace, Name: args.Name,
		Rationale:    args.Rationale,
		CurrentYAML:  current.YAML,
		ProposedYAML: args.ProposedYAML,
	})
	if err != nil {
		return "", nil, err
	}
	result := fmt.Sprintf("Proposed a change to %s %s/%s (proposal %s). It is now pending the user's approval — it has NOT been applied. An approval card is shown to the user directly in this chat.",
		args.Kind, args.Namespace, args.Name, p.ID)
	return result, &p, nil
}

func marshalJSON(v interface{}) (string, error) {
	data, err := json.Marshal(v)
	if err != nil {
		return "", fmt.Errorf("encoding tool result: %w", err)
	}
	return capString(string(data)), nil
}

func capString(s string) string {
	if len(s) <= maxToolResultLen {
		return s
	}
	return s[:maxToolResultLen] + "\n... (truncated)"
}
