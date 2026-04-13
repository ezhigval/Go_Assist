package controlplane

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHTTPHandlerExposesControlPlaneEndpoints(t *testing.T) {
	handler := NewHandler(NewService())

	recorder := performRequest(t, handler, http.MethodGet, "/api/control-plane", nil)
	if recorder.Code != http.StatusOK {
		t.Fatalf("GET /api/control-plane status = %d, want 200", recorder.Code)
	}

	var snapshot Snapshot
	if err := json.Unmarshal(recorder.Body.Bytes(), &snapshot); err != nil {
		t.Fatalf("decode snapshot returned error: %v", err)
	}
	if len(snapshot.Brokers) == 0 {
		t.Fatalf("snapshot brokers empty: %+v", snapshot)
	}
}

func TestHTTPHandlerMutatesScopeModulePluginAndBroker(t *testing.T) {
	handler := NewHandler(NewService())

	recorder := performRequest(t, handler, http.MethodPost, "/api/scopes", bytes.NewBufferString(`{"segment":"health","tags":["focus","audit"]}`))
	if recorder.Code != http.StatusCreated {
		t.Fatalf("POST /api/scopes status = %d, want 201", recorder.Code)
	}

	var created ScopePreset
	if err := json.Unmarshal(recorder.Body.Bytes(), &created); err != nil {
		t.Fatalf("decode created scope returned error: %v", err)
	}

	recorder = performRequest(t, handler, http.MethodPatch, "/api/control-plane/modules/tracker", bytes.NewBufferString(`{"enabled":false}`))
	if recorder.Code != http.StatusOK {
		t.Fatalf("PATCH module status = %d, want 200", recorder.Code)
	}

	recorder = performRequest(t, handler, http.MethodPatch, "/api/control-plane/plugins/audit-mirror", bytes.NewBufferString(`{"status":"enabled"}`))
	if recorder.Code != http.StatusOK {
		t.Fatalf("PATCH plugin status = %d, want 200", recorder.Code)
	}

	recorder = performRequest(t, handler, http.MethodPost, "/api/control-plane/brokers/runtime-core/cycle", nil)
	if recorder.Code != http.StatusOK {
		t.Fatalf("POST broker cycle status = %d, want 200", recorder.Code)
	}

	recorder = performRequest(t, handler, http.MethodDelete, "/api/scopes/"+ScopeKey(created), nil)
	if recorder.Code != http.StatusNoContent {
		t.Fatalf("DELETE scope status = %d, want 204", recorder.Code)
	}
}

func performRequest(t *testing.T, handler http.Handler, method, path string, body io.Reader) *httptest.ResponseRecorder {
	t.Helper()

	req := httptest.NewRequest(method, path, body)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, req)
	return recorder
}
