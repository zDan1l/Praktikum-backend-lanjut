package service

import (
	"errors"
	"strings"
	"testing"

	"pemrograman-code/app/model"
)

func TestBuildMessage(t *testing.T) {
	user := model.User{ID: 7, Username: "mhsuji", Email: "mhsuji@student.unair.ac.id"}
	event := model.EnrollmentEvent{
		UserID: 7, CourseID: 4, CourseCode: "IF302", CourseName: "Keamanan Informasi",
		FromCourseCode: "IF102", FromCourseName: "Basis Data Lanjut",
	}

	tests := []struct {
		eventType   string
		wantSubject string
	}{
		{model.EventEnrollmentCreated, "KRS: berhasil terdaftar di IF302 Keamanan Informasi"},
		{model.EventEnrollmentDropped, "KRS: keluar dari IF302 Keamanan Informasi"},
		{model.EventEnrollmentSwapped, "KRS: pindah dari IF102 Basis Data Lanjut ke IF302 Keamanan Informasi"},
	}

	for _, tt := range tests {
		t.Run(tt.eventType, func(t *testing.T) {
			msg, err := BuildMessage(tt.eventType, event, user)
			if err != nil {
				t.Fatalf("error tidak diharapkan: %v", err)
			}
			if msg.Subject != tt.wantSubject {
				t.Errorf("subject = %q, seharusnya %q", msg.Subject, tt.wantSubject)
			}
			if msg.To != user.Email {
				t.Errorf("to = %q, seharusnya %q", msg.To, user.Email)
			}
			if !strings.Contains(msg.Body, user.Username) {
				t.Errorf("body tidak menyebut username: %q", msg.Body)
			}
		})
	}

	// PENYESUAIAN REPO (bukan kesalahan modul): BuildMessage di repo ini
	// memiliki cabang tambahan course.quota_changed dari Tugas Mandiri D.2
	// Modul 11. Tanpa skenario ini coverage BuildMessage tidak 100% (T3).
	t.Run(model.EventCourseQuotaChanged, func(t *testing.T) {
		quota := event
		quota.NewQuota = 45
		msg, err := BuildMessage(model.EventCourseQuotaChanged, quota, user)
		if err != nil {
			t.Fatalf("error tidak diharapkan: %v", err)
		}
		if want := "KRS: kuota IF302 Keamanan Informasi berubah menjadi 45"; msg.Subject != want {
			t.Errorf("subject = %q, seharusnya %q", msg.Subject, want)
		}
		if msg.To != user.Email || !strings.Contains(msg.Body, user.Username) {
			t.Errorf("pesan kuota salah alamat atau tidak menyebut username: %+v", msg)
		}
	})

	t.Run("event tidak dikenal", func(t *testing.T) {
		_, err := BuildMessage("enrollment.unknown", event, user)
		if !errors.Is(err, errUnknownEvent) {
			t.Errorf("error = %v, seharusnya membungkus errUnknownEvent", err)
		}
	})
}
