package model

import "time"

// Prestasi - entitas utama
type Prestasi struct {
	ID           int        `json:"id"`
	ID_Student   int        `json:"id_student"`
	NamaPrestasi string     `json:"nama_prestasi"`
	Juara        int        `json:"juara"`
	CreatedAt    *time.Time `json:"created_at"`
}

// DTO untuk POST /prestasi
type CreatePrestasiRequest struct {
	IDStudent    int    `json:"id_student"`
	NamaPrestasi string `json:"nama_prestasi"`
	Juara        int    `json:"juara"`
}

// DTO untuk PUT /prestasi/:id (ganti semua field)
type ReplacePrestasiRequest struct {
	IDStudent    int    `json:"id_student"`
	NamaPrestasi string `json:"nama_prestasi"`
	Juara        int    `json:"juara"`
}

// DTO untuk PATCH /prestasi/:id (salah satu field)
type PatchPrestasiRequest struct {
	IDStudent    *int    `json:"id_student"`
	NamaPrestasi *string `json:"nama_prestasi"`
	Juara        *int    `json:"juara"`
}
