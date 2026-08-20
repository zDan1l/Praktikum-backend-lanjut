package main

type Student struct {
	ID       int
	Name     string
	Grade    float64
	IsActive bool
}

// POST — semua field wajib
type CreateStudentRequest struct {
	Name     string  `json:"name"`
	Grade    float64 `json:"grade"`
	IsActive bool    `json:"is_active"`
}

// PUT — ganti seluruh isi, jadi field bertipe biasa dan semuanya wajib
type ReplaceStudentRequest struct {
	Name     string  `json:"name"`
	Grade    float64 `json:"grade"`
	IsActive bool    `json:"is_active"`
}

// PATCH — ubah sebagian, jadi field bertipe pointer supaya bisa dibedakan
// antara "tidak dikirim" (nil) dan "dikirim bernilai kosong"
type PatchStudentRequest struct {
	Name     *string  `json:"name,omitempty"`
	Grade    *float64 `json:"grade,omitempty"`
	IsActive *bool    `json:"is_active,omitempty"`
}

// Amplop baku untuk semua respons
type WebResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
	Data    any    `json:"data,omitempty"`
	Meta    *Meta  `json:"meta,omitempty"`
	Errors  any    `json:"errors,omitempty"`
}

type Meta struct {
	Page       int `json:"page"`
	Limit      int `json:"limit"`
	Total      int `json:"total"`
	TotalPages int `json:"total_pages"`
}

type ListQuery struct {
	Page     int
	Limit    int
	Search   string
	Sort     string
	Order    string
	IsActive *bool
}
