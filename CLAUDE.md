@AGENTS.md
@MEMORY.md

## Específico de Claude Code
- Lee `AGENTS.md` (reglas, comandos, diseño «papel y píxel») y `MEMORY.md` (estado y decisiones) antes de tocar nada; `PLAN.md` dice qué toca después.
- Subagentes del repo en `.claude/agents/`: `revisor-seguridad` (Action, workflows, CSP, escape del SVG) y `guardian-compatibilidad` (que la Action y las tarjetas no cambien sin querer).
- Skills en `.claude/skills/`: `publicar-version` (CHANGELOG, versión de la CLI, qué etiquetas mueve Alberto) y `probar-web` (galería + wasm servidos con el prefijo `/commitling/` y comprobación en navegador).
- Antes de dar algo por terminado: `gofmt -l .` vacío, `go vet ./...`, `go test ./...` y, si tocaste la web, probarla en un navegador (la CSP no perdona: mira la consola).
- Nunca crees etiquetas ni releases ni hagas push a `main`: lo hace Alberto.
