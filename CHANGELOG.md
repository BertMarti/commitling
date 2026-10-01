# Registro de cambios

Todos los cambios notables de commitling se anotan en este archivo.

El formato sigue [Keep a Changelog](https://keepachangelog.com/es-ES/1.1.0/) y el proyecto usa [versionado semántico](https://semver.org/lang/es/).

## [Sin publicar]

## [0.7.0] - 2026-10-01

### Cambiado

- **WebAssembly un 45 % más pequeño** (#47): de 4.624.571 a 2.566.093 bytes (de 1.261.906 a 734.870 con gzip; unos 6,3 s a unos 3,7 s en Slow-4G). `encoding/json` (con `reflect`) pesaba 1,6 MB y `fmt` 0,45 MB, así que los campos de los eventos se leen a mano (`internal/events/scan.go`, comparado con `encoding/json` por un test diferencial) y `render` y `generate` usan `strconv` y concatenación. Las tarjetas siguen siendo idénticas byte a byte: las 96 huellas SHA-256 de v0.5.0 pasan y la galería entera y la CLI coinciden con las de v0.6.0. `Event.Payload` pasa de `json.RawMessage` a `[]byte`. El CI compila con `-trimpath`, falla si el wasm pasa de 3 MB y un test impide que vuelvan `encoding/json`, `fmt`, `reflect`, `regexp`, `net/http`, `os` o las plantillas a lo que entra en el wasm.
- **Límite de la API de GitHub en el generador** (#48): con las 60 peticiones por hora agotadas, el aviso dice **a qué hora se restablece** (cabecera `X-RateLimit-Reset`, en la hora local del navegador) y la web ofrece «Ver demo». Los eventos de un usuario se guardan 10 minutos en `sessionStorage` (por pestaña), de modo que redibujar, cambiar tamaño, especie o tema o recargar no gasta peticiones, y el nombre de usuario se valida en JavaScript antes de cargar el wasm o pedir nada (misma regla que `events.ValidLogin`, comprobada por un test).
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

[Sin publicar]: https://github.com/BertMarti/commitling/compare/v0.7.0...HEAD
[0.7.0]: https://github.com/BertMarti/commitling/compare/v0.6.0...v0.7.0
[0.6.0]: https://github.com/BertMarti/commitling/compare/v0.5.0...v0.6.0
[0.5.0]: https://github.com/BertMarti/commitling/compare/v0.4.0...v0.5.0
[0.4.0]: https://github.com/BertMarti/commitling/compare/v0.3.0...v0.4.0
[0.3.0]: https://github.com/BertMarti/commitling/compare/v0.2.0...v0.3.0
[0.2.0]: https://github.com/BertMarti/commitling/compare/v0.1.0...v0.2.0
[0.1.0]: https://github.com/BertMarti/commitling/releases/tag/v0.1.0
