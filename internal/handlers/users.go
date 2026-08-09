package handlers

import (
	"context"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"transport-app/internal/domain"
	"transport-app/internal/middleware"
)

// UserHandlers handles user management (admin only).
type UserHandlers struct {
	*App
}

func (h *UserHandlers) Routes(r chi.Router) {
	r.With(middleware.ResourcePermission(h.AuthSrv, "users", "read")).Get("/", h.List)
	r.With(middleware.ResourcePermission(h.AuthSrv, "users", "create")).Get("/new", h.New)
	r.With(middleware.ResourcePermission(h.AuthSrv, "users", "create")).Post("/new", h.Create)
	r.With(middleware.ResourcePermission(h.AuthSrv, "users", "update")).Get("/{id}/edit", h.Edit)
	r.With(middleware.ResourcePermission(h.AuthSrv, "users", "update")).Post("/{id}/edit", h.Update)
	r.With(middleware.ResourcePermission(h.AuthSrv, "users", "delete")).Post("/{id}/delete", h.Delete)
	r.With(middleware.ResourcePermission(h.AuthSrv, "users", "update")).Post("/{id}/reset-password", h.ResetPassword)
}

func (h *UserHandlers) List(w http.ResponseWriter, r *http.Request) {
	session, _ := h.getUserFromContext(r)
	pp := parsePaginationParams(r)

	list, total, err := h.Services.Users.ListUsers(r.Context(), pp.Query, pp.Status, pp.Limit, pp.Offset)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	pd := newPaginationData(pp, total, "/users")

	if isDatastarRequest(r) {
		h.renderFragment(w, "user_list_table.html", map[string]interface{}{
			"Users":        list,
			"Pagination":   pd,
			"Query":        pp.Query,
			"StatusFilter": pp.Status,
		})
		return
	}

	h.renderPage(w, r, "user_list.html", PageData{
		Title: "Users",
		User:  session,
		Extra: map[string]interface{}{"Users": list, "Pagination": pd, "Query": pp.Query, "StatusFilter": pp.Status},
	})
}

func (h *UserHandlers) New(w http.ResponseWriter, r *http.Request) {
	session, _ := h.getUserFromContext(r)
	roles, _ := h.Services.Users.ListRoles(r.Context())
	h.renderForm(w, r, "user_edit.html", PageData{Title: "New User", User: session, Roles: roles})
}

func (h *UserHandlers) Create(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	roleID, _ := strconv.ParseInt(r.PostFormValue("role_id"), 10, 64)

	pwd := r.PostFormValue("password")
	var created domain.User
	var err error

	if pwd != "" {
		created, err = h.Services.Users.CreateUserWithPassword(
			r.Context(),
			r.PostFormValue("email"),
			r.PostFormValue("name"),
			r.PostFormValue("phone"),
			pwd,
			roleID,
			domain.UserStatus(r.PostFormValue("status")),
		)
	} else {
		created, err = h.Services.Users.CreateUser(
			r.Context(),
			r.PostFormValue("email"),
			r.PostFormValue("name"),
			r.PostFormValue("phone"),
			roleID,
			domain.UserStatus(r.PostFormValue("status")),
		)
	}

	if err != nil {
		roles, _ := h.Services.Users.ListRoles(r.Context())
		h.renderForm(w, r, "user_edit.html", PageData{Title: "New User", FlashError: err.Error(), Roles: roles})
		return
	}

	// Update Casbin policy for RBAC
	roleName := h.getRoleNameByID(r.Context(), roleID)
	_ = h.AuthSrv.AddRoleForUser(created.ID.String(), roleName)

	if isDatastarRequest(r) {
		w.Header().Set("Location", "/users")
		w.WriteHeader(http.StatusSeeOther)
		return
	}
	http.Redirect(w, r, "/users", http.StatusSeeOther)
}

func (h *UserHandlers) Edit(w http.ResponseWriter, r *http.Request) {
	id := domain.UserID(chi.URLParam(r, "id"))
	session, _ := h.getUserFromContext(r)

	user, err := h.Services.Users.GetUser(r.Context(), id)
	if err != nil {
		http.Error(w, "User not found", http.StatusNotFound)
		return
	}

	roles, _ := h.Services.Users.ListRoles(r.Context())

	h.renderForm(w, r, "user_edit.html", PageData{Title: "Edit User", User: session, UserDetail: user, Roles: roles})
}

func (h *UserHandlers) Update(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	id := domain.UserID(chi.URLParam(r, "id"))
	roleID, _ := strconv.ParseInt(r.PostFormValue("role_id"), 10, 64)

	updated, err := h.Services.Users.UpdateUser(
		r.Context(), id,
		r.PostFormValue("email"),
		r.PostFormValue("name"),
		r.PostFormValue("phone"),
		roleID,
		domain.UserStatus(r.PostFormValue("status")),
	)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Update RBAC role in Casbin
	roleName := h.getRoleNameByID(r.Context(), roleID)
	_ = h.AuthSrv.DeleteRolesForUser(updated.ID.String())
	_ = h.AuthSrv.AddRoleForUser(updated.ID.String(), roleName)

	http.Redirect(w, r, "/users", http.StatusSeeOther)
}

func (h *UserHandlers) getRoleNameByID(ctx context.Context, roleID int64) string {
	roles, err := h.Services.Users.ListRoles(ctx)
	if err != nil {
		return "viewer"
	}
	for _, r := range roles {
		if r.ID == roleID {
			return string(r.Name)
		}
	}
	return "viewer"
}

func (h *UserHandlers) Delete(w http.ResponseWriter, r *http.Request) {
	id := domain.UserID(chi.URLParam(r, "id"))
	if err := h.Services.Users.DeleteUser(r.Context(), id); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, "/users", http.StatusSeeOther)
}

func (h *UserHandlers) ResetPassword(w http.ResponseWriter, r *http.Request) {
	id := domain.UserID(chi.URLParam(r, "id"))
	_, err := h.Services.Users.GetUser(r.Context(), id)
	if err != nil {
		http.Error(w, "User not found", http.StatusNotFound)
		return
	}

	if err := h.Services.Users.ResetPassword(r.Context(), id); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if isDatastarRequest(r) {
		w.WriteHeader(http.StatusOK)
		return
	}
	http.Redirect(w, r, "/users", http.StatusSeeOther)
}
