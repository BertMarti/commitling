# PLAN.md · commitling

Hoja de ruta priorizada. Fechada el 24/10/2026 y actualizada el 31/10/2026 (versión 1.1.0). Las etiquetas, releases y mover `v1` los hace Alberto (o el orquestador en su nombre).

## Hecho: versión 1.0.0 (26/10/2026, de v0.7.0 a v1.0.0; la 0.8.0 nunca se publicó)
- **Fase 1 (24/10, «Blindada»)**: CSP estricta en la web y sin scripts en línea, CI con job «seguridad» (govulncheck, staticcheck, gitleaks), permisos mínimos, dependabot, `SECURITY.md`, `go 1.27.3` (la Action compilaba con Go 1.27.0, con 11 vulnerabilidades), escritura atómica del SVG, README para personas usuarias y `docs/DESARROLLO.md`.
- **Fase 2 (24/10, tarde)**: generador del workflow en la web (6 pasos y «Si algo falla»), `commitling workflow`, errores de la API en español y accionables con anotación `::error`, token enmascarado, CSP sin `'unsafe-inline'` en estilos.
- **Fase 3 (25-26/10)**: tercera especie **cactus**, selector de tema de la web, Trusted Types, acciones por SHA y binario precompilado preparado.
- **Fase 4 (26/10)**: README, `docs/USO.md`, CHANGELOG y web sin contradicciones.
- **Fase 5 (26/10, rama `v1-2026-10-26`)**: versión 1.0.0 en la CLI, la web y la documentación; CHANGELOG `[1.0.0]`; `release/checksums.txt` de la 1.0.0 (binario precompilado activo al publicar); el job `action` del CI fuerza la ruta de `go run` y el job `binario` prueba también la release real; fuera la falsa promesa del Marketplace y el aviso obsoleto de «`v1` aún no existe» en `docs/USO.md`.

## Hecho: versión 1.1.0 (31/10/2026, rama `v1.1-2026-10-31`)
- **Cuarta especie: la calabaza** (`species: pumpkin`), de la lista «Después de la 1.0»: Pepita, Plántula, Calabacita, Calabaza y Calabaza gigante, en la Action, la CLI, la galería, los generadores de la web y la imagen OG (dos especies por fila). Musgo, hongo y cactus byte a byte iguales.
- Mantenimiento: Go 1.27.3 sigue siendo el último parche (sin cambios en `go.mod`), govulncheck limpio, 0 alertas y 0 PRs de Dependabot.

## Siguiente (por orden)
0. **Orquestador/Alberto (1.1.0)**: fusionar el PR de `v1.1-2026-10-31`, publicar `v1.1.0` sobre el SHA del merge, esperar a que «Release» suba los binarios, mover `v1` y fusionar el PR de `v1.1-tras-publicar` (pasos exactos en el informe de la fase 6).
1. **Orquestador/Alberto**: fusionar el PR de `v1-2026-10-26`, publicar la release `v1.0.0` desde ese commit (título y notas en el informe de la fase 5) y mover `v1` a ella (orden exacta en el README, «Lo que tienes que hacer tú»). Comprobar que el workflow «Release» sube los 6 binarios y `checksums.txt`.
2. ~~**Orquestador**: PR de la rama `v1-tras-publicar`~~ Hecho: avisos de «aún no publicada» fuera, ejemplos `@v1.0.0` y `release/checksums.txt` borrado de `main`.
3. Tras el despliegue: abrir https://bertmarti.github.io/commitling/ con la consola y comprobar 0 violaciones de CSP/Trusted Types (demo, generador con un usuario real, selector de tema). La primera ejecución del perfil de BertMarti tras mover `v1` no debe tener el paso de `setup-go` (usa el binario).
4. Subir la línea `go` de `go.mod` con cada parche de Go (y el mínimo de `TestGoModPinsAPatchedGo`); govulncheck en el CI avisará.

## Después de la 1.0
- **Alberto**: publicar la Action en el GitHub Marketplace (hay que aceptar el acuerdo de desarrollador en la web al editar la release: casilla «Publish this Action to the GitHub Marketplace»). `action.yml` ya cumple los requisitos (`TestActionMetadataIsMarketplaceReady`).
- ~~Una cuarta especie~~ Hecho en la 1.1.0 (calabaza).
- Modo «año en píxeles» con la actividad de todo el año (requiere otra fuente de datos: la API de eventos solo da 90 días; probablemente GraphQL con token).
- Atestaciones de procedencia de los binarios (`actions/attest-build-provenance`) como capa extra para quien quiera verificarlos a mano; la Action seguiría confiando solo en `release/checksums.txt`.
