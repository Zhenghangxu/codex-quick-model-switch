package router

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"sort"
	"strings"

	"codex-quick-model-switch/internal/config"
	"codex-quick-model-switch/internal/hook"
	"codex-quick-model-switch/internal/patchjson"
	"codex-quick-model-switch/internal/state"
)

type Handler struct {
	cfg    config.Config
	store  *state.Store
	client *http.Client
	logger logger
}

type logger interface {
	Printf(format string, v ...any)
}

func New(cfg config.Config, store *state.Store, client *http.Client) http.Handler {
	return NewWithLogger(cfg, store, client, log.Default())
}

func NewWithLogger(cfg config.Config, store *state.Store, client *http.Client, logger logger) http.Handler {
	if client == nil {
		client = http.DefaultClient
	}
	return &Handler{cfg: cfg, store: store, client: client, logger: logger}
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch {
	case r.Method == http.MethodGet && r.URL.Path == "/healthz":
		w.WriteHeader(http.StatusNoContent)
	case r.Method == http.MethodGet && r.URL.Path == "/switches":
		if !h.authorized(w, r) {
			return
		}
		h.handleSwitches(w)
	case r.Method == http.MethodGet && r.URL.Path == "/state":
		if !h.authorized(w, r) {
			return
		}
		h.handleState(w)
	case r.Method == http.MethodPost && r.URL.Path == "/switch":
		if !h.authorized(w, r) {
			return
		}
		h.handleSwitch(w, r)
	case r.Method == http.MethodPost && (r.URL.Path == "/v1/responses" || r.URL.Path == "/v1/chat/completions"):
		if !h.authorized(w, r) {
			return
		}
		h.handleProxy(w, r)
	default:
		http.NotFound(w, r)
	}
}

func (h *Handler) handleSwitches(w http.ResponseWriter) {
	st, err := h.store.Load()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	switches := h.cfg.SwitchOrder
	if len(switches) == 0 {
		for _, sw := range h.cfg.Switches {
			switches = append(switches, sw)
		}
		sort.Slice(switches, func(i, j int) bool {
			return switches[i].Shortcut < switches[j].Shortcut
		})
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"switches": switches,
		"active":   st.Active,
	})
}

func (h *Handler) authorized(w http.ResponseWriter, r *http.Request) bool {
	if h.cfg.RouterAPIKey == "" {
		return true
	}
	if bearerToken(r.Header.Get("Authorization")) == h.cfg.RouterAPIKey {
		return true
	}
	http.Error(w, "invalid api key", http.StatusUnauthorized)
	return false
}

func (h *Handler) handleState(w http.ResponseWriter) {
	st, err := h.store.Load()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(st)
}

func (h *Handler) handleSwitch(w http.ResponseWriter, r *http.Request) {
	var req hook.SwitchRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	sw, ok := h.cfg.Switches[req.Shortcut]
	if !ok {
		http.Error(w, "unknown shortcut", http.StatusBadRequest)
		return
	}
	if err := h.store.Save(state.ActiveState{Active: sw}); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) handleProxy(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	st, err := h.store.Load()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	requestedModel := requestedModel(body)
	patchedBody, wasPatched, err := patchjson.PatchRequest(body, h.cfg.VirtualModel, st.Active)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	h.logProxy(wasPatched, requestedModel, patchedBody, st.Active)

	target, err := h.upstreamURL(r.URL)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	req, err := http.NewRequestWithContext(r.Context(), r.Method, target, bytes.NewReader(patchedBody))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	req.Header = r.Header.Clone()
	req.Header.Del("Host")
	req.Header.Del("Content-Length")
	if h.cfg.UpstreamAPIKey != "" {
		req.Header.Set("Authorization", "Bearer "+h.cfg.UpstreamAPIKey)
	}
	req.ContentLength = int64(len(patchedBody))
	resp, err := h.client.Do(req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	for key, values := range resp.Header {
		for _, value := range values {
			w.Header().Add(key, value)
		}
	}
	w.WriteHeader(resp.StatusCode)
	_, _ = io.Copy(w, resp.Body)
}

func (h *Handler) logProxy(patched bool, requestedModel string, forwardedBody []byte, active config.Switch) {
	if h.logger == nil {
		return
	}
	summary := forwardedSummary(forwardedBody)
	h.logger.Printf(
		"qms proxy patched=%t requested_model=%q forwarded_model=%q forwarded_effort_field=%q forwarded_effort=%q forwarded_service_tier=%q active_shortcut=%q active_model=%q active_effort=%q active_service_tier=%q",
		patched,
		requestedModel,
		summary.model,
		summary.effortField,
		summary.effort,
		summary.serviceTier,
		active.Shortcut,
		active.Model,
		active.Effort,
		active.ServiceTier,
	)
}

type requestSummary struct {
	model       string
	effortField string
	effort      string
	serviceTier string
}

func forwardedSummary(body []byte) requestSummary {
	var req struct {
		Model     string `json:"model"`
		Reasoning struct {
			Effort string `json:"effort"`
		} `json:"reasoning"`
		ReasoningEffort string `json:"reasoning_effort"`
		ServiceTier     string `json:"service_tier"`
	}
	if err := json.Unmarshal(body, &req); err != nil {
		return requestSummary{}
	}
	summary := requestSummary{model: req.Model, serviceTier: req.ServiceTier}
	if req.Reasoning.Effort != "" {
		summary.effortField = "reasoning.effort"
		summary.effort = req.Reasoning.Effort
	} else if req.ReasoningEffort != "" {
		summary.effortField = "reasoning_effort"
		summary.effort = req.ReasoningEffort
	}
	return summary
}

func requestedModel(body []byte) string {
	var req struct {
		Model string `json:"model"`
	}
	if err := json.Unmarshal(body, &req); err != nil {
		return ""
	}
	return req.Model
}

func (h *Handler) upstreamURL(requestURL *url.URL) (string, error) {
	base, err := url.Parse(strings.TrimRight(h.cfg.UpstreamBaseURL, "/"))
	if err != nil {
		return "", err
	}
	if base.Scheme == "" || base.Host == "" {
		return "", fmt.Errorf("invalid upstream base url %q", h.cfg.UpstreamBaseURL)
	}
	base.Path = strings.TrimRight(base.Path, "/") + "/" + strings.TrimLeft(strings.TrimPrefix(requestURL.Path, "/v1"), "/")
	base.RawQuery = requestURL.RawQuery
	return base.String(), nil
}

func bearerToken(header string) string {
	const prefix = "Bearer "
	if !strings.HasPrefix(header, prefix) {
		return ""
	}
	return strings.TrimSpace(strings.TrimPrefix(header, prefix))
}
