# PLAN.md · commitling

Hoja de ruta priorizada. Fechada el 24/10/2026. Las etiquetas, releases y mover `v1` los hace Alberto.

## Hecho hoy (v0.8.0 «Blindada», rama `mejoras-2026-10-24`)
- CSP estricta en la web y sin scripts en línea (`page.js`), `theme-color`.
- CI con job «seguridad» (govulncheck, staticcheck, gitleaks), permisos mínimos por job, `persist-credentials: false`, dependabot semanal agrupado y `SECURITY.md`.
- `go 1.27.3` en `go.mod`: la Action y el CI compilaban con Go 1.27.0 (11 vulnerabilidades de la biblioteca estándar).
- Tests de regresión: sin `${{ }}` en `run:`, workflows con permisos y escape de logins hostiles en todas las tarjetas.
- README para personas usuarias con cabecera pixel-art (sprites reales) y `docs/DESARROLLO.md`; `.claude/agents` y `.claude/skills`; corregido `@v1.0.0` inexistente en `docs/USO.md`.

## Hecho en la fase 2 (24/10/2026, tarde, rama `fase2-2026-10-24`)
- Generador del workflow en la web (usuario → especie/tema/tamaño → YAML y README; 6 pasos y «Si algo falla»), `commitling workflow`.
- Errores de la API en español y accionables, anotación `::error` en Actions, aviso de permisos en el `git push` del workflow de ejemplo.
- Token enmascarado en la Action, actionlint sobre los 13 workflows generables, CSP sin `'unsafe-inline'` en estilos.

## Siguiente (por orden)
1. **Alberto**: fusionar el PR, crear la release `v0.8.0` con las notas del CHANGELOG y mover `v1` a ella (la Action es compatible: mismas entradas, salida y SVG byte a byte).
2. **Alberto**: activar «Private vulnerability reporting» en Settings → Security (lo enlaza `SECURITY.md`).
3. Tras el despliegue: abrir https://bertmarti.github.io/commitling/ con la consola abierta y comprobar que no hay violaciones de CSP (demo, generador con un usuario y descarga).
4. Publicar la Action en el Marketplace si aún no lo está (nombre `commitling`, branding ya declarado).
5. Subir la línea `go` de `go.mod` con cada parche de Go (y el mínimo de `TestGoModPinsAPatchedGo`) (govulncheck en el CI avisará).
6. Fijar las acciones de terceros por SHA en los workflows (hoy por versión mayor) si se quiere el máximo de reproducibilidad; dependabot sabe actualizar SHAs.

## Ideas (sin compromiso)
- Una tercera especie de diseño propio (receta en `CONTRIBUTING.md`).
- Precompilar el binario en las releases para que la Action no compile con `go run` en cada ejecución (ahorra unos segundos; habría que mantener la compatibilidad de `action.yml`).
- Modo «año en píxeles» con la actividad de todo el año (requiere otra fuente de datos: la API de eventos solo da 90 días).
- Selector manual claro/oscuro en la web (hoy sigue al sistema).
