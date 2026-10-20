# Vimeo Transcript Fetcher 🚀

CLI ultrarrápida en Go para descargar directamente transcripciones de videos y showcases de Vimeo como archivos Markdown (`.md`) con timestamps, sin necesidad de abrir el navegador ni reproducir los videos.

## Características

- ⚡ **Descarga directa:** Extrae los subtítulos `.vtt` de Vimeo y los convierte a párrafos legibles con marcas de tiempo en menos de 1 segundo.
- 🔒 **Soporte para videos con contraseña:** Pasa la contraseña mediante `-p "tu_password"` y se autentica automáticamente.
- 📂 **Soporte para Showcases:** Pasa la URL de un showcase y descargará automáticamente todas las clases/videos.
- 🎓 **Estructura automática para clases de ORT:** Detecta títulos de materias y fechas (ej. `FI-6411-Finanzas y val. de proy... | 15-10-2026`) y guarda en `output/<Materia>/<aaaa-mm-dd>.md`.
- 📦 **Binario único e independiente:** Compilado en Go, sin dependencias externas de Node, Python ni navegadores.

## Instalación y compilación

```bash
git clone https://github.com/Noborita9/Vimeo-Transcript-Fetcher.git
cd Vimeo-Transcript-Fetcher
go build -o vimeo-fetcher ./cmd/vimeo-fetcher
```

*(Opcional: podés mover `vimeo-fetcher` a tu `/usr/local/bin` o agregar la carpeta al `PATH`)*.

## Uso

### Video individual (público)
```bash
./vimeo-fetcher "https://vimeo.com/123456789"
```

### Video protegido con contraseña
```bash
./vimeo-fetcher -u "https://vimeo.com/123456789" -p "M1awjQrt"
```

### Showcases completos (lotes de clases)
```bash
./vimeo-fetcher "https://vimeo.com/showcase/9876543" -p "M1awjQrt"
```

### Opciones disponibles

| Flag | Descripción | Por defecto |
|---|---|---|
| `-u, --url` | URL o ID del video / showcase | Argumento posicional |
| `-p, --password` | Contraseña del video (si está protegido) | `""` |
| `-o, --output` | Carpeta de destino | `./output` |
| `-l, --lang` | Idioma de subtítulos preferido | `es` |
| `-r, --referer` | Referer HTTP (ej. `https://aulas.ort.edu.uy/`) | `""` |
| `--list-tracks` | Solo lista los idiomas disponibles sin descargar | `false` |
