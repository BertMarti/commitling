# Guía de uso de commitling

Esta guía es para ti si quieres tener tu propia criatura de commitling en el perfil de GitHub. No hace falta saber programar: son unos pasos de copiar y pegar en la web de GitHub.

## Índice

1. [Qué es commitling](#1-qué-es-commitling)
2. [Ver la galería](#2-ver-la-galería)
3. [Ponerla en tu perfil, paso a paso](#3-ponerla-en-tu-perfil-paso-a-paso)
4. [Tema oscuro](#4-tema-oscuro)
5. [Las reglas: XP, fases, ánimos y accesorios](#5-las-reglas-xp-fases-ánimos-y-accesorios)
6. [Versiones de la Action y cómo actualizar](#6-versiones-de-la-action-y-cómo-actualizar)
7. [Desde la línea de órdenes y la imagen de vista previa](#7-desde-la-línea-de-órdenes-y-la-imagen-de-vista-previa)
8. [Preguntas frecuentes](#8-preguntas-frecuentes)
9. [Solución de problemas](#9-solución-de-problemas)

---

## 1. Qué es commitling

commitling es una mascota pixel-art que vive en el README de tu perfil de GitHub. Empieza siendo una semilla y, según lo que hagas en GitHub, crece hasta convertirse en un árbol ancestral. Además cambia de ánimo (duerme si llevas días sin aparecer, salta de alegría si tienes racha) y puede llevar gorro, bufanda o una flor.

Cada día, un pequeño programa de GitHub (una «Action») mira tu actividad **pública**, calcula unos números y redibuja la criatura como una imagen. Tú solo tienes que configurarlo una vez.

## 2. Ver la galería

Antes de instalar nada puedes ver todas las fases y ánimos en la web del proyecto:

https://bertmarti.github.io/commitling/

Ahí verás las dos especies, cada una con sus 5 fases y los 4 ánimos de cada una (20 tarjetas por especie, en versión clara y oscura), los accesorios, las reglas y las instrucciones de instalación con botones para copiar. También hay una criatura «en vivo» del autor del proyecto, que se actualiza cada día.

## 3. Ponerla en tu perfil, paso a paso

Necesitas una cuenta de GitHub. Todo se hace desde el navegador.

### Paso 1. Crea (o abre) tu repositorio de perfil

GitHub muestra en tu perfil el README de un repositorio especial: el que se llama **igual que tu usuario**. Si tu usuario es `ana-lopez`, el repositorio debe llamarse `ana-lopez/ana-lopez`.

- Si ya lo tienes, ábrelo y pasa al paso 2.
- Si no lo tienes: pulsa **New repository** (el `+` de arriba a la derecha), escribe tu usuario como nombre, márcalo como **Public** y marca **Add a README file**. Al crearlo, GitHub indica que es un repositorio especial.

### Paso 2. Crea el workflow

Un «workflow» es el archivo que le dice a GitHub qué hacer y cuándo.

1. En tu repositorio de perfil pulsa **Add file → Create new file**.
2. En el nombre escribe exactamente `.github/workflows/commitling.yml` (al escribir las barras `/`, GitHub crea las carpetas por ti).
3. Pega este contenido tal cual:

```yaml
name: commitling

on:
  schedule:
    - cron: "23 5 * * *" # cada día a las 05:23 UTC
  workflow_dispatch:

permissions:
  contents: write

jobs:
  commitling:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v7
      - uses: BertMarti/commitling@v1
        with:
          out: commitling.svg
      - name: Guardar el SVG si ha cambiado
        run: |
          git add commitling.svg
          if git diff --cached --quiet; then
            echo "Sin cambios."
            exit 0
          fi
          git config user.name "github-actions[bot]"
          git config user.email "41898282+github-actions[bot]@users.noreply.github.com"
          git commit -m "chore: actualiza commitling"
          git push
```

4. Pulsa **Commit changes…** y confirma con **Commit changes**.

No tienes que cambiar tu nombre de usuario en ningún sitio: la Action usa por defecto el dueño del repositorio, es decir, tú.

Qué hace este archivo, en cristiano:

- `schedule` / `cron`: lo ejecuta solo cada día a las 05:23 UTC (07:23 en el horario de verano de la España peninsular, 06:23 en invierno). GitHub puede retrasar unos minutos las ejecuciones programadas.
- `workflow_dispatch`: añade el botón **Run workflow** para lanzarlo a mano.
- `permissions: contents: write`: le permite guardar la imagen en tu repositorio. Sin esto fallaría al guardar.
- `BertMarti/commitling@v1`: es la Action de commitling, que crea el archivo `commitling.svg`. `@v1` significa «la versión 1»: recibes las mejoras compatibles sin tocar nada. Si prefieres fijar una versión exacta, usa por ejemplo `@v1.0.0`.
- El último paso guarda `commitling.svg` en tu repositorio solo si ha cambiado, con un commit «chore: actualiza commitling».

### Paso 3. Añade la imagen a tu README

1. Abre el `README.md` de tu repositorio de perfil y pulsa el lápiz para editarlo.
2. Pega esta línea donde quieras que aparezca la criatura:

```markdown
![commitling](./commitling.svg)
```

3. Guarda con **Commit changes**.

Hasta el paso 4 verás una imagen rota: es normal, porque el archivo aún no existe.

### Paso 4. Ejecuta el workflow a mano la primera vez

Si no lo lanzas, tendrías que esperar a la próxima ejecución diaria.

1. Ve a la pestaña **Actions** de tu repositorio de perfil.
2. En la lista de la izquierda elige **commitling**.
3. Pulsa **Run workflow** y confirma con el botón verde **Run workflow**.
4. Espera un minuto o dos (la primera vez compila el programa). Cuando aparezca un círculo verde, abre tu perfil (`github.com/tu-usuario`) y recarga la página: ahí está tu criatura.

Si el botón **Run workflow** no aparece, comprueba que el archivo está en `.github/workflows/` de la rama principal y que contiene `workflow_dispatch:`.

A partir de aquí, se actualiza sola cada día. La criatura solo genera un commit nuevo en tu repositorio cuando cambia de verdad (fase, ánimo, números…).

### Opciones de la Action (opcional)

Dentro de `with:` puedes ajustar:

| Opción | Por defecto | Qué hace |
|---|---|---|
| `user` | el dueño del repositorio | Usuario de GitHub cuya actividad se dibuja |
| `out` | `commitling.svg` | Nombre del archivo de imagen |
| `theme` | `light` | `light` (papel) o `dark` (tinta) |
| `species` | `moss` | Especie de la criatura: `moss` (brote de musgo) o `mushroom` (hongo) |
| `token` | el token del propio workflow | Suele bastar; no hace falta que lo cambies |
| `fixture` | vacío | Archivo de eventos de ejemplo para probar sin conexión (solo para desarrollo) |

### Elegir la especie

Hay dos especies, las dos de diseño propio y con las mismas reglas (XP, ánimos y accesorios). Por defecto tu criatura es un **brote de musgo** (`moss`). Si prefieres el **hongo** (`mushroom`), una seta pequeña de sombrero miel, añade `species: mushroom` en `with:`:

```yaml
      - uses: BertMarti/commitling@v1
        with:
          out: commitling.svg
          species: mushroom
```

Solo cambian el dibujo y el nombre de las fases:

| Fase | Desde | Brote de musgo (`moss`) | Hongo (`mushroom`) |
|---|---|---|---|
| 1 | 0 XP | Semilla | Espora |
| 2 | 100 XP | Brote | Botón |
| 3 | 400 XP | Retoño | Seta |
| 4 | 1.000 XP | Arbusto | Seta grande |
| 5 | 2.500 XP | Árbol ancestral | Corro de setas |

Valores que acepta `species` (sin distinguir mayúsculas ni espacios alrededor):

| Especie | Valores válidos |
|---|---|
| Brote de musgo | `moss`, `musgo`, `brote de musgo` (y vacío, que es la predeterminada) |
| Hongo | `mushroom`, `hongo`, `seta`, `setas` |

Cualquier otro valor (por ejemplo `dragon`) hace fallar el paso a propósito, con el mensaje `especie no válida "dragon" (usa moss o mushroom)`, para que un error tipográfico no pase desapercibido, y no escribe ningún archivo. Puedes ver todas las fases de las dos especies en la [galería](https://bertmarti.github.io/commitling/#galeria).

¿Las dos a la vez? Sí: usa la Action dos veces con `out` distinto (por ejemplo `commitling.svg` y `commitling-hongo.svg`, cada uno con su `species`), suma el segundo archivo al `git add` y enseña la imagen que quieras en el README.

## 4. Tema oscuro

Si tu perfil lo ve gente con el tema oscuro de GitHub, puedes ofrecer una imagen para cada tema. GitHub elige la adecuada según quién mira.

**a) Cambia el workflow** para generar dos imágenes. Sustituye la sección `steps:` por esta (lo demás del archivo no cambia):

```yaml
    steps:
      - uses: actions/checkout@v7
      - uses: BertMarti/commitling@v1
        with:
          out: commitling.svg
      - uses: BertMarti/commitling@v1
        with:
          out: commitling-dark.svg
          theme: dark
      - name: Guardar los SVG si han cambiado
        run: |
          git add commitling.svg commitling-dark.svg
          if git diff --cached --quiet; then
            echo "Sin cambios."
            exit 0
          fi
          git config user.name "github-actions[bot]"
          git config user.email "41898282+github-actions[bot]@users.noreply.github.com"
          git commit -m "chore: actualiza commitling"
          git push
```

**b) Cambia la línea de tu README** por este bloque:

```html
<picture>
  <source media="(prefers-color-scheme: dark)" srcset="./commitling-dark.svg">
  <img alt="commitling" src="./commitling.svg">
</picture>
```

Después ejecuta el workflow a mano (paso 4 de la sección anterior). En el tema oscuro la tarjeta tiene fondo de tinta y la silueta de la criatura se dibuja clara, como una pegatina.

## 5. Las reglas: XP, fases, ánimos y accesorios

Todo se calcula con tu actividad **pública** de los **últimos 90 días** (la API de GitHub da como máximo 300 eventos). Los días se cuentan en **UTC**, que no coincide con la medianoche de España: el día UTC empieza a las 02:00 en horario de verano y a la 01:00 en invierno.

La regla de oro: mismos datos, misma criatura. No hay azar.

### Experiencia (XP)

| Actividad | XP |
|---|---|
| Cada commit de un push | 10 |
| Pull request abierta | 25 |
| Issue abierta | 5 |
| Cualquier otro evento público (estrellas, forks, comentarios, revisiones…) | 2 |

Detalles útiles:

- Un push con 3 commits da 30 XP.
- Abrir una pull request o una issue da XP; cerrarla, comentarla o fusionarla cuenta solo como «otro evento» (2 XP).
- Si GitHub no dice cuántos commits tiene un push, cuenta como 1.
- Como solo se miran los últimos 90 días, **la XP puede bajar**: lo que hiciste hace más de 90 días deja de contar.

Ejemplo: en los últimos 90 días has hecho 20 commits (200 XP), abierto 2 pull requests (50 XP), abierto 3 issues (15 XP) y dado 10 estrellas (20 XP). Total: **285 XP**.

### Fases

| Fase | Desde |
|---|---|
| Semilla | 0 XP |
| Brote | 100 XP |
| Retoño | 400 XP |
| Arbusto | 1.000 XP |
| Árbol ancestral | 2.500 XP |

En el hongo, las cinco fases se llaman Espora, Botón, Seta, Seta grande y Corro de setas, con las mismas XP. Con 285 XP estarías en **Brote** (fase 2 de 5) y te faltarían 115 XP para ser Retoño. La tarjeta muestra la XP total, la meta siguiente y una barra de progreso (24 casillas) hasta la próxima fase; en el Árbol ancestral la barra va siempre llena.

### Ánimos

El ánimo depende de cuántos días UTC han pasado desde tu última actividad y de tu racha (días seguidos con actividad).

| Ánimo | Cuándo | Cómo se nota |
|---|---|---|
| Durmiendo | 7 días o más sin actividad, o ninguna actividad | Ojos cerrados, respiración lenta y «zzz» |
| Aburrido | De 3 a 6 días sin actividad | Párpados caídos y boca recta |
| Contento | Actividad hoy, ayer o anteayer | Sonrisa, mejillas y parpadeo |
| Radiante | Racha de 5 días seguidos o más | Ojos felices, saltitos y destellos |

Se comprueban en este orden: primero «Durmiendo», luego «Aburrido», luego «Radiante» y, si no se cumple nada, «Contento».

Ejemplos (hoy es viernes):

- Hiciste un commit el miércoles y nada después: han pasado 2 días, así que está **Contento**.
- Tu último commit fue el lunes: han pasado 4 días, así que está **Aburrido**.
- Tu último commit fue hace 10 días: está **Durmiendo**.
- Llevas commits de lunes a viernes, cinco días seguidos: está **Radiante**.

**La racha** son los días seguidos con al menos una actividad, contando hacia atrás desde hoy. Se mantiene hasta el final del día siguiente a tu última actividad, para que no se rompa por la mañana antes de tu primer commit: si ayer hiciste algo y hoy todavía no, la racha sigue viva. Si pasan dos días completos sin actividad, vuelve a cero. En un mismo día, da igual cuántas cosas hagas: cuenta como un día.

### Accesorios

| Accesorio | Se desbloquea con |
|---|---|
| Gorro | **Más de** 30 días activos en los últimos 90 (31 o más) |
| Bufanda | Racha de 10 días o más |
| Flor | 5 repositorios distintos o más con commits, pull requests o issues |

- «Días activos» son los días UTC en los que hiciste algo público. La tarjeta muestra los de los últimos 30 días como `activo 30 d`, pero el gorro se calcula con los 90.
- «Repositorios distintos» solo cuenta repositorios en los que hiciste commits, abriste una pull request o una issue. Dar una estrella no cuenta. La tarjeta lo muestra como `repos`.
- Los accesorios se pueden tener a la vez y se pierden si dejas de cumplir la condición.

## 6. Versiones de la Action y cómo actualizar

En el workflow, la línea `uses: BertMarti/commitling@...` decide qué versión de commitling se ejecuta:

| Escribes | Qué significa | Cuándo usarla |
|---|---|---|
| `@v1` | La versión mayor 1: la etiqueta `v1` se mueve con cada versión 1.x compatible, así que recibes mejoras y correcciones sin tocar nada | Lo recomendado |
| `@v1.0.0` | Una versión exacta, que nunca cambia | Si quieres controlar tú cuándo actualizas |
| `@<sha>` (huella de un commit) | Una copia exacta del código, la opción más reproducible | Si te importa la máxima seguridad |
| `@main` | La rama de desarrollo, que puede cambiar en cualquier momento (y romper tu workflow) | Solo para probar cambios |

**Actualizar desde `@main` a `@v1`.** Si copiaste un workflow antiguo que usaba `BertMarti/commitling@main`:

1. Abre `.github/workflows/commitling.yml` en tu repositorio de perfil y pulsa el lápiz.
2. Cambia `BertMarti/commitling@main` por `BertMarti/commitling@v1` en cada sitio donde aparezca (dos veces si tienes la variante con tema oscuro).
3. Guarda con **Commit changes** y lanza el workflow a mano (**Actions → commitling → Run workflow**).

No hay que tocar nada más: las entradas (`out`, `theme`, `user`, `token`) siguen igual, y `species` es opcional (sin ella, sigues con el brote de musgo).

Si el workflow falla con «unable to find version `v1`» es que la etiqueta `v1` aún no existe (la crea el responsable del proyecto al publicar la versión 1). Mientras tanto usa `@main`.

La Action aparece en el [GitHub Marketplace](https://github.com/marketplace?type=actions) buscando «commitling»; el uso es el mismo que el del ejemplo del paso 2.

## 7. Desde la línea de órdenes y la imagen de vista previa

Esto es para quien quiera probar commitling en su ordenador (hace falta Go 1.27, sin dependencias externas); no hace falta para usarlo en tu perfil. Desde una copia del repositorio:

```sh
# Tu criatura (GITHUB_TOKEN es opcional, pero sube el límite de peticiones)
go run ./cmd/commitling render --user tu-usuario --out out/commitling.svg

# El hongo en tema oscuro
go run ./cmd/commitling render --user tu-usuario --species mushroom --theme dark --out out/hongo-dark.svg

# Sin red, con datos de ejemplo de un usuario ficticio
go run ./cmd/commitling render --fixture testdata/events.json --out out/commitling.svg

# Todas las fases, ánimos y accesorios como web estática
go run ./cmd/commitling gallery --out site/

# La imagen de vista previa (PNG de 1200x630)
go run ./cmd/commitling og --out out/og.png

go run ./cmd/commitling version
```

`--species` acepta los mismos valores que `species` en la Action (sección 3) y `--out -` escribe en la salida estándar.

**La imagen de vista previa (`og`).** Es el PNG de 1200×630 píxeles que muestran Slack, X, LinkedIn o Telegram cuando alguien pega el enlace de la web del proyecto (formato «Open Graph»). Enseña una criatura grande y las cinco fases de las dos especies, en la paleta de la criatura, sin suavizado y en unos 3 KB. `commitling og --out og.png` la dibuja donde le digas (con `--out -` va a la salida estándar); no necesita red ni usuario, y sale siempre igual. Solo te hace falta si mantienes tu propia web: `commitling gallery` ya la genera como `og.png` dentro de la carpeta de salida y las etiquetas `og:image` de la página apuntan a ella. Ten en cuenta que las redes sociales guardan las vistas previas en caché, así que un cambio puede tardar en verse.

## 8. Preguntas frecuentes

**¿Por qué no se actualiza?**
La Action se ejecuta una vez al día (05:23 UTC) y solo guarda un archivo nuevo si la criatura ha cambiado, así que muchos días no verás ningún commit nuevo. Además, GitHub tarda un poco en publicar tus eventos y el navegador puede tener la imagen antigua en caché. Para forzar una actualización: pestaña **Actions → commitling → Run workflow** y recarga tu perfil con Ctrl+F5 (o Cmd+Mayús+R en Mac). Si el workflow acaba en verde, la imagen está al día. Más pistas en [Solución de problemas](#9-solución-de-problemas).

**¿Usa mis repositorios privados?**
No. commitling solo lee la API **pública** de eventos de GitHub, es decir, lo mismo que cualquiera puede ver de ti. Tu actividad en repositorios privados no cuenta, ni siquiera aunque tengas activada la opción de mostrar contribuciones privadas en tu perfil. No guarda nada fuera de tu propio repositorio y solo necesita el permiso `contents: write` del workflow.

**¿Por qué dice «Durmiendo»?**
Porque no ve actividad pública tuya en los últimos 7 días o más. Las causas habituales: llevas una semana sin actividad, tu trabajo está en repositorios privados (no se ven), o los eventos son de hace más de 90 días. Si acabas de hacer commits en un repositorio público, espera un poco y vuelve a ejecutar el workflow.

**¿Cuánto tarda en cambiar de fase?**
Depende solo de tu XP. Como referencia, un commit da 10 XP: llegar a Brote (100 XP) son unos 10 commits; a Retoño, 40; a Arbusto, 100; y al Árbol ancestral, 250 commits, todo dentro de una ventana de 90 días (o menos commits si abres pull requests, que dan 25 XP). La criatura se redibuja cuando se ejecuta el workflow, normalmente una vez al día. Y recuerda: si tu actividad baja, la XP puede bajar y la criatura volver a una fase anterior.

**¿Cuánto tarda en cambiar de ánimo?**
El ánimo se recalcula en cada ejecución. Con la actividad de hoy pasas a «Contento» en la siguiente ejecución, y necesitas 5 días seguidos para «Radiante».

**¿Se puede dibujar a otra persona?**
Sí. Añade `user: nombre-de-usuario` dentro del `with:` de la Action. Solo se usa su actividad pública.

**¿Qué especie elijo?**
La que más te guste: el dibujo cambia, pero la XP, los ánimos, los accesorios y los umbrales de fase son idénticos. Puedes cambiar de especie cuando quieras editando `species:` en el workflow; la criatura conserva la misma fase porque solo depende de tu actividad.

**¿Qué pasa si GitHub falla justo cuando se ejecuta?**
commitling repite la petición por su cuenta antes de fallar; el detalle está en [El workflow falla por el límite de la API](#el-workflow-falla-por-el-límite-de-la-api). Si aun así se rinde, el workflow queda en rojo, no se guarda nada y al día siguiente se vuelve a intentar.

**¿Cómo pasó de `@main` a `@v1`?**
Es solo cambiar una palabra en el workflow; los pasos están en la sección [Versiones de la Action y cómo actualizar](#6-versiones-de-la-action-y-cómo-actualizar).

**¿Cómo la quito?**
1. Borra la línea `![commitling](./commitling.svg)` (o el bloque `<picture>`) de tu README.
2. Borra el archivo `.github/workflows/commitling.yml` (abre el archivo, menú `⋯`, **Delete file**).
3. Opcional: borra `commitling.svg` (y `commitling-dark.svg`) de tu repositorio.

Si solo quieres pausarlo sin borrar nada: **Actions → commitling → menú ⋯ → Disable workflow**.

## 9. Solución de problemas

### La imagen del README aparece rota

- Comprueba que has ejecutado el workflow al menos una vez (paso 4) y que acabó en verde.
- Comprueba que existe el archivo `commitling.svg` en la raíz de tu repositorio de perfil.
- Comprueba que la ruta del README coincide con el `out:` del workflow (por defecto, `./commitling.svg`).

### El workflow falla al guardar con un error de permisos

Verás algo como «Permission denied» o `403` en el paso «Guardar el SVG si ha cambiado». Revisa:

1. Que tu workflow contiene `permissions:` con `contents: write` (está en el ejemplo).
2. Que en el repositorio, **Settings → Actions → General → Workflow permissions**, está marcado **Read and write permissions**. Algunas organizaciones lo limitan a solo lectura y esa opción manda sobre el archivo.
3. Que no hay una regla de protección de la rama principal que impida a `github-actions[bot]` hacer push directamente.

### El workflow falla por el límite de la API

El mensaje dice que GitHub ha rechazado la petición (403 o 429) y que puede ser el límite de peticiones, o que la API ha fallado (500, 502, 503 o 504) o no ha respondido. commitling ya reintenta por su cuenta antes de rendirse:

- **Qué se reintenta:** los errores del servidor (500, 502, 503 y 504), el límite de peticiones (429, o 403 con `Retry-After` o `X-RateLimit-Remaining: 0`) y los errores de red (DNS, conexión rechazada o cortada, tiempo agotado; el mensaje dice «no se pudo contactar con la API de GitHub»). Solo se repite la página que falla, no todas.
- **Cuántas veces:** hasta 3 reintentos (4 intentos en total), con esperas de 1, 2 y 4 segundos, o las que indique GitHub en `Retry-After` o `X-RateLimit-Reset`. En total, unos 7 segundos de espera como mucho.
- **Cuándo se rinde:** al agotar los 3 reintentos (el error dice «tras 3 reintentos»), o de inmediato si GitHub pide esperar más de 30 segundos (por ejemplo, hasta que se reinicie la cuota dentro de media hora): esperar tanto colgaría el workflow. Tampoco reintenta lo que no tiene arreglo esperando: usuario no encontrado (404), 422 y un 403 sin cabeceras de límite (que es un problema de permisos).

En un workflow normal no debería pasar, porque la Action usa el token del propio workflow, que tiene más margen. Si falla, es casi siempre pasajero: espera y vuelve a ejecutarlo (o deja que lo haga el cron del día siguiente). Comprueba también que no has puesto un `token:` propio caducado, y que el usuario de `user:` existe. Si la usas fuera de GitHub Actions (en la línea de órdenes), define la variable `GITHUB_TOKEN` para subir el límite.

### El workflow termina en verde pero no hace commit

Es lo esperado cuando no hay cambios: el último paso muestra «Sin cambios.» y termina sin hacer nada. commitling solo guarda un archivo nuevo si la criatura es distinta a la de ayer (no incluye fechas ni nada variable a propósito). Si esperabas un cambio, puede que la criatura siga igual porque tu actividad pública no ha cambiado su fase, su ánimo ni sus números.

### Los workflows programados dejaron de ejecutarse

GitHub desactiva las ejecuciones programadas de un repositorio público si pasan unos 60 días sin actividad en él. Como commitling solo hace commit cuando la criatura cambia, esto puede pasar en repositorios de perfil muy quietos. Entra en **Actions**: si ves un aviso de workflow desactivado, pulsa **Enable workflow**. También puedes lanzarlo a mano cuando quieras con **Run workflow**.

### No aparece el botón «Run workflow»

El archivo debe estar en la rama principal, dentro de `.github/workflows/` y con la línea `workflow_dispatch:`. Si acabas de crearlo, recarga la página de **Actions**.

### Dudas o errores

Si nada de esto ayuda, abre una issue en https://github.com/BertMarti/commitling/issues con el enlace a la ejecución fallida del workflow.
