package storage

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCollectTranscripts(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "collect_test_*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Crear estructura de prueba
	// tmpDir/
	//   Materia A/2026-03-10/transcript.md
	//   Materia A/2026-03-12/transcript.md
	//   Materia B/2026-08-18/transcript.md
	//   Video Suelto/transcript.md
	files := map[string]string{
		"Materia A/2026-03-10/transcript.md": "Contenido clase 1",
		"Materia A/2026-03-12/transcript.md": "Contenido clase 2",
		"Materia B/2026-08-18/transcript.md": "Contenido materia B",
		"Video Suelto/transcript.md":          "Contenido suelto",
	}

	for relPath, content := range files {
		fullPath := filepath.Join(tmpDir, relPath)
		if err := os.MkdirAll(filepath.Dir(fullPath), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(fullPath, []byte(content), 0644); err != nil {
			t.Fatal(err)
		}
	}

	destDir := filepath.Join(tmpDir, "transcripts")
	results, err := CollectTranscripts(CollectOptions{
		BaseDir:       tmpDir,
		GlobalDestDir: destDir,
		PerSubject:    true,
	})
	if err != nil {
		t.Fatalf("error en CollectTranscripts: %v", err)
	}

	if len(results) != 4 {
		t.Fatalf("se esperaban 4 archivos recopilados, se obtuvieron %d", len(results))
	}

	// Verificar existencia de archivos en el destino global
	expectedGlobalFiles := []string{
		"Materia A - 2026-03-10.md",
		"Materia A - 2026-03-12.md",
		"Materia B - 2026-08-18.md",
		"Video Suelto.md",
	}

	for _, name := range expectedGlobalFiles {
		p := filepath.Join(destDir, name)
		if !FileExists(p) {
			t.Errorf("no se encontró archivo global esperado: %s", p)
		}
	}

	// Verificar existencia en per-materia
	materiaAPath := filepath.Join(tmpDir, "Materia A", "transcripts", "2026-03-10.md")
	if !FileExists(materiaAPath) {
		t.Errorf("no se encontró archivo per-materia esperado: %s", materiaAPath)
	}

	// Test con filtro de materia
	filteredDest := filepath.Join(tmpDir, "filtered_transcripts")
	filteredResults, err := CollectTranscripts(CollectOptions{
		BaseDir:       tmpDir,
		GlobalDestDir: filteredDest,
		SubjectFilter: "Materia B",
		PerSubject:    false,
	})
	if err != nil {
		t.Fatalf("error en CollectTranscripts filtrado: %v", err)
	}

	if len(filteredResults) != 1 {
		t.Errorf("se esperaba 1 archivo filtrado, se obtuvieron %d", len(filteredResults))
	}
}
