package model

// Course adalah satu kelas yang dapat diikuti mahasiswa.
type Course struct {
	ID            int    `json:"id"`
	Code          string `json:"code"`
	Name          string `json:"name"`
	Capacity      int    `json:"capacity"`
	EnrolledCount int    `json:"enrolled_count"`
}

// SwapRequest adalah body POST /enrollments/swap: pindah dari satu kelas
// ke kelas lain dalam satu tindakan.
type SwapRequest struct {
	FromCourseID int `json:"from_course_id"`
	ToCourseID   int `json:"to_course_id"`
}

// UpdateQuotaRequest adalah body PATCH /courses/:id (Modul 9) untuk mengubah
// kuota kelas. Dipakai ulang oleh tugas mandiri D.2.
type UpdateQuotaRequest struct {
	Capacity int `json:"capacity"`
}
