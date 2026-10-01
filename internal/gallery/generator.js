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
  var form = document.getElementById('gen-form');
  var userInput = document.getElementById('gen-user');
  var button = document.getElementById('gen-go');
  var statusEl = document.getElementById('gen-status');
  var errorEl = document.getElementById('gen-error');
  var img = document.getElementById('gen-img');
  var empty = document.getElementById('gen-empty');
  var caption = document.getElementById('gen-caption');
  var after = document.getElementById('gen-after');
  var dl = document.getElementById('gen-dl');
  var wf = document.getElementById('gen-wf');

  var busy = false;
  var current = null; // {user, events, at} of the last successful search
  var blobURL = '';

  root.hidden = false;

  if (typeof WebAssembly !== 'object' || typeof fetch !== 'function') {
    errorEl.textContent = 'Tu navegador no admite WebAssembly, que el generador necesita. Prueba con una versión reciente de Firefox, Chrome, Safari o Edge.';
    button.setAttribute('aria-disabled', 'true');
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

  function instantiate(go) {
    var url = 'commitling.wasm';
    var viaBuffer = function () {
      return fetch(url).then(function (r) {
        if (!r.ok) throw new Error('commitling.wasm: ' + r.status);
        return r.arrayBuffer();
      }).then(function (b) { return WebAssembly.instantiate(b, go.importObject); });
    };
    if (!WebAssembly.instantiateStreaming) return viaBuffer();
    // Falls back to a plain download when the server does not send application/wasm.
    return WebAssembly.instantiateStreaming(fetch(url), go.importObject).catch(viaBuffer);
  }

  function ensureWasm() {
    if (!loading) {
      loading = loadScript('wasm_exec.js').then(function () {
        var go = new Go();
        return instantiate(go).then(function (result) {
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

  function show(result, user, noActivity) {
    var url = URL.createObjectURL(new Blob([result.svg], { type: 'image/svg+xml' }));
    img.src = url;
    if (blobURL) URL.revokeObjectURL(blobURL);
    blobURL = url;
    img.alt = 'commitling de @' + user;
    img.hidden = false;
    empty.hidden = true;
    caption.textContent = '@' + user + ': ' + result.description;
    // The file is the SVG on screen; user has passed commitling.check, so the
    // name only holds letters, digits and dashes.
    dl.href = url;
    dl.download = 'commitling-' + user + '.svg';
    var w = window.commitling.workflow(user, checked('species'), checked('theme'));
    wf.textContent = w.error ? '' : w.workflow;
    after.hidden = !!w.error;
    say(noActivity
      ? '@' + user + ' no tiene actividad pública en los últimos 90 días: su criatura duerme.'
      : 'Listo: esta es la criatura de @' + user + '.');
  }

  function redraw() {
    var r = window.commitling.render(JSON.stringify(current.events), current.user, checked('species'), checked('theme'));
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
    setBusy(true);
    say('Cargando el generador…');
    try {
      try {
        await ensureWasm();
      } catch (e) {
        fail('No se pudo cargar el generador. Comprueba tu conexión e inténtalo de nuevo.');
        return;
      }
      var invalid = window.commitling.check(user, checked('species'), checked('theme'));
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

  form.addEventListener('submit', function (e) {
    e.preventDefault();
    draw();
  });

  // Start downloading as soon as someone types in the form (submit and change
  // load it too); merely tabbing through the page does not.
  form.addEventListener('input', function () {
    ensureWasm().catch(function () {});
  });

  // Species and theme redraw what is already on screen, without asking GitHub again.
  form.addEventListener('change', function (e) {
    if (e.target.type !== 'radio' || busy) return;
    ensureWasm().then(function () {
      if (current) redraw();
    }, function () {
      fail('No se pudo cargar el generador. Comprueba tu conexión e inténtalo de nuevo.');
    });
  });
})();
