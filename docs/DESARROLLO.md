# Desarrollo de commitling

Esta es la referencia técnica del proyecto. Para instalar la Action sin entrar en el código, consulta la [guía de uso](USO.md).

## CLI

La línea de órdenes requiere Go 1.27 y no tiene dependencias externas:

```sh
# Tu criatura (GITHUB_TOKEN es opcional, pero sube el límite de peticiones)
go run ./cmd/commitling render --user BertMarti --out out/commitling.svg

# Sin red, con datos de ejemplo de un usuario ficticio
go run ./cmd/commitling render --fixture testdata/events.json --out out/commitling.svg

# Tema oscuro y fecha de referencia fija, a la salida estándar
go run ./cmd/commitling render --fixture testdata/events.json --theme dark --now 2026-10-01T12:00:00Z --out -

# Otra especie (moss por defecto, mushroom o cactus)
go run ./cmd/commitling render --fixture testdata/events.json --species mushroom --out out/hongo.svg

# Si la API tiene una caída transitoria y out/commitling.svg ya existe, lo conserva (aviso, código de salida 0)
go run ./cmd/commitling render --user BertMarti --keep-on-error --out out/commitling.svg

# Web estática con todas las fases, ánimos y accesorios
go run ./cmd/commitling gallery --out site/

# Imagen PNG 1200x630 para las vistas previas en redes (Open Graph)
go run ./cmd/commitling og --out out/og.png

go run ./cmd/commitling version
```

Con `--fixture` y sin `--now`, se usa la fecha del último evento del fixture. `--species` acepta `moss`, `mushroom` o `cactus` y `--size` acepta `full` o `compact`.

## Stack

- Go 1.27 y solo biblioteca estándar.
- SVG escrito a mano, con animación CSS y píxeles nítidos.
- WebAssembly (`GOOS=js GOARCH=wasm`, `syscall/js`) para el generador en vivo.
- GitHub Action compuesta (`action.yml`), CI en Ubuntu y Windows y despliegue a GitHub Pages.
- La galería es HTML, CSS y JavaScript estáticos generados por el propio binario.

## Estructura

```text
cmd/commitling/      CLI (render, gallery, og, version)
cmd/wasm/            capa fina de syscall/js para el navegador
internal/generate/   eventos a SVG, workflow, errores y demo sintética
internal/events/     parser de eventos y actividades, sin red
internal/github/     cliente de la API de eventos públicos, con reintentos
internal/stats/      XP, racha, días activos y repositorios
internal/creature/   especies, fases, ánimos, accesorios y mapas de píxeles
internal/render/     generación del SVG
internal/gallery/    web estática de la galería y generador
internal/og/         imagen PNG 1200x630 para Open Graph
testdata/            eventos de ejemplo de «octoexample»
action.yml           la GitHub Action
```

## Desarrollo

Formato, análisis y tests:

```sh
gofmt -l .      # debe salir vacío
go vet ./...
go test ./...
```

Seguridad (lo mismo que el job «seguridad» del CI; versiones fijadas en `.github/workflows/ci.yml`):

```sh
go run golang.org/x/vuln/cmd/govulncheck@v1.9.0 ./...
go run honnef.co/go/tools/cmd/staticcheck@2026.2.2 ./...
GOOS=js GOARCH=wasm go run honnef.co/go/tools/cmd/staticcheck@2026.2.2 ./cmd/wasm
gitleaks git . --redact && gitleaks dir . --redact
```

La web declara una Content-Security-Policy en `internal/gallery/index.html.tmpl` (solo scripts propios y `'wasm-unsafe-eval'`, conexiones a sí misma y a `api.github.com`). Por eso **no puede haber scripts en línea ni atributos `on…=`**: lo pequeño va en `internal/gallery/page.js` y el generador en `generator.js`. `TestPageHasStrictCSP` lo vigila.

Para generar la web y el WebAssembly localmente (los artefactos no se suben al repositorio):

```sh
go run ./cmd/commitling gallery --out site
GOOS=js GOARCH=wasm go build -trimpath -ldflags="-s -w" -o site/commitling.wasm ./cmd/wasm
cp "$(go env GOROOT)/lib/wasm/wasm_exec.js" site/
python -m http.server 8080 --directory site   # http://localhost:8080/
```

El generador carga el WebAssembly solo cuando se interactúa con el formulario o con la demo. El archivo pesa aproximadamente 2,6 MB (0,73 MB comprimido). El navegador pide hasta tres páginas de eventos públicos a `https://api.github.com/users/<usuario>/events/public`, sin token, y entrega el JSON al mismo código Go compilado a WebAssembly. Los eventos se guardan diez minutos en `sessionStorage` de la pestaña para evitar peticiones al cambiar tamaño, especie o tema. GitHub limita a 60 peticiones por hora y conexión sin autenticación; el generador muestra la hora local del reinicio y ofrece la demo sintética cuando llega un 403 o 429. La Action usa el token de su workflow y no tiene ese límite.

## Reintentos y conservación

El cliente repite hasta tres veces, con esperas de 1, 2 y 4 segundos. Respeta `Retry-After` y `X-RateLimit-Reset` cuando indican una espera de hasta 30 segundos. Se reintentan:

- Errores de red, como DNS, conexión rechazada, conexión cortada o timeout.
- Respuestas 500, 502, 503 y 504.
- Límite de peticiones: 429 o 403 con cabeceras que indiquen límite.

Solo se repite la página que falla. Un 404, 422, otro 4xx o un 403 sin cabeceras de límite no se reintenta. Si se agotan los reintentos, el error explica la causa.

La entrada de la Action `keep-on-error` está activa por defecto. Si la caída es transitoria y el archivo de salida ya existe, es un SVG completo y el workflow hizo `checkout` antes, se conserva intacto, se muestra un aviso y el paso termina correctamente. La CLI ofrece el mismo comportamiento con `--keep-on-error`. Esto no se aplica a un 401, 404, 422, 403 de permisos, fixture inexistente, especie inválida, archivo vacío o cortado, ni a la salida estándar. Con `keep-on-error: false`, cualquier caída transitoria termina en error. Así los fallos permanentes no quedan ocultos y una caída pasajera no deja el perfil sin imagen.

## Versiones y GitHub Marketplace

- `BertMarti/commitling@v1` sigue la versión mayor 1: la etiqueta móvil `v1` apunta a la última versión compatible y la mueve Alberto al publicar.
- Una etiqueta exacta (`@v0.7.0`) o el SHA de un commit es lo más reproducible.
- `@main` funciona, pero es la rama de desarrollo; los ejemplos nunca la usan (`TestExamplesUseV1`).
- La Action declara nombre, descripción y `branding` (icono `feather`, color `green`) en `action.yml`, que es lo que muestra el Marketplace. **La interfaz de `action.yml` (entradas, salida `path`, archivos y formato del SVG) no se rompe**: hay perfiles que la usan a diario.

## Binario precompilado

Sin binario, la Action compila la CLI con `go run` en cada ejecución. Desde la v1.0.0 usa en su lugar el binario precompilado de la release, **solo** si el árbol de la Action (la etiqueta a la que apunta `@v1`) contiene `release/checksums.txt` y el binario descargado tiene exactamente ese SHA-256. Si falta el archivo, la release aún no tiene binarios, la plataforma no está en la lista, falla la red o el SHA-256 no coincide, la Action compila con Go como siempre (con un aviso en el log); nunca falla por esto. El commit de la v1.0.0 lleva su `release/checksums.txt`, así que entre la fusión y la publicación de la release (y si el workflow «Release» fallara) los perfiles siguen compilando con Go.

- `scripts/build-release.sh <carpeta>` compila los 6 binarios (Linux, macOS y Windows; amd64 y arm64) de forma **reproducible** (`CGO_ENABLED=0`, `-trimpath`, `-buildvcs=false`, `-buildid=`, la versión exacta de Go de `go.mod`): el mismo código da los mismos bytes en cualquier máquina, así que el SHA-256 se puede subir **antes** de que exista la release.
- `--write` copia los SHA-256 a `release/checksums.txt`; `--check` falla si no coinciden.
- `.github/workflows/release.yml` se dispara al **publicar** una release: comprueba que la etiqueta es `v` + la versión de la CLI, recompila con `--check` y sube los binarios y `checksums.txt` a la release. Sin `release/checksums.txt` no sube nada.
- El CI recompila y compara en cada PR si el archivo existe (un cambio de código sin regenerarlo falla). El job `binario` de `action-test.yml` prueba primero la Action con el `release/checksums.txt` del repositorio y la release real (antes de publicarla compila con Go; después usa el binario; el SVG es el mismo) y luego las tres rutas (binario bueno, manipulado y sin release) contra un servidor local. El job `action` fuerza siempre la ruta de `go run` (`COMMITLING_RELEASE_URL` a un puerto cerrado) para seguir comprobando el Go parcheado de cada llamada.

Para cada versión (ver `.claude/skills/publicar-version`): en la rama de la versión, **después del último cambio de código** (la web va embebida en el binario: también cuenta la plantilla), `GOFLAGS=-p=2 nice -n 19 scripts/build-release.sh out/rel --write` y commit de `release/checksums.txt`; se fusiona y se publica la release `vX.Y.Z` desde ese commit. Tras publicarla, el primer PR que cambie código **borra** el archivo (`git rm release/checksums.txt`): describe los binarios de la release ya publicada, no los de `main`, y regenerarlo con la misma versión daría SHA-256 que no coinciden con los subidos. La versión siguiente lo vuelve a generar.

## Cómo se ha hecho

commitling se ha construido con un equipo de agentes de IA coordinados por issues, con una rama y un pull request por issue:

- **lead** y **builder** de la v0.1.0: Claude Opus.
- **qa**, **docs** y el trabajo de la v0.2.0: Claude Sonnet.
- **OpenCode** participa como delegado cuando el entorno permite su ejecución autónoma.
- **Alberto** supervisa y fusiona; nadie más hace commit ni push a `main`, ni crea etiquetas o releases.
- **24/10/2026: revisión nocturna de seguridad, documentación y accesibilidad con Claude Code (Opus) como jefe y OpenCode (GPT-5.6 Luna y modelos Zen) como delegados.**

Las reglas del equipo están en [AGENTS.md](../AGENTS.md), el estado del proyecto en [MEMORY.md](../MEMORY.md) y el flujo para colaborar en [CONTRIBUTING.md](../CONTRIBUTING.md). La [guía de uso](USO.md) explica las opciones desde el punto de vista de quien instala la Action.
