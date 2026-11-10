package helper

// PermissionSet menyimpan pemetaan role -> permission di memory.
//
// Diisi SEKALI saat aplikasi menyala dari RoleRepository.LoadPermissions.
// Permission ditulis sebagai "resource:action", misalnya "queue:monitor".
// Unit test dapat membuatnya langsung dari map tanpa database.
type PermissionSet struct {
	byRole map[string]map[string]bool
}

func NewPermissionSet(raw map[string][]string) *PermissionSet {
	set := &PermissionSet{byRole: make(map[string]map[string]bool, len(raw))}
	for role, permissions := range raw {
		m := make(map[string]bool, len(permissions))
		for _, p := range permissions {
			m[p] = true
		}
		set.byRole[role] = m
	}
	return set
}

// Has melaporkan apakah role memiliki permission. PermissionSet nil tidak
// memberi izin apa pun (fail closed).
func (p *PermissionSet) Has(role, permission string) bool {
	if p == nil {
		return false
	}
	return p.byRole[role][permission]
}
