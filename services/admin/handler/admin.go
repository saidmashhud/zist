package handler

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	zistauth "github.com/saidmashhud/zist/internal/auth"
	"github.com/saidmashhud/zist/internal/httputil"
	"github.com/saidmashhud/zist/services/admin/store"
)

// ─── Feature Flags ────────────────────────────────────────────────────────────

// ListFlags handles GET /admin/flags.
func (h *Handler) ListFlags(w http.ResponseWriter, r *http.Request) {
	p := zistauth.FromContext(r.Context())
	if !requireAdmin(p) {
		httputil.WriteError(w, http.StatusForbidden, "admin scope required")
		return
	}
	flags, err := h.Store.ListFlags(r.Context())
	if err != nil {
		httputil.WriteError(w, http.StatusInternalServerError, "db error")
		return
	}
	httputil.WriteJSON(w, http.StatusOK, map[string]any{"flags": flags})
}

// UpsertFlag handles POST /admin/flags.
func (h *Handler) UpsertFlag(w http.ResponseWriter, r *http.Request) {
	p := zistauth.FromContext(r.Context())
	if !requireAdmin(p) {
		httputil.WriteError(w, http.StatusForbidden, "admin scope required")
		return
	}

	var req struct {
		Name     string  `json:"name"`
		Enabled  bool    `json:"enabled"`
		Rollout  int     `json:"rollout"`
		TenantID *string `json:"tenantId"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Name == "" {
		httputil.WriteError(w, http.StatusBadRequest, "name is required")
		return
	}
	if req.Rollout < 0 || req.Rollout > 100 {
		req.Rollout = 100
	}

	flag, err := h.Store.UpsertFlag(r.Context(), req.Name, req.Enabled, req.Rollout, req.TenantID)
	if err != nil {
		httputil.WriteError(w, http.StatusInternalServerError, "failed to upsert flag")
		return
	}

	h.Store.AddAudit(r.Context(), p.UserID, "upsert_flag", "feature_flag:"+req.Name, //nolint:errcheck
		"enabled="+strconv.FormatBool(req.Enabled), p.TenantID)

	httputil.WriteJSON(w, http.StatusOK, flag)
}

// ─── Audit Log ────────────────────────────────────────────────────────────────

// ListAudit handles GET /admin/audit.
func (h *Handler) ListAudit(w http.ResponseWriter, r *http.Request) {
	p := zistauth.FromContext(r.Context())
	if !requireAdmin(p) {
		httputil.WriteError(w, http.StatusForbidden, "admin scope required")
		return
	}
	actorFilter := r.URL.Query().Get("actor_id")
	limit := 100
	if n, err := strconv.Atoi(r.URL.Query().Get("limit")); err == nil && n > 0 {
		limit = n
	}
	entries, err := h.Store.ListAudit(r.Context(), actorFilter, limit)
	if err != nil {
		httputil.WriteError(w, http.StatusInternalServerError, "db error")
		return
	}
	httputil.WriteJSON(w, http.StatusOK, map[string]any{"entries": entries})
}

// ─── Tenant Config ────────────────────────────────────────────────────────────

// GetTenantConfig handles GET /admin/tenants/{id}.
func (h *Handler) GetTenantConfig(w http.ResponseWriter, r *http.Request) {
	p := zistauth.FromContext(r.Context())
	if !requireAdmin(p) {
		httputil.WriteError(w, http.StatusForbidden, "admin scope required")
		return
	}
	tenantID := chi.URLParam(r, "id")
	cfg, err := h.Store.GetTenantConfig(r.Context(), tenantID)
	if err != nil {
		httputil.WriteError(w, http.StatusInternalServerError, "db error")
		return
	}
	httputil.WriteJSON(w, http.StatusOK, cfg)
}

// UpsertTenantConfig handles PUT /admin/tenants/{id}.
func (h *Handler) UpsertTenantConfig(w http.ResponseWriter, r *http.Request) {
	p := zistauth.FromContext(r.Context())
	if !requireAdmin(p) {
		httputil.WriteError(w, http.StatusForbidden, "admin scope required")
		return
	}
	tenantID := chi.URLParam(r, "id")

	var req store.TenantConfig
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httputil.WriteError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	req.TenantID = tenantID

	cfg, err := h.Store.UpsertTenantConfig(r.Context(), req)
	if err != nil {
		httputil.WriteError(w, http.StatusInternalServerError, "failed to update tenant config")
		return
	}

	h.Store.AddAudit(r.Context(), p.UserID, "update_tenant_config", "tenant:"+tenantID, //nolint:errcheck
		"", p.TenantID)

	httputil.WriteJSON(w, http.StatusOK, cfg)
}
