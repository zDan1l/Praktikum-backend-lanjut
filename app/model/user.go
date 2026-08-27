package model

import "time"

// Offset menghitung berapa baris yang dilewati untuk halaman ini.
// Perhitungan ini pindah ke sini karena kini dipakai langsung oleh SQL.
func (q ListQuery) Offset() int {
 return (q.Page - 1) * q.Limit
}


type User struct { ID int; Username string; Email string; Password string; IsActive bool; CreatedAt time.Time }

type ListQuery struct { Page int; Limit int; Search string; Sort string; Order string; IsActive *bool }

type Meta struct { Page int; Limit int; Total int; TotalPages int }

type WebResponse struct { Success bool; Message string; Data any; Meta *Meta; Errors any } // atau tetap di main jika pilih opsi B

type CreateUserRequest struct { Username string `json:"username"`; Email string `json:"email"`; Password string `json:"password"` }
type ReplaceUserRequest struct { Username string `json:"username"`; Email string `json:"email"`; IsActive bool `json:"is_active"` }
type PatchUserRequest struct { Username *string `json:"username"`; Email *string `json:"email"`; IsActive *bool `json:"is_active"` }