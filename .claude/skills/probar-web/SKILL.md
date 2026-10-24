---
name: probar-web
description: Construye la web de commitling (galería + generador wasm + criatura en vivo de ejemplo), la sirve con el prefijo /commitling/ como GitHub Pages y la comprueba en un navegador (CSP, demo, generador, móvil y modo oscuro).
---

# Probar la web de commitling en local

1. Construye (de uno en uno; en la Raspberry Pi con `GOFLAGS=-p=2`):
   ```sh
   R=out/serve; rm -rf $R; mkdir -p $R
   go run ./cmd/commitling gallery --out $R/commitling
   GOOS=js GOARCH=wasm go build -trimpath -ldflags="-s -w" -o $R/commitling/commitling.wasm ./cmd/wasm
   cp "$(go env GOROOT)/lib/wasm/wasm_exec.js" $R/commitling/
   mkdir -p $R/commitling/live
   go run ./cmd/commitling render --fixture testdata/events.json --out $R/commitling/live/BertMarti.svg
   go run ./cmd/commitling render --fixture testdata/events.json --theme dark --out $R/commitling/live/BertMarti-dark.svg
   python3 -m http.server 18417 --bind 127.0.0.1 --directory $R   # http://127.0.0.1:18417/commitling/
   ```
   (Elige otro puerto si está ocupado.)
2. En el navegador, abre la consola y comprueba:
   - **Sin violaciones de CSP** (escucha `securitypolicyviolation` o mira los errores de consola).
   - «Ver demo: míralo crecer» arranca la demo; Pausa/Reanudar/Detener funcionan también con teclado y el foco vuelve a «Ver demo».
   - El generador dibuja con un usuario real (gasta 1-3 de las 60 peticiones por hora), «Descargar SVG» y «Copiar» funcionan.
   - Sin `live/`, aparece el aviso «La criatura en vivo aún no está disponible» (fuerza la carga si el `loading="lazy"` no se dispara).
   - A 320 y 390 px de ancho no hay desplazamiento lateral; en modo oscuro todo se lee.
3. Capturas sin cabeza: `chromium --headless=new --disable-gpu --hide-scrollbars --window-size=1366,850 --virtual-time-budget=8000 --screenshot=out/web.png http://127.0.0.1:18417/commitling/` (y `--window-size=390,844` para móvil). Mira la imagen antes de darla por buena.
