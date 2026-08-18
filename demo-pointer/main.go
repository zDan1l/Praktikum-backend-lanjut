package main

import "fmt"

func swap(a, b *int) {
	tmp := *a
	*a = *b
	*b = tmp
}

func updateSlice(s *[]string, newItem string) {
	*s = append(*s, newItem)
}

func main() {
	x, y := 10, 20
	fmt.Println(" swap(a, b *int) ")
	fmt.Printf("Sebelum swap: x=%d, y=%d\n", x, y)
	swap(&x, &y)
	fmt.Printf("Setelah swap: x=%d, y=%d\n", x, y)
	fmt.Println()

	items := []string{"apel", "jeruk"}
	fmt.Println(" updateSlice(s *[]string, newItem string) ")
	fmt.Printf("Sebelum update: %v\n", items)
	updateSlice(&items, "mangga")
	fmt.Printf("Setelah update: %v (len=%d, cap=%d)\n", items, len(items), cap(items))
	fmt.Println()

	fmt.Println(" Perbandingan Pass by Value vs Pass by Pointer ")
	v1, v2 := 5, 7
	fmt.Printf("Awal: v1=%d, v2=%d\n", v1, v2)
	swapByValue(v1, v2)
	fmt.Printf("Setelah pass by value : v1=%d, v2=%d (tidak berubah)\n", v1, v2)
	swap(&v1, &v2)
	fmt.Printf("Setelah pass by pointer: v1=%d, v2=%d (berubah)\n", v1, v2)
	fmt.Println()

	s1 := []string{"a"}
	s2 := []string{"a"}
	addByValue(s1, "b")
	fmt.Printf("Setelah pass by value : %v (tidak berubah)\n", s1)
	addByPointer(&s2, "b")
	fmt.Printf("Setelah pass by pointer: %v (berubah)\n", s2)
}

func swapByValue(a, b int) {
	tmp := a
	a = b
	b = tmp
}

func addByValue(s []string, item string) {
	s = append(s, item)
}

func addByPointer(s *[]string, item string) {
	*s = append(*s, item)
}
