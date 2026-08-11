package calendars

import (
	"encoding/json"
	"errors"
	"net/http"
	"regexp"

	"github.com/go-chi/chi/v5"

	"neuroboost/api-go/internal/middleware"
	"neuroboost/api-go/internal/util"
)

type createRequest struct {
	Name  string  `json:"name"`
	Color *string `json:"color"`
}

type updateRequest struct {
	Name  *string `json:"name"`
	Color *string `json:"color"`
}

// hexColorRE matches a strict 6-digit hex colour, e.g. #7c3aed. An empty
// string does not match, so it cannot be used to clear a colour — clearing is
// deliberately not expressible in v1 (see UpdateFields doc comment).
var hexColorRE = regexp.MustCompile(`^#[0-9a-fA-F]{6}$`)

// validColor reports whether a supplied colour is acceptable. A nil pointer
// is always valid — it means "not supplied" (absent on create, unchanged on
// update). A non-nil pointer must be a strict hex colour; an empty string is
// rejected rather than treated as "no colour".
func validColor(color *string) bool {
	if color == nil {
		return true
	}
	return hexColorRE.MatchString(*color)
}

// caller resolves the authenticated user, writing the 401 itself. Returns ""
// when it has already responded.
func caller(w http.ResponseWriter, r *http.Request) string {
	if db == nil {
		util.RespondError(w, http.StatusInternalServerError, "DB_NOT_INITIALIZED", "Database not initialized")
		return ""
	}
	userID := middleware.UserIDFromContext(r.Context())
	if userID == "" {
		util.RespondError(w, http.StatusUnauthorized, "NOT_AUTHENTICATED", "Not authenticated")
	}
	return userID
}

// ListHandler handles GET /api/calendars
func ListHandler(w http.ResponseWriter, r *http.Request) {
	userID := caller(w, r)
	if userID == "" {
		return
	}
	list, err := ListFor(r.Context(), userID)
	if err != nil {
		util.RespondError(w, http.StatusInternalServerError, "LIST_ERROR", "Failed to list calendars")
		return
	}
	util.RespondJSON(w, http.StatusOK, list)
}

// CreateHandler handles POST /api/calendars
func CreateHandler(w http.ResponseWriter, r *http.Request) {
	userID := caller(w, r)
	if userID == "" {
		return
	}
	var req createRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		util.RespondError(w, http.StatusBadRequest, "INVALID_REQUEST", "Invalid request body")
		return
	}
	if _, ok := NormalizeName(req.Name); !ok {
		util.RespondError(w, http.StatusBadRequest, "INVALID_NAME", "Name must be 1-100 characters")
		return
	}
	if !validColor(req.Color) {
		util.RespondError(w, http.StatusBadRequest, "INVALID_COLOR", "Color must be a hex value like #7c3aed")
		return
	}
	c, err := Create(r.Context(), userID, req.Name, req.Color)
	if err != nil {
		respondCalendarError(w, err)
		return
	}
	util.RespondJSON(w, http.StatusCreated, c)
}

// UpdateHandler handles PATCH /api/calendars/{id}
func UpdateHandler(w http.ResponseWriter, r *http.Request) {
	userID := caller(w, r)
	if userID == "" {
		return
	}
	var req updateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		util.RespondError(w, http.StatusBadRequest, "INVALID_REQUEST", "Invalid request body")
		return
	}
	if req.Name != nil {
		if _, ok := NormalizeName(*req.Name); !ok {
			util.RespondError(w, http.StatusBadRequest, "INVALID_NAME", "Name must be 1-100 characters")
			return
		}
	}
	if !validColor(req.Color) {
		util.RespondError(w, http.StatusBadRequest, "INVALID_COLOR", "Color must be a hex value like #7c3aed")
		return
	}
	c, err := Update(r.Context(), userID, chi.URLParam(r, "id"), UpdateFields{Name: req.Name, Color: req.Color})
	if err != nil {
		respondCalendarError(w, err)
		return
	}
	util.RespondJSON(w, http.StatusOK, c)
}

// DeleteHandler handles DELETE /api/calendars/{id}
func DeleteHandler(w http.ResponseWriter, r *http.Request) {
	userID := caller(w, r)
	if userID == "" {
		return
	}
	if err := Delete(r.Context(), userID, chi.URLParam(r, "id")); err != nil {
		respondCalendarError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// respondCalendarError maps the store's typed errors onto status codes in one
// place, so create, update and delete cannot disagree about what a stranger
// is told.
func respondCalendarError(w http.ResponseWriter, err error) {
	var ne *NotEmptyError
	switch {
	case errors.Is(err, ErrCalendarNotFound):
		// Deliberately 404, not 403: a 403 would confirm to a stranger that
		// this calendar exists.
		util.RespondError(w, http.StatusNotFound, "CALENDAR_NOT_FOUND", "Календарь не найден")
	case errors.Is(err, ErrNotCalendarOwner):
		util.RespondError(w, http.StatusForbidden, "NOT_CALENDAR_OWNER", "Только владелец может изменить календарь")
	case errors.Is(err, ErrCalendarIsPersonal):
		util.RespondError(w, http.StatusConflict, "CALENDAR_IS_PERSONAL", "Личный календарь удалить нельзя")
	case errors.Is(err, ErrInvalidName):
		// The handler already validates with NormalizeName before calling the
		// store, so this path is a backstop, not the primary route to a 400 —
		// but the store can return it too, and no path may fall through to a
		// generic 500 on a validation failure.
		util.RespondError(w, http.StatusBadRequest, "INVALID_NAME", "Name must be 1-100 characters")
	case errors.As(err, &ne):
		respondNotEmpty(w, ne)
	default:
		util.RespondError(w, http.StatusInternalServerError, "CALENDAR_ERROR", "Failed to process calendar request")
	}
}

// respondNotEmpty writes the 409 the spec requires with the counts inline.
// util.RespondError has no field for them, so the envelope is built by hand —
// keep its shape identical to util's, or the frontend error parser will miss
// this case entirely.
func respondNotEmpty(w http.ResponseWriter, ne *NotEmptyError) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusConflict)
	_ = json.NewEncoder(w).Encode(map[string]any{
		"error": map[string]any{
			"code":    "CALENDAR_NOT_EMPTY",
			"message": "Календарь не пуст",
			"events":  ne.Events,
			"tasks":   ne.Tasks,
		},
	})
}
