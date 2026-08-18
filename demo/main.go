package main

import "fmt"

type Mahasiswa struct {
	NIM   string
	Prodi string
	IPK   float64
}

func main() {
	nama := "Budi"
	umur := 20
	tinggi := 175.5
	lulus := true
	hobi := []string{"membaca", "bersepeda", "ngoding"}

	fmt.Println("=== Lima Variabel dengan Tipe Berbeda ===")
	fmt.Printf("string : %q (tipe %T)\n", nama, nama)
	fmt.Printf("int    : %d (tipe %T)\n", umur, umur)
	fmt.Printf("float64: %.2f (tipe %T)\n", tinggi, tinggi)
	fmt.Printf("bool   : %v (tipe %T)\n", lulus, lulus)
	fmt.Printf("slice  : %v (tipe %T)\n", hobi, hobi)
	fmt.Println()

	mahasiswa := make(map[string]Mahasiswa)

	mahasiswa["Budi"] = Mahasiswa{NIM: "22001", Prodi: "Informatika", IPK: 3.75}
	mahasiswa["Siti"] = Mahasiswa{NIM: "22002", Prodi: "Sistem Informasi", IPK: 3.90}

	fmt.Println("=== Tambah Data ===")
	mahasiswa["Andi"] = Mahasiswa{NIM: "22003", Prodi: "Informatika", IPK: 3.50}
	fmt.Printf("Setelah ditambah, jumlah mahasiswa: %d\n", len(mahasiswa))
	fmt.Println()

	fmt.Println("=== Baca dengan Pengecekan Keberadaan ===")
	namaCari := "Siti"
	if m, ok := mahasiswa[namaCari]; ok {
		fmt.Printf("Ditemukan %s: NIM=%s, Prodi=%s, IPK=%.2f\n", namaCari, m.NIM, m.Prodi, m.IPK)
	} else {
		fmt.Printf("%s tidak ditemukan\n", namaCari)
	}

	namaCari = "Rina"
	if m, ok := mahasiswa[namaCari]; ok {
		fmt.Printf("Ditemukan %s: NIM=%s, Prodi=%s, IPK=%.2f\n", namaCari, m.NIM, m.Prodi, m.IPK)
	} else {
		fmt.Printf("%s tidak ditemukan\n", namaCari)
	}
	fmt.Println()

	fmt.Println("=== Hapus Data ===")
	delete(mahasiswa, "Andi")
	fmt.Printf("Setelah dihapus, jumlah mahasiswa: %d\n", len(mahasiswa))
	if _, ok := mahasiswa["Andi"]; !ok {
		fmt.Println("Andi sudah dihapus dari map")
	}
	fmt.Println()

	fmt.Println("=== Telusuri Seluruh Isi Map ===")
	for nama, m := range mahasiswa {
		fmt.Printf("%s -> NIM=%s, Prodi=%s, IPK=%.2f\n", nama, m.NIM, m.Prodi, m.IPK)
	}
}
