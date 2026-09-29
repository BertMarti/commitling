# Contribuir a commitling

Gracias por querer ayudar. Esta guía es corta; las reglas del equipo de agentes están en [AGENTS.md](AGENTS.md) y el estado del proyecto en [MEMORY.md](MEMORY.md) (léelo antes de empezar y actualízalo al terminar).

## Requisitos

- Go 1.27. Solo biblioteca estándar: no añadas dependencias sin justificarlo en «Decisiones» de `MEMORY.md`.
- Git y, si quieres abrir PR desde la terminal, la CLI `gh`.
- La interfaz y la documentación van en español; los nombres del código, en inglés.

## Comandos

```sh
gofmt -l .      # debe salir vacío
go vet ./...
go test ./...

# Dibuja una criatura sin red (usuario ficticio «octoexample»)
go run ./cmd/commitling render --fixture testdata/events.json --out out/commitling.svg

# Con tu usuario real (GITHUB_TOKEN es opcional, sube el límite de peticiones)
go run ./cmd/commitling render --user tu-usuario --out out/commitling.svg

# Web con todas las fases, ánimos y accesorios
go run ./cmd/commitling gallery --out site/
```

Las carpetas `out/` y `site/` están ignoradas por git. En Windows, si el Control de aplicaciones bloquea los binarios de Go, usa una carpeta temporal dentro del proyecto: `export GOTMPDIR="$PWD/out/gotmp"` (crea la carpeta antes).

## Ramas y pull requests

- Nunca hagas commit directo a `main`. Trabaja en una rama (`agent/<nombre>` para los agentes, `feat/...` o `fix/...` para personas) y abre un pull request.
- Antes de abrirlo: `gofmt`, `go vet` y `go test` en verde. El CI (Ubuntu y Windows) debe estar en verde.
- Rellena la plantilla del PR: qué cambia, cómo se ha verificado, agente y rama, y la checklist.
- Solo el responsable del proyecto fusiona.

## Commits

Commits convencionales y en español, pequeños y con sentido propio: `feat:`, `fix:`, `test:`, `docs:`, `ci:`, `chore:`, `refactor:`.

```
fix(render): el texto de la tarjeta ya no se sale del panel
```

No subas claves, tokens, `.env` ni datos personales: el repositorio es público.

## Cómo añadir un accesorio

1. `internal/creature/creature.go`: añade el campo a `Accessories`, su umbral (constante junto a `HatActiveDays90`, `ScarfStreak`, `FlowerRepos`), la condición en `AccessoriesFor` y su nombre en español en `Names`.
2. `internal/creature/sprites.go`: dibuja su mapa de píxeles (como `hatArt` o `flowerArt`), añade su posición en cada fase (`stageArt`), estámpalo en `Draw` y, si tapa la silueta, inclúyelo en `touched`.
3. `internal/gallery/gallery.go`: añade una tarjeta a `extras` para que salga en la galería.
4. Actualiza la tabla de accesorios del `README.md` y de `docs/USO.md`, y la web si la regla se menciona en la plantilla `internal/gallery/index.html.tmpl`.
5. Añade tests (umbrales justo en el límite, y que el SVG resultante sigue cabiendo en la tarjeta).

## Cómo añadir una fase

1. `internal/creature/creature.go`: añade la constante de `Stage`, su XP mínima, y entradas en `Stages`, `stageXP`, `stageNames` y `stageSlugs` (mantén el orden de menor a mayor).
2. `internal/creature/sprites.go`: añade su entrada en `art` con el mapa 16×16 y las posiciones de cara, gorro, bufanda y flor.
3. La galería genera una fila por fase automáticamente (`internal/gallery/gallery.go`), pero comprueba que `Sample` produce una XP intermedia coherente.
4. Revisa `internal/render/render.go` (la tarjeta dice «fase N de M» y calcula la barra con `creature.Progress`).
5. Actualiza las tablas de fases del `README.md` y de `docs/USO.md`, y los tests de fases y del SVG.

Los ánimos se tocan igual: `Mood`, `MoodFor`, `eyes` y `mouths` en `sprites.go`, y sus animaciones en `internal/render/render.go`.

## Regla de diseño: personaje original

La criatura es **diseño propio**: un brote de musgo con ojos, dibujado píxel a píxel. Está prohibido imitar criaturas de videojuegos, anime o marcas, y no se aceptan sprites, nombres ni siluetas tomados de otras obras.

Respeta también el estilo «papel y píxel» de [AGENTS.md](AGENTS.md): paleta de la criatura limitada a 5 colores (`#2b2724`, `#f3efe6`, `#7fb069`, `#e6aa68`, `#ca3c25`), píxeles nítidos, sin degradados ni sombras, animaciones suaves y respetando `prefers-reduced-motion`.

## Si cambias las reglas

Las reglas (XP, fases, ánimos, accesorios) son deterministas y están documentadas en tres sitios que deben decir lo mismo: el código (`internal/stats`, `internal/creature`), el `README.md` y `docs/USO.md`. Además, el workflow del README debe ser idéntico a `gallery.WorkflowSnippet` en `internal/gallery/gallery.go`; un test lo comprueba, así que si cambias uno, cambia el otro.
