package handler

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"
	httputil "github.com/saidmashhud/zist/internal/httputil"
	"github.com/saidmashhud/zist/services/listings/store"
)

func (h *Handler) ListPhotos(w http.ResponseWriter, r *http.Request) {
	photos, _ := h.Store.GetPhotos(r.Context(), listingID(r))
	if photos == nil {
		photos = nil
	}
	httputil.WriteJSON(w, http.StatusOK, map[string]any{"photos": photos})
}

func (h *Handler) AddPhoto(w http.ResponseWriter, r *http.Request) {
	id := listingID(r)
	if h.requireOwner(w, r, id) == "" {
		return
	}

	var req struct {
		URL     string `json:"url"`
		Caption string `json:"caption"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httputil.WriteError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.URL == "" {
		httputil.WriteError(w, http.StatusUnprocessableEntity, "url is required")
		return
	}

	count, _ := h.Store.PhotoCount(r.Context(), id)
	if count >= 20 {
		httputil.WriteError(w, http.StatusUnprocessableEntity, "photo limit exceeded (max 20)")
		return
	}

	photo, err := h.Store.AddPhoto(r.Context(), id, req.URL, req.Caption, count)
	if err != nil {
		httputil.WriteError(w, http.StatusInternalServerError, "insert photo failed")
		return
	}
	httputil.WriteJSON(w, http.StatusCreated, photo)
}

func (h *Handler) ReorderPhotos(w http.ResponseWriter, r *http.Request) {
	id := listingID(r)
	if h.requireOwner(w, r, id) == "" {
		return
	}

	var req []struct {
		ID        string `json:"id"`
		SortOrder int    `json:"sortOrder"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httputil.WriteError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	items := make([]struct {
		ID        string
		SortOrder int
	}, len(req))
	for i, v := range req {
		items[i] = struct {
			ID        string
			SortOrder int
		}{v.ID, v.SortOrder}
	}

	if err := h.Store.ReorderPhotos(r.Context(), id, items); err != nil {
		httputil.WriteError(w, http.StatusInternalServerError, "reorder failed")
		return
	}
	httputil.WriteJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (h *Handler) DeletePhoto(w http.ResponseWriter, r *http.Request) {
	id := listingID(r)
	photoID := chi.URLParam(r, "photoId")
	if h.requireOwner(w, r, id) == "" {
		return
	}
	if err := h.Store.DeletePhoto(r.Context(), id, photoID); errors.Is(err, store.ErrNotFound) {
		httputil.WriteError(w, http.StatusNotFound, "photo not found")
		return
	} else if err != nil {
		httputil.WriteError(w, http.StatusInternalServerError, "delete failed")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
