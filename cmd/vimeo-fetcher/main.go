package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/Noborita9/Vimeo-Transcript-Fetcher/pkg/storage"
	"github.com/Noborita9/Vimeo-Transcript-Fetcher/pkg/transcript"
	"github.com/Noborita9/Vimeo-Transcript-Fetcher/pkg/vimeo"
)

func main() {
	var (
		rawURL     string
		password   string
		outputDir  string
		langPref   string
		referer    string
		listTracks bool
	)

	flag.StringVar(&rawURL, "u", "", "URL o ID del video / showcase de Vimeo")
	flag.StringVar(&rawURL, "url", "", "URL o ID del video / showcase de Vimeo")
	flag.StringVar(&password, "p", "", "Contraseña del video (si está protegido)")
	flag.StringVar(&password, "password", "", "Contraseña del video (si está protegido)")
	flag.StringVar(&outputDir, "o", "./output", "Directorio de salida para guardar las transcripciones")
	flag.StringVar(&outputDir, "output", "./output", "Directorio de salida para guardar las transcripciones")
	flag.StringVar(&langPref, "l", "es", "Código de idioma preferido para subtítulos (default: es)")
	flag.StringVar(&langPref, "lang", "es", "Código de idioma preferido para subtítulos (default: es)")
	flag.StringVar(&referer, "r", "", "Referer HTTP (ej: https://aulas.ort.edu.uy/)")
	flag.StringVar(&referer, "referer", "", "Referer HTTP (ej: https://aulas.ort.edu.uy/)")
	flag.BoolVar(&listTracks, "list-tracks", false, "Solo listar los tracks de subtítulos disponibles")

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Uso: vimeo-fetcher [opciones] <URL o ID>\n\n")
		fmt.Fprintf(os.Stderr, "Ejemplos:\n")
		fmt.Fprintf(os.Stderr, "  vimeo-fetcher -u \"https://vimeo.com/123456789\" -p \"secreto\"\n")
		fmt.Fprintf(os.Stderr, "  vimeo-fetcher -u \"https://vimeo.com/showcase/987654\"\n")
		fmt.Fprintf(os.Stderr, "  vimeo-fetcher \"123456789\"\n\n")
		fmt.Fprintf(os.Stderr, "Opciones:\n")
		flag.PrintDefaults()
	}

	flag.Parse()

	// Si no se pasó por flag -u, tomar el primer argumento posicional
	if rawURL == "" && flag.NArg() > 0 {
		rawURL = flag.Arg(0)
	}

	if rawURL == "" {
		flag.Usage()
		os.Exit(1)
	}

	target, err := vimeo.ParseURL(rawURL)
	if err != nil {
		fmt.Fprintf(os.Stderr, "❌ Error: %v\n", err)
		os.Exit(1)
	}

	client, err := vimeo.NewClient(referer)
	if err != nil {
		fmt.Fprintf(os.Stderr, "❌ Error al iniciar cliente: %v\n", err)
		os.Exit(1)
	}

	if target.IsShowcase {
		fmt.Printf("📂 Detectado Showcase ID: %s\n", target.ShowcaseID)
		fmt.Println("🔍 Obteniendo lista de videos del showcase...")
		videos, err := client.FetchShowcaseVideos(target.ShowcaseID)
		if err != nil {
			fmt.Fprintf(os.Stderr, "❌ Error al obtener videos del showcase: %v\n", err)
			os.Exit(1)
		}

		if len(videos) == 0 {
			fmt.Println("⚠️ No se encontraron videos en el showcase.")
			return
		}

		fmt.Printf("✅ Se encontraron %d videos. Procesando...\n\n", len(videos))
		successCount := 0
		for i, vid := range videos {
			fmt.Printf("[%d/%d] 🎬 %s (ID: %s)\n", i+1, len(videos), vid.Title, vid.ID)
			savedPath, err := processSingleVideo(client, vid.ID, "", password, outputDir, langPref, listTracks)
			if err != nil {
				fmt.Printf("   ⚠️ Error: %v\n\n", err)
			} else if !listTracks {
				fmt.Printf("   💾 Guardado en: %s\n\n", savedPath)
				successCount++
			}
		}
		fmt.Printf("🎉 Proceso finalizado: %d/%d transcripciones guardadas.\n", successCount, len(videos))
		return
	}

	// Video individual
	fmt.Printf("🎬 Procesando video ID: %s\n", target.VideoID)
	savedPath, err := processSingleVideo(client, target.VideoID, target.UnlistedHash, password, outputDir, langPref, listTracks)
	if err != nil {
		fmt.Fprintf(os.Stderr, "❌ Error: %v\n", err)
		os.Exit(1)
	}

	if !listTracks {
		fmt.Printf("✅ ¡Listo! Transcripción guardada en:\n👉 %s\n", savedPath)
	}
}

func processSingleVideo(client *vimeo.Client, videoID, unlistedHash, password, outputDir, langPref string, listOnly bool) (string, error) {
	config, err := client.GetPlayerConfig(videoID, unlistedHash, password)
	if err != nil {
		return "", fmt.Errorf("no se pudo obtener la configuración del video: %w", err)
	}

	tracks := config.Request.TextTracks
	if len(tracks) == 0 {
		return "", fmt.Errorf("el video no tiene pistas de subtítulos o transcripción disponibles")
	}

	if listOnly {
		fmt.Println("   Pistas de subtítulos disponibles:")
		for _, t := range tracks {
			fmt.Printf("   - [%s] %s (default: %v, provenance: %s)\n", t.Lang, t.Label, t.Default, t.Provenance)
		}
		return "", nil
	}

	// Seleccionar el track adecuado
	selectedTrack := selectTrack(tracks, langPref)
	if selectedTrack == nil {
		return "", fmt.Errorf("no se encontró una pista de subtítulos adecuada")
	}

	fmt.Printf("   📝 Descargando transcripción (%s - %s)...\n", selectedTrack.Lang, selectedTrack.Label)
	vttRaw, err := client.DownloadTrack(selectedTrack.URL)
	if err != nil {
		return "", fmt.Errorf("error al descargar subtítulos: %w", err)
	}

	cues := transcript.ParseVTT(vttRaw)
	if len(cues) == 0 {
		return "", fmt.Errorf("el archivo de subtítulos no contiene fragmentos de texto válidos")
	}

	paragraphs := transcript.GroupIntoParagraphs(cues)
	title := config.Video.Title
	if title == "" {
		title = fmt.Sprintf("video-%s", videoID)
	}

	md := transcript.BuildMarkdown(title, videoID, paragraphs)
	savedFile, err := storage.SaveTranscript(outputDir, title, md)
	if err != nil {
		return "", fmt.Errorf("error al guardar archivo Markdown: %w", err)
	}

	return savedFile, nil
}

func selectTrack(tracks []vimeo.TextTrack, langPref string) *vimeo.TextTrack {
	langPref = strings.ToLower(strings.TrimSpace(langPref))

	// 1. Coincidencia exacta de idioma
	for _, t := range tracks {
		if strings.ToLower(t.Lang) == langPref {
			return &t
		}
	}

	// 2. Coincidencia de prefijo (ej: "es-es" con "es")
	for _, t := range tracks {
		if strings.HasPrefix(strings.ToLower(t.Lang), langPref) {
			return &t
		}
	}

	// 3. Track marcado como default
	for _, t := range tracks {
		if t.Default {
			return &t
		}
	}

	// 4. Primer track disponible
	if len(tracks) > 0 {
		return &tracks[0]
	}

	return nil
}
