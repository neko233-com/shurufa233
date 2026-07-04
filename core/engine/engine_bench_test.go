package engine

import "testing"

func BenchmarkPreviewExact(b *testing.B) {
	e := New(DefaultConfig())
	for i := 0; i < b.N; i++ {
		_ = e.Preview("nihao")
	}
}

func BenchmarkPreviewPrefix(b *testing.B) {
	e := New(DefaultConfig())
	for i := 0; i < b.N; i++ {
		_ = e.Preview("zhong")
	}
}

func BenchmarkPreviewSegmented(b *testing.B) {
	e := New(DefaultConfig())
	for i := 0; i < b.N; i++ {
		_ = e.Preview("womende")
	}
}
