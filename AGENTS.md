# AGENTS.md · commitling

## Qué es
Una mascota pixel-art **original** que vive en el README de tu perfil de GitHub. Una GitHub Action lee tu actividad pública (commits, racha, días activos) y dibuja un SVG animado: la criatura crece por fases con tu experiencia, cambia de ánimo según tu actividad reciente y desbloquea accesorios.

## Tecnología (propia de este proyecto)
- **Go 1.27**, solo biblioteca estándar (`net/http`, `encoding/json`, `encoding/xml` para validar).
- SVG generado a mano con animación **CSS dentro del SVG** (GitHub no ejecuta JavaScript en los README).
- Tests con `go test`; formato con `gofmt` y análisis con `go vet`.
- **GitHub Action compuesta** (`action.yml`) reutilizable desde cualquier repositorio.
- Web del proyecto en **GitHub Pages**: galería de fases y ánimos generada por el propio binario (`commitling gallery`), un generador en vivo (wasm), un **modo demo** («Ver demo») y una criatura en vivo de BertMarti regenerada a diario.

## Comandos
- Tests: `go test ./...`
- Generador (comportamiento, Node): `node internal/gallery/testdata/generator.test.js <generator.js construido por la galería>`
- Análisis: `go vet ./...` y `gofmt -l .` (debe salir vacío)
- Generar: `go run ./cmd/commitling render --user BertMarti --out out/commitling.svg`
- Sin red: `go run ./cmd/commitling render --fixture testdata/events.json --out out/commitling.svg`
- Galería: `go run ./cmd/commitling gallery --out site/`
- wasm (generador en vivo): `GOOS=js GOARCH=wasm go build -trimpath -ldflags="-s -w" -o site/commitling.wasm ./cmd/wasm` y copiar `wasm_exec.js` de `$(go env GOROOT)/lib/wasm/` a `site/` (no se suben al repositorio)

## Estructura
- `cmd/commitling/` CLI.
- `cmd/wasm/` capa fina para el navegador (`syscall/js`) sobre `internal/generate`.
- `internal/generate/` función compartida CLI y wasm: eventos a SVG, workflow relleno, mensajes de error de la API y `Demo` (timelapse sintético de 90 días de `octoexample`).
- `internal/events/` parser de eventos y actividades (sin red).
- `internal/github/` cliente de la API pública de eventos (token opcional por `GITHUB_TOKEN`).
- `internal/stats/` cálculo de experiencia, racha y días activos.
- `internal/creature/` fases, ánimos, accesorios y mapas de píxeles (cuadrículas ASCII).
- `internal/render/` generación del SVG.
- `testdata/` datos de ejemplo sin información personal real.

## Demo en tiempo real (v0.5.0)
- `generate.Demo(day, Options)` dibuja el día 0..90 de un usuario ficticio (`octoexample`) con `stats.Compute` y el render de siempre; `commitling.demo(day, species, theme)` es la capa fina del wasm. Datos sintéticos y deterministas: nada de personas reales, nada de red.
- En `generator.js`: «Ver demo» (y el enlace de la cabecera) es una acción de la persona, nunca autoarranca; el día sale del tiempo transcurrido (15 s), no del número de cuadros; Pausa/Reanudar/Repetir y Detener están siempre en la página (deshabilitados, no ocultos); se pausa si la pestaña se oculta.
- Con `prefers-reduced-motion: reduce` no hay timelapse: solo el control deslizante de día.
- Solo `#demo-phase` es región viva (`aria-live`): el contador y el pie cambian en cada cuadro y no lo son (nada de `<output>`).
- Para probarla: servir la galería con el wasm y `wasm_exec.js`, pulsar «Ver demo», comprobar las 5 fases, Pausa/Reanudar/Detener (también con teclado: tras Detener el foco vuelve a «Ver demo»), el deslizante y `prefers-reduced-motion`.

## Ajustes visuales de v0.5.0
- La cabecera muestra la tarjeta real (`svg/acc-all.svg`, claro y oscuro) en un `<picture>`; el propio SVG apaga la animación con `prefers-reduced-motion`.
- Regla anti-desborde: toda rejilla declara sus columnas con `minmax(0,1fr)` (una columna implícita es `auto` y crece con el contenido) y los hijos `min-width:0`; el código en línea lleva `overflow-wrap:anywhere` (un nombre de archivo largo sin espacios desbordó a 320 px) y la primera columna de las tablas puede envolver en móvil; sin `width` fijo en CSS para las imágenes (hay tests de intención). Medir siempre a 320 px: a 360 y 375 no se veía.
- El escenario del generador es una tarjeta en blanco 12:5 con marco de 1 px y radio 8 px, como el SVG; sin sombras ni bordes discontinuos.
- No se toca `render.go` ni los sprites: el contorno es la silueta del cuerpo y se mueve con él a propósito.

## Tarjeta compacta (v0.6.0)
- Entrada `size` de la Action (`full` por defecto, `compact`) y `--size` en la CLI: la compacta es una insignia de 200x60 (criatura, fase, ánimo «ánimo · fase n/5» y barra de progreso) con la misma paleta, los mismos sprites y la misma animación a escala; sin zzz ni destellos. `render.Size` tiene `Full` como valor cero: sin `size` el SVG es byte a byte el de v0.5.0 (`testdata/golden/full-sha256.txt` guarda el SHA-256 de las 96 tarjetas de la galería de v0.5.0 y un test las compara; no cambies `fullLayout` ni el dibujo completo sin regenerarlas a propósito).
- El wasm recibe `size` como último argumento de `render`, `workflow`, `check` y `demo`; el generador de la web tiene el selector «Tamaño» y adapta la vista previa (`data-size` en `#gen-stage`). La galería enseña la compacta en `#compacta` (`svg/compact-*.svg`).
- Carga del wasm: empieza con `pointerdown` o `input` en el formulario (nunca al tabular ni al abrir la página), con `fetch(WASM_URL)` en paralelo a `wasm_exec.js`; el fallback usa `res.clone()` (hay tests).
- Los tests de CSS de la galería no comparan bloques literales: `css_test.go` lee las reglas (`cssRules(...).prop(selector, propiedad)`) y las aserciones nombran regla y propiedad.

## Ligero (v0.7.0)
- **Lo que entra en el wasm no puede importar `encoding/json`, `fmt`, `reflect`, `regexp`, `net/http`, `os` ni plantillas** (`TestWasmPackagesAvoidHeavyImports` lee los imports de `events`, `generate`, `render`, `stats`, `creature` y `cmd/wasm`; el CI falla si el wasm pasa de 3 MB). Texto: `strconv` y concatenación; errores: `errors.New`; eventos: `internal/events/scan.go` (lector a mano que sigue a `encoding/json`; `scan_test.go` lo compara con él, y ahí sí se puede importar `encoding/json`). `Event.Payload` es `[]byte`. CLI, galería y cliente HTTP (`internal/github`, `gallery`, `og`) pueden usar lo que quieran: no entran en el wasm.
- Medir antes de optimizar: `go tool nm` no lee wasm; el desglose por paquete sale de leer las secciones de código y de nombres de un wasm sin recortar (ver `docs/specs/v0.7.md`). Pesos (Go 1.27, `-trimpath -ldflags="-s -w"`): 4.624.571 B (1.261.906 gzip) en v0.6.0 y 2.569.892 B (735.538 gzip) en v0.7.0; el resto es el `runtime` de Go.
- Cualquier reescritura que toque `render` o `generate` se comprueba con las huellas (`testdata/golden/full-sha256.txt`: 96 tarjetas de v0.5.0 y `favicon.svg`) y comparando la galería entera y la CLI con las de `main` (`diff -r`).
- Generador y límite de la API: `generator.js` valida el login con la regla de `events.ValidLogin` (un test compara la regexp de JS con la función de Go), cachea los eventos 10 min en `sessionStorage` (`commitling:events:<login en minúsculas>`; todo con `try/catch`; no hay más almacenamiento, ni `localStorage` ni cookies) y, con un 403 o 429, enseña `generate.FetchError` (hora local de `X-RateLimit-Reset`) y el botón «Ver demo» (`#gen-demo-alt`).
- Tests de comportamiento del generador en Node: `node internal/gallery/testdata/generator.test.js <generator.js>` (DOM, `fetch`, `sessionStorage` y wasm simulados, sin dependencias); lo lanza `TestGeneratorBehaviourInNode`, que en el CI falla si no hay `node` y en otros sitios se salta.
- El lector de CSS de los tests (`css_test.go`) ignora comentarios y cadenas y entiende `@media` y `@supports`; sigue siendo un lector mínimo, no un analizador completo.

## Reglas de contenido
- El personaje es **diseño propio**. Prohibido imitar criaturas de videojuegos, anime o marcas.
- Las reglas de fase y ánimo están documentadas en el README y son deterministas (mismos datos, mismo SVG).

## Diseño: «papel y píxel»
Minimalista, cálido y lúdico, como un cuaderno con un sprite.
- Fondo papel `#f3efe6`, tinta `#2b2724`, líneas `#ddd5c6`.
- Paleta de la criatura limitada a 5 colores: `#2b2724`, `#f3efe6`, `#7fb069` (verde musgo), `#e6aa68` (miel), `#ca3c25` (acento).
- Píxeles nítidos (`shape-rendering: crispEdges`), sin degradados ni sombras.
- Tipografía monoespaciada del sistema; mucho espacio en blanco.
- Animaciones suaves y breves (respirar, parpadear) y respetando `prefers-reduced-motion`.

## Equipo de agentes y ramas
Los tres proyectos se desarrollan en paralelo con un equipo de agentes. El trabajo se guía por **issues del hito** (v0.2.0 en adelante): cada issue lleva la etiqueta del agente que la ejecuta (`agent:builder`, `agent:qa` o `agent:docs`) y sus criterios de aceptación son el contrato. Todo entra en `main` mediante pull request y **solo Alberto fusiona**.

| Agente | Herramienta | Etiqueta | Cometido |
|---|---|---|---|
| lead | Claude Code (sesión principal) | (crea y prioriza las issues) | Plan, revisión de PRs, integración y documentación final |
| builder | Claude Code (subagente) | `agent:builder` | Implementa funciones, tests, CI y despliegue |
| qa | Claude Code (subagente) | `agent:qa` | Revisa el código, añade tests de casos límite, corrige fallos y accesibilidad |
| docs | Claude Code (subagente, Sonnet); OpenCode cuando se permita su ejecución autónoma | `agent:docs` | Documentación para personas usuarias (`docs/USO.md`, README) |

Flujo por issue:
1. **Una rama y un PR por issue.** Rama: `agent/<rol>/<n>-<slug>` (por ejemplo `agent/builder/4-reintentos-5xx`).
2. Si varias issues del mismo agente dependen unas de otras, cada rama puede partir de la anterior, pero **todos los PR van contra `main`** (`gh pr create --base main`) y se fusionan en orden; el PR lo indica («Se fusiona después de #N»).
3. La descripción del PR lleva `Closes #<n>`, qué cambia, cómo se verificó y, si aplica, el orden de fusión.
4. Nadie hace commit ni push a `main`, ni crea etiquetas o releases: eso lo hace Alberto.
5. Los commits terminan con una línea `Agente: <rol> (Claude Code · Sonnet)` y la línea `Co-Authored-By`.

## Reglas para todos los agentes
1. **Lee `MEMORY.md` antes de empezar** y **actualízalo siempre al terminar** (estado, decisiones, siguiente paso y una línea en «Registro de sesiones» con fecha, agente y rama). Una sesión sin `MEMORY.md` actualizado no está terminada.
2. Nunca hagas commit directo a `main`. Trabaja en la rama de tu issue y abre un pull request contra `main`.
3. Commits convencionales en español: `feat:`, `fix:`, `test:`, `docs:`, `ci:`, `chore:`, `refactor:`. Cambios pequeños y con sentido propio.
4. No subas claves, tokens, `.env` ni datos personales. El repositorio es público.
5. No añadas dependencias sin justificarlo en «Decisiones» de `MEMORY.md`.
6. Si algo es ambiguo, elige la opción más simple, anótala en `MEMORY.md` y sigue.
7. La interfaz y la documentación, en español. El código (nombres), en inglés.

## Terminado significa
- Lint y tests en verde en local y en el CI.
- La aplicación funciona desplegada en GitHub Pages.
- README al día y `MEMORY.md` actualizado.
