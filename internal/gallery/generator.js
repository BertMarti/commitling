// Live generator of the website. The drawing is done by the same Go code as
// the command line, compiled to WebAssembly (commitling.wasm); this file only
// downloads the public events of a user from api.github.com, hands them to the
// wasm and shows the result. No dependencies, no credentials, no HTML built
// from outside data.
(function () {
  'use strict';

  var root = document.getElementById('gen');
  if (!root) return;

  var API = 'https://api.github.com/users/';
  var PAGES = 3; // 3 pages of 100: the 300 events the API keeps (about 90 days)
  var TIMEOUT_MS = 15000; // same as the command line client
  var REUSE_MS = 60000; // the API caches for about a minute: do not ask again sooner
  var DEMO_MS = 15000; // the whole timelapse of the demo
  var form = document.getElementById('gen-form');
  var userInput = document.getElementById('gen-user');
  var button = document.getElementById('gen-go');
  var statusEl = document.getElementById('gen-status');
  var errorEl = document.getElementById('gen-error');
  var img = document.getElementById('gen-img');
  var stage = document.getElementById('gen-stage');
  var empty = document.getElementById('gen-empty');
  var caption = document.getElementById('gen-caption');
  var after = document.getElementById('gen-after');
  var dl = document.getElementById('gen-dl');
  var wf = document.getElementById('gen-wf');
  var demoGo = document.getElementById('demo-go');
  var demoPause = document.getElementById('demo-pause');
  var demoStop = document.getElementById('demo-stop');
  var demoDay = document.getElementById('demo-day');
  var demoSlider = document.getElementById('demo-slider');
  var demoRange = document.getElementById('demo-range');
  var demoPhase = document.getElementById('demo-phase');
  var heroDemo = document.getElementById('hero-demo');

  var busy = false;
  var current = null; // {user, events, at} of the last successful search
  var blobURL = '';
  var demo = null; // {day, days, phase, playing, elapsed, t0, raf} while the demo is on screen

  root.hidden = false;

  if (typeof WebAssembly !== 'object' || typeof fetch !== 'function') {
    errorEl.textContent = 'Tu navegador no admite WebAssembly, que el generador necesita. Prueba con una versión reciente de Firefox, Chrome, Safari o Edge.';
    button.setAttribute('aria-disabled', 'true');
    demoGo.disabled = true;
    return;
  }

  function checked(name) {
    return form.elements[name].value;
  }

  function say(text) {
    errorEl.textContent = '';
    statusEl.textContent = text;
  }

  function fail(text) {
    statusEl.textContent = '';
    errorEl.textContent = text;
  }

  function setBusy(on) {
    busy = on;
    form.setAttribute('aria-busy', on ? 'true' : 'false');
    button.setAttribute('aria-disabled', on ? 'true' : 'false');
  }

  // The wasm and wasm_exec.js are only downloaded the first time they are
  // needed, which is when the person interacts with the generator.
  var loading = null;

  function loadScript(src) {
    return new Promise(function (resolve, reject) {
      var s = document.createElement('script');
      s.src = src;
      s.onload = resolve;
      s.onerror = function () { reject(new Error('no se pudo cargar ' + src)); };
      document.head.appendChild(s);
    });
  }

  var WASM_URL = 'commitling.wasm';

  // Instantiates the wasm from a request that is already under way. The same
  // response serves the fallback (a clone), so a server that does not send
  // application/wasm costs no second download.
  function instantiate(go, pending) {
    var viaBuffer = function (res) {
      if (!res.ok) throw new Error(WASM_URL + ': ' + res.status);
      return res.arrayBuffer().then(function (b) { return WebAssembly.instantiate(b, go.importObject); });
    };
    return pending.then(function (res) {
      if (!WebAssembly.instantiateStreaming) return viaBuffer(res);
      var copy = res.clone();
      return WebAssembly.instantiateStreaming(res, go.importObject).catch(function () { return viaBuffer(copy); });
    });
  }

  function ensureWasm() {
    if (!loading) {
      // The .wasm (the heavy part) is requested now, while wasm_exec.js loads.
      var wasm = fetch(WASM_URL);
      wasm.catch(function () {}); // if the script fails first, this is not an unhandled rejection
      loading = loadScript('wasm_exec.js').then(function () {
        var go = new Go();
        return instantiate(go, wasm).then(function (result) {
          go.run(result.instance).catch(function () {});
          if (!window.commitling) throw new Error('el generador no se inició');
        });
      });
      loading.catch(function () { loading = null; }); // allow trying again
    }
    return loading;
  }

  // A signal that aborts the request after TIMEOUT_MS (none on old browsers).
  function timeoutSignal() {
    return typeof AbortSignal !== 'undefined' && typeof AbortSignal.timeout === 'function'
      ? AbortSignal.timeout(TIMEOUT_MS) : undefined;
  }

  // Reads the public events of user: up to 3 pages, newest first. A failure
  // is rejected as {status, remaining, reset}; status 0 means no answer at all.
  async function fetchEvents(user) {
    var all = [];
    for (var page = 1; page <= PAGES; page++) {
      var res;
      var batch;
      try {
        res = await fetch(API + encodeURIComponent(user) + '/events/public?per_page=100&page=' + page,
          { headers: { Accept: 'application/vnd.github+json' }, signal: timeoutSignal() });
        if (res.status === 422 && page > 1) break; // past the last page
        if (!res.ok) {
          throw { status: res.status, remaining: res.headers.get('X-RateLimit-Remaining'), reset: res.headers.get('X-RateLimit-Reset') };
        }
        batch = await res.json();
        if (!Array.isArray(batch)) throw { status: 502 }; // a 200 that is not a list of events
      } catch (e) {
        throw e && typeof e.status === 'number' ? e : { status: 0 };
      }
      all = all.concat(batch);
      if (batch.length < 100) break;
    }
    return all;
  }

  // The preview takes the shape of the card that is asked for: 480x200 or the
  // 200x60 badge. Called when the size changes, even before anything is drawn.
  function sizeStage() {
    var compact = checked('size') === 'compact';
    stage.dataset.size = compact ? 'compact' : 'full';
    img.width = compact ? 200 : 480;
    img.height = compact ? 60 : 200;
  }

  // Puts an SVG on the stage as an image (never as markup) and frees the previous one.
  function paint(svg) {
    var url = URL.createObjectURL(new Blob([svg], { type: 'image/svg+xml' }));
    sizeStage();
    img.src = url;
    if (blobURL) URL.revokeObjectURL(blobURL);
    blobURL = url;
    img.hidden = false;
    empty.hidden = true;
    return url;
  }

  function show(result, user, noActivity) {
    var url = paint(result.svg);
    img.alt = 'commitling de @' + user;
    caption.textContent = '@' + user + ': ' + result.description;
    // The file is the SVG on screen; user has passed commitling.check, so the
    // name only holds letters, digits and dashes.
    dl.href = url;
    dl.download = 'commitling-' + user + '.svg';
    var w = window.commitling.workflow(user, checked('species'), checked('theme'), checked('size'));
    wf.textContent = w.error ? '' : w.workflow;
    after.hidden = !!w.error;
    say(noActivity
      ? '@' + user + ' no tiene actividad pública en los últimos 90 días: su criatura duerme.'
      : 'Listo: esta es la criatura de @' + user + '.');
  }

  function redraw() {
    var r = window.commitling.render(JSON.stringify(current.events), current.user, checked('species'), checked('theme'), 0, checked('size'));
    if (r.error) {
      fail(r.error);
      return;
    }
    show(r, current.user, current.events.length === 0);
  }

  async function draw() {
    if (busy) return;
    var user = userInput.value.trim().replace(/^@/, '');
    if (!user) {
      fail('Escribe tu usuario de GitHub.');
      userInput.focus();
      return;
    }
    stopDemo(false);
    setBusy(true);
    say('Cargando el generador…');
    try {
      try {
        await ensureWasm();
      } catch (e) {
        fail('No se pudo cargar el generador. Comprueba tu conexión e inténtalo de nuevo.');
        return;
      }
      var invalid = window.commitling.check(user, checked('species'), checked('theme'), checked('size'));
      if (invalid) {
        fail(invalid);
        userInput.focus();
        return;
      }
      var events;
      if (current && current.user.toLowerCase() === user.toLowerCase() && Date.now() - current.at < REUSE_MS) {
        events = current.events; // same user a moment ago: nothing new to ask GitHub
      } else {
        say('Buscando la actividad pública de @' + user + '…');
        try {
          events = await fetchEvents(user);
        } catch (f) {
          fail(window.commitling.explain(f.status, f.remaining, f.reset));
          return;
        }
        current = { user: user, events: events, at: Date.now() };
      }
      current.user = user;
      redraw();
    } finally {
      setBusy(false);
    }
  }


  // The demo: 90 fictitious days drawn by commitling.demo, one card per day,
  // painted in turn. No network. The day comes from the elapsed time, so a slow
  // tab skips cards instead of slowing the story down.
  function reducedMotion() {
    return typeof matchMedia === 'function' && matchMedia('(prefers-reduced-motion: reduce)').matches;
  }

  function demoButtons() {
    var still = reducedMotion(); // with reduced motion there is no timelapse to pause
    demoPause.disabled = !demo || still;
    demoStop.disabled = !demo;
    demoPause.textContent = !demo || still || demo.playing ? 'Pausa' : demo.day >= demo.days ? 'Repetir' : 'Reanudar';
  }

  function frame(day) {
    var r = window.commitling.demo(day, checked('species'), checked('theme'), checked('size'));
    if (r.error) {
      stopDemo(false);
      fail(r.error);
      return;
    }
    paint(r.svg);
    img.alt = 'Demo de commitling: una criatura de ejemplo';
    demo.day = r.day;
    demo.days = r.days;
    demoRange.max = r.days;
    demoRange.value = r.day;
    demoDay.textContent = 'Día ' + r.day + ' de ' + r.days;
    caption.textContent = 'Demo con @' + r.login + ' (usuario ficticio), día ' + r.day + ': ' + r.description;
    // Only a new phase is announced; the rest of the text is not a live region.
    if (r.phase !== demo.phase) {
      demo.phase = r.phase;
      demoPhase.textContent = 'Fase: ' + r.phase + ' (día ' + r.day + ')';
    }
  }

  function tick(now) {
    if (!demo || !demo.playing) return;
    if (demo.t0 === null) demo.t0 = now - demo.elapsed;
    demo.elapsed = now - demo.t0;
    // The epsilon keeps a slider position from rounding down a day when playback resumes.
    var day = Math.min(demo.days, Math.floor(demo.elapsed / DEMO_MS * demo.days + 1e-6));
    if (day !== demo.day) frame(day);
    if (!demo) return;
    if (demo.day >= demo.days) {
      demo.playing = false;
      demoButtons();
      return;
    }
    demo.raf = requestAnimationFrame(tick);
  }

  function play() {
    if (demo.day >= demo.days) {
      demo.elapsed = 0;
      frame(0);
    }
    demo.playing = true;
    demo.t0 = null;
    demoButtons();
    demo.raf = requestAnimationFrame(tick);
  }

  function pause() {
    demo.playing = false;
    cancelAnimationFrame(demo.raf);
    demoButtons();
  }

  // Leaves the demo. With restore, the stage goes back to what the person had.
  function stopDemo(restore) {
    if (!demo) return;
    cancelAnimationFrame(demo.raf);
    demo = null;
    // A button that is about to be disabled must not keep the keyboard focus.
    if (document.activeElement === demoStop || document.activeElement === demoPause) demoGo.focus();
    demoSlider.hidden = true;
    demoDay.textContent = '';
    demoPhase.textContent = '';
    demoButtons();
    if (!restore) return;
    if (current) {
      redraw();
    } else {
      if (blobURL) URL.revokeObjectURL(blobURL);
      blobURL = '';
      img.removeAttribute('src');
      img.hidden = true;
      empty.hidden = false;
      caption.textContent = '';
      after.hidden = true;
      say('Demo detenida.');
    }
  }

  async function startDemo() {
    if (busy) return;
    setBusy(true);
    say('Cargando la demo…');
    try {
      await ensureWasm();
    } catch (e) {
      fail('No se pudo cargar la demo. Comprueba tu conexión e inténtalo de nuevo.');
      return;
    } finally {
      setBusy(false);
    }
    stopDemo(false);
    say('');
    after.hidden = true;
    demo = { day: -1, days: 0, phase: '', playing: false, elapsed: 0, t0: null, raf: 0 };
    demoSlider.hidden = false;
    frame(0); // fills in demo.days from the wasm
    if (!demo) return; // frame stopped the demo on an error
    if (reducedMotion()) {
      demoButtons();
      demoRange.focus();
    } else {
      play();
    }
  }

  demoGo.addEventListener('click', startDemo);
  demoPause.addEventListener('click', function () {
    if (!demo) return;
    if (demo.playing) pause(); else play();
  });
  demoStop.addEventListener('click', function () { stopDemo(true); });
  demoRange.addEventListener('input', function () {
    if (!demo) return;
    if (demo.playing) pause();
    demo.elapsed = Number(demoRange.value) / demo.days * DEMO_MS;
    frame(Number(demoRange.value));
    demoButtons();
  });
  // A tab nobody is looking at does not keep playing (and does not jump ahead on return).
  document.addEventListener('visibilitychange', function () {
    if (document.hidden && demo && demo.playing) pause();
  });
  heroDemo.hidden = false;
  heroDemo.addEventListener('click', startDemo);

  form.addEventListener('submit', function (e) {
    e.preventDefault();
    draw();
  });

  // Start downloading as soon as someone touches or types in the form (submit
  // and change load it too); merely tabbing through the page does not.
  function warmUp() {
    ensureWasm().catch(function () {});
  }
  form.addEventListener('pointerdown', warmUp);
  form.addEventListener('input', warmUp);

  // Species, theme and size redraw what is already on screen, without asking GitHub again.
  form.addEventListener('change', function (e) {
    if (e.target.type !== 'radio') return;
    sizeStage();
    if (busy) return;
    ensureWasm().then(function () {
      if (demo) frame(demo.day);
      else if (current) redraw();
    }, function () {
      fail('No se pudo cargar el generador. Comprueba tu conexión e inténtalo de nuevo.');
    });
  });
})();
