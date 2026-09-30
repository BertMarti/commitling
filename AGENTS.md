# AGENTS.md · commitling

## Qué es
Una mascota pixel-art **original** que vive en el README de tu perfil de GitHub. Una GitHub Action lee tu actividad pública (commits, racha, días activos) y dibuja un SVG animado: la criatura crece por fases con tu experiencia, cambia de ánimo según tu actividad reciente y desbloquea accesorios.

## Tecnología (propia de este proyecto)
- **Go 1.27**, solo biblioteca estándar (`net/http`, `encoding/json`, `encoding/xml` para validar).
- SVG generado a mano con animación **CSS dentro del SVG** (GitHub no ejecuta JavaScript en los README).
- Tests con `go test`; formato con `gofmt` y análisis con `go vet`.
- **GitHub Action compuesta** (`action.yml`) reutilizable desde cualquier repositorio.
- Web del proyecto en **GitHub Pages**: galería de fases y ánimos generada por el propio binario (`commitling gallery`) más una criatura en vivo de BertMarti regenerada a diario.

## Comandos
- Tests: `go test ./...`
- Análisis: `go vet ./...` y `gofmt -l .` (debe salir vacío)
- Generar: `go run ./cmd/commitling render --user BertMarti --out out/commitling.svg`
- Sin red: `go run ./cmd/commitling render --fixture testdata/events.json --out out/commitling.svg`
- Galería: `go run ./cmd/commitling gallery --out site/`

## Estructura
- `cmd/commitling/` CLI.
- `internal/github/` cliente de la API pública de eventos (token opcional por `GITHUB_TOKEN`).
- `internal/stats/` cálculo de experiencia, racha y días activos.
- `internal/creature/` fases, ánimos, accesorios y mapas de píxeles (cuadrículas ASCII).
- `internal/render/` generación del SVG.
- `testdata/` datos de ejemplo sin información personal real.

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
