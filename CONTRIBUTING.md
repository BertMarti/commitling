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

# Otra especie (moss por defecto, o mushroom)
go run ./cmd/commitling render --fixture testdata/events.json --species mushroom --out out/hongo.svg

# Web con todas las fases, ánimos y accesorios (y og.png)
go run ./cmd/commitling gallery --out site/

# Imagen PNG 1200x630 para Open Graph
go run ./cmd/commitling og --out out/og.png
```

Las carpetas `out/` y `site/` están ignoradas por git. En Windows, si el Control de aplicaciones bloquea los binarios de Go, usa una carpeta temporal dentro del proyecto: `export GOTMPDIR="$PWD/out/gotmp"` (crea la carpeta antes).

## Flujo por issues

Desde la v0.2.0 el trabajo se guía por **issues del hito**. Cada issue lleva la etiqueta del rol que la ejecuta (`agent:builder`, `agent:qa` o `agent:docs`) y sus criterios de aceptación son el contrato: la issue está terminada cuando todas las casillas se cumplen.

1. Elige una issue abierta de tu rol (o abre una con criterios de aceptación y la etiqueta del hito).
2. **Una rama y un PR por issue.** Rama `agent/<rol>/<n>-<slug>` (por ejemplo `agent/docs/9-documentacion-v0.2`); las personas pueden usar `feat/...` o `fix/...`.
3. Si varias issues dependen unas de otras, cada rama puede partir de la anterior, pero **todos los PR van contra `main`** (`gh pr create --base main`), se fusionan en orden y el PR lo dice («Se fusiona después de #N»).
4. La descripción del PR lleva `Closes #<n>`, qué cambia, cómo se ha verificado y el orden de fusión si aplica.
5. Al terminar, actualiza `MEMORY.md` (estado, decisiones con fecha, siguiente paso y una línea en «Registro de sesiones») y, si el cambio se nota, `CHANGELOG.md`.
6. Solo el responsable del proyecto fusiona y crea las etiquetas y las releases.

## Ramas y pull requests

- Nunca hagas commit directo a `main`. Trabaja en la rama de tu issue y abre un pull request.
- Antes de abrirlo: `gofmt`, `go vet` y `go test` en verde. El CI (Ubuntu y Windows) debe estar en verde.
- Rellena la plantilla del PR: qué cambia, cómo se ha verificado, agente y rama, y la checklist.
- Solo el responsable del proyecto fusiona.

## Commits

Commits convencionales y en español, pequeños y con sentido propio: `feat:`, `fix:`, `test:`, `docs:`, `ci:`, `chore:`, `refactor:`.

```
fix(render): el texto de la tarjeta ya no se sale del panel
```

Los agentes terminan el mensaje con una línea en blanco y luego `Agente: <rol> (Claude Code · Sonnet)` y la línea `Co-Authored-By`.

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

## Cómo añadir una especie

Archivos exactos, en este orden:

1. `internal/creature/creature.go`: añade la constante de `Species` **después** de `Mushroom` (así el valor cero sigue siendo el brote de musgo) y una entrada, en el mismo orden, en cada uno de estos cuatro sitios: `AllSpecies`, `speciesNames` (nombre en español), `speciesSlugs` (ASCII: es lo que se escribe en `species:` y en `--species`) y `speciesStageNames` (los nombres en español de sus cinco fases). Si quieres alias, añádelos al `switch` de `SpeciesByName`.
2. Un archivo nuevo en `internal/creature/` (como `mushroom.go`) con su `[...]stageArt` de cinco mapas 16×16 (`body`, `face`, `hat`, `scarf`, `flower`), y engánchalo en `artFor` de `internal/creature/sprites.go` (hoy es un `if sp == Mushroom`; con tres especies pasa a `switch`). La cara (8×4) va sobre un relleno, la fila de la bufanda debe ser un tramo continuo y el gorro y la flor no pueden tapar la cara. Usa solo la paleta de 5 colores y no copies mapas de otra especie ni de ninguna obra ajena.
3. `cmd/commitling/main.go`: el mensaje «especie no válida … (usa moss o mushroom)» y la ayuda de `--species` (en el comentario del archivo y en `usage`) listan los valores; añade el nuevo. Igual el texto de la entrada `species` de `action.yml`.
4. La galería (`internal/gallery`), la tarjeta SVG y la imagen Open Graph (`internal/og`) recorren `AllSpecies`, así que la nueva especie sale sin más cambios: genera la web y la imagen y revísalas en claro y oscuro.
5. Actualiza las tablas de especies del `README.md` y de `docs/USO.md` (valores de `species` y nombres de las fases), el `CHANGELOG.md` y `MEMORY.md`.

Tests que deben pasar (y qué tocar en ellos):

- `internal/creature`: `TestSpeciesNamesAndSlugs` (fija `len(AllSpecies) != 2`: sube el número; comprueba nombres, slugs únicos y cinco nombres de fase), `TestSpeciesByName` (añade los alias), `TestAnchorsFitTheBody` (cara sobre relleno, bufanda continua, gorro y flor sin tapar la cara, para todas las especies y fases), `TestDrawUsesPalette` (solo colores de la paleta), `TestSpeciesDrawDifferently` (cada especie se dibuja distinta y de forma determinista, y cada fase distinta de las demás) y un test como `TestMushroomMapsAreNotTheMossOnes` para tu especie.
- `internal/render`: la tarjeta es XML válido, determinista y el texto cabe, para todas las especies (`TestMushroomCard` es el modelo).
- `internal/gallery`: `TestBuildShowsBothSpecies`, `TestBuildIsDeterministic`, `TestGalleryAltsAreUnique` y `TestPageHasSkipLinkAndSpeciesNav`.
- `internal/og`: las cinco fases de cada especie aparecen (`TestStageRowsShowAllStages`) y todo cabe en el marco.
- `cmd/commitling`: la especie inválida sigue fallando antes de tocar la red y el mensaje lista los valores.
- `internal/gallery/action_test.go`: `action.yml` y la tabla de entradas del README siguen coincidiendo.
- `.github/workflows/action-test.yml` prueba `species: mushroom`; añade un paso para la nueva.

## Regla de diseño: personaje original

Las criaturas son **diseño propio**: un brote de musgo con ojos y una seta pequeña con ojos, dibujados píxel a píxel. Está prohibido imitar criaturas de videojuegos, anime o marcas, y no se aceptan sprites, nombres ni siluetas tomados de otras obras.

Respeta también el estilo «papel y píxel» de [AGENTS.md](AGENTS.md): paleta de la criatura limitada a 5 colores (`#2b2724`, `#f3efe6`, `#7fb069`, `#e6aa68`, `#ca3c25`), píxeles nítidos, sin degradados ni sombras, animaciones suaves y respetando `prefers-reduced-motion`.

## Si cambias las reglas

Las reglas (XP, fases, ánimos, accesorios) son deterministas y están documentadas en tres sitios que deben decir lo mismo: el código (`internal/stats`, `internal/creature`), el `README.md` y `docs/USO.md`. Anota también el cambio en `CHANGELOG.md`. Además, el workflow del README debe ser idéntico a `gallery.WorkflowSnippet` en `internal/gallery/gallery.go`; un test lo comprueba, así que si cambias uno, cambia el otro.
