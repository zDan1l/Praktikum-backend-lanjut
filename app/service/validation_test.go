package service

import (
	"testing"

	"pemrograman-code/app/model"
)

func TestValidateCreate(t *testing.T) {
	cases := []struct {
		name string
		req  model.CreateStudentRequest
		want int // jumlah error
	}{
		{"valid", model.CreateStudentRequest{NIM: "22001", Name: "Budi", Grade: 85.5}, 0},
		{"nim kosong", model.CreateStudentRequest{NIM: " ", Name: "Budi", Grade: 85.5}, 1},
		{"name kosong", model.CreateStudentRequest{NIM: "22001", Name: " ", Grade: 85.5}, 1},
		{"grade negatif", model.CreateStudentRequest{NIM: "22001", Name: "Budi", Grade: -1}, 1},
		{"grade >100", model.CreateStudentRequest{NIM: "22001", Name: "Budi", Grade: 150}, 1},
		{"semua kosong", model.CreateStudentRequest{NIM: "", Name: "", Grade: 200}, 3},
	}
	for _, tc := range cases {
		errs := ValidateCreate(tc.req)
		if tc.want == 0 && errs != nil {
			t.Errorf("%s: harap tidak ada error, dapat %v", tc.name, errs)
		}
		if tc.want != 0 && len(errs) != tc.want {
			t.Errorf("%s: harap %d error, dapat %d (%v)", tc.name, tc.want, len(errs), errs)
		}
	}
}

func TestValidateReplace(t *testing.T) {
	cases := []struct {
		name string
		req  model.ReplaceStudentRequest
		want int
	}{
		{"valid", model.ReplaceStudentRequest{NIM: "22001", Name: "Budi", Grade: 90}, 0},
		{"nim kosong", model.ReplaceStudentRequest{NIM: "", Name: "Budi", Grade: 90}, 1},
		{"name kosong", model.ReplaceStudentRequest{NIM: "22001", Name: "", Grade: 90}, 1},
		{"grade invalid", model.ReplaceStudentRequest{NIM: "22001", Name: "Budi", Grade: -5}, 1},
	}
	for _, tc := range cases {
		errs := ValidateReplace(tc.req)
		if tc.want == 0 && errs != nil {
			t.Errorf("%s: harap tidak ada error, dapat %v", tc.name, errs)
		}
		if tc.want != 0 && len(errs) != tc.want {
			t.Errorf("%s: harap %d error, dapat %d (%v)", tc.name, tc.want, len(errs), errs)
		}
	}
}

func TestApplyPatch(t *testing.T) {
	gradeBaru := 95.5
	nimBaru := "22099"
	gradeInvalid := 150.0
	current := model.Student{ID: 1, NIM: "22001", Name: "Sari", Grade: 80, IsActive: true}

	// Patch grade saja
	updated, errs := ApplyPatch(current, model.PatchStudentRequest{Grade: &gradeBaru})
	if errs != nil {
		t.Fatalf("tidak seharusnya error: %v", errs)
	}
	if updated.Grade != 95.5 {
		t.Errorf("grade harus 95.5, dapat %v", updated.Grade)
	}
	if updated.Name != "Sari" {
		t.Error("field yang tidak dikirim seharusnya tidak berubah")
	}

	// Patch nim
	updated, errs = ApplyPatch(current, model.PatchStudentRequest{NIM: &nimBaru})
	if errs != nil {
		t.Fatalf("tidak seharusnya error: %v", errs)
	}
	if updated.NIM != "22099" {
		t.Errorf("nim harus 22099, dapat %s", updated.NIM)
	}

	// Patch invalid grade
	_, errs = ApplyPatch(current, model.PatchStudentRequest{Grade: &gradeInvalid})
	if len(errs) == 0 {
		t.Error("seharusnya error untuk grade invalid")
	}

	// Patch is_active
	inactive := false
	updated, errs = ApplyPatch(current, model.PatchStudentRequest{IsActive: &inactive})
	if errs != nil {
		t.Fatalf("tidak seharusnya error: %v", errs)
	}
	if updated.IsActive {
		t.Error("is_active seharusnya false")
	}
}

func TestIsEmptyPatch(t *testing.T) {
	if !IsEmptyPatch(model.PatchStudentRequest{}) {
		t.Error("seharusnya true untuk patch kosong")
	}
	nim := "22001"
	if IsEmptyPatch(model.PatchStudentRequest{NIM: &nim}) {
		t.Error("seharusnya false jika ada field")
	}
}

func TestCountTotalPages(t *testing.T) {
	cases := []struct{ total, limit, want int }{
		{0, 10, 0},
		{1, 10, 1},
		{10, 10, 1},
		{11, 10, 2},
		{137, 20, 7},
		{100, 0, 0},
	}
	for _, tc := range cases {
		if got := CountTotalPages(tc.total, tc.limit); got != tc.want {
			t.Errorf("total=%d limit=%d: harap %d, dapat %d", tc.total, tc.limit, tc.want, got)
		}
	}
}
