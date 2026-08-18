package main

import "fmt"

type Student struct {
	ID       int
	Name     string
	Grade    float64
	IsActive bool
}

// GetInfo: value receiver karena hanya membaca data, tidak mengubah struct.
func (s Student) GetInfo() string {
	status := "non-aktif"
	if s.IsActive {
		status = "aktif"
	}
	return fmt.Sprintf("ID: %d | Nama: %s | Nilai: %.2f | Status: %s", s.ID, s.Name, s.Grade, status)
}

// UpdateGrade: pointer receiver karena mengubah nilai field Grade.
func (s *Student) UpdateGrade(grade float64) {
	s.Grade = grade
}

// Activate: pointer receiver karena mengubah field IsActive menjadi true.
func (s *Student) Activate() {
	s.IsActive = true
}

// Deactivate: pointer receiver karena mengubah field IsActive menjadi false.
func (s *Student) Deactivate() {
	s.IsActive = false
}

func main() {
	s := Student{ID: 1, Name: "Budi", Grade: 85.5, IsActive: true}

	fmt.Println(s.GetInfo())

	s.UpdateGrade(92.75)
	s.Deactivate()

	fmt.Println(s.GetInfo())

	s.Activate()

	fmt.Println(s.GetInfo())
}
