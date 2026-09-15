package service

import (
	"testing"

	"pemrograman-code/app/model"
)

func TestValidateRegister(t *testing.T) {
	cases := []struct {
		name string
		req  model.RegisterRequest
		want int
	}{
		{"valid", model.RegisterRequest{Username: "sari", Email: "sari@example.com", Password: "rahasia123"}, 0},
		{"username pendek", model.RegisterRequest{Username: "ab", Email: "a@b.com", Password: "rahasia123"}, 1},
		{"username kosong", model.RegisterRequest{Username: " ", Email: "a@b.com", Password: "rahasia123"}, 1},
		{"email invalid", model.RegisterRequest{Username: "sari", Email: "not-email", Password: "rahasia123"}, 1},
		{"password lemah", model.RegisterRequest{Username: "sari", Email: "sari@example.com", Password: "password1"}, 1},
		{"password pendek", model.RegisterRequest{Username: "sari", Email: "sari@example.com", Password: "pendek1"}, 1},
	}
	for _, tc := range cases {
		errs := ValidateRegister(tc.req)
		if tc.want == 0 && errs != nil {
			t.Errorf("%s: harap tidak ada error, dapat %v", tc.name, errs)
		}
		if tc.want != 0 && len(errs) != tc.want {
			t.Errorf("%s: harap %d error, dapat %d (%v)", tc.name, tc.want, len(errs), errs)
		}
	}
}

func TestValidateLogin(t *testing.T) {
	cases := []struct {
		name string
		req  model.LoginRequest
		want int
	}{
		{"valid", model.LoginRequest{Username: "sari", Password: "rahasia123"}, 0},
		{"username kosong", model.LoginRequest{Username: " ", Password: "x"}, 1},
		{"password kosong", model.LoginRequest{Username: "sari", Password: ""}, 1},
		{"semua kosong", model.LoginRequest{Username: "", Password: ""}, 2},
	}
	for _, tc := range cases {
		errs := ValidateLogin(tc.req)
		if tc.want == 0 && errs != nil {
			t.Errorf("%s: harap tidak ada error, dapat %v", tc.name, errs)
		}
		if tc.want != 0 && len(errs) != tc.want {
			t.Errorf("%s: harap %d error, dapat %d (%v)", tc.name, tc.want, len(errs), errs)
		}
	}
}

func TestCheckPasswordStrength(t *testing.T) {
	if msg := checkPasswordStrength("rahasia123"); msg != "" {
		t.Errorf("password kuat seharusnya lolos, dapat %s", msg)
	}
	if msg := checkPasswordStrength("12345678"); msg == "" {
		t.Error("password angka saja harus gagal")
	}
	if msg := checkPasswordStrength("pendek1"); msg == "" {
		t.Error("password pendek harus gagal")
	}
}
