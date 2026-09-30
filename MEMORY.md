# MEMORY.md · commitling
Última actualización: 2026-09-30 por qa

## Estado actual
Revisión QA en `agent/qa` (PR #2 hacia `agent/builder`; se fusiona después de #1 y luego hay que reorientarlo a `main`): acciones sin Node 20, texto del SVG que ya no desborda, cliente HTTP endurecido, eventos duplicados descartados, galería con Open Graph y botones de copiar accesibles. Ver «Decisiones» del 2026-09-30.

MVP completo en la rama `agent/builder` (PR #1 abierto a `main`, pendiente de revisión del lead):
- `internal/stats`: XP, días activos (30 y 90), racha, días desde la última actividad y repos distintos. Puro, con «ahora» inyectable.
- `internal/github`: cliente de `/users/<u>/events/public` (paginado, hasta 300 eventos, `GITHUB_TOKEN` opcional), parser y conversión a actividades. Fixture ficticio en `testdata/events.json` (usuario «octoexample»).
- `internal/creature`: 5 fases, 4 ánimos, 3 accesorios y mapas de píxeles 16×16 originales (brote de musgo con ojos).
- `internal/render`: tarjeta SVG 480×200 con animación CSS (respirar/saltar, parpadeo, zzz, destellos), `prefers-reduced-motion`, temas `light` y `dark`.
- `internal/gallery`: web estática (`index.html` + 48 SVG claro/oscuro + favicon + hero) con reglas, instalación y hueco para `live/BertMarti.svg`.
- `cmd/commitling`: `render`, `gallery`, `version`.
- `action.yml` (compuesta) y workflows `ci.yml`, `deploy.yml` (Pages, diario) y `action-test.yml`.
- README completo. Tests en todos los paquetes.

## Decisiones (por qué)
- 2026-09-29: Animación con CSS dentro del SVG porque GitHub no ejecuta JavaScript en los README.
- 2026-09-29: Go con biblioteca estándar: binario sin dependencias y stack distinto al de los otros proyectos.
- 2026-09-29: Solo datos públicos de la API de eventos; nada de repos privados.
- 2026-09-29 (builder): `go 1.27.0` en go.mod; setup-go ya ofrece 1.27.x, así que CI y Action usan `go-version-file`. `cache: false` porque no hay go.sum.
- 2026-09-29 (builder): un PushEvent sin `size` ni `commits` (GitHub está recortando esos payloads) cuenta como 1 commit.
- 2026-09-29 (builder): PR e issues solo dan su XP al abrirse (`action: opened`); el resto de acciones cuentan como «otra actividad» (2 XP).
- 2026-09-29 (builder): «repos distintos» solo cuenta repos con commits, PR o issues (dar una estrella no es «tocar» un repo).
- 2026-09-29 (builder): la racha se mantiene hasta el final del día UTC siguiente a la última actividad; todo se calcula en días UTC y solo con eventos de los últimos 90 días y no posteriores a «ahora».
- 2026-09-29 (builder): con `--fixture` y sin `--now`, «ahora» es la fecha del último evento del fixture, para que el resultado sea estable día a día.
- 2026-09-29 (builder): el SVG no incluye fechas ni nada variable; así la Action solo provoca commit cuando la criatura cambia.
- 2026-09-29 (builder): tema oscuro: fondo #2b2724, tinta #f3efe6, líneas #4a433d y apagado #a39a8e (derivados). La silueta de la criatura se dibuja en papel en el tema oscuro (capa `Outline` aparte), como una pegatina; la paleta sigue siendo de 5 colores.
- 2026-09-29 (builder): paquete extra `internal/gallery` (no estaba en la estructura de AGENTS.md) para la web; la plantilla va embebida con `go:embed`.
- 2026-09-29 (builder): en `action.yml` la ruta de go.mod se toma de `$GITHUB_ACTION_PATH` en un paso `run` y se pasa como salida a setup-go, porque `github.action_path` no es fiable en los `with:` de una action compuesta. Entrada extra `fixture` para probar sin red.
- 2026-09-29 (builder): el workflow de ejemplo usa `BertMarti/commitling@main` (aún no hay etiqueta `v1`).
- 2026-09-29 (builder): si la API falla en el despliegue, la criatura en vivo se genera con el fixture y muestra «@octoexample» (no se hace pasar por datos reales); queda un aviso en el log y en el resumen del job.
- 2026-09-29 (builder): un test comprueba que el workflow del README es idéntico a `gallery.WorkflowSnippet`; si se cambia uno, hay que cambiar el otro.

- 2026-09-30 (qa): acciones subidas a `checkout@v7`, `setup-go@v7`, `configure-pages@v6`, `upload-pages-artifact@v5` y `deploy-pages@v5` (workflows, `action.yml`, README y galería; el test que ata README y galería sigue en verde).
- 2026-09-30 (qa): el texto del panel del SVG se ajusta de forma determinista (`fit` en `internal/render`): tamaño decreciente, redacción más corta («meta: N XP») y, como último recurso, recorte con «…». Anchura estimada a 0,62 em por carácter (cota superior de las fuentes de la pila). Un login de 39 caracteres se muestra entero a 9-10 px y, si es muy largo, se oculta el rótulo «commitling» de la derecha (el `<title>` lo conserva). Tests: nada sale del panel ni se solapa, para todas las fases, ánimos, temas y logins de 0 a 80 caracteres, con XP de 7 cifras y los tres accesorios.
- 2026-09-30 (qa): pila de fuentes `ui-monospace, SFMono-Regular, Menlo, Consolas, "DejaVu Sans Mono", "Liberation Mono", monospace` (SVG y web; se quitó «SF Mono»).
- 2026-09-30 (qa): gris atenuado de la tarjeta clara `#8a8178` -> `#6f675e` y `--muted` de la web `#7a7168` -> `#6b6259` para llegar a contraste AA (4,5:1); no es un color de la criatura ni cambia el diseño. Test de contraste.
- 2026-09-30 (qa): cliente HTTP: `User-Agent: commitling`, `X-GitHub-Api-Version: 2022-11-28`, timeout de 15 s (también si se construye el `Client` a mano). Un 422 pasada la página 1 es fin de datos; en la página 1 sigue siendo error.
- 2026-09-30 (qa): eventos con el mismo id se cuentan una vez (`github.Dedupe`, en `FetchEvents` y en `Activities`): al llegar un evento nuevo mientras se pagina, la página 2 repetía el último de la 1 y duplicaba su XP. Eventos sin id no se fusionan. `stats.Compute` ya ignoraba el orden y las fechas futuras; se añadieron tests.
- 2026-09-30 (qa): galería: `og:title`, `og:description`, `og:image` = `https://bertmarti.github.io/commitling/hero.svg`, `og:url`, `og:type`, `twitter:card`, canonical; botones de copiar con `aria-label`, región `aria-live`, texto seleccionado y mensaje si falla el portapapeles, ocultos sin JavaScript; los bloques `pre` son enfocables con teclado.

## Siguiente paso
1. lead: revisar y fusionar el PR de `agent/builder`; tras el merge, comprobar el despliegue en https://bertmarti.github.io/commitling/ y la imagen en vivo del README.
2. lead: fusionar #1, reorientar el PR #2 (qa) a `main` y fusionarlo; comprobar el despliegue.
2b. qa (pendiente para otra ronda): reintentos ante 5xx intermitentes en el cliente; revisión visual real de la galería en navegador (esta ronda solo se verificó con tests y sintaxis del JS con `node --check`); comprobar el resto de la web (foco, orden de tabulación) en móvil.
3. docs (`agent/opencode-docs`): escribir `docs/USO.md` a partir de las secciones «Úsalo en tu perfil», «Reglas» y «CLI» del README y de la web; recordar que el workflow del README está atado a `internal/gallery/gallery.go` por un test.
4. Pendiente: publicar una etiqueta `v1` para que la gente use `BertMarti/commitling@v1` en lugar de `@main`.

## Problemas conocidos
- En el Windows local de Alberto, el Control de aplicaciones (Smart App Control) bloquea a veces los binarios que genera Go (`go test`, `go run`, `go build`). Solución: `GOTMPDIR="$PWD/out/gotmp"` y reintentar; en el CI no pasa.
- `live/BertMarti.svg` solo existe tras el primer despliegue desde `main`; hasta entonces la imagen del README aparece rota y la web muestra un aviso.
- La API de eventos solo da 90 días / 300 eventos: perfiles muy activos pueden ver su XP recortada a lo que cabe en esos 300 eventos.
- La Action compila con `go run` en cada ejecución (unos segundos extra); aceptable para un cron diario.
- `og:image` apunta a un SVG (lo pedido): X/Twitter, Facebook y LinkedIn no muestran SVG en las vistas previas; para que salga imagen habría que publicar un PNG 1200×630 (sin dependencias no hay rasterizador en la biblioteca estándar; habría que generarlo aparte).
- El ancho del texto del SVG se estima (0,62 em por carácter), no se mide: con una fuente más ancha que las de la pila (p. ej. una monoespaciada del sistema fuera de la lista) podría rozar el borde con logins de casi 39 caracteres.
- El cliente no reintenta ante 5xx: una caída puntual de la API hace fallar la ejecución diaria (se reintenta al día siguiente).

## Registro de sesiones
- 2026-09-29 lead (main): creación del repositorio y reparto del equipo.
- 2026-09-29 builder (agent/builder): MVP completo (stats, cliente GitHub, criatura, SVG, CLI, galería, Action, CI, Pages, README) y PR a main.
- 2026-09-30 qa · Claude Code Sonnet (agent/qa): versiones de acciones, desbordes del SVG con tests, cliente HTTP (cabeceras, timeout, 422), eventos duplicados, contraste AA y galería (Open Graph, copiar accesible). PR #2 hacia agent/builder.
