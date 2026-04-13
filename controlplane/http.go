package controlplane

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"strings"
)

type httpHandler struct {
	api API
}

// NewHandler публикует operator-facing HTTP surface как по корню, так и под `/api`.
func NewHandler(api API) http.Handler {
	handler := &httpHandler{api: api}
	mux := http.NewServeMux()
	handler.register(mux, "")
	handler.register(mux, "/api")
	return withCORS(mux)
}

func (h *httpHandler) register(mux *http.ServeMux, base string) {
	mux.HandleFunc(base+"/health", func(w http.ResponseWriter, r *http.Request) {
		h.handleHealth(w, r)
	})
	mux.HandleFunc(base+"/scopes", func(w http.ResponseWriter, r *http.Request) {
		h.handleScopes(w, r)
	})
	mux.HandleFunc(base+"/scopes/", func(w http.ResponseWriter, r *http.Request) {
		h.handleScopeByID(w, r, base+"/scopes/")
	})
	mux.HandleFunc(base+"/control-plane", func(w http.ResponseWriter, r *http.Request) {
		h.handleSnapshot(w, r)
	})
	mux.HandleFunc(base+"/control-plane/plugins/reload", func(w http.ResponseWriter, r *http.Request) {
		h.handlePluginReload(w, r)
	})
	mux.HandleFunc(base+"/control-plane/modules/", func(w http.ResponseWriter, r *http.Request) {
		h.handleModuleByID(w, r, base+"/control-plane/modules/")
	})
	mux.HandleFunc(base+"/control-plane/plugins/", func(w http.ResponseWriter, r *http.Request) {
		h.handlePluginByID(w, r, base+"/control-plane/plugins/")
	})
	mux.HandleFunc(base+"/control-plane/brokers/", func(w http.ResponseWriter, r *http.Request) {
		h.handleBrokerCycle(w, r, base+"/control-plane/brokers/")
	})
}

func (h *httpHandler) handleHealth(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeMethodNotAllowed(w)
		return
	}
	status, err := h.api.Health(r.Context())
	if err != nil {
		writeError(w, http.StatusServiceUnavailable, err)
		return
	}
	writeJSON(w, http.StatusOK, status)
}

func (h *httpHandler) handleScopes(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		scopes, err := h.api.ListScopes(r.Context())
		if err != nil {
			writeError(w, http.StatusInternalServerError, err)
			return
		}
		writeJSON(w, http.StatusOK, scopes)
	case http.MethodPost:
		var scope ScopePreset
		if err := decodeJSON(r, &scope); err != nil {
			writeError(w, http.StatusBadRequest, err)
			return
		}
		created, err := h.api.CreateScope(r.Context(), scope)
		if err != nil {
			writeError(w, statusFromError(err), err)
			return
		}
		writeJSON(w, http.StatusCreated, created)
	default:
		writeMethodNotAllowed(w)
	}
}

func (h *httpHandler) handleScopeByID(w http.ResponseWriter, r *http.Request, prefix string) {
	id, ok := decodePathID(r.URL.Path, prefix)
	if !ok {
		http.NotFound(w, r)
		return
	}

	switch r.Method {
	case http.MethodPatch:
		var patch struct {
			Tags []string `json:"tags"`
		}
		if err := decodeJSON(r, &patch); err != nil {
			writeError(w, http.StatusBadRequest, err)
			return
		}
		scope, err := h.api.UpdateScopeTags(r.Context(), id, patch.Tags)
		if err != nil {
			writeError(w, statusFromError(err), err)
			return
		}
		writeJSON(w, http.StatusOK, scope)
	case http.MethodDelete:
		if err := h.api.DeleteScope(r.Context(), id); err != nil {
			writeError(w, statusFromError(err), err)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	default:
		writeMethodNotAllowed(w)
	}
}

func (h *httpHandler) handleSnapshot(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeMethodNotAllowed(w)
		return
	}
	snapshot, err := h.api.Snapshot(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, snapshot)
}

func (h *httpHandler) handlePluginReload(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeMethodNotAllowed(w)
		return
	}

	snapshot, err := h.api.ReloadPlugins(r.Context())
	if err != nil {
		writeError(w, statusFromError(err), err)
		return
	}
	writeJSON(w, http.StatusOK, snapshot)
}

func (h *httpHandler) handleModuleByID(w http.ResponseWriter, r *http.Request, prefix string) {
	if r.Method != http.MethodPatch {
		writeMethodNotAllowed(w)
		return
	}
	id, ok := decodePathID(r.URL.Path, prefix)
	if !ok {
		http.NotFound(w, r)
		return
	}
	var patch ModulePatch
	if err := decodeJSON(r, &patch); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	module, err := h.api.UpdateModule(r.Context(), id, patch)
	if err != nil {
		writeError(w, statusFromError(err), err)
		return
	}
	writeJSON(w, http.StatusOK, module)
}

func (h *httpHandler) handlePluginByID(w http.ResponseWriter, r *http.Request, prefix string) {
	if r.Method != http.MethodPatch {
		writeMethodNotAllowed(w)
		return
	}
	id, ok := decodePathID(r.URL.Path, prefix)
	if !ok {
		http.NotFound(w, r)
		return
	}
	var patch PluginPatch
	if err := decodeJSON(r, &patch); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	plugin, err := h.api.UpdatePlugin(r.Context(), id, patch)
	if err != nil {
		writeError(w, statusFromError(err), err)
		return
	}
	writeJSON(w, http.StatusOK, plugin)
}

func (h *httpHandler) handleBrokerCycle(w http.ResponseWriter, r *http.Request, prefix string) {
	if r.Method != http.MethodPost {
		writeMethodNotAllowed(w)
		return
	}

	trimmed := strings.TrimPrefix(r.URL.Path, prefix)
	parts := strings.Split(strings.Trim(trimmed, "/"), "/")
	if len(parts) != 2 || parts[1] != "cycle" {
		http.NotFound(w, r)
		return
	}
	id, err := url.PathUnescape(parts[0])
	if err != nil || id == "" {
		http.NotFound(w, r)
		return
	}

	broker, err := h.api.CycleBrokerMode(r.Context(), id)
	if err != nil {
		writeError(w, statusFromError(err), err)
		return
	}
	writeJSON(w, http.StatusOK, broker)
}

func decodePathID(path, prefix string) (string, bool) {
	idPart := strings.TrimPrefix(path, prefix)
	if idPart == path || idPart == "" {
		return "", false
	}
	if strings.Contains(strings.Trim(idPart, "/"), "/") {
		return "", false
	}
	id, err := url.PathUnescape(strings.Trim(idPart, "/"))
	if err != nil || id == "" {
		return "", false
	}
	return id, true
}

func decodeJSON(r *http.Request, target any) error {
	defer r.Body.Close()
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return err
	}
	return nil
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

func writeError(w http.ResponseWriter, status int, err error) {
	writeJSON(w, status, map[string]any{
		"error": err.Error(),
	})
}

func writeMethodNotAllowed(w http.ResponseWriter) {
	writeError(w, http.StatusMethodNotAllowed, errors.New("method not allowed"))
}

func statusFromError(err error) int {
	switch {
	case errors.Is(err, context.DeadlineExceeded), errors.Is(err, context.Canceled):
		return http.StatusRequestTimeout
	case errors.Is(err, ErrScopeNotFound), errors.Is(err, ErrModuleNotFound), errors.Is(err, ErrPluginNotFound), errors.Is(err, ErrBrokerNotFound):
		return http.StatusNotFound
	case errors.Is(err, ErrPluginReloadUnavailable):
		return http.StatusConflict
	case errors.Is(err, ErrInvalidScope), errors.Is(err, ErrScopeConflict), errors.Is(err, ErrLastScope):
		return http.StatusBadRequest
	default:
		return http.StatusInternalServerError
	}
}

func withCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		w.Header().Set("Access-Control-Allow-Methods", "GET,POST,PATCH,DELETE,OPTIONS")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}
