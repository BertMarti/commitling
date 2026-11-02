# Registro de cambios

Todos los cambios notables de commitling se anotan en este archivo.

El formato sigue [Keep a Changelog](https://keepachangelog.com/es-ES/1.1.0/) y el proyecto usa [versionado semántico](https://semver.org/lang/es/).

## [Sin publicar]

### Cambiado

- Con la v1.1.0 publicada y `v1` apuntando a ella, el README, la guía de uso y el instalador de la web dejan de avisar de que la calabaza necesita una versión aún sin publicar, y los ejemplos de versión exacta pasan a `@v1.1.0`.
- `release/checksums.txt` sale de `main`: describe los binarios de la v1.1.0 ya publicada (va en su etiqueta) y no los de `main`, que ha cambiado. La próxima versión lo vuelve a generar en su commit.

## [1.1.0] - 2026-10-31

Una especie nueva y nada más que cambie para quien ya usa commitling: **la interfaz de `action.yml` es la misma** (mismas entradas con los mismos valores por defecto, misma salida `path`, mismos archivos; solo cambia el texto de ayuda de `species` para nombrar la nueva) y las tarjetas del brote de musgo, el hongo y el cactus son byte a byte las de la v1.0.0. Con `@v1` no hay que hacer nada.

### Añadido

- **Cuarta especie: la calabaza** (`species: pumpkin`, también vale `calabaza`), de diseño propio: una calabaza de huerto con ojos que pasa por **Pepita** (una pepita con un brote), **Plántula** (dos hojas redondas con la cáscara de la pepita encima), **Calabacita** (verde y rayada), **Calabaza** (naranja, con gajos, hoja y zarcillo) y **Calabaza gigante**, con las mismas reglas, ánimos y accesorios. Está en la Action, en `--species` de la CLI (`render` y `workflow`), en la galería de la web (claro, oscuro y tarjeta compacta), en los dos generadores de la web y en la imagen de vista previa.

### Cambiado

- La imagen de vista previa (`og.png`) coloca las fases de dos especies por fila para que quepan las cuatro.
- La web describía el cactus como «una semilla» al empezar; su primera fase es la **yema**.
- Con la v1.0.0 publicada y `v1` apuntando a ella, el README, la guía de uso y la web dejaron de avisar de que el cactus y `go run …@latest workflow` necesitaban una versión aún sin publicar, y los ejemplos de versión exacta pasaron a `@v1.0.0`.
- `release/checksums.txt` vuelve con los SHA-256 de los binarios de la 1.1.0 (tras la 1.0.0 había salido de `main`).
- Versión de la CLI: `1.1.0`.

## [1.0.0] - 2026-10-26

Primera versión estable (la 0.8.0 preparada nunca se publicó: todo sale aquí). Reúne todo lo hecho desde la v0.7.0: las fases de seguridad y documentación, la instalación guiada paso a paso, la tercera especie cactus, el selector de tema de la web y el binario precompilado. **La interfaz de `action.yml` no cambia** (mismas entradas, misma salida `path`, mismos archivos) y las tarjetas SVG del brote de musgo y del hongo son byte a byte las de v0.7.0 (las 96 huellas de v0.5.0 pasan). Quien usa `BertMarti/commitling@v1` la recibe sin tocar nada.

### Añadido

- **Tercera especie: el cactus** (`species: cactus`), de diseño propio: un cactus con ojos en su maceta de barro que pasa por **Yema**, **Bolita**, **Cactus**, **Cactus con brazos** y **Cactus centenario**, con las mismas reglas, ánimos y accesorios. Sale en la galería (claro, oscuro y compacta), en los dos generadores de la web, en `--species` y en la imagen de vista previa.
- **Selector de tema de la web** (Sistema · Claro · Oscuro) en la cabecera: cambia la página y todas las tarjetas sin recargar, se aplica antes de pintar (sin parpadeo) y se recuerda en el navegador con la clave `commitling:theme` (solo `light` o `dark`; «Sistema» la borra y cualquier otro valor se ignora). Sin JavaScript, la web sigue al sistema como antes.
- **Binario precompilado, activo desde esta versión**: `scripts/build-release.sh` compila los binarios (Linux, macOS y Windows; amd64 y arm64) de forma reproducible, `release.yml` los sube al publicar la release y la Action los usa en vez de instalar Go y compilar **solo** si el SHA-256 coincide con el `release/checksums.txt` de su propio árbol (el de la 1.0.0 va en este commit); si no hay binario o algo falla, compila con Go como siempre. Los perfiles se ahorran el `setup-go` y la compilación en cada ejecución.
- **Generador del workflow en la web** («Instálalo en tu perfil»): escribes tu usuario (o pegas el enlace de tu perfil), eliges especie, tema (claro, oscuro o **automático**, con las dos tarjetas y un `<picture>`) y tamaño, y copias el `.github/workflows/commitling.yml` y el fragmento del README ya rellenos. Seis pasos explicados para quien nunca ha usado Actions (repositorio de perfil, crear el archivo, permisos, primera ejecución con «Run workflow», README y resultado), enlaces directos a cada paso en GitHub (incluido «Abrir GitHub con el archivo ya relleno») y una lista «Si algo falla» con cada mensaje y qué hacer. Funciona sin red y sin wasm (`install.js`), no guarda nada y produce byte a byte lo mismo que `generate.Install` (test en Node y Go).
- **`commitling workflow`** (alias `init`): imprime ese mismo workflow (`--user`, `--species`, `--theme light|dark|both`, `--size`) y dice qué pegar en el README.
- **Mensajes de error que dicen qué hacer**, en español: usuario inexistente (404), token rechazado (401), acceso denegado (403), límite de peticiones (con la hora UTC a la que se restablece y si había token) y caídas de GitHub (5xx); el login no válido explica por qué («quita la @ del principio»…). Dentro de GitHub Actions el error sale también como anotación `::error` en el resumen de la ejecución.
- El workflow de ejemplo explica en el propio log por qué falla el `git push` («No se pudo guardar la criatura en el repositorio»: `permissions: contents: write` o rama protegida). El paso se llama ahora «Guardar la criatura si ha cambiado».
- CI: los 19 workflows que pueden generarse se validan con **actionlint** (con shellcheck) y deben usar `@v1`; la prueba de la Action comprueba que un usuario inexistente falla.

- **Content-Security-Policy estricta en la web**: solo scripts propios (más `'wasm-unsafe-eval'` para el generador), conexiones solo a la propia web y a `api.github.com`, sin `object`, `base` ni formularios externos; `referrer` `strict-origin-when-cross-origin` y `theme-color` para claro y oscuro. El script de los botones «Copiar» y el aviso de criatura en vivo no disponible pasan a `page.js` (antes eran un `<script>` y un `onerror` en línea, que la CSP bloquearía). Test `TestPageHasStrictCSP`.
- **Job «seguridad» en el CI**: `govulncheck` (sin vulnerabilidades con Go 1.27.3), `staticcheck` (también del wasm, sin avisos) y `gitleaks` sobre todo el historial (binario oficial con SHA-256 fijado).
- `.github/dependabot.yml` (Actions y Go, semanal y agrupado) y `SECURITY.md`.
- Tests de regresión: ninguna expresión `${{ }}` dentro de un script `run:` de `action.yml` o de los workflows (`TestNoExpressionsInRunScripts`), y todo workflow con `permissions` y `persist-credentials: false` (`TestWorkflowsAreLeastPrivilege`). Tests de escape de logins hostiles en todas las tarjetas.
- README para personas usuarias con cabecera pixel-art propia (`docs/img/banner.svg`, hecha con los sprites reales) y preguntas frecuentes; lo técnico pasa a `docs/DESARROLLO.md`.
- Documentación para agentes: `PLAN.md`, `.claude/agents/` y `.claude/skills/`.

### Seguridad

- **Trusted Types en la web** (`require-trusted-types-for 'script'; trusted-types commitling`): ninguna cadena puede llegar a `innerHTML`, `script.src`, `eval` ni similares; la única política, «commitling», solo crea la URL de `wasm_exec.js` (probado en Chromium: 0 violaciones con la demo, el generador y la criatura en vivo, y una asignación de prueba a `innerHTML` queda bloqueada).
- **Acciones de terceros fijadas por SHA** (con la versión en un comentario, que dependabot mantiene) en los workflows y en `action.yml` (`TestThirdPartyActionsArePinnedBySHA`).
- La Action **enmascara el token** (`::add-mask::`) antes de ejecutar nada, también si alguien lo pega en claro en `token:`; los mensajes de error nunca lo incluyen (test).
- **CSP sin `'unsafe-inline'` en estilos**: el CSS pasa a `style.css` y desaparecen los atributos `style=` (todas las webs de `bertmarti.github.io` comparten origen). La web no usa cookies; en `localStorage` solo guarda, si la eliges, la preferencia de tema (`commitling:theme`).

- **La Action, el CI y Pages compilaban con Go 1.27.0**, que tiene 11 vulnerabilidades conocidas en la biblioteca estándar que commitling usa (`net/http`, `net/http/internal/http2`, `net/textproto`, `crypto/tls`, `crypto/x509` y `html/template`; GO-2026-6599 a GO-2026-6809), porque `setup-go` instalaba exactamente la versión de `go.mod`. `go.mod` declara ahora `go 1.27.3` (la última), así que todo se compila con ella, también cuando la Action se usa varias veces en el mismo job (claro y oscuro): una línea `toolchain` no bastaba, porque el primer `setup-go` exporta `GOTOOLCHAIN=local` y el segundo volvía a instalar 1.27.0. Lo destapó el nuevo job de govulncheck; la prueba de la Action comprueba ahora la versión de Go tras varias llamadas.

### Cambiado

- Workflows con permisos mínimos: el de Pages no da nada por defecto (`build` solo lee; `deploy` tiene `pages` e `id-token`; `configure-pages` se quita porque nada usaba sus salidas) y ningún `checkout` guarda el token en `.git/config`.
- La prueba de la Action pasa `github.repository_owner` al script por `env:` en vez de interpolarlo.
- La CLI (y por tanto la Action) **escribe el SVG de forma atómica** (archivo temporal y renombrado): una ejecución cancelada a medias ya no puede dejar un SVG cortado, que `keep-on-error` necesita entero. Mismo contenido, mismos permisos (0644).
- `.gitignore` ignora `.env*` (salvo `.env.example`).
- Versión de la CLI: `1.0.0` (la Action y la CLI van alineadas).
- CI: el job `action` prueba siempre la ruta de `go run` (y el Go parcheado de cada llamada), sea cual sea el estado de la release, y el job `binario` prueba además la Action con el `release/checksums.txt` del repositorio contra la release real y exige que coincida con los binarios recompilados.

### Corregido

- `docs/USO.md` recomendaba fijar `@v1.0.0` cuando esa etiqueta aún no existía (el workflow fallaba): ahora remite a las releases.
- `docs/USO.md` decía que la Action aparece en el GitHub Marketplace (aún no está publicada allí) y que la etiqueta `v1` podía no existir (existe y apunta a la última versión publicada): ahora explica que basta la línea `uses:` y qué revisar si sale «unable to find version».

## [0.7.0] - 2026-10-01

### Cambiado

- **WebAssembly un 44 % más pequeño** (#47): de 4.624.571 a 2.569.892 bytes (de 1.261.906 a 735.538 con gzip; unos 6,3 s a unos 3,7 s en Slow-4G). `encoding/json` (con `reflect`) pesaba 1,6 MB y `fmt` 0,45 MB, así que los campos de los eventos se leen a mano (`internal/events/scan.go`, comparado con `encoding/json` por un test diferencial) y `render` y `generate` usan `strconv` y concatenación. Las tarjetas siguen siendo idénticas byte a byte: las 96 huellas SHA-256 de v0.5.0 pasan y la galería entera y la CLI coinciden con las de v0.6.0. `Event.Payload` pasa de `json.RawMessage` a `[]byte`. El CI compila con `-trimpath`, falla si el wasm pasa de 3 MB y un test impide que vuelvan `encoding/json`, `fmt`, `reflect`, `regexp`, `net/http`, `os` o las plantillas a lo que entra en el wasm.
- **Límite de la API de GitHub en el generador** (#48): con las 60 peticiones por hora agotadas, el aviso dice **a qué hora se restablece** (cabecera `X-RateLimit-Reset`, en la hora local del navegador) y la web ofrece «Ver demo». Los eventos de un usuario se guardan 10 minutos en `sessionStorage` (por pestaña), de modo que redibujar, cambiar tamaño, especie o tema o recargar no gasta peticiones, y el nombre de usuario se valida en JavaScript antes de cargar el wasm o pedir nada (misma regla que `events.ValidLogin`, comprobada por un test).
- **El texto de error ante unos eventos que no son JSON válido cambia** (por ejemplo, un fixture roto en `--fixture`): ahora dice «no se pudieron leer los eventos: JSON no válido en el byte N: …» en lugar del mensaje de `encoding/json`. Los códigos de salida son los mismos.
- Los datos de cada usuario se guardan 10 minutos en la pestaña: un commit muy reciente puede tardar en verse (abre la web en otra pestaña para empezar de cero). La pista del formulario y `docs/USO.md` lo dicen.
- Versión de la CLI: `0.7.0`. `action.yml` no cambia.

### Añadido

- `docs/specs/v0.7.md`, con la medición por paquete del wasm, qué se aplicó y qué se descartó.
- Test de comportamiento del generador en Node (`internal/gallery/testdata/generator.test.js`) con un DOM, `fetch`, `sessionStorage` y wasm simulados.

### Corregido

- Los tests de CSS (#49): el lector de reglas ignora los comentarios `/* */`, entiende `@supports` y no se traga la regla que sigue a un `@charset` o `@import`; `favicon.svg` entra en las huellas de compatibilidad (la de v0.5.0).

## [0.6.0] - 2026-10-01

### Añadido

- **Tarjeta compacta opcional** (#37 y #38): entrada `size` en la Action (`full` por defecto o `compact`) y `--size` en la CLI. La compacta es una insignia de 200x60 con la criatura, la fase, el ánimo («Contento · fase 3/5») y una barra de progreso, para firmas, barras laterales o la cabecera de un repositorio; misma paleta, mismos sprites y misma animación a escala (sin zzz ni destellos). Un valor desconocido falla con un mensaje claro. Sin `size`, el SVG es **byte a byte** el de v0.5.0: `testdata/golden/full-sha256.txt` guarda el SHA-256 de las 96 tarjetas de la galería de v0.5.0 y un test las compara.
- `render.Size` (`Full` es el valor cero), `generate.Options.Size` (también en `Check` y en el workflow relleno, que solo escribe `size: compact` si no es el predeterminado) y argumento `size`, siempre el último, en `commitling.render`, `workflow`, `check` y `demo` del wasm.
- La galería enseña la compacta en las cinco fases de las dos especies (`#compacta`, claro y oscuro) y el generador de la web tiene un selector «Tamaño» que redibuja en vivo, se aplica a la demo y al workflow relleno y adapta la vista previa.
- `docs/specs/v0.6.md`, con la causa del desborde medida antes y después.

### Corregido

- **Desborde lateral a 320 px** (#39): el documento medía 338 px (a 360 y 375 px no se veía). La causa real era un `<code>` sin puntos de corte (`.github/workflows/commitling.yml`) en un paso de «Instálalo»; además las rejillas del generador sin columnas declaradas (columna implícita `auto`) y las tablas con la primera columna en `nowrap`. Ahora el código en línea parte (`overflow-wrap:anywhere`), toda rejilla usa `minmax(0,1fr)` y las tablas envuelven en móvil. Medido después: 320 px.

### Cambiado

- Los tests de CSS de la galería comprueban intenciones (lector de reglas `css_test.go`) en lugar de cadenas exactas de CSS.
- Versión de la CLI: `0.6.0`.
- **Generador más rápido de arrancar** (#45): la descarga del wasm empieza también al tocar el formulario (`pointerdown`; tabular no cuenta) y se lanza en paralelo con `wasm_exec.js` en lugar de esperar a que cargue, reutilizando esa misma petición en `instantiateStreaming` (y su copia en el fallback, sin pedirlo dos veces). La criatura en vivo de la galería carga en diferido (`loading="lazy"`).

## [0.5.0] - 2026-10-01

### Añadido

- **Modo demo «Míralo crecer»** (#28 y #29): el botón «Ver demo» de la galería (y un enlace en la cabecera) reproduce un timelapse de unos 15 s con 90 días ficticios de `octoexample`: la criatura nace como semilla, pasa por las cinco fases y los cuatro ánimos y desbloquea los tres accesorios, con contador de días y la tarjeta SVG redibujada en vivo por el mismo código Go (wasm). Sin red, sin datos de personas reales y sin autoarranque; con Pausa/Reanudar/Repetir y Detener siempre visibles, un control deslizante de día (único mando con `prefers-reduced-motion`) y `aria-live` solo para los cambios de fase.
- `generate.Demo` (actividades sintéticas deterministas calculadas con `stats.Compute` y el render de siempre) y `commitling.demo(day, species, theme)` en el wasm, probados en Go y en Node.
- `docs/specs/v0.5.md`: especificación de la versión, con la auditoría de UX de OpenCode verificada contra el código.

### Cambiado

- **Cabecera** (#30): muestra la tarjeta real animada (clara y oscura) en lugar del sprite quieto de 112 px; se deja de generar `hero.svg`.
- **Galería sin desborde en móvil**: a 375 px el documento medía 730 px por los bloques `<pre>` de «Instálalo» en una rejilla sin `minmax(0,1fr)`; ahora no hay desplazamiento horizontal a 360 y 375 px.
- El escenario del generador es una tarjeta en blanco (proporción 12:5 y marco redondeado como el SVG) en lugar de una caja discontinua con hueco.
- La versión de la CLI pasa a `0.5.0`. Las entradas y salidas de `action.yml` no cambian.

## [0.4.0] - 2026-10-01

### Añadido

- **Generador en vivo con WebAssembly** (v0.4.0 «Pro», #19 a #22): la galería tiene una sección «Pruébalo con tu usuario» donde cualquiera escribe su usuario de GitHub, elige especie y tema y ve su criatura al instante, dibujada en su navegador por el mismo código Go que la CLI, compilado a WebAssembly (`GOOS=js GOARCH=wasm`, `syscall/js`). Los eventos públicos se piden desde el navegador a `api.github.com` (sin token; 60 peticiones por hora y por conexión, con un mensaje claro y los minutos hasta el reinicio si se agotan). Incluye botón para descargar el SVG y el workflow ya relleno con el usuario, la especie y el tema, con botón de copiar. El `.wasm` (unos 4,6 MB, 1,3 MB con gzip) solo se descarga al interactuar con el formulario.
- `internal/generate`: función compartida `Render` (eventos a SVG), `Workflow` (workflow relleno) y `FetchError` (mensajes de la API en español); la CLI dibuja con ella y un test comprueba que da los mismos bytes que el wasm.
- `cmd/wasm`: capa fina de `syscall/js` (`commitling.render`, `workflow`, `explain`, `check`), probada dentro de Node en el CI.
- El CI comprueba que el wasm compila y muestra su peso; el despliegue de Pages genera `commitling.wasm` y copia `wasm_exec.js` de la misma versión de Go (ninguno de los dos está en el repositorio).

### Cambiado

- La versión de la CLI pasa a `0.4.0`.
- `commitling.workflow` (y el workflow que muestra el generador) escribe `user` siempre entre comillas, para que un login como `007`, `1e3`, `null` o `true` no se lea como número, booleano o nulo en YAML, y escribe el tema con su nombre canónico.
- Las regiones de estado y de error del generador ya no se ocultan cuando están vacías, para que existan siempre en el árbol de accesibilidad.
- `ValidLogin` se comprueba a mano en lugar de con una expresión regular (mismas reglas), lo que reduce el wasm.
- El generador cancela las peticiones a GitHub a los 15 s, comprueba que cada página sea una lista, reutiliza los eventos durante un minuto si se vuelve a dibujar el mismo usuario, empieza a descargar el wasm al escribir (no al enfocar) y usa un `alt` corto.
- El parser de eventos pasa a `internal/events`, sin `net/http`, para que el wasm pese 5,1 MB en lugar de 7,1 MB; `internal/github` conserva alias con la misma API.
- `docs/specs/v0.4.md`: especificación de la versión.

## [0.3.0] - 2026-09-30

### Añadido

- **Conservar el último SVG válido ante caídas transitorias de la API** (#16): opción `--keep-on-error` en `commitling render` y entrada `keep-on-error` en la Action (por defecto `true`; los workflows existentes la reciben sin cambiar nada). Si la API falla tras los reintentos por una causa transitoria (error de red, 5xx, 429, o 403 con cabeceras de límite) y el archivo de salida ya es un SVG completo, se conserva intacto y el paso termina en verde con un aviso (`::warning::` en GitHub Actions). Los fallos permanentes (401, 404, 422, 403 sin cabeceras de límite), la ausencia de archivo previo, un archivo vacío o cortado y `--out -` siguen siendo un error, igual que un `fixture` inexistente o una especie no válida.
- `github.IsTransient` y `APIError.RateLimited` en el cliente de la API.

### Cambiado

- Los workflows existentes dejan de fallar ante caídas transitorias de la API (errores de red, 5xx o límite de peticiones) si el SVG de salida ya existe: ahora lo conservan y avisan, porque `keep-on-error` está activada por defecto. Es un cambio de comportamiento compatible: para recuperar el fallo anterior, `keep-on-error: false`. Los errores permanentes (token inválido, usuario inexistente) siguen haciendo fallar el paso.

## [0.2.0] - 2026-09-30

Versión guiada por issues (#4 a #9). Los cambios llegan en los pull requests #10 a #14 y la documentación en el #15.

### Añadido

- **Segunda especie: el hongo** (#6, PR #12). Una seta pequeña de diseño propio con cinco fases (Espora, Botón, Seta, Seta grande y Corro de setas), los mismos cuatro ánimos y tres accesorios que el brote de musgo, y las mismas reglas y umbrales. Se elige con la entrada `species` de la Action o con `--species` en la CLI; acepta `moss`, `musgo`, `mushroom`, `hongo` y `seta`. El brote de musgo sigue siendo la especie predeterminada.
- **Imagen Open Graph** (#5, PR #11): nuevo comando `commitling og --out` que dibuja una vista previa PNG de 1200×630 (paleta indexada de 7 colores, sin dependencias externas). La galería genera `og.png` y las etiquetas `og:image` y `twitter:card` (`summary_large_image`) apuntan a ella.
- **Reintentos en el cliente de la API** (#4, PR #10): hasta 3 reintentos con esperas de 1, 2 y 4 segundos ante los errores 500, 502, 503 y 504 y ante el límite de peticiones (429, o 403 con `Retry-After` o `X-RateLimit-Remaining: 0`). Respeta `Retry-After` y `X-RateLimit-Reset` con un tope de 30 s.
- **Action lista para el Marketplace y para `@v1`** (#7, PR #13): nombre, descripción, autor y `branding` (icono `feather`, color `green`) en `action.yml`, y la sección «GitHub Marketplace y versiones» en el README. Tests que vigilan `action.yml`.
- La galería tiene una sección por especie, enlace «Saltar al contenido» y navegación entre especies (#8, PR #14).
- Tests nuevos de red y tiempo de espera, de cabeceras de límite extremas, de contraste de la web, de especie inválida y de `action.yml` (#8, PR #14).
- `CHANGELOG.md`, guía de uso ampliada (comando `og`, especies, reintentos, `@v1`) y guía de contribución con la receta para añadir una especie (#9).

### Cambiado

- Los ejemplos de workflow del README, la galería y `docs/USO.md` usan `BertMarti/commitling@v1` en lugar de `@main`. La etiqueta `v1` la crea la persona responsable del proyecto tras fusionar los PR.
- El tiempo máximo de la CLI sube de 1 a 3 minutos para dejar sitio a los reintentos.
- Una especie desconocida en `species` o `--species` es un error de uso (código 2) que lista los valores válidos, en vez de caer en la predeterminada.
- La versión de la CLI pasa a `0.2.0`.
- Flujo de trabajo por issues (etiquetas `agent:*`, una rama y un PR por issue, siempre contra `main`) descrito en `AGENTS.md`.

### Corregido

- Un `Retry-After` o `X-RateLimit-Reset` enorme desbordaba la duración y provocaba una espera negativa, es decir, un reintento inmediato. Ahora supera el tope de 30 s y no se reintenta (#8, PR #14).
- Los errores de red (DNS, conexión rechazada o cortada, tiempo agotado) no se reintentaban; ahora sí, con el mismo límite y las mismas esperas que los 5xx (#8, PR #14).
- Contraste de la web: el color de acento no llegaba a AA como texto y como contorno de foco (4,37:1 en claro y 2,95:1 en oscuro); ahora es `#b8341f` en claro y `#ee6e55` en oscuro (#8, PR #14).
- `og:image:alt` menciona las dos especies (#8, PR #14).

## [0.1.0] - 2026-09-30

Primera versión: el MVP.

### Añadido

- **MVP de commitling** (PR #1): cliente de la API pública de eventos de GitHub (hasta 300 eventos, `GITHUB_TOKEN` opcional), cálculo de XP, racha, días activos y repositorios, criatura pixel-art original (brote de musgo) con 5 fases, 4 ánimos y 3 accesorios, tarjeta SVG animada con CSS y temas claro y oscuro, CLI (`render`, `gallery`, `version`), GitHub Action compuesta (`action.yml`), galería estática y flujos de CI y de despliegue en GitHub Pages.
- **Revisión de calidad** (PR #2): tests de casos límite, texto del SVG que se ajusta sin salirse del panel, cabeceras y tiempo máximo en el cliente HTTP, eventos duplicados contados una sola vez, contraste AA del gris atenuado y metadatos Open Graph y accesibilidad de la galería.
- **Documentación** (PR #3): guía de uso para personas usuarias (`docs/USO.md`), guía de contribución (`CONTRIBUTING.md`) y enlaces desde el README.

### Cambiado

- Las acciones de los workflows suben de versión para evitar Node 20 (`checkout@v7`, `setup-go@v7`, `configure-pages@v6`, `upload-pages-artifact@v5` y `deploy-pages@v5`) (PR #2).

[Sin publicar]: https://github.com/BertMarti/commitling/compare/v1.1.0...HEAD
[1.1.0]: https://github.com/BertMarti/commitling/compare/v1.0.0...v1.1.0
[1.0.0]: https://github.com/BertMarti/commitling/compare/v0.7.0...v1.0.0
[0.7.0]: https://github.com/BertMarti/commitling/compare/v0.6.0...v0.7.0
[0.6.0]: https://github.com/BertMarti/commitling/compare/v0.5.0...v0.6.0
[0.5.0]: https://github.com/BertMarti/commitling/compare/v0.4.0...v0.5.0
[0.4.0]: https://github.com/BertMarti/commitling/compare/v0.3.0...v0.4.0
[0.3.0]: https://github.com/BertMarti/commitling/compare/v0.2.0...v0.3.0
[0.2.0]: https://github.com/BertMarti/commitling/compare/v0.1.0...v0.2.0
[0.1.0]: https://github.com/BertMarti/commitling/releases/tag/v0.1.0
