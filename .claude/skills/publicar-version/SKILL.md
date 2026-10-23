---
name: publicar-version
description: Prepara una versión nueva de commitling en una rama (CHANGELOG, versión de la CLI, docs y notas de release) sin crear etiquetas. Úsala al cerrar un hito o una tanda de mejoras.
---

# Preparar una versión de commitling

Las etiquetas (`vX.Y.Z`), las releases y mover la etiqueta móvil `v1` las hace **solo Alberto** tras fusionar. Tú dejas la rama lista.

1. **Elige el número** (semver): corrección → parche; algo nuevo compatible → menor. Si cambia la interfaz de `action.yml` (entradas, salida, archivos) es mayor y eso exige `v2`: no lo hagas sin que Alberto lo pida.
2. **Versión de la CLI**: `var version` en `cmd/commitling/main.go` y su test en `cmd/commitling/main_test.go` (`TestVersionAndGallery`).
3. **CHANGELOG.md**: mueve lo de «Sin publicar» a `## [X.Y.Z] - AAAA-MM-DD` con Añadido / Cambiado / Corregido en español y actualiza los enlaces de comparación del final (`[Sin publicar]` → `vX.Y.Z...HEAD` y una línea `[X.Y.Z]`).
4. **Docs**: README (solo lo de usuario), `docs/USO.md`, `docs/DESARROLLO.md`, `AGENTS.md` si cambian reglas, `MEMORY.md` (estado, decisiones con fecha, registro de sesiones) y `PLAN.md`.
5. **Cabecera del README** (solo si cambian los sprites): genera la galería (`go run ./cmd/commitling gallery --out out/site`) y rehaz `docs/img/banner.svg` copiando el grupo `<g class="bob">` de `svg/{seed,sprout,sapling,shrub}-happy.svg` y `svg/ancient-radiant.svg`, cada uno dentro de `<g transform="translate(x y) scale(0.6)">` (la animación va en el grupo interior, nunca en el que lleva `transform`, o la animación CSS lo anula).
6. **Comprueba**: `gofmt -l .`, `go vet ./...`, `GOFLAGS=-p=2 go test ./...`, el subagente `guardian-compatibilidad` y, si tocaste la web, la skill `probar-web`.
7. **Notas para Alberto** en el PR: «tras fusionar: release `vX.Y.Z` con las notas del CHANGELOG y, si es compatible, mover `v1` a ella».
