package model

import "time"

type Prestasi struct {
	ID           int        `json:"id"`
	ID_Student   int        `json:"id_student"`
	NamaPrestasi string     `json:"nama_prestasi"`
	Juara        int        `json:"juara"`
	CreatedAt    *time.Time `json:"created_at"`
}


