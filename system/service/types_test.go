package service

import "testing"

func TestByteSize(t *testing.T) {
	if v := ByteSize("20GB").ToGB(); v != 20 {
		t.Fatalf("20GB, 20 != %d", v)
	}
	if v := ByteSize("1024MB").ToGB(); v != 1 {
		t.Fatalf("1024MB, 1 != %d", v)
	}
	if v := ByteSize("1548MB").ToGB(); v != 2 {
		t.Fatalf("1548MB, 2 != %d", v)
	}
}
