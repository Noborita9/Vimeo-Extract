# Vimeo Transcript Fetcher 🚀

CLI ultrarrápida en Go para descargar directamente transcripciones de videos y showcases de Vimeo como archivos Markdown (`.md`) con timestamps, sin necesidad de abrir el navegador ni reproducir los videos.

## Características

- ⚡ **Descarga directa:** Extrae los subtítulos `.vtt` de Vimeo y los convierte a párrafos legibles con marcas de tiempo en menos de 1 segundo.
- 🔒 **Soporte para videos y showcases protegidos:** Pasa la contraseña mediante `-p "tu_password"` y se autentica automáticamente contra Vimeo.
- 📂 **Soporte para Showcases (Lotes):** Descarga automáticamente todas las clases de una carpeta/showcase o un video particular dentro de él.
- ⏭️ **Salteo inteligente:** Si una clase ya fue descargada en `output/`, se saltea automáticamente. Usá `-f` para forzar la re-descarga.
- 🎓 **Estructura automática para clases de ORT:** Detecta materias y fechas (ej. `FI-7669-Fundamentos de Ingeniería... | 14-10-2026`) y guarda como `output/<Materia>/<aaaa-mm-dd>.md`.
- 📦 **Binario único e independiente:** Compilado en Go para macOS / Linux, sin dependencias externas de Node, Python ni navegadores.

## Instalación y compilación

```bash
cd /Users/noborita/workspace/github.com/Noborita9/Vimeo-Transcript-Fetcher
go build -o vimeo-fetcher ./cmd/vimeo-fetcher
```

## Ejemplos de uso

### 1. Video particular dentro de un showcase
```bash
./vimeo-fetcher -u "https://vimeo.com/showcase/12367680?video=1236100586" -p "M1awjQrt"
```

### 2. Todo un showcase (todas las clases del semestre)
```bash
./vimeo-fetcher -u "https://vimeo.com/showcase/12367680" -p "M1awjQrt"
```

### 3. Video individual de Vimeo
```bash
./vimeo-fetcher -u "https://vimeo.com/123456789" -p "secreto"
```

### 4. Forzar re-descarga si ya existe
```bash
./vimeo-fetcher -u "https://vimeo.com/showcase/12367680" -p "M1awjQrt" -f
```

### Opciones disponibles

| Flag | Descripción | Por defecto |
|---|---|---|
| `-u, --url` | URL o ID del video / showcase | Argumento posicional |
| `-p, --password` | Contraseña del video o showcase | `""` |
| `-f, --force` | Sobrescribir si ya existe en `output/` | `false` |
| `-o, --output` | Carpeta de destino | `./output` |
| `-l, --lang` | Idioma de subtítulos preferido | `es` |
| `-r, --referer` | Referer HTTP si está restringido | `""` |
| `--list-tracks` | Solo lista los idiomas disponibles sin descargar | `false` |
