package model

// Course adalah satu kelas yang dapat diikuti mahasiswa.
//
// Credits adalah bobot SKS (aturan batas 24 SKS, Modul 9). Version dipakai
// optimistic locking saat admin mengubah kuota: perubahan hanya diterima bila
// version yang dikirim sama dengan version di database.
type Course struct {
	ID            int    `json:"id"`
	Code          string `json:"code"`
	Name          string `json:"name"`
	Credits       int    `json:"credits"`
	Quota         int    `json:"quota"`
	EnrolledCount int    `json:"enrolled_count"`
	Version       int    `json:"version"`
}

// SwapRequest adalah body POST /enrollments/swap: pindah dari satu kelas
// ke kelas lain dalam satu tindakan.
type SwapRequest struct {
	FromCourseID int `json:"from_course_id" validate:"required,gt=0"`
	ToCourseID   int `json:"to_course_id" validate:"required,gt=0,nefield=FromCourseID"`
}

// UpdateQuotaRequest adalah body PATCH /courses/:id untuk mengubah kuota
// kelas. Version wajib diisi dengan version yang terakhir dilihat admin.
type UpdateQuotaRequest struct {
	Quota   int `json:"quota" validate:"min=0,max=500"`
	Version int `json:"version" validate:"required,gt=0"`
}
