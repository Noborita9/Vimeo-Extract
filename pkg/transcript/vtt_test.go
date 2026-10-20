package transcript

import (
	"testing"
)

func TestParseVTT(t *testing.T) {
	vtt := `WEBVTT

00:00:01.000 --> 00:00:04.500
<v Teacher>Buenas tardes a todos.

00:00:05.000 --> 00:00:08.000
Hoy vamos a ver accesibilidad.
`
	cues := ParseVTT(vtt)
	if len(cues) != 2 {
		t.Fatalf("se esperaban 2 cues, se obtuvieron %d", len(cues))
	}

	if cues[0].Content != "Buenas tardes a todos." {
		t.Errorf("contenido inesperado en cue 0: %s", cues[0].Content)
	}
	if cues[0].Start != 1.0 || cues[0].End != 4.5 {
		t.Errorf("tiempos incorrectos en cue 0: start=%f, end=%f", cues[0].Start, cues[0].End)
	}
}

func TestFormatTimestamp(t *testing.T) {
	tests := []struct {
		seconds  float64
		expected string
	}{
		{5, "00:05"},
		{65, "01:05"},
		{3665, "1:01:05"},
	}

	for _, tt := range tests {
		got := FormatTimestamp(tt.seconds)
		if got != tt.expected {
			t.Errorf("para %f seg: esperado %s, obtenido %s", tt.seconds, tt.expected, got)
		}
	}
}
