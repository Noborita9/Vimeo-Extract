package storage

import (
	"path/filepath"
	"testing"
)

func TestDetermineItemDir(t *testing.T) {
	base := "/tmp/output"

	tests := []struct {
		title    string
		expected string
	}{
		{
			title:    "FI-7669-Fundamentos de Ingeniería  SF-78522-ID_ N4A0826 _FIS | 14-10-2026",
			expected: filepath.Join(base, "Fundamentos de Ingeniería SF", "2026-10-14"),
		},
		{
			title:    "FI-7669-Fundamentos de Ingeniería SF-78522-13-10-2026",
			expected: filepath.Join(base, "Fundamentos de Ingeniería SF", "2026-10-13"),
		},
		{
			title:    "Video de prueba sin fecha",
			expected: filepath.Join(base, "Video de prueba sin fecha"),
		},
	}

	for _, tt := range tests {
		got := DetermineItemDir(base, tt.title)
		if got != tt.expected {
			t.Errorf("para '%s':\nesperado: %s\nobtenido: %s", tt.title, tt.expected, got)
		}
	}
}

func TestSanitizeFilename(t *testing.T) {
	input := "Hola: / * <Materia> ? \"Test\" |"
	expected := "Hola- - - -Materia- - -Test- -"
	got := SanitizeFilename(input)
	if got != expected {
		t.Errorf("esperado '%s', obtenido '%s'", expected, got)
	}
}
