package models

import "time"

type AdminRole string

const (
	AdminRoleOwner AdminRole = "owner"
	AdminRoleStaff AdminRole = "staff"
)

type Admin struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Role      AdminRole `json:"role"`
	CreatedAt time.Time `json:"created_at"`
}
