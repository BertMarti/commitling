# Registro de cambios

Todos los cambios notables de commitling se anotan en este archivo.

El formato sigue [Keep a Changelog](https://keepachangelog.com/es-ES/1.1.0/) y el proyecto usa [versionado semántico](https://semver.org/lang/es/).

## [Sin publicar]

### Añadido

- **Conservar el último SVG válido ante caídas largas de la API** (#16): opción `--keep-on-error` en `commitling render` y entrada `keep-on-error` en la Action (por defecto `true`, compatible con los workflows existentes, que la reciben sin cambiar nada). Si la API falla tras los reintentos y el archivo de salida ya es un SVG completo, se conserva intacto y el paso termina en verde con un aviso (`::warning::` en GitHub Actions). Sin archivo previo, con un archivo vacío o cortado, o con `--out -`, el error se mantiene. Un `fixture` inexistente o una especie no válida siguen siendo errores.

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

[Sin publicar]: https://github.com/BertMarti/commitling/compare/v0.2.0...HEAD
[0.2.0]: https://github.com/BertMarti/commitling/compare/v0.1.0...v0.2.0
[0.1.0]: https://github.com/BertMarti/commitling/releases/tag/v0.1.0
