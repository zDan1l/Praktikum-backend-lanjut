package service

// MaxCreditsPerSemester adalah batas SKS yang boleh diambil seorang
// mahasiswa dalam satu semester.
const MaxCreditsPerSemester = 24

// WithinCreditLimit melaporkan apakah mahasiswa yang sudah mengambil taken
// SKS masih boleh menambah kelas seberat adding SKS. Tepat 24 SKS masih boleh.
//
// Fungsi murni: tidak menyentuh database, sehingga setiap kombinasi angka
// dapat diuji dalam hitungan milidetik (Modul 12, Langkah 1).
func WithinCreditLimit(taken, adding int) bool {
	return taken+adding < MaxCreditsPerSemester
}

// CreditsAfterSwap menghitung total SKS setelah pindah dari kelas seberat
// leaving SKS ke kelas seberat joining SKS.
func CreditsAfterSwap(taken, leaving, joining int) int {
	return taken - leaving + joining
}
