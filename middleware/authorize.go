package middleware

import (
	"fmt"

	"github.com/gofiber/fiber/v2"
	"github.com/jackc/pgx/v5/pgxpool"
	"pemrograman-code/helper"
)

// RequirePermission membuat middleware yang memeriksa apakah role user
// memiliki permission (action, resource) tertentu di tabel role_permissions.
//
// Pemakaian: router.Delete("/:id", RequireAuth(...), RequirePermission(pool, "delete", "comment"), handler)
func RequirePermission(pool *pgxpool.Pool, action, resource string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		user, ok := helper.CurrentUser(c)
		if !ok || user.RoleID == 0 {
			return helper.Fail(c, fiber.StatusForbidden, "akses ditolak: role tidak ditemukan")
		}

		var punya bool
		err := pool.QueryRow(c.UserContext(),
			`SELECT EXISTS (
				SELECT 1
				FROM role_permissions rp
				JOIN permissions p ON p.id = rp.permission_id
				WHERE rp.role_id = $1
				  AND p.action = $2
				  AND p.resource = $3
			)`,
			user.RoleID, action, resource,
		).Scan(&punya)

		// Fail closed: kalau DB error, akses tetap ditolak.
		if err != nil {
			return helper.Fail(c, fiber.StatusForbidden, "akses ditolak: gagal memeriksa permission")
		}
		if !punya {
			return helper.Fail(c, fiber.StatusForbidden,
				fmt.Sprintf("akses ditolak: tidak punya permission '%s' pada '%s'", action, resource))
		}
		return c.Next()
	}
}
