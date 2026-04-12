package controlplane

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHTTPHandlerExposesControlPlaneEndpoints(t *testing.T) {
	server := httptest.NewServer(NewHandler(NewService()))
	defer server.Close()

	resp, err := http.Get(server.URL + "/api/control-plane")
	if err != nil {
		t.Fatalf("GET /api/control-plane returned error: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("GET /api/control-plane status = %d, want 200", resp.StatusCode)
	}

	var snapshot Snapshot
	if err := json.NewDecoder(resp.Body).Decode(&snapshot); err != nil {
		t.Fatalf("decode snapshot returned error: %v", err)
	}
	if len(snapshot.Brokers) == 0 {
		t.Fatalf("snapshot brokers empty: %+v", snapshot)
	}
}

func TestHTTPHandlerMutatesScopeModulePluginAndBroker(t *testing.T) {
	server := httptest.NewServer(NewHandler(NewService()))
	defer server.Close()

	body := bytes.NewBufferString(`{"segment":"health","tags":["focus","audit"]}`)
	resp, err := http.Post(server.URL+"/api/scopes", "application/json", body)
	if err != nil {
		t.Fatalf("POST /api/scopes returned error: %v", err)
	}
	if resp.StatusCode != http.StatusCreated {
		defer resp.Body.Close()
		t.Fatalf("POST /api/scopes status = %d, want 201", resp.StatusCode)
	}
	var created ScopePreset
	if err := json.NewDecoder(resp.Body).Decode(&created); err != nil {
		resp.Body.Close()
		t.Fatalf("decode created scope returned error: %v", err)
	}
	resp.Body.Close()

	req, err := http.NewRequest(http.MethodPatch, server.URL+"/api/control-plane/modules/tracker", bytes.NewBufferString(`{"enabled":false}`))
	if err != nil {
		t.Fatalf("NewRequest(module patch) returned error: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err = http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("PATCH module returned error: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("PATCH module status = %d, want 200", resp.StatusCode)
	}

	req, err = http.NewRequest(http.MethodPatch, server.URL+"/api/control-plane/plugins/audit-mirror", bytes.NewBufferString(`{"status":"enabled"}`))
	if err != nil {
		t.Fatalf("NewRequest(plugin patch) returned error: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err = http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("PATCH plugin returned error: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("PATCH plugin status = %d, want 200", resp.StatusCode)
	}

	req, err = http.NewRequest(http.MethodPost, server.URL+"/api/control-plane/brokers/runtime-core/cycle", nil)
	if err != nil {
		t.Fatalf("NewRequest(broker cycle) returned error: %v", err)
	}
	resp, err = http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("POST broker cycle returned error: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("POST broker cycle status = %d, want 200", resp.StatusCode)
	}

	req, err = http.NewRequest(http.MethodDelete, server.URL+"/api/scopes/"+ScopeKey(created), nil)
	if err != nil {
		t.Fatalf("NewRequest(scope delete) returned error: %v", err)
	}
	resp, err = http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("DELETE scope returned error: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("DELETE scope status = %d, want 204", resp.StatusCode)
	}
}
