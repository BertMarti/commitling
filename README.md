<div align="center">

![commitling](https://bertmarti.github.io/commitling/live/BertMarti.svg)

# commitling

**Una mascota pixel-art que vive en el README de tu perfil de GitHub y crece con tus commits.**

[![CI](https://github.com/BertMarti/commitling/actions/workflows/ci.yml/badge.svg)](https://github.com/BertMarti/commitling/actions/workflows/ci.yml)
[![Pages](https://github.com/BertMarti/commitling/actions/workflows/deploy.yml/badge.svg)](https://github.com/BertMarti/commitling/actions/workflows/deploy.yml)
[![Licencia MIT](https://img.shields.io/badge/licencia-MIT-7fb069)](LICENSE)

[Galería y generador en vivo](https://bertmarti.github.io/commitling/) · [Instalar](#úsalo-en-tu-perfil) · [Reglas](#reglas)

</div>

Una GitHub Action lee tu actividad **pública** (commits, pull requests, issues…), calcula tu experiencia, tu racha y tus días activos, y dibuja un SVG animado: un pequeño brote de musgo con ojos que empieza siendo una semilla y acaba convertido en un árbol ancestral (o, si prefieres, un hongo que crece de espora a corro de setas). Cambia de ánimo según lo que hayas hecho estos días y desbloquea accesorios. Es un diseño original, dibujado píxel a píxel.

- Animación con CSS dentro del SVG (GitHub no ejecuta JavaScript en los README) y respetuosa con `prefers-reduced-motion`.
- Determinista: mismos datos, mismo SVG, así que tu repositorio solo recibe un commit cuando la criatura cambia de verdad.
- Tema claro y oscuro, estética «papel y píxel».

## Pruébalo con tu usuario (generador en vivo)

En la [web del proyecto](https://bertmarti.github.io/commitling/#generador) escribes tu usuario de GitHub, eliges especie y tema, pulsas **Dibujar** y ves tu criatura al momento, sin instalar nada. Puedes **descargar el SVG** y copiar el **workflow ya relleno con tu usuario**.

- **¿Sin usuario a mano? «Ver demo: míralo crecer».** Un timelapse de unos 15 segundos (90 días ficticios de `octoexample`) en el que la criatura nace como semilla, pasa por las cinco fases y los cuatro ánimos y desbloquea gorro, bufanda y flor, redibujada en vivo por el mismo código. No escribes nada y no se pide nada a GitHub. Tiene **Pausa** y **Detener**, un control deslizante para recorrer los días y no arranca sola; con `prefers-reduced-motion` no hay timelapse, solo el deslizante.
- Se dibuja **en tu navegador** con el mismo código Go que la CLI y la Action, compilado a WebAssembly (`GOOS=js GOARCH=wasm`, `syscall/js`): para los mismos eventos y la misma fecha, el SVG es idéntico byte a byte (lo comprueba un test).
- Lo único que sale de tu ordenador es la petición pública a `https://api.github.com/users/<usuario>/events/public` (hasta 3 páginas de 100 eventos, sin token). GitHub deja **60 peticiones por hora y por conexión** sin iniciar sesión; si se agotan, la web dice cuántos minutos faltan. La Action, con el token de tu workflow, no tiene ese límite.
- El WebAssembly (unos 4,6 MB, 1,3 MB comprimido) **solo se descarga cuando interactúas con el formulario o pulsas «Ver demo»**, no al abrir la página. El `.wasm` y `wasm_exec.js` los genera el workflow de despliegue; no están en el repositorio.
- Necesita JavaScript y un navegador con WebAssembly. Sin ellos, la galería sigue funcionando.

## Úsalo en tu perfil

1. Abre (o crea) tu repositorio de perfil: el que se llama igual que tu usuario, `tu-usuario/tu-usuario`.
2. Crea `.github/workflows/commitling.yml` con este contenido:

```yaml
name: commitling

on:
  schedule:
    - cron: "23 5 * * *" # cada día a las 05:23 UTC
  workflow_dispatch:

permissions:
  contents: write

jobs:
  commitling:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v7
      - uses: BertMarti/commitling@v1
        with:
          out: commitling.svg
      - name: Guardar el SVG si ha cambiado
        run: |
          git add commitling.svg
          if git diff --cached --quiet; then
            echo "Sin cambios."
            exit 0
          fi
          git config user.name "github-actions[bot]"
          git config user.email "41898282+github-actions[bot]@users.noreply.github.com"
          git commit -m "chore: actualiza commitling"
          git push
```

3. Añade la criatura a tu `README.md`:

```markdown
![commitling](./commitling.svg)
```

4. En la pestaña **Actions**, elige «commitling» y pulsa **Run workflow**. A partir de ahí se actualiza sola cada día.

### GitHub Marketplace y versiones

commitling está preparada para el [GitHub Marketplace](https://github.com/marketplace?type=actions) como Action (la publica el responsable del proyecto al crear la versión 1). Una vez publicada se encuentra buscando «commitling»; el uso es el del ejemplo de arriba:

- `BertMarti/commitling@v1` sigue la versión mayor 1: recibes correcciones y mejoras compatibles sin tocar tu workflow (la etiqueta `v1` se mueve con cada versión 1.x).
- `BertMarti/commitling@v1.0.0` fija una versión exacta, y una huella de commit (`@<sha>`) es lo más reproducible.
- `@main` funciona, pero es la rama de desarrollo y puede cambiar en cualquier momento. Si tu workflow aún termina en `@main`, cámbialo por `@v1` (pasos en la [guía de uso](docs/USO.md#6-versiones-de-la-action-y-cómo-actualizar)).

La Action declara su nombre, descripción y `branding` (icono `feather`, color `green`) en `action.yml`, que es lo que el Marketplace muestra.

### Entradas de la Action

| Entrada | Por defecto | Qué hace |
|---|---|---|
| `user` | `${{ github.repository_owner }}` | Usuario de GitHub cuya actividad se dibuja |
| `out` | `commitling.svg` | Ruta del SVG, relativa al repositorio |
| `theme` | `light` | `light` (papel) o `dark` (tinta) |
| `species` | `moss` | Especie: `moss` (brote de musgo) o `mushroom` (hongo); un valor desconocido hace fallar el paso |
| `token` | `${{ github.token }}` | Token para la API; basta con el del propio workflow |
| `fixture` | vacío | Archivo JSON de eventos para probar sin red |
| `keep-on-error` | `true` | Ante una caída transitoria de la API de GitHub (red, errores 5xx o límite de peticiones), si el SVG de `out` ya existe lo conserva y termina con un aviso en vez de fallar; `false` para que falle siempre. Un token caducado (401) o un usuario inexistente (404) siguen fallando |

La Action tiene una salida, `path` (ruta absoluta del SVG generado). Solo genera el archivo; el commit lo hace tu workflow (como en el ejemplo), así controlas cuándo y cómo se guarda.

### Especies

Hay dos especies, las dos de diseño propio y con las mismas reglas (fases, ánimos y accesorios). La predeterminada es el **brote de musgo**; con `species: mushroom` (o `--species mushroom` en la CLI) sale el **hongo**, una seta pequeña que crece de espora a corro de setas.

| Fase | Desde | Brote de musgo (`moss`) | Hongo (`mushroom`) |
|---|---|---|---|
| 1 | 0 XP | Semilla | Espora |
| 2 | 100 XP | Brote | Botón |
| 3 | 400 XP | Retoño | Seta |
| 4 | 1.000 XP | Arbusto | Seta grande |
| 5 | 2.500 XP | Árbol ancestral | Corro de setas |

<table>
  <tr>
    <td align="center"><img alt="Árbol ancestral radiante (brote de musgo)" src="https://bertmarti.github.io/commitling/svg/ancient-radiant.svg" width="380"><br><sub><code>moss</code>: árbol ancestral, radiante</sub></td>
    <td align="center"><img alt="Corro de setas radiante (hongo)" src="https://bertmarti.github.io/commitling/svg/mushroom-ancient-radiant.svg" width="380"><br><sub><code>mushroom</code>: corro de setas, radiante</sub></td>
  </tr>
</table>

Capturas de la [galería](https://bertmarti.github.io/commitling/), generada por el propio binario (`commitling gallery`). `species` acepta también `musgo`, `hongo` y `seta`, sin distinguir mayúsculas.

### Tema oscuro (opcional)

Añade un segundo paso con `theme: dark` y `out: commitling-dark.svg`, súmalo al `git add` y usa esto en el README para que GitHub elija según el tema de quien lo mira:

```html
<picture>
  <source media="(prefers-color-scheme: dark)" srcset="./commitling-dark.svg">
  <img alt="commitling" src="./commitling.svg">
</picture>
```

## Reglas

Todo se calcula con los eventos públicos de los últimos 90 días (lo que ofrece la API, hasta 300 eventos). Los días se cuentan en **UTC**.

**Experiencia (XP)**

| Actividad | XP |
|---|---|
| Commit (cada commit de un push) | 10 |
| Pull request abierta | 25 |
| Issue abierta | 5 |
| Cualquier otro evento público (estrellas, forks, comentarios, revisiones…) | 2 |

**Fases**

| Fase | Desde |
|---|---|
| Semilla | 0 XP |
| Brote | 100 XP |
| Retoño | 400 XP |
| Arbusto | 1.000 XP |
| Árbol ancestral | 2.500 XP |

**Ánimos**

| Ánimo | Cuándo | Cómo se nota |
|---|---|---|
| Durmiendo | 7 días o más sin actividad (o ninguna) | Ojos cerrados, respiración lenta y «zzz» |
| Aburrido | De 3 a 6 días sin actividad | Párpados caídos y boca recta |
| Contento | Actividad en los últimos 3 días | Sonrisa, mejillas y parpadeo |
| Radiante | Racha de 5 días seguidos o más | Ojos felices, saltitos y destellos |

**Accesorios**

| Accesorio | Se desbloquea con |
|---|---|
| Gorro | Más de 30 días activos en los últimos 90 |
| Bufanda | Racha de 10 días o más |
| Flor | 5 repositorios distintos o más con commits, PR o issues |

Reintentos: si la API de GitHub falla (500, 502, 503 o 504), limita las peticiones (429, o 403 con cabeceras de límite) o no responde, commitling repite la petición hasta 3 veces, con esperas de 1, 2 y 4 s (o las que indique GitHub, hasta 30 s). Si pide esperar más de 30 s o se agotan los reintentos, se rinde con un error claro; 404, 422 y otros 4xx no se reintentan.

Caídas largas de la API: cuando se agotan los reintentos por una caída **transitoria** (errores de red, 5xx o límite de peticiones: 429, o 403 con cabeceras de límite), la Action **conserva el SVG anterior** (el que tu workflow ya tiene en el repositorio tras el `checkout`) y termina en verde con un aviso (`::warning::`) en el log, en vez de fallar y dejar el perfil sin actualizar. Es la entrada `keep-on-error` (activada por defecto) o `--keep-on-error` en la CLI. Lo **permanente** sigue fallando aunque haya SVG previo, para que lo veas: un token caducado o inválido (401), un usuario inexistente (404), un 422 y un 403 sin cabeceras de límite (permisos). Tampoco se conserva nada sin archivo previo (o con uno vacío o cortado), y un `fixture` inexistente o un `species` inválido siguen siendo errores.

Detalles: la racha cuenta días seguidos con actividad y se mantiene hasta el final del día siguiente (no se rompe por la mañana antes de tu primer commit). Si GitHub no indica cuántos commits lleva un push, cuenta como uno.

## CLI

Requiere Go 1.27. Sin dependencias externas.

```sh
# Tu criatura (GITHUB_TOKEN es opcional, pero sube el límite de peticiones)
go run ./cmd/commitling render --user BertMarti --out out/commitling.svg

# Sin red, con datos de ejemplo de un usuario ficticio
go run ./cmd/commitling render --fixture testdata/events.json --out out/commitling.svg

# Tema oscuro y fecha de referencia fija, a la salida estándar
go run ./cmd/commitling render --fixture testdata/events.json --theme dark --now 2026-10-01T12:00:00Z --out -

# La otra especie (moss por defecto, o mushroom)
go run ./cmd/commitling render --fixture testdata/events.json --species mushroom --out out/hongo.svg

# Si la API tiene una caída transitoria y out/commitling.svg ya existe, lo conserva (aviso, código de salida 0)
go run ./cmd/commitling render --user BertMarti --keep-on-error --out out/commitling.svg

# Web estática con todas las fases, ánimos y accesorios
go run ./cmd/commitling gallery --out site/

# Imagen PNG 1200x630 para las vistas previas en redes (Open Graph)
go run ./cmd/commitling og --out out/og.png

go run ./cmd/commitling version
```

Con `--fixture` y sin `--now`, la fecha de referencia es la del último evento del archivo, para que el resultado no cambie de un día a otro.

## Privacidad

commitling solo usa la API **pública** de eventos de GitHub: lo mismo que cualquiera puede ver en tu perfil. No lee repositorios privados, no guarda nada fuera de tu repositorio y no necesita más permisos que el `GITHUB_TOKEN` de tu propio workflow. Los datos de ejemplo de `testdata/` son inventados.

## Stack

- Go 1.27, solo biblioteca estándar (`net/http`, `encoding/json`, `html/template`; `encoding/xml` en los tests).
- SVG escrito a mano, con animación CSS y píxeles nítidos (`shape-rendering="crispEdges"`).
- WebAssembly (`GOOS=js GOARCH=wasm`, `syscall/js`) para el generador en vivo de la web, sin JavaScript de terceros.
- GitHub Action compuesta (`action.yml`), CI en Ubuntu y Windows y despliegue a GitHub Pages.

## Estructura

```
cmd/commitling/      CLI (render, gallery, og, version)
cmd/wasm/            capa fina de syscall/js para el navegador (solo con GOOS=js)
internal/generate/   eventos a SVG, workflow relleno, mensajes de error y la demo sintética (`Demo`); lo usan la CLI y el wasm
internal/events/     parser de eventos y actividades (sin red)
internal/github/     cliente de la API de eventos públicos (con reintentos)
internal/stats/      XP, racha, días activos y repos (puro, con «ahora» inyectable)
internal/creature/   especies, fases, ánimos, accesorios y mapas de píxeles
internal/render/     generación del SVG
internal/gallery/    web estática de la galería
internal/og/         imagen PNG 1200x630 para Open Graph (solo biblioteca estándar)
testdata/            eventos de ejemplo del usuario ficticio «octoexample»
action.yml           la GitHub Action
```

## Desarrollo

```sh
gofmt -l .      # debe salir vacío
go vet ./...
go test ./...
```

El generador en vivo (el `.wasm` y `wasm_exec.js` no se suben al repositorio):

```sh
go run ./cmd/commitling gallery --out site
GOOS=js GOARCH=wasm go build -ldflags="-s -w" -o site/commitling.wasm ./cmd/wasm
cp "$(go env GOROOT)/lib/wasm/wasm_exec.js" site/
python -m http.server 8080 --directory site   # http://localhost:8080/
```

## Documentación

- [Guía de uso](docs/USO.md): instalación paso a paso, tema oscuro, elección de especie, versiones de la Action, comando `og`, reglas con ejemplos, preguntas frecuentes y solución de problemas.
- [Contribuir](CONTRIBUTING.md): requisitos, comandos, flujo por issues, ramas, commits y cómo añadir un accesorio, una fase o una especie.
- [Registro de cambios](CHANGELOG.md): qué trae cada versión.

## Cómo se ha hecho

commitling se ha construido con un equipo de agentes de IA de Claude Code, coordinados por un agente lead y guiados por issues del hito (una rama y un PR por issue):

- **lead** y **builder** de la v0.1.0 (MVP): Claude Opus.
- **qa**, **docs** y todo el trabajo de la v0.2.0 (reintentos, imagen Open Graph, hongo, Action para v1, revisión y documentación): Claude Sonnet.
- **OpenCode** estaba previsto para la documentación, pero no pudo ejecutarse en modo autónomo (el sistema de permisos no lo permite), así que la escribió Claude Code.
- **Alberto** supervisa y fusiona: nadie más hace commit ni push a `main`, ni crea etiquetas o releases.

Las reglas del equipo están en [AGENTS.md](AGENTS.md) y el estado del proyecto en [MEMORY.md](MEMORY.md).

## Licencia

[MIT](LICENSE).
