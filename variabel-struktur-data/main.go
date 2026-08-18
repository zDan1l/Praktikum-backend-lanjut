package main

import "fmt"

type Mahasiswa struct {
	NIM   string
	Kelas string
	IPK   float64
	Lulus bool
}

func main() {
	nama := "Andi Prasetyo"
	umur := 20
	tinggiBadan := 172.5
	aktif := true
	matakuliah := []string{"Backend", "Database", "Jaringan"}

	fmt.Println("=== 5 Variabel dengan Tipe Berbeda ===")
	fmt.Printf("string  : nama        = %s\n", nama)
	fmt.Printf("int     : umur        = %d\n", umur)
	fmt.Printf("float64 : tinggiBadan = %.1f\n", tinggiBadan)
	fmt.Printf("bool    : aktif       = %t\n", aktif)
	fmt.Printf("slice   : matakuliah  = %v\n\n", matakuliah)

	dataMahasiswa := make(map[string]Mahasiswa)

	dataMahasiswa["Andi"] = Mahasiswa{NIM: "A001", Kelas: "Backend", IPK: 3.75, Lulus: false}
	dataMahasiswa["Budi"] = Mahasiswa{NIM: "A002", Kelas: "Backend", IPK: 3.90, Lulus: false}
	dataMahasiswa["Citra"] = Mahasiswa{NIM: "A003", Kelas: "Database", IPK: 3.50, Lulus: true}

	fmt.Println("=== Map Data Mahasiswa ===")

	fmt.Println("Data awal (3 mahasiswa):")
	for nama, mhs := range dataMahasiswa {
		fmt.Printf("  %-6s NIM=%s Kelas=%-8s IPK=%.2f Lulus=%t\n",
			nama, mhs.NIM, mhs.Kelas, mhs.IPK, mhs.Lulus)
	}

	dataMahasiswa["Dewi"] = Mahasiswa{NIM: "A004", Kelas: "Jaringan", IPK: 3.85, Lulus: false}
	fmt.Printf("\nSetelah menambah 'Dewi', jumlah data = %d\n", len(dataMahasiswa))

	fmt.Println("\nMembaca data dengan pengecekan keberadaan:")
	if mhs, ok := dataMahasiswa["Budi"]; ok {
		fmt.Printf("  'Budi' ditemukan: NIM=%s, IPK=%.2f\n", mhs.NIM, mhs.IPK)
	} else {
		fmt.Println("  'Budi' tidak ditemukan")
	}

	if mhs, ok := dataMahasiswa["Eko"]; ok {
		fmt.Printf("  'Eko' ditemukan: NIM=%s, IPK=%.2f\n", mhs.NIM, mhs.IPK)
	} else {
		fmt.Println("  'Eko' tidak ditemukan")
	}

	delete(dataMahasiswa, "Andi")
	fmt.Printf("\nSetelah menghapus 'Andi', jumlah data = %d\n", len(dataMahasiswa))

	if _, ok := dataMahasiswa["Andi"]; ok {
		fmt.Println("  'Andi' masih ada")
	} else {
		fmt.Println("  'Andi' sudah tidak ada")
	}

	fmt.Println("\n=== Menelusuri Seluruh Isi Map ===")
	fmt.Println("Isi akhir data mahasiswa:")
	for nama, mhs := range dataMahasiswa {
		fmt.Printf("  %-6s NIM=%s Kelas=%-8s IPK=%.2f Lulus=%t\n",
			nama, mhs.NIM, mhs.Kelas, mhs.IPK, mhs.Lulus)
	}
}
