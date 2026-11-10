package repository

import (
	"context"
	"fmt"
)

// RoleRepository membaca pemetaan role -> permission dari tabel RBAC.
type RoleRepository struct{ db DBTX }

func NewRoleRepository(db DBTX) *RoleRepository { return &RoleRepository{db: db} }

// LoadPermissions mengembalikan seluruh permission per nama role, ditulis
// sebagai "resource:action" (mis. "queue:monitor", "review:moderate").
//
// Dibaca SEKALI saat aplikasi menyala lalu disimpan di helper.PermissionSet,
// sehingga pemeriksaan permission tidak lagi butuh query per request.
func (r *RoleRepository) LoadPermissions(ctx context.Context) (map[string][]string, error) {
	rows, err := r.db.Query(ctx, `
		SELECT ro.name, p.resource || ':' || p.action
		FROM role_permissions rp
		JOIN roles ro ON ro.id = rp.role_id
		JOIN permissions p ON p.id = rp.permission_id
		ORDER BY ro.name, p.resource, p.action`)
	if err != nil {
		return nil, fmt.Errorf("memuat permission: %w", err)
	}
	defer rows.Close()

	result := map[string][]string{}
	for rows.Next() {
		var role, permission string
		if err := rows.Scan(&role, &permission); err != nil {
			return nil, fmt.Errorf("membaca permission: %w", err)
		}
		result[role] = append(result[role], permission)
	}
	return result, rows.Err()
}
