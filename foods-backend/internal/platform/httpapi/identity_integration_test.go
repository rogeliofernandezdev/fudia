package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"golang.org/x/crypto/bcrypt"
)

func seedIdentityAdministrator(t *testing.T) (*API, scope, string) {
	t.Helper()
	pool := integrationPool(t)
	s := seedInventoryScope(t, pool)
	hash, err := bcrypt.GenerateFromPassword([]byte("AdminPass123"), bcrypt.DefaultCost)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = pool.Exec(context.Background(), `
		UPDATE users SET password_hash=$2 WHERE id=$1
	`, s.UserID, string(hash)); err != nil {
		t.Fatal(err)
	}
	var roleID string
	if err = pool.QueryRow(context.Background(), `
		INSERT INTO roles(organization_id,name,description,menu_access,permissions)
		VALUES($1,$2,'Administrador funcional',ARRAY['usuarios'],ARRAY['users.read','users.manage'])
		RETURNING id
	`, s.OrganizationID, fmt.Sprintf("Admin funcional %d", time.Now().UnixNano())).Scan(&roleID); err != nil {
		t.Fatal(err)
	}
	if _, err = pool.Exec(context.Background(), `
		INSERT INTO user_roles(user_id,role_id,location_id) VALUES($1,$2,$3)
	`, s.UserID, roleID, s.LocationID); err != nil {
		t.Fatal(err)
	}
	return New(pool), s, roleID
}

func TestIdentityLoginRequiresActiveAssignment(t *testing.T) {
	pool := integrationPool(t)
	s := seedInventoryScope(t, pool)
	api := New(pool)
	password := "LoginPass123"
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		t.Fatal(err)
	}
	email := fmt.Sprintf("login-%d@example.test", time.Now().UnixNano())
	if _, err = pool.Exec(context.Background(), `
		UPDATE users SET email=$2,password_hash=$3 WHERE id=$1
	`, s.UserID, email, string(hash)); err != nil {
		t.Fatal(err)
	}

	login := func() *httptest.ResponseRecorder {
		req := httptest.NewRequest("POST", "/v1/auth/login", bytes.NewReader([]byte(fmt.Sprintf(`{"email":%q,"password":%q}`, email, password))))
		rec := httptest.NewRecorder()
		api.login(rec, req)
		return rec
	}

	blocked := login()
	if blocked.Code != 403 || !strings.Contains(blocked.Body.String(), "account_without_access") {
		t.Fatalf("expected account_without_access, got %d %s", blocked.Code, blocked.Body.String())
	}

	var roleID string
	if err = pool.QueryRow(context.Background(), `
		INSERT INTO roles(organization_id,name,menu_access,permissions)
		VALUES($1,$2,ARRAY['dashboard'],ARRAY['dashboard.read']) RETURNING id
	`, s.OrganizationID, fmt.Sprintf("Login role %d", time.Now().UnixNano())).Scan(&roleID); err != nil {
		t.Fatal(err)
	}
	if _, err = pool.Exec(context.Background(), `
		INSERT INTO user_roles(user_id,role_id,location_id) VALUES($1,$2,$3)
	`, s.UserID, roleID, s.LocationID); err != nil {
		t.Fatal(err)
	}

	ok := login()
	if ok.Code != 200 || !strings.Contains(ok.Header().Get("Set-Cookie"), "foods_session=") {
		t.Fatalf("expected successful login with session, got %d %s cookie=%s", ok.Code, ok.Body.String(), ok.Header().Get("Set-Cookie"))
	}
}

func TestIdentityRejectsUnknownRoleCatalogValues(t *testing.T) {
	api, s, _ := seedIdentityAdministrator(t)
	body := []byte(`{"name":"Rol inválido","description":"Prueba","menuAccess":["usuarios"],"permissions":["users.read","permission.invented"]}`)
	req := httptest.NewRequest("POST", "/v1/admin/roles", bytes.NewReader(body))
	req = req.WithContext(context.WithValue(req.Context(), scopeKey{}, s))
	rec := httptest.NewRecorder()
	api.createRole(rec, req)
	if rec.Code != 400 || !strings.Contains(rec.Body.String(), "invalid_role") {
		t.Fatalf("expected invalid_role, got %d %s", rec.Code, rec.Body.String())
	}
}

func TestIdentityInvalidAssignmentPreservesPreviousAccess(t *testing.T) {
	api, s, adminRoleID := seedIdentityAdministrator(t)
	pool := api.db
	ctx := context.Background()

	hash, _ := bcrypt.GenerateFromPassword([]byte("UserPass123"), bcrypt.DefaultCost)
	var targetID string
	if err := pool.QueryRow(ctx, `
		INSERT INTO users(organization_id,email,full_name,password_hash)
		VALUES($1,$2,'Usuario objetivo',$3) RETURNING id
	`, s.OrganizationID, fmt.Sprintf("target-%d@example.test", time.Now().UnixNano()), string(hash)).Scan(&targetID); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO user_roles(user_id,role_id,location_id) VALUES($1,$2,$3)`, targetID, adminRoleID, s.LocationID); err != nil {
		t.Fatal(err)
	}
	var inactiveRoleID string
	if err := pool.QueryRow(ctx, `
		INSERT INTO roles(organization_id,name,description,menu_access,permissions,active)
		VALUES($1,$2,'Inactivo',ARRAY['dashboard'],ARRAY['dashboard.read'],false) RETURNING id
	`, s.OrganizationID, fmt.Sprintf("Inactive %d", time.Now().UnixNano())).Scan(&inactiveRoleID); err != nil {
		t.Fatal(err)
	}

	body := []byte(fmt.Sprintf(`{"fullName":"Usuario objetivo","email":"target-%d@example.test","password":"","assignments":[{"roleId":%q,"locationId":%q}]}`, time.Now().UnixNano(), inactiveRoleID, s.LocationID))
	req := httptest.NewRequest("PATCH", "/v1/admin/users/"+targetID, bytes.NewReader(body))
	req.SetPathValue("id", targetID)
	req = req.WithContext(context.WithValue(req.Context(), scopeKey{}, s))
	rec := httptest.NewRecorder()
	api.updateUser(rec, req)
	if rec.Code != 400 || !strings.Contains(rec.Body.String(), "invalid_assignment") {
		t.Fatalf("expected invalid_assignment, got %d %s", rec.Code, rec.Body.String())
	}

	var count int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM user_roles WHERE user_id=$1 AND role_id=$2 AND location_id=$3`, targetID, adminRoleID, s.LocationID).Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 1 {
		t.Fatalf("previous assignment must remain after rejected update, got %d", count)
	}
}

func TestIdentityTenantAdminCannotModifyPlatformAdmin(t *testing.T) {
	api, s, adminRoleID := seedIdentityAdministrator(t)
	pool := api.db
	ctx := context.Background()
	hash, _ := bcrypt.GenerateFromPassword([]byte("PlatformPass123"), bcrypt.DefaultCost)
	var platformID string
	email := fmt.Sprintf("platform-%d@example.test", time.Now().UnixNano())
	if err := pool.QueryRow(ctx, `
		INSERT INTO users(organization_id,email,full_name,password_hash,platform_admin)
		VALUES($1,$2,'Platform Admin',$3,true) RETURNING id
	`, s.OrganizationID, email, string(hash)).Scan(&platformID); err != nil {
		t.Fatal(err)
	}

	body := []byte(fmt.Sprintf(`{"fullName":"Platform alterado","email":%q,"password":"","assignments":[{"roleId":%q,"locationId":%q}]}`, email, adminRoleID, s.LocationID))
	req := httptest.NewRequest("PATCH", "/v1/admin/users/"+platformID, bytes.NewReader(body))
	req.SetPathValue("id", platformID)
	req = req.WithContext(context.WithValue(req.Context(), scopeKey{}, s))
	rec := httptest.NewRecorder()
	api.updateUser(rec, req)
	if rec.Code != 403 || !strings.Contains(rec.Body.String(), "protected_platform_admin") {
		t.Fatalf("expected protected_platform_admin edit, got %d %s", rec.Code, rec.Body.String())
	}

	statusReq := httptest.NewRequest("PATCH", "/v1/admin/users/"+platformID+"/status", bytes.NewReader([]byte(`{"active":false}`)))
	statusReq.SetPathValue("id", platformID)
	statusReq = statusReq.WithContext(context.WithValue(statusReq.Context(), scopeKey{}, s))
	statusRec := httptest.NewRecorder()
	api.updateUserStatus(statusRec, statusReq)
	if statusRec.Code != 403 || !strings.Contains(statusRec.Body.String(), "protected_platform_admin") {
		t.Fatalf("expected protected_platform_admin status, got %d %s", statusRec.Code, statusRec.Body.String())
	}
}

func TestIdentityCannotDisableLastAdministrativeRole(t *testing.T) {
	api, s, adminRoleID := seedIdentityAdministrator(t)
	req := httptest.NewRequest("PATCH", "/v1/admin/roles/"+adminRoleID+"/status", bytes.NewReader([]byte(`{"active":false}`)))
	req.SetPathValue("id", adminRoleID)
	req = req.WithContext(context.WithValue(req.Context(), scopeKey{}, s))
	rec := httptest.NewRecorder()
	api.updateRoleStatus(rec, req)
	if rec.Code != 409 || !strings.Contains(rec.Body.String(), "last_administrator") {
		t.Fatalf("expected last_administrator, got %d %s", rec.Code, rec.Body.String())
	}
	var active bool
	if err := api.db.QueryRow(context.Background(), `SELECT active FROM roles WHERE id=$1`, adminRoleID).Scan(&active); err != nil {
		t.Fatal(err)
	}
	if !active {
		t.Fatal("last administrative role must remain active")
	}
}

func TestIdentityDefaultRolesProtectAdministrator(t *testing.T) {
	api, s, _ := seedIdentityAdministrator(t)
	tx, err := api.db.Begin(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	adminID, err := seedOrganizationRoles(context.Background(), tx, s.OrganizationID)
	if err != nil {
		_ = tx.Rollback(context.Background())
		t.Fatal(err)
	}
	if err = tx.Commit(context.Background()); err != nil {
		t.Fatal(err)
	}
	var key,name string
	var permissions, menus []string
	if err := api.db.QueryRow(context.Background(), `SELECT system_key,name,permissions,menu_access FROM roles WHERE id=$1`, adminID).Scan(&key,&name,&permissions,&menus); err != nil {
		t.Fatal(err)
	}
	contains:=func(values []string,want string)bool{for _,value:=range values{if value==want{return true}};return false}
	if key != "administrator" || name != "Administrador de empresa" || contains(permissions,"*") || contains(menus,"*") || !contains(permissions,"users.manage") || !contains(permissions,"organizations.manage") {
		t.Fatalf("company administrator must be explicit and tenant-scoped: key=%q name=%q perms=%v menus=%v", key,name,permissions,menus)
	}
	var defaults int
	if err := api.db.QueryRow(context.Background(), `SELECT count(*) FROM roles WHERE organization_id=$1 AND system_key IS NOT NULL`, s.OrganizationID).Scan(&defaults); err != nil {
		t.Fatal(err)
	}
	if defaults != len(defaultOrganizationRoles) {
		t.Fatalf("expected %d default roles, got %d", len(defaultOrganizationRoles), defaults)
	}
}

func TestIdentityAvailableLocationsFollowAssignments(t *testing.T) {
	api, s, roleID := seedIdentityAdministrator(t)
	ctx := context.Background()
	var secondLocation string
	if err := api.db.QueryRow(ctx, `
		INSERT INTO locations(organization_id,name,code,address)
		VALUES($1,'Secundaria',$2,'') RETURNING id
	`, s.OrganizationID, fmt.Sprintf("ID-%d", time.Now().UnixNano()%1000000)).Scan(&secondLocation); err != nil {
		t.Fatal(err)
	}

	list := func() []locationSummary {
		req := httptest.NewRequest("GET", "/v1/admin/locations/available", nil)
		req = req.WithContext(context.WithValue(req.Context(), scopeKey{}, s))
		rec := httptest.NewRecorder()
		api.listAvailableLocations(rec, req)
		if rec.Code != 200 {
			t.Fatalf("locations: %d %s", rec.Code, rec.Body.String())
		}
		var out struct{ Items []locationSummary `json:"items"` }
		if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
			t.Fatal(err)
		}
		return out.Items
	}
	if got := len(list()); got != 1 {
		t.Fatalf("expected only assigned location, got %d", got)
	}
	if _, err := api.db.Exec(ctx, `INSERT INTO user_roles(user_id,role_id,location_id) VALUES($1,$2,$3)`, s.UserID, roleID, secondLocation); err != nil {
		t.Fatal(err)
	}
	if got := len(list()); got != 2 {
		t.Fatalf("expected both assigned locations, got %d", got)
	}
}

func TestIdentityProfilePasswordChange(t *testing.T) {
	api, s, _ := seedIdentityAdministrator(t)
	body := []byte(`{"fullName":"Administrador Renombrado","currentPassword":"AdminPass123","newPassword":"NewAdminPass123"}`)
	req := httptest.NewRequest("PATCH", "/v1/admin/me", bytes.NewReader(body))
	req = req.WithContext(context.WithValue(req.Context(), scopeKey{}, s))
	rec := httptest.NewRecorder()
	api.updateMyProfile(rec, req)
	if rec.Code != 200 {
		t.Fatalf("profile update: %d %s", rec.Code, rec.Body.String())
	}
	var name, hash string
	if err := api.db.QueryRow(context.Background(), `SELECT full_name,password_hash FROM users WHERE id=$1`, s.UserID).Scan(&name,&hash); err != nil {
		t.Fatal(err)
	}
	if name != "Administrador Renombrado" || bcrypt.CompareHashAndPassword([]byte(hash), []byte("NewAdminPass123")) != nil {
		t.Fatalf("profile/password did not update correctly name=%q", name)
	}
}


func TestIdentityProfileLoadsAfterRoleChange(t *testing.T) {
	api, adminScope, _ := seedIdentityAdministrator(t)
	pool := api.db
	ctx := context.Background()

	var firstRoleID, nextRoleID string
	if err := pool.QueryRow(ctx, `
		INSERT INTO roles(organization_id,name,description,menu_access,permissions)
		VALUES($1,$2,'Rol inicial',ARRAY['pedidos'],ARRAY['orders.read'])
		RETURNING id
	`, adminScope.OrganizationID, fmt.Sprintf("Mesero %d", time.Now().UnixNano())).Scan(&firstRoleID); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(ctx, `
		INSERT INTO roles(organization_id,name,description,menu_access,permissions)
		VALUES($1,$2,'Rol actualizado',ARRAY['dashboard','usuarios'],ARRAY['dashboard.read','users.read'])
		RETURNING id
	`, adminScope.OrganizationID, fmt.Sprintf("Administrador prueba %d", time.Now().UnixNano())).Scan(&nextRoleID); err != nil {
		t.Fatal(err)
	}

	hash, err := bcrypt.GenerateFromPassword([]byte("GastonPass123"), bcrypt.DefaultCost)
	if err != nil {
		t.Fatal(err)
	}
	email := fmt.Sprintf("gaston-%d@example.test", time.Now().UnixNano())
	var targetID string
	if err = pool.QueryRow(ctx, `
		INSERT INTO users(organization_id,email,full_name,password_hash)
		VALUES($1,$2,'Gaston Acurio',$3)
		RETURNING id
	`, adminScope.OrganizationID, email, string(hash)).Scan(&targetID); err != nil {
		t.Fatal(err)
	}
	if _, err = pool.Exec(ctx, `
		INSERT INTO user_roles(user_id,role_id,location_id)
		VALUES($1,$2,$3)
	`, targetID, firstRoleID, adminScope.LocationID); err != nil {
		t.Fatal(err)
	}

	updateBody := []byte(fmt.Sprintf(
		`{"fullName":"Gaston Acurio","email":%q,"password":"","assignments":[{"roleId":%q,"locationId":%q}]}`,
		email, nextRoleID, adminScope.LocationID,
	))
	updateReq := httptest.NewRequest("PATCH", "/v1/admin/users/"+targetID, bytes.NewReader(updateBody))
	updateReq.SetPathValue("id", targetID)
	updateReq = updateReq.WithContext(context.WithValue(updateReq.Context(), scopeKey{}, adminScope))
	updateRec := httptest.NewRecorder()
	api.updateUser(updateRec, updateReq)
	if updateRec.Code != 200 {
		t.Fatalf("role change failed: %d %s", updateRec.Code, updateRec.Body.String())
	}

	targetScope := scope{
		UserID: targetID,
		OrganizationID: adminScope.OrganizationID,
		LocationID: adminScope.LocationID,
		Name: "Gaston Acurio",
	}
	profileReq := httptest.NewRequest("GET", "/v1/admin/me", nil)
	profileReq = profileReq.WithContext(context.WithValue(profileReq.Context(), scopeKey{}, targetScope))
	profileRec := httptest.NewRecorder()
	api.getMyProfile(profileRec, profileReq)
	if profileRec.Code != 200 {
		t.Fatalf("profile after role change: %d %s", profileRec.Code, profileRec.Body.String())
	}

	var profile myProfileView
	if err := json.Unmarshal(profileRec.Body.Bytes(), &profile); err != nil {
		t.Fatal(err)
	}
	if profile.FullName != "Gaston Acurio" {
		t.Fatalf("unexpected profile name %q", profile.FullName)
	}
	if len(profile.RoleNames) != 1 || !strings.HasPrefix(profile.RoleNames[0], "Administrador prueba ") {
		t.Fatalf("expected updated role in profile, got %v", profile.RoleNames)
	}
	if len(profile.Permissions) != 2 || !containsString(profile.Permissions, "dashboard.read") || !containsString(profile.Permissions, "users.read") {
		t.Fatalf("expected updated permissions in profile, got %v", profile.Permissions)
	}
}

func containsString(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}


func TestIdentityListUsersIncludesAssignments(t *testing.T) {
	api, s, roleID := seedIdentityAdministrator(t)
	ctx := context.Background()
	hash, err := bcrypt.GenerateFromPassword([]byte("ListUserPass123"), bcrypt.DefaultCost)
	if err != nil {
		t.Fatal(err)
	}
	email := fmt.Sprintf("list-user-%d@example.test", time.Now().UnixNano())
	var userID string
	if err = api.db.QueryRow(ctx, `
		INSERT INTO users(organization_id,email,full_name,password_hash)
		VALUES($1,$2,'Usuario listado',$3)
		RETURNING id
	`, s.OrganizationID, email, string(hash)).Scan(&userID); err != nil {
		t.Fatal(err)
	}
	if _, err = api.db.Exec(ctx, `
		INSERT INTO user_roles(user_id,role_id,location_id)
		VALUES($1,$2,$3)
	`, userID, roleID, s.LocationID); err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest("GET", "/v1/admin/users?q="+email+"&page=1&pageSize=20", nil)
	req = req.WithContext(context.WithValue(req.Context(), scopeKey{}, s))
	rec := httptest.NewRecorder()
	api.listUsers(rec, req)
	if rec.Code != 200 {
		t.Fatalf("list users: %d %s", rec.Code, rec.Body.String())
	}
	var out struct {
		Items []userView `json:"items"`
		Total int        `json:"total"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatal(err)
	}
	if out.Total != 1 || len(out.Items) != 1 {
		t.Fatalf("expected one filtered user, total=%d items=%d", out.Total, len(out.Items))
	}
	if out.Items[0].ID != userID || len(out.Items[0].Assignments) != 1 {
		t.Fatalf("expected assignment for listed user: %+v", out.Items[0])
	}
	if out.Items[0].Assignments[0].RoleID != roleID || out.Items[0].Assignments[0].LocationID != s.LocationID {
		t.Fatalf("unexpected assignment: %+v", out.Items[0].Assignments[0])
	}
}
