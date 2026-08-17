package calendars

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"

	"neuroboost/api-go/internal/util"
)

type inviteRequest struct {
	Email string `json:"email"`
	Role  string `json:"role"`
}

type inviteLinkRequest struct {
	Role string `json:"role"`
}

type respondRequest struct {
	Accept bool `json:"accept"`
}

type acceptLinkRequest struct {
	Token string `json:"token"`
}

// ListMembersHandler handles GET /api/calendars/{id}/members
func ListMembersHandler(w http.ResponseWriter, r *http.Request) {
	userID := caller(w, r)
	if userID == "" {
		return
	}
	members, err := ListMembers(r.Context(), userID, chi.URLParam(r, "id"))
	if err != nil {
		respondMemberError(w, err)
		return
	}
	util.RespondJSON(w, http.StatusOK, members)
}

// InviteHandler handles POST /api/calendars/{id}/invites — by email.
func InviteHandler(w http.ResponseWriter, r *http.Request) {
	userID := caller(w, r)
	if userID == "" {
		return
	}
	var req inviteRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		util.RespondError(w, http.StatusBadRequest, "INVALID_REQUEST", "Invalid request body")
		return
	}
	if req.Email == "" {
		util.RespondError(w, http.StatusBadRequest, "INVALID_EMAIL", "Email is required")
		return
	}
	member, err := InviteByEmail(r.Context(), userID, chi.URLParam(r, "id"), req.Email, req.Role)
	if err != nil {
		respondMemberError(w, err)
		return
	}
	util.RespondJSON(w, http.StatusCreated, member)
}

// InviteLinkHandler handles POST /api/calendars/{id}/invite-links.
func InviteLinkHandler(w http.ResponseWriter, r *http.Request) {
	userID := caller(w, r)
	if userID == "" {
		return
	}
	var req inviteLinkRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		util.RespondError(w, http.StatusBadRequest, "INVALID_REQUEST", "Invalid request body")
		return
	}
	invite, err := CreateInviteLink(r.Context(), userID, chi.URLParam(r, "id"), req.Role)
	if err != nil {
		respondMemberError(w, err)
		return
	}
	util.RespondJSON(w, http.StatusCreated, invite)
}

// RespondInvitationHandler handles POST /api/calendars/{id}/invitation —
// the invitee accepting or declining an invitation addressed to them.
func RespondInvitationHandler(w http.ResponseWriter, r *http.Request) {
	userID := caller(w, r)
	if userID == "" {
		return
	}
	var req respondRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		util.RespondError(w, http.StatusBadRequest, "INVALID_REQUEST", "Invalid request body")
		return
	}
	if err := RespondToInvitation(r.Context(), userID, chi.URLParam(r, "id"), req.Accept); err != nil {
		respondMemberError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// AcceptInviteLinkHandler handles POST /api/calendars/invite-links/accept.
//
// The token travels in the BODY, not the path. A path lands in access logs,
// browser history and any referrer header the page emits — and this token is a
// bearer credential for someone's calendar.
func AcceptInviteLinkHandler(w http.ResponseWriter, r *http.Request) {
	userID := caller(w, r)
	if userID == "" {
		return
	}
	var req acceptLinkRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		util.RespondError(w, http.StatusBadRequest, "INVALID_REQUEST", "Invalid request body")
		return
	}
	if req.Token == "" {
		util.RespondError(w, http.StatusBadRequest, "INVALID_REQUEST", "Token is required")
		return
	}
	c, err := AcceptInviteLink(r.Context(), userID, req.Token)
	if err != nil {
		respondMemberError(w, err)
		return
	}
	util.RespondJSON(w, http.StatusOK, c)
}

// RemoveMemberHandler handles DELETE /api/calendars/{id}/members/{userId}.
// The owner removing someone, or anyone removing themselves.
func RemoveMemberHandler(w http.ResponseWriter, r *http.Request) {
	userID := caller(w, r)
	if userID == "" {
		return
	}
	err := RemoveMember(r.Context(), userID, chi.URLParam(r, "id"), chi.URLParam(r, "userId"))
	if err != nil {
		respondMemberError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// respondMemberError maps this file's errors in one place, and falls through to
// respondCalendarError for the ones shared with create/update/delete — so the
// two cannot disagree about what a stranger is told.
func respondMemberError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, ErrCalendarIsPersonalShare):
		util.RespondError(w, http.StatusConflict, "CALENDAR_IS_PERSONAL",
			"Личный календарь нельзя расшарить — создайте общий")
	case errors.Is(err, ErrInviteeNotFound):
		util.RespondError(w, http.StatusNotFound, "INVITEE_NOT_FOUND",
			"Пользователя с таким email нет — отправьте ссылку-приглашение")
	case errors.Is(err, ErrAlreadyMember):
		util.RespondError(w, http.StatusConflict, "ALREADY_MEMBER", "Этот человек уже в календаре")
	case errors.Is(err, ErrInvalidRole):
		util.RespondError(w, http.StatusBadRequest, "INVALID_ROLE", "Роль должна быть editor или viewer")
	case errors.Is(err, ErrInviteInvalid):
		// Deliberately one message for unknown, expired and used: which of the
		// three it is would tell the holder whether the token ever existed.
		util.RespondError(w, http.StatusGone, "INVITE_INVALID",
			"Ссылка недействительна — она живёт 2 часа и срабатывает один раз")
	case errors.Is(err, ErrNoInvitation):
		util.RespondError(w, http.StatusNotFound, "NO_INVITATION", "Приглашения нет")
	case errors.Is(err, ErrCannotRemoveOwner):
		util.RespondError(w, http.StatusConflict, "CANNOT_REMOVE_OWNER", "Владельца убрать нельзя")
	default:
		respondCalendarError(w, err)
	}
}
