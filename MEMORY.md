# MEMORY.md · commitling
Última actualización: 2026-09-30 por builder

## Estado actual
v0.1.0 está en `main` (MVP, revisión QA y guía de uso ya fusionados). Fase v0.2.0, guiada por issues (ver «Equipo de agentes y ramas» en AGENTS.md): el builder entrega un PR por issue, todos contra `main` y en este orden de fusión:
1. #4 `agent/builder/4-reintentos-5xx`: el cliente de la API reintenta (hecho, PR #10).
2. #5 `agent/builder/5-og-png`: PNG 1200x630 para Open Graph (hecho, PR #11).
3. #6 `agent/builder/6-nueva-especie`: segunda especie seleccionable (hecho, PR abierto).
4. #7 `agent/builder/7-action-v1`: Action lista para v1 y el Marketplace (pendiente).

Lo que hay hoy en `main` (v0.1.0): `internal/stats` (XP, días activos, racha), `internal/github` (cliente de eventos públicos, parser, dedupe), `internal/creature` (5 fases, 4 ánimos, 3 accesorios, sprites 16x16), `internal/render` (tarjeta SVG 480x200 con animación CSS, temas claro y oscuro), `internal/gallery` (web estática), CLI `render`/`gallery`/`version`, `action.yml`, workflows `ci`, `deploy` (Pages, diario) y `action-test`, README, `docs/USO.md` y `CONTRIBUTING.md`.

Cambios de #6: nueva especie **hongo** (`mushroom`, "Espora, Botón, Seta, Seta grande, Corro de setas") junto al brote de musgo (`moss`, predeterminada). `creature.Species` (`MossSprout` es el valor cero), `Creature.Species`, `FromStatsAs`, `Creature.StageName()`, `SpeciesByName` (acepta slug o nombre en español). Mapas en `internal/creature/mushroom.go`. CLI `render --species`, entrada `species` de la Action, galería con una sección por especie (100 archivos) y la imagen Open Graph muestra las fases de las dos especies.

Cambios de #5: nuevo paquete `internal/og` (PNG 1200x630 con `image`, `image/color` e `image/png`, paleta indexada de 7 colores, escala entera, fuente de mapa de bits propia 5x7), comando `commitling og --out`, la galería genera `og.png` y `og:image`/`twitter:card` apuntan a él.

Cambios de #4: el cliente reintenta hasta 3 veces (esperas de 1, 2 y 4 s) ante 500, 502, 503 y 504 y ante límite de peticiones (429, o 403 con `Retry-After` o `X-RateLimit-Remaining: 0`), respeta `Retry-After` (segundos o fecha) y `X-RateLimit-Reset` y no reintenta 404, 422 ni otros 4xx.

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
- 2026-09-30 (docs): la guía de uso la escribe Claude Code (Sonnet) y no OpenCode, porque el sistema de permisos no permite lanzar OpenCode en modo autónomo; la fila del agente docs de `AGENTS.md` y la rama (`agent/docs`, no `agent/opencode-docs`) se actualizaron. OpenCode se retomará cuando se permita su ejecución autónoma.
- 2026-09-30 (docs): `docs/USO.md` repite el workflow del README (y una variante con tema oscuro). Solo el del README está vigilado por test frente a `gallery.WorkflowSnippet`; si se cambia el workflow hay que cambiar los tres sitios. La guía también advierte de que la XP puede bajar (solo cuentan los últimos 90 días) y de que GitHub desactiva los workflows programados tras unos 60 días sin actividad.
- 2026-09-30 (builder): flujo por issues (#4 a #7), una rama y un PR por issue, cada rama parte de la anterior pero todos los PR van contra `main`; AGENTS.md actualizado.
- 2026-09-30 (builder): reintentos (#4): 3 reintentos como máximo (4 peticiones por página), backoff 1, 2, 4 s. Si el servidor indica una espera con `Retry-After` o `X-RateLimit-Reset` se usa esa; si supera el tope de 30 s (`DefaultMaxWait`) **no se reintenta** y se devuelve el error con su explicación (esperar 40 minutos a que se reinicie la cuota colgaría el workflow). Un 403 sin cabeceras de límite es de permisos y no se reintenta; un 429 siempre. Solo se repite la página que falla. El error final dice «tras N reintentos». `Sleep` y `Now` son inyectables en `Client` para testear sin esperar. El tiempo máximo de la CLI pasa de 1 a 3 minutos para dejar sitio a las esperas.
- 2026-09-30 (builder): PNG Open Graph (#5) en el paquete nuevo `internal/og`, sin dependencias. Imagen indexada (`image.Paletted`, 7 colores: los 5 de la criatura más línea `#ddd5c6` y gris atenuado `#6f675e` de la tarjeta), sin suavizado: la criatura grande va a escala 20 y las cinco fases pequeñas a escala 3, el texto con una fuente propia 5x7 dibujada con bloques (letras, dígitos y algo de puntuación, sin acentos, así que el texto de la tarjeta evita tildes). Codificación `png.BestCompression`, determinista (mismo binario, mismos bytes). La criatura de la imagen es `og.Hero` (retoño radiante con flor, el mismo que `hero.svg`); pesa unos 3 KB.
- 2026-09-30 (builder): `og:image` pasa a `https://bertmarti.github.io/commitling/og.png` con `og:image:type/width/height` y `twitter:card=summary_large_image`. `hero.svg` se mantiene (lo usa la web).
- 2026-09-30 (builder): especie nueva (#6) = **hongo**, diseño propio: seta de sombrero miel con motas de musgo, cara en el sombrero, láminas de papel bajo el sombrero y pie de musgo; la fase máxima («Corro de setas») suma dos setas pequeñas a los lados. Deliberadamente NO es un sombrero rojo con puntos blancos ni lleva la cara en el pie (para no parecerse a ninguna mascota de videojuegos). Misma paleta de 5 colores, mismas reglas y umbrales; solo cambian el dibujo y los nombres de las fases. Las constantes de especie se llaman `MossSprout` y `Mushroom` porque `Moss` ya es el color.
- 2026-09-30 (builder): `Creature{}` (valor cero) sigue siendo el brote de musgo, así que `render.NewCard` y `creature.FromStats` no cambian; la CLI asigna `card.Creature.Species`. Los archivos de la especie predeterminada en la galería conservan sus nombres (`svg/<fase>-<ánimo>.svg`); los del hongo llevan el prefijo `mushroom-`.
- 2026-09-30 (builder): `species` inválido en la Action/CLI es error de uso (código 2) en lugar de caer en la predeterminada, para que un error tipográfico no pase desapercibido. Se aceptan `moss`, `musgo`, `mushroom`, `hongo` y `seta` sin distinguir mayúsculas.
- 2026-09-30 (builder): la web de la galería pasa a una sección por especie (`h3.species` + `h4` por fase); la tabla de fases de las reglas muestra los dos nombres («Semilla / Espora»). La cara y los accesorios de cada especie tienen test de anclaje (la cara sobre relleno, la bufanda en una fila continua, el gorro y la flor sin tapar la cara).

## Siguiente paso
1. Alberto: fusionar los PR del builder en orden (#4, #5, #6, #7) y, tras cada uno, comprobar el CI.
2. Alberto: tras fusionar el de #7, crear la etiqueta `v1` y publicar en el Marketplace (instrucciones en el PR de #7).
3. qa (otra ronda): revisión visual real de la galería en navegador y del foco/orden de tabulación en móvil.
4. Pendiente menor: la sección «Cómo se ha hecho» del README aún cita OpenCode como parte del equipo.

## Problemas conocidos
- En el Windows local de Alberto, el Control de aplicaciones (Smart App Control) bloquea a veces los binarios que genera Go (`go test`, `go run`, `go build`). Solución: `GOTMPDIR="$PWD/out/gotmp"` y reintentar; en el CI no pasa.
- `live/BertMarti.svg` solo existe tras el primer despliegue desde `main`; hasta entonces la imagen del README aparece rota y la web muestra un aviso.
- La API de eventos solo da 90 días / 300 eventos: perfiles muy activos pueden ver su XP recortada a lo que cabe en esos 300 eventos.
- La Action compila con `go run` en cada ejecución (unos segundos extra); aceptable para un cron diario.
- `action-test.yml` prueba la especie hongo con el fixture, pero un fallo ahí solo se ve en el CI del PR.
- Las redes sociales cachean las vistas previas: tras desplegar, puede tardar en verse el nuevo `og.png` (se puede forzar con el depurador de Facebook o el Card Validator de LinkedIn).
- El ancho del texto del SVG se estima (0,62 em por carácter), no se mide: con una fuente más ancha que las de la pila (p. ej. una monoespaciada del sistema fuera de la lista) podría rozar el borde con logins de casi 39 caracteres.
- El cliente no reintenta los errores de red (DNS, conexión rechazada, timeout); solo las respuestas 5xx y de límite de peticiones. Una caída larga de la API sigue haciendo fallar la ejecución diaria (se reintenta al día siguiente).

## Registro de sesiones
- 2026-09-29 lead (main): creación del repositorio y reparto del equipo.
- 2026-09-29 builder (agent/builder): MVP completo (stats, cliente GitHub, criatura, SVG, CLI, galería, Action, CI, Pages, README) y PR a main.
- 2026-09-30 qa · Claude Code Sonnet (agent/qa): versiones de acciones, desbordes del SVG con tests, cliente HTTP (cabeceras, timeout, 422), eventos duplicados, contraste AA y galería (Open Graph, copiar accesible). PR #2 hacia agent/builder.
- 2026-09-30 docs · Claude Code Sonnet (agent/docs): `docs/USO.md` (guía para personas usuarias contrastada con el código), `CONTRIBUTING.md`, enlaces desde el README y fila del agente docs en AGENTS.md. PR hacia agent/qa.
- 2026-09-30 builder · Claude Code Sonnet (agent/builder/4-reintentos-5xx): reintentos ante 5xx y límite de peticiones (#4) y flujo por issues en AGENTS.md.
- 2026-09-30 builder · Claude Code Sonnet (agent/builder/5-og-png): PNG 1200x630 para Open Graph (#5): `internal/og`, comando `og`, galería y metadatos.
- 2026-09-30 builder · Claude Code Sonnet (agent/builder/6-nueva-especie): especie hongo (#6): sprites, `--species`, entrada `species`, galería con dos especies, Open Graph con ambas y tests.

