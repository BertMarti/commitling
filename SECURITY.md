# Seguridad

## Versiones con soporte

| Versión | Soporte |
|---|---|
| `v1` (la etiqueta móvil de la Action) y la última versión publicada | ✅ |
| Versiones anteriores | ❌ (actualiza a `@v1`) |

## Cómo avisar de un problema

**No abras una issue pública** para un fallo de seguridad. Usa el aviso privado de GitHub:
pestaña **Security → Report a vulnerability** de [BertMarti/commitling](https://github.com/BertMarti/commitling/security/advisories/new).
Cuenta qué pasa, cómo reproducirlo y qué versión usas. Si esa opción no aparece, abre una issue pidiendo un canal privado **sin dar detalles del fallo**.
Respondo en cuanto pueda (proyecto personal, sin plazos garantizados).

## Qué cubre

- La **GitHub Action** (`action.yml`): solo lee la API pública de eventos con el token que le pases; no escribe en tu repositorio (el commit lo hace tu workflow) y sus entradas llegan al script por variables de entorno, nunca interpoladas en el código.
- El **SVG generado**: todo texto que viene de fuera (usuario, nombres) se escapa como XML; el SVG no lleva scripts ni enlaces externos.
- Los **binarios de las releases**: se compilan de forma reproducible y la Action solo ejecuta uno si su SHA-256 coincide con `release/checksums.txt` de su propia etiqueta (si no, compila con Go). A partir de la 1.2.0 llevan además una atestación de procedencia firmada por GitHub: `gh attestation verify <binario> --repo BertMarti/commitling`.
- La **web** (GitHub Pages): sin dependencias de terceros, con Content-Security-Policy estricta; Trusted Types (la única política crea la URL de `wasm_exec.js`); el generador solo habla con `api.github.com`, no hay cookies y en `localStorage` solo se guarda la preferencia de tema (`commitling:theme`, `light` o `dark`).

## Buenas prácticas para quien usa la Action

- Fíjala a `@v1`, a una versión exacta (por ejemplo `@v1.2.0`) o a un SHA, nunca a `@main`.
- Da a tu workflow solo `permissions: contents: write` (lo necesita para guardar el SVG) y nada más.
- No hace falta ningún token personal: basta el `GITHUB_TOKEN` del propio workflow.
