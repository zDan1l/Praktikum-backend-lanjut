package service

import "strings"

// NormalizeTags merapikan tag ulasan: spasi di tepi dibuang, huruf dijadikan
// kecil, string kosong dibuang, dan duplikat (setelah dinormalisasi) dihapus
// dengan mempertahankan urutan kemunculan pertama.
//
// Hasilnya tidak pernah nil, agar response JSON berisi [] dan bukan null.
func NormalizeTags(tags []string) []string {
	result := make([]string, 0, len(tags))
	seen := make(map[string]bool, len(tags))
	for _, tag := range tags {
		tag = strings.ToLower(strings.TrimSpace(tag))
		if tag == "" || seen[tag] {
			continue
		}
		seen[tag] = true
		result = append(result, tag)
	}
	return result
}
