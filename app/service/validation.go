package service

import (
	"strings"

	"pemrograman-code/app/model"
)

// ValidateCreate validasi untuk POST — business rules murni tanpa Fiber.
func ValidateCreate(req model.CreateStudentRequest) map[string]string {
	req.NIM = strings.TrimSpace(req.NIM)
	req.Name = strings.TrimSpace(req.Name)
	errs := map[string]string{}
	if req.NIM == "" {
		errs["nim"] = "wajib diisi"
	}
	if req.Name == "" {
		errs["name"] = "wajib diisi"
	}
	if req.Grade < 0 || req.Grade > 100 {
		errs["grade"] = "harus 0 - 100"
	}
	if len(errs) == 0 {
		return nil
	}
	return errs
}

// ValidateReplace validasi untuk PUT — business rules murni tanpa Fiber.
func ValidateReplace(req model.ReplaceStudentRequest) map[string]string {
	req.NIM = strings.TrimSpace(req.NIM)
	req.Name = strings.TrimSpace(req.Name)
	errs := map[string]string{}
	if req.NIM == "" {
		errs["nim"] = "wajib diisi pada PUT"
	}
	if req.Name == "" {
		errs["name"] = "wajib diisi pada PUT"
	}
	if req.Grade < 0 || req.Grade > 100 {
		errs["grade"] = "harus 0 - 100"
	}
	if len(errs) == 0 {
		return nil
	}
	return errs
}

// ApplyPatch menerapkan perubahan PATCH pada student yang sudah ada.
// Mengembalikan student hasil merge dan error validasi (nil jika valid).
// Business rules murni tanpa Fiber.
func ApplyPatch(current model.Student, req model.PatchStudentRequest) (model.Student, map[string]string) {
	if req.NIM != nil {
		if strings.TrimSpace(*req.NIM) == "" {
			return current, map[string]string{"nim": "tidak boleh kosong"}
		}
		current.NIM = strings.TrimSpace(*req.NIM)
	}
	if req.Name != nil {
		if strings.TrimSpace(*req.Name) == "" {
			return current, map[string]string{"name": "tidak boleh kosong"}
		}
		current.Name = strings.TrimSpace(*req.Name)
	}
	if req.Grade != nil {
		if *req.Grade < 0 || *req.Grade > 100 {
			return current, map[string]string{"grade": "harus 0 - 100"}
		}
		current.Grade = *req.Grade
	}
	if req.IsActive != nil {
		current.IsActive = *req.IsActive
	}
	return current, nil
}

// IsEmptyPatch mengecek apakah PATCH tidak mengubah apa pun.
func IsEmptyPatch(req model.PatchStudentRequest) bool {
	return req.NIM == nil && req.Name == nil && req.Grade == nil && req.IsActive == nil
}

// CountTotalPages menghitung total halaman (bulat ke atas).
func CountTotalPages(total, limit int) int {
	if limit <= 0 {
		return 0
	}
	return (total + limit - 1) / limit
}
