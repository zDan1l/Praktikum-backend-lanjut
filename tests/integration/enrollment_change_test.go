//go:build integration

package integration_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"pemrograman-code/app/model"
)

// Tugas Mandiri D.2: integration test untuk POST /enrollments/swap dan
// DELETE /courses/:id/enroll terhadap PostgreSQL sungguhan.

// send mengirim request ber-body JSON atas nama student dan mengembalikan
// status code beserta field "code" pada response (kosong untuk 204).
func (a api) send(t *testing.T, student model.User, method, path, body string) (int, string) {
	t.Helper()

	token, err := a.jwt.GenerateAccessToken(model.AuthUser{UserID: student.ID, Username: student.Username, Role: "user"})
	if err != nil {
		t.Fatalf("membuat token: %v", err)
	}
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := a.app.Test(req, -1)
	if err != nil {
		t.Fatalf("%s %s: %v", method, path, err)
	}
	defer resp.Body.Close()

	var parsed struct {
		Code string `json:"code"`
	}
	_ = json.NewDecoder(resp.Body).Decode(&parsed)
	return resp.StatusCode, parsed.Code
}

func enrolledCount(t *testing.T, courseID int) int {
	t.Helper()
	return countRows(t, `SELECT enrolled_count FROM courses WHERE id = $1`, courseID)
}

func isEnrolled(t *testing.T, courseID, userID int) bool {
	t.Helper()
	return countRows(t, `SELECT COUNT(*) FROM enrollments WHERE course_id = $1 AND user_id = $2`, courseID, userID) == 1
}

// Pindah ke kelas yang sudah penuh harus ditolak SELURUHNYA: mahasiswa tetap
// di kelas asal, kedua counter tidak berubah, dan tidak ada event pindah.
// Inilah jaminan "satu tindakan" milik transaction: langkah keluar dari kelas
// asal yang sudah sempat dijalankan ikut dibatalkan.
func TestSwapEndpoint_TargetFullKeepsOriginalEnrollment(t *testing.T) {
	a := newAPI(t)
	from := seedCourse(t, 3, 40)
	to := seedCourse(t, 3, 1) // kuota satu kursi
	student := seedStudent(t)
	other := seedStudent(t)

	if status, code := a.enroll(t, student, from.ID); status != http.StatusCreated {
		t.Fatalf("mendaftar kelas asal: dapat %d %s, seharusnya 201", status, code)
	}
	if status, code := a.enroll(t, other, to.ID); status != http.StatusCreated {
		t.Fatalf("mengisi kursi terakhir kelas tujuan: dapat %d %s, seharusnya 201", status, code)
	}
	fromBefore, toBefore := enrolledCount(t, from.ID), enrolledCount(t, to.ID)

	body := `{"from_course_id": ` + strconv.Itoa(from.ID) + `, "to_course_id": ` + strconv.Itoa(to.ID) + `}`
	status, code := a.send(t, student, http.MethodPost, "/api/v1/enrollments/swap", body)
	if status != http.StatusConflict || code != "COURSE_FULL" {
		t.Fatalf("swap ke kelas penuh: dapat %d %s, seharusnya 409 COURSE_FULL", status, code)
	}

	if !isEnrolled(t, from.ID, student.ID) {
		t.Errorf("mahasiswa tidak lagi terdaftar di kelas asal; seharusnya TETAP terdaftar")
	}
	if isEnrolled(t, to.ID, student.ID) {
		t.Errorf("mahasiswa terdaftar di kelas tujuan yang penuh")
	}
	if got := enrolledCount(t, from.ID); got != fromBefore {
		t.Errorf("enrolled_count kelas asal = %d, seharusnya tetap %d", got, fromBefore)
	}
	if got := enrolledCount(t, to.ID); got != toBefore {
		t.Errorf("enrolled_count kelas tujuan = %d, seharusnya tetap %d", got, toBefore)
	}
	swapped := countRows(t,
		`SELECT COUNT(*) FROM outbox WHERE (payload->>'user_id')::INT = $1 AND event_type = $2`,
		student.ID, model.EventEnrollmentSwapped)
	if swapped != 0 {
		t.Errorf("event enrollment.swapped = %d, seharusnya 0 (swap yang gagal tidak boleh meninggalkan jejak)", swapped)
	}
}

// Keluar dari kelas: 204, baris enrollment hilang, kursi kembali satu, dan
// tepat satu event enrollment.dropped tercatat di outbox.
func TestDropEndpoint_ReleasesSeatAndWritesOneEvent(t *testing.T) {
	a := newAPI(t)
	course := seedCourse(t, 3, 40)
	student := seedStudent(t)

	if status, code := a.enroll(t, student, course.ID); status != http.StatusCreated {
		t.Fatalf("mendaftar: dapat %d %s, seharusnya 201", status, code)
	}
	before := enrolledCount(t, course.ID)

	status, code := a.send(t, student, http.MethodDelete, "/api/v1/courses/"+strconv.Itoa(course.ID)+"/enroll", "")
	if status != http.StatusNoContent {
		t.Fatalf("drop: dapat %d %s, seharusnya 204", status, code)
	}

	if isEnrolled(t, course.ID, student.ID) {
		t.Errorf("baris enrollment masih ada setelah drop")
	}
	if got := enrolledCount(t, course.ID); got != before-1 {
		t.Errorf("enrolled_count = %d, seharusnya %d (berkurang satu)", got, before-1)
	}
	dropped := countRows(t,
		`SELECT COUNT(*) FROM outbox WHERE (payload->>'user_id')::INT = $1 AND event_type = $2`,
		student.ID, model.EventEnrollmentDropped)
	if dropped != 1 {
		t.Errorf("event enrollment.dropped = %d, seharusnya tepat 1", dropped)
	}
}
