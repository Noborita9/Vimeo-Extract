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
		force      bool
		listTracks bool
	)

	flag.StringVar(&rawURL, "u", "", "URL o ID del video / showcase de Vimeo")
	flag.StringVar(&rawURL, "url", "", "URL o ID del video / showcase de Vimeo")
	flag.StringVar(&password, "p", "", "Contraseña del video o showcase (si está protegido)")
	flag.StringVar(&password, "password", "", "Contraseña del video o showcase (si está protegido)")
	flag.StringVar(&outputDir, "o", "./output", "Directorio de salida para guardar las transcripciones")
	flag.StringVar(&outputDir, "output", "./output", "Directorio de salida para guardar las transcripciones")
	flag.StringVar(&langPref, "l", "es", "Código de idioma preferido para subtítulos (default: es)")
	flag.StringVar(&langPref, "lang", "es", "Código de idioma preferido para subtítulos (default: es)")
	flag.StringVar(&referer, "r", "", "Referer HTTP (ej: https://aulas.ort.edu.uy/)")
	flag.StringVar(&referer, "referer", "", "Referer HTTP (ej: https://aulas.ort.edu.uy/)")
	flag.BoolVar(&force, "f", false, "Sobrescribir si ya existe la transcripción")
	flag.BoolVar(&force, "force", false, "Sobrescribir si ya existe la transcripción")
	flag.BoolVar(&listTracks, "list-tracks", false, "Solo listar los tracks de subtítulos disponibles")

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Uso: vimeo-fetcher [opciones] <URL o ID>\n\n")
		fmt.Fprintf(os.Stderr, "Ejemplos:\n")
		fmt.Fprintf(os.Stderr, "  # Descargar un video particular dentro de un showcase:\n")
		fmt.Fprintf(os.Stderr, "  vimeo-fetcher -u \"https://vimeo.com/showcase/12367680?video=1236100586\" -p \"M1awjQrt\"\n\n")
		fmt.Fprintf(os.Stderr, "  # Descargar todo un showcase (todas las clases):\n")
		fmt.Fprintf(os.Stderr, "  vimeo-fetcher -u \"https://vimeo.com/showcase/12367680\" -p \"M1awjQrt\"\n\n")
		fmt.Fprintf(os.Stderr, "  # Descargar un video individual de Vimeo:\n")
		fmt.Fprintf(os.Stderr, "  vimeo-fetcher -u \"https://vimeo.com/123456789\" -p \"secreto\"\n\n")
		fmt.Fprintf(os.Stderr, "Opciones:\n")
		flag.PrintDefaults()
	}

	flag.Parse()

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

	// Caso 1: Showcase completo
	if target.IsShowcase {
		fmt.Printf("📂 Detectado Showcase ID: %s\n", target.ShowcaseID)
		var hashedPass string
		if password != "" {
			fmt.Println("🔑 Autenticando con contraseña en el showcase...")
			h, err := client.AuthenticateShowcase(target.ShowcaseID, password)
			if err != nil {
				fmt.Fprintf(os.Stderr, "❌ Error al autenticar showcase: %v\n", err)
				os.Exit(1)
			}
			hashedPass = h
			fmt.Println("✅ Showcase autenticado.")
		}

		fmt.Println("🔍 Obteniendo lista de videos del showcase...")
		videos, err := client.FetchShowcaseVideos(target.ShowcaseID, hashedPass)
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
		skippedCount := 0
		showcaseReferer := fmt.Sprintf("https://vimeo.com/showcase/%s", target.ShowcaseID)

		for i, vid := range videos {
			fmt.Printf("[%d/%d] 🎬 %s (ID: %s)\n", i+1, len(videos), vid.Title, vid.ID)

			if !force && storage.TranscriptExists(outputDir, vid.Title) {
				fmt.Printf("   ⏭️ Ya existe en output/, salteando. (Usá -f para forzar descarga)\n\n")
				skippedCount++
				continue
			}

			savedPath, err := processShowcaseVideo(client, target.ShowcaseID, vid.ID, hashedPass, showcaseReferer, outputDir, langPref, force, listTracks)
			if err != nil {
				fmt.Printf("   ⚠️ Error: %v\n\n", err)
			} else if !listTracks {
				fmt.Printf("   💾 Guardado en: %s\n\n", savedPath)
				successCount++
			}
		}
		fmt.Printf("🎉 Proceso finalizado: %d guardados, %d ya existían (total: %d videos).\n", successCount, skippedCount, len(videos))
		return
	}

	// Caso 2: Video específico dentro de un showcase (?video=ID)
	if target.ShowcaseID != "" && target.VideoID != "" {
		fmt.Printf("🎬 Procesando video ID: %s (dentro del showcase %s)\n", target.VideoID, target.ShowcaseID)
		var hashedPass string
		if password != "" {
			fmt.Println("🔑 Autenticando con contraseña en el showcase...")
			h, err := client.AuthenticateShowcase(target.ShowcaseID, password)
			if err != nil {
				fmt.Fprintf(os.Stderr, "❌ Error al autenticar showcase: %v\n", err)
				os.Exit(1)
			}
			hashedPass = h
			fmt.Println("✅ Showcase autenticado.")
		}

		showcaseReferer := fmt.Sprintf("https://vimeo.com/showcase/%s", target.ShowcaseID)
		savedPath, err := processShowcaseVideo(client, target.ShowcaseID, target.VideoID, hashedPass, showcaseReferer, outputDir, langPref, force, listTracks)
		if err != nil {
			fmt.Fprintf(os.Stderr, "❌ Error: %v\n", err)
			os.Exit(1)
		}

		if !listTracks {
			fmt.Printf("✅ ¡Listo! Transcripción guardada en:\n👉 %s\n", savedPath)
		}
		return
	}

	// Caso 3: Video individual estándar
	fmt.Printf("🎬 Procesando video ID: %s\n", target.VideoID)
	config, err := client.GetPlayerConfig(target.VideoID, target.UnlistedHash, password)
	if err != nil {
		fmt.Fprintf(os.Stderr, "❌ Error al obtener configuración del video: %v\n", err)
		os.Exit(1)
	}

	savedPath, err := downloadAndSaveTranscript(client, config, target.VideoID, outputDir, langPref, force, listTracks)
	if err != nil {
		fmt.Fprintf(os.Stderr, "❌ Error: %v\n", err)
		os.Exit(1)
	}

	if !listTracks {
		fmt.Printf("✅ ¡Listo! Transcripción guardada en:\n👉 %s\n", savedPath)
	}
}

func processShowcaseVideo(client *vimeo.Client, showcaseID, videoID, hashedPass, referer, outputDir, langPref string, force, listOnly bool) (string, error) {
	embedURL, _, err := client.GetShowcaseVideoEmbedURL(showcaseID, videoID, hashedPass)
	if err != nil {
		return "", fmt.Errorf("no se pudo obtener la URL del reproductor: %w", err)
	}

	config, err := client.GetPlayerConfigFromEmbedURL(embedURL, referer)
	if err != nil {
		return "", fmt.Errorf("no se pudo obtener la configuración del reproductor: %w", err)
	}

	return downloadAndSaveTranscript(client, config, videoID, outputDir, langPref, force, listOnly)
}

func downloadAndSaveTranscript(client *vimeo.Client, config *vimeo.PlayerConfig, videoID, outputDir, langPref string, force, listOnly bool) (string, error) {
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
	savedFile, err := storage.SaveTranscript(outputDir, title, md, force)
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

	// 2. Coincidencia de prefijo (ej: "es-es" o "es-x-autogen" con "es")
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
