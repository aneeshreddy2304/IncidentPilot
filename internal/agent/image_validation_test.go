package agent

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestValidatePublicDockerHubImage(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/library/httpd/tags/2.4-alpine" {
			w.WriteHeader(http.StatusOK)
			return
		}
		if r.URL.Path == "/library/httpd/tags" {
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"results":[{"name":"2.4-alpine3.24"},{"name":"2.4-alpine"},{"name":"latest"}]}`))
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	client := server.Client()
	if err := validatePublicDockerHubImageAt(context.Background(), client, server.URL, "httpd:2.4-alpine"); err != nil {
		t.Fatalf("valid tag rejected: %v", err)
	}
	if err := validatePublicDockerHubImageAt(context.Background(), client, server.URL, "docker.io/library/httpd:2.4-alpin-slim"); err == nil {
		t.Fatal("nonexistent tag accepted")
	} else if !strings.Contains(err.Error(), "2.4-alpine") {
		t.Fatalf("missing verified tag suggestion: %v", err)
	}
	if err := validatePublicDockerHubImageAt(context.Background(), client, server.URL, "httpd:2.4-alpin-20260929"); err == nil {
		t.Fatal("short Docker Hub name with nonexistent tag bypassed validation")
	}
	if err := validatePublicDockerHubImageAt(context.Background(), client, server.URL, "private.example.com/team/httpd:custom"); err != nil {
		t.Fatalf("private registry should be deferred to human review: %v", err)
	}
}
