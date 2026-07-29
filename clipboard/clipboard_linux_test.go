package clipboard

import (
	"errors"
	"testing"
)

func TestWriteLinuxStringUsesX11WhenNotWayland(t *testing.T) {
	calledX11 := false

	err := writeLinuxString("hello", "", func(string) (string, error) {
		t.Fatal("lookPath should not be called outside Wayland")
		return "", nil
	}, func(string) error {
		t.Fatal("writeWayland should not be called outside Wayland")
		return nil
	}, func(str string) error {
		calledX11 = true
		if str != "hello" {
			t.Fatalf("writeX11 got %q, want hello", str)
		}
		return nil
	})

	if err != nil {
		t.Fatal(err)
	}
	if !calledX11 {
		t.Fatal("writeX11 was not called")
	}
}

func TestWriteLinuxStringUsesWaylandWhenAvailable(t *testing.T) {
	calledWayland := false

	err := writeLinuxString("hello", "wayland-1", func(name string) (string, error) {
		if name != "wl-copy" {
			t.Fatalf("lookPath got %q, want wl-copy", name)
		}
		return "/bin/wl-copy", nil
	}, func(str string) error {
		calledWayland = true
		if str != "hello" {
			t.Fatalf("writeWayland got %q, want hello", str)
		}
		return nil
	}, func(string) error {
		t.Fatal("writeX11 should not be called when Wayland succeeds")
		return nil
	})

	if err != nil {
		t.Fatal(err)
	}
	if !calledWayland {
		t.Fatal("writeWayland was not called")
	}
}

func TestWriteLinuxStringFallsBackWhenWlCopyMissing(t *testing.T) {
	calledX11 := false

	err := writeLinuxString("hello", "wayland-1", func(string) (string, error) {
		return "", errors.New("not found")
	}, func(string) error {
		t.Fatal("writeWayland should not be called when wl-copy is missing")
		return nil
	}, func(string) error {
		calledX11 = true
		return nil
	})

	if err != nil {
		t.Fatal(err)
	}
	if !calledX11 {
		t.Fatal("writeX11 was not called")
	}
}

func TestWriteLinuxStringFallsBackWhenWaylandFails(t *testing.T) {
	calledX11 := false

	err := writeLinuxString("hello", "wayland-1", func(string) (string, error) {
		return "/bin/wl-copy", nil
	}, func(string) error {
		return errors.New("wayland failed")
	}, func(string) error {
		calledX11 = true
		return nil
	})

	if err != nil {
		t.Fatal(err)
	}
	if !calledX11 {
		t.Fatal("writeX11 was not called")
	}
}

func TestWriteLinuxStringReturnsX11Error(t *testing.T) {
	want := errors.New("x11 failed")

	err := writeLinuxString("hello", "wayland-1", func(string) (string, error) {
		return "/bin/wl-copy", nil
	}, func(string) error {
		return errors.New("wayland failed")
	}, func(string) error {
		return want
	})

	if !errors.Is(err, want) {
		t.Fatalf("got %v, want %v", err, want)
	}
}
