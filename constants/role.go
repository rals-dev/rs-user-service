package constants

const (
	Admin    = 1
	Customer = 2
)

// AdminRoleCode is the lowercase role code (see database/seeders/role_seeder.go
// and services/user.Login, which lowercase the stored role code) used to grant
// admin-only bypass of ownership checks, e.g. in the user IDOR authorization logic.
const AdminRoleCode = "admin"
