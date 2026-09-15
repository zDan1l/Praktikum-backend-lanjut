package service

import (
	"strings"

	"pemrograman-code/app/model"
)

// Validasi murni tanpa Fiber - gampang di-test & di-copy untuk resource baru

func ValidatePrestasiCreate(req model.CreatePrestasiRequest) map[string]string {
	errs := map[string]string{}
	if req.IDStudent < 1 {
		errs["id_student"] = "wajib diisi dan >0"
	}
	if strings.TrimSpace(req.NamaPrestasi) == "" {
		errs["nama_prestasi"] = "wajib diisi"
	}
	if req.Juara < 1 {
		errs["juara"] = "harus >=1"
	}
	if len(errs) == 0 {
		return nil
	}
	return errs
}

func ValidatePrestasiReplace(req model.ReplacePrestasiRequest) map[string]string {
	errs := map[string]string{}
	if req.IDStudent < 1 {
		errs["id_student"] = "wajib diisi dan >0"
	}
	if strings.TrimSpace(req.NamaPrestasi) == "" {
		errs["nama_prestasi"] = "wajib diisi"
	}
	if req.Juara < 1 {
		errs["juara"] = "harus >=1"
	}
	if len(errs) == 0 {
		return nil
	}
	return errs
}

func ApplyPrestasiPatch(current model.Prestasi, req model.PatchPrestasiRequest) (model.Prestasi, map[string]string) {
	if req.IDStudent != nil {
		if *req.IDStudent < 1 {
			return current, map[string]string{"id_student": "harus >=1"}
		}
		current.ID_Student = *req.IDStudent
	}
	if req.NamaPrestasi != nil {
		if strings.TrimSpace(*req.NamaPrestasi) == "" {
			return current, map[string]string{"nama_prestasi": "tidak boleh kosong"}
		}
		current.NamaPrestasi = strings.TrimSpace(*req.NamaPrestasi)
	}
	if req.Juara != nil {
		if *req.Juara < 1 {
			return current, map[string]string{"juara": "harus >=1"}
		}
		current.Juara = *req.Juara
	}
	return current, nil
}
