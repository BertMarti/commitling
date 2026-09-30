<div align="center">

![commitling](https://bertmarti.github.io/commitling/live/BertMarti.svg)

# commitling

**Una mascota pixel-art que vive en el README de tu perfil de GitHub y crece con tus commits.**

[![CI](https://github.com/BertMarti/commitling/actions/workflows/ci.yml/badge.svg)](https://github.com/BertMarti/commitling/actions/workflows/ci.yml)
[![Pages](https://github.com/BertMarti/commitling/actions/workflows/deploy.yml/badge.svg)](https://github.com/BertMarti/commitling/actions/workflows/deploy.yml)
[![Licencia MIT](https://img.shields.io/badge/licencia-MIT-7fb069)](LICENSE)

[Galería de fases y ánimos](https://bertmarti.github.io/commitling/) · [Instalar](#úsalo-en-tu-perfil) · [Reglas](#reglas)

</div>

Una GitHub Action lee tu actividad **pública** (commits, pull requests, issues…), calcula tu experiencia, tu racha y tus días activos, y dibuja un SVG animado: un pequeño brote de musgo con ojos que empieza siendo una semilla y acaba convertido en un árbol ancestral. Cambia de ánimo según lo que hayas hecho estos días y desbloquea accesorios. Es un diseño original, dibujado píxel a píxel.

- Animación con CSS dentro del SVG (GitHub no ejecuta JavaScript en los README) y respetuosa con `prefers-reduced-motion`.
- Determinista: mismos datos, mismo SVG, así que tu repositorio solo recibe un commit cuando la criatura cambia de verdad.
- Tema claro y oscuro, estética «papel y píxel».

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
      - uses: BertMarti/commitling@main
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

### Entradas de la Action

| Entrada | Por defecto | Qué hace |
|---|---|---|
| `user` | `${{ github.repository_owner }}` | Usuario de GitHub cuya actividad se dibuja |
| `out` | `commitling.svg` | Ruta del SVG, relativa al repositorio |
| `theme` | `light` | `light` (papel) o `dark` (tinta) |
| `token` | `${{ github.token }}` | Token para la API; basta con el del propio workflow |
| `fixture` | vacío | Archivo JSON de eventos para probar sin red |

La Action solo genera el archivo; el commit lo hace tu workflow (como en el ejemplo), así controlas cuándo y cómo se guarda.

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
- GitHub Action compuesta (`action.yml`), CI en Ubuntu y Windows y despliegue a GitHub Pages.

## Estructura

```
cmd/commitling/      CLI (render, gallery, og, version)
internal/github/     cliente de la API de eventos públicos y parser
internal/stats/      XP, racha, días activos y repos (puro, con «ahora» inyectable)
internal/creature/   fases, ánimos, accesorios y mapas de píxeles
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

## Documentación

- [Guía de uso](docs/USO.md): instalación paso a paso, tema oscuro, reglas con ejemplos, preguntas frecuentes y solución de problemas.
- [Contribuir](CONTRIBUTING.md): requisitos, comandos, ramas, commits y cómo añadir un accesorio o una fase.

## Cómo se ha hecho

commitling se ha construido con un equipo de agentes de IA de Claude Code: builder (Opus), qa y docs (Sonnet), coordinados por un agente lead. Cada uno trabajó en su rama y todo entra en `main` mediante pull request, con la supervisión y la fusión de Alberto. OpenCode estaba previsto para la documentación, pero no pudo ejecutarse en modo autónomo. Las reglas del equipo están en [AGENTS.md](AGENTS.md) y el estado del proyecto en [MEMORY.md](MEMORY.md).

## Licencia

[MIT](LICENSE).
