# Seguridad

## Versiones con soporte

| Versión | Soporte |
|---|---|
| `v1` (la etiqueta móvil de la Action) y la última `v0.x` publicada | ✅ |
| Versiones anteriores | ❌ (actualiza a `@v1`) |

## Cómo avisar de un problema

**No abras una issue pública** para un fallo de seguridad. Usa el aviso privado de GitHub:
pestaña **Security → Report a vulnerability** de [BertMarti/commitling](https://github.com/BertMarti/commitling/security/advisories/new).
Cuenta qué pasa, cómo reproducirlo y qué versión usas. Si esa opción no aparece, abre una issue pidiendo un canal privado **sin dar detalles del fallo**.
Respondo en cuanto pueda (proyecto personal, sin plazos garantizados).

## Qué cubre

- La **GitHub Action** (`action.yml`): solo lee la API pública de eventos con el token que le pases; no escribe en tu repositorio (el commit lo hace tu workflow) y sus entradas llegan al script por variables de entorno, nunca interpoladas en el código.
- El **SVG generado**: todo texto que viene de fuera (usuario, nombres) se escapa como XML; el SVG no lleva scripts ni enlaces externos.
- La **web** (GitHub Pages): sin dependencias de terceros, con Content-Security-Policy estricta; el generador solo habla con `api.github.com` y no usa cookies ni `localStorage`.

## Buenas prácticas para quien usa la Action

- Fíjala a `@v1`, a una versión exacta (por ejemplo `@v0.7.0`) o a un SHA, nunca a `@main`.
- Da a tu workflow solo `permissions: contents: write` (lo necesita para guardar el SVG) y nada más.
- No hace falta ningún token personal: basta el `GITHUB_TOKEN` del propio workflow.
