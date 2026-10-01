// Behaviour test of generator.js, run by Go (generator_limit_test.go) or by hand:
//
//   node internal/gallery/testdata/generator.test.js internal/gallery/generator.js
//
// The script is loaded into a fake page: a DOM that answers to what the
// script touches, a fetch that counts the requests to api.github.com, a
// sessionStorage shared between "page loads" and a stand-in for the wasm. No
// dependencies: only node:fs, node:vm and node:assert.
'use strict';
const fs = require('node:fs');
const vm = require('node:vm');
const assert = require('node:assert/strict');

const source = fs.readFileSync(process.argv[2] || 'generator.js', 'utf8');
const MIN = 60 * 1000;

function element(id) {
  const listeners = {};
  const attrs = {};
  const el = {
    id, hidden: false, disabled: false, textContent: '', value: '', dataset: {}, style: {}, children: [],
    addEventListener(type, fn) { (listeners[type] = listeners[type] || []).push(fn); },
    async fire(type, ev) { for (const fn of listeners[type] || []) await fn(Object.assign({ preventDefault() {}, target: el }, ev)); },
    setAttribute(k, v) { attrs[k] = String(v); },
    getAttribute(k) { return attrs[k] === undefined ? null : attrs[k]; },
    removeAttribute(k) { delete attrs[k]; },
    focus() { el.focused = true; },
    appendChild(c) { el.children.push(c); if (c.tagName === 'SCRIPT') queueMicrotask(() => c.onload()); return c; },
  };
  return el;
}

// A page load. storage is the sessionStorage (shared by the loads of a tab).
function loadPage(storage, opts = {}) {
  const els = {};
  const byId = (id) => (els[id] = els[id] || element(id));
  const requests = { api: [], wasm: 0 };
  const calls = { render: [], demo: [], check: [] };
  const form = byId('gen-form');
  form.elements = { species: { value: 'moss' }, theme: { value: 'light' }, size: { value: 'full' } };
  byId('gen').hidden = true;
  byId('demo-slider').hidden = true;
  byId('gen-demo-alt').hidden = true; // as in the markup

  const page = {
    doc: null,
    api: opts.api || (() => ({ status: 200, body: [] })),
    requests, calls, els, form,
    user: byId('gen-user'),
    status: () => byId('gen-status').textContent,
    error: () => byId('gen-error').textContent,
    alt: byId('gen-demo-alt'),
  };

  const sandbox = {
    console, JSON, Date, Promise, Math, Array, Object, Number, String, Error, Boolean, RegExp, Blob: class {}, encodeURIComponent, queueMicrotask,
    URL: { createObjectURL: () => 'blob:fake', revokeObjectURL() {} },
    document: {
      getElementById: byId,
      createElement: (tag) => ({ tagName: tag.toUpperCase() }),
      head: element('head'),
      addEventListener() {},
      hidden: false,
      activeElement: null,
    },
    requestAnimationFrame: () => 1,
    cancelAnimationFrame() {},
    WebAssembly: { instantiate: async () => ({ instance: {} }) },
    Go: class {
      constructor() { this.importObject = {}; }
      run() { sandbox.window.commitling = stub(); return Promise.resolve(); }
    },
    fetch: async (url) => {
      if (String(url).includes('commitling.wasm')) {
        requests.wasm++;
        return { ok: true, status: 200, arrayBuffer: async () => new ArrayBuffer(1) };
      }
      requests.api.push(String(url));
      const r = page.api(String(url), requests.api.length);
      return {
        ok: r.status >= 200 && r.status < 300,
        status: r.status,
        headers: { get: (h) => (r.headers && r.headers[h]) || null },
        json: async () => r.body,
      };
    },
  };
  Object.defineProperty(sandbox, 'sessionStorage', {
    get() {
      if (opts.storageThrows) throw new Error('SecurityError');
      return storage;
    },
  });
  sandbox.window = sandbox;

  function stub() {
    return {
      check(user) { calls.check.push(user); return ''; },
      render(json, user, species, theme, now, size) {
        calls.render.push({ user, species, theme, size, events: JSON.parse(json) });
        if (JSON.parse(json).some((e) => e.id === 'bad')) return { error: 'EVENTOS NO VALIDOS' };
        return { svg: '<svg/>', description: 'descripción', login: user };
      },
      workflow() { return { workflow: 'yaml' }; },
      explain(status, remaining, reset) { return 'EXPLAIN ' + status + ' ' + remaining + ' ' + reset; },
      demo(day) {
        calls.demo.push(day);
        return { svg: '<svg/>', description: 'd', login: 'octoexample', day, days: 90, phase: 'Brote' };
      },
    };
  }

  page.doc = sandbox.document;
  vm.runInNewContext(source, sandbox);
  return page;
}

function memoryStorage(initial) {
  const data = Object.assign({}, initial);
  return {
    data,
    getItem: (k) => (k in data ? data[k] : null),
    setItem(k, v) { data[k] = String(v); },
    removeItem(k) { delete data[k]; },
    get length() { return Object.keys(data).length; },
    key: (i) => Object.keys(data)[i] || null,
  };
}

const wait = (ms) => new Promise((r) => setTimeout(r, ms));

async function submit(page, user) {
  page.user.value = user;
  await page.form.fire('submit');
  for (let i = 0; i < 50 && page.form.getAttribute('aria-busy') !== 'false'; i++) await wait(2);
}

const push = (id) => ({ id: String(id), type: 'PushEvent', created_at: '2026-09-30T10:00:00Z', actor: { login: 'octo' }, repo: { name: 'octo/x' }, payload: { size: 2 } });
const KEY = (u) => 'commitling:events:' + u;
const ok = (body) => () => ({ status: 200, body: body });

const tests = {
  async 'a name that is not a GitHub login never reaches the network or the wasm'() {
    for (const bad of ['a b', '-octo', 'x'.repeat(40), 'ñandú', 'a/b', '../etc', 'octo?x=1', 'a_b', 'a.b', 'a\nb', 'octo cat']) {
      const p = loadPage(memoryStorage());
      await submit(p, bad);
      assert.equal(p.requests.api.length, 0, JSON.stringify(bad) + ' must not be requested');
      assert.equal(p.requests.wasm, 0, JSON.stringify(bad) + ' must not even load the wasm');
      assert.match(p.error(), /usuario/, bad);
    }
    const p = loadPage(memoryStorage());
    await submit(p, '');
    assert.match(p.error(), /Escribe tu usuario/);
    assert.equal(p.requests.api.length, 0);
  },

  async 'a valid name, with or without @ or spaces, is requested once'() {
    for (const name of ['octo', '@octo', '  octo  ', 'a', '0', 'x'.repeat(39), 'Octo-Cat', 'octo-']) {
      const p = loadPage(memoryStorage(), { api: ok([push(1)]) });
      await submit(p, name);
      assert.equal(p.requests.api.length, 1, name);
      assert.equal(p.calls.render.length, 1, name);
    }
  },

  async 'changing species, theme or size, or drawing again, costs no request'() {
    const p = loadPage(memoryStorage(), { api: ok([push(1)]) });
    await submit(p, 'octo');
    assert.equal(p.requests.api.length, 1);
    for (const [name, value] of [['theme', 'dark'], ['species', 'mushroom'], ['size', 'compact']]) {
      p.form.elements[name].value = value;
      await p.form.fire('change', { target: { type: 'radio' } });
      await wait(2);
    }
    await submit(p, 'octo');
    await submit(p, 'OCTO');
    assert.equal(p.requests.api.length, 1, 'the events of the same user are reused');
    const last = p.calls.render[p.calls.render.length - 1];
    assert.deepEqual([last.theme, last.species, last.size], ['dark', 'mushroom', 'compact']);
    assert.ok(p.calls.render.length >= 5);
  },

  async 'the events are cached in sessionStorage and survive a reload'() {
    const storage = memoryStorage();
    const first = loadPage(storage, { api: ok([push(1), push(2)]) });
    await submit(first, 'Octo');
    const saved = JSON.parse(storage.data[KEY('octo')]);
    assert.equal(saved.events.length, 2);
    assert.ok(Math.abs(saved.at - Date.now()) < MIN);
    const second = loadPage(storage, { api: () => assert.fail('the cache must be used') });
    await submit(second, 'octo');
    assert.equal(second.requests.api.length, 0);
    assert.equal(second.calls.render[0].events.length, 2);
    // Another user is another entry.
    const third = loadPage(storage, { api: ok([]) });
    await submit(third, 'other');
    assert.equal(third.requests.api.length, 1);
    assert.ok(storage.data[KEY('other')]);
  },

  async 'an old or broken cache entry is asked for again and replaced'() {
    for (const stale of [
      JSON.stringify({ at: Date.now() - 11 * MIN, events: [push(9)] }), // older than the cache lives
      JSON.stringify({ at: Date.now() + 60 * MIN, events: [push(9)] }), // from the future: a clock that went back
      'not json', '{"at":1}', '{"events":[]}', 'null', JSON.stringify({ at: Date.now(), events: 'x' }),
    ]) {
      const storage = memoryStorage({ [KEY('octo')]: stale });
      const p = loadPage(storage, { api: ok([push(1)]) });
      await submit(p, 'octo');
      assert.equal(p.requests.api.length, 1, stale.slice(0, 30));
      assert.equal(JSON.parse(storage.data[KEY('octo')]).events[0].id, '1', 'the entry is replaced by the fresh one');
    }
    // A recent one is used.
    const storage = memoryStorage({ [KEY('octo')]: JSON.stringify({ at: Date.now() - 9 * MIN, events: [push(7)] }) });
    const p = loadPage(storage, { api: () => assert.fail('9 minutes is still fresh') });
    await submit(p, 'octo');
    assert.equal(p.requests.api.length, 0);
  },

  async 'a failed search is never cached'() {
    const storage = memoryStorage();
    const p = loadPage(storage, { api: () => ({ status: 404 }) });
    await submit(p, 'ghost');
    assert.equal(storage.length, 0);
    assert.match(p.error(), /EXPLAIN 404/);
    assert.equal(p.alt.hidden, true, 'the demo is offered for the rate limit, not for a 404');
  },

  async 'without a usable sessionStorage everything still works'() {
    const throwing = loadPage(null, { storageThrows: true, api: ok([push(1)]) });
    await submit(throwing, 'octo');
    assert.equal(throwing.calls.render.length, 1);
    await submit(throwing, 'octo');
    assert.equal(throwing.requests.api.length, 1, 'still reused from memory');

    const full = memoryStorage();
    full.setItem = () => { throw new Error('QuotaExceededError'); };
    const p = loadPage(full, { api: ok([push(1)]) });
    await submit(p, 'octo');
    assert.equal(p.calls.render.length, 1);
    assert.equal(p.error(), '');
  },

  async 'when the storage is full, the other users are dropped to make room'() {
    const storage = memoryStorage({ [KEY('old')]: JSON.stringify({ at: Date.now(), events: [push(5)] }), unrelated: 'keep' });
    const setItem = storage.setItem;
    let first = true;
    storage.setItem = (k, v) => {
      if (first && k === KEY('octo')) { first = false; throw new Error('QuotaExceededError'); }
      setItem(k, v);
    };
    const p = loadPage(storage, { api: ok([push(1)]) });
    await submit(p, 'octo');
    assert.ok(storage.data[KEY('octo')], 'saved after making room');
    assert.equal(storage.data[KEY('old')], undefined);
    assert.equal(storage.data.unrelated, 'keep', 'what is not ours is not touched');
  },

  async 'the rate limit shows the explanation, offers the demo and does not cache'() {
    const reset = String(Math.floor(Date.now() / 1000) + 17 * 60);
    const storage = memoryStorage();
    const p = loadPage(storage, { api: () => ({ status: 403, headers: { 'X-RateLimit-Remaining': '0', 'X-RateLimit-Reset': reset } }) });
    await submit(p, 'octo');
    assert.equal(p.error(), 'EXPLAIN 403 0 ' + reset, 'the wasm gets the raw headers to write the message');
    assert.equal(p.alt.hidden, false, '«Ver demo» is offered');
    assert.equal(storage.length, 0);
    // It starts the same demo as the other button, without touching the network.
    await p.alt.fire('click');
    await wait(5);
    assert.ok(p.calls.demo.length >= 1, 'the demo runs');
    assert.equal(p.requests.api.length, 1, 'and asks GitHub for nothing');
    assert.equal(p.error(), '', 'the alert is cleared');
    assert.equal(p.alt.hidden, true);
  },

  async 'with the keyboard on «Ver demo» of the notice, the focus moves to the demo button, not to <body>'() {
    const p = loadPage(memoryStorage(), { api: () => ({ status: 403 }) });
    await submit(p, 'octo');
    assert.equal(p.alt.hidden, false);
    p.doc.activeElement = p.alt; // the button that has the focus is about to be hidden
    await p.alt.fire('click');
    await wait(5);
    assert.equal(p.alt.hidden, true);
    assert.equal(p.els['demo-go'].focused, true, 'the focus goes to the demo button');
  },

  async 'what came from the cache and does not draw is dropped from it'() {
    const storage = memoryStorage({ [KEY('octo')]: JSON.stringify({ at: Date.now(), events: [{ id: 'bad' }] }) });
    const p = loadPage(storage, { api: () => assert.fail('it came from the cache') });
    await submit(p, 'octo');
    assert.equal(p.error(), 'EVENTOS NO VALIDOS');
    assert.equal(storage.data[KEY('octo')], undefined, 'the broken entry is removed');
    // And the next try asks GitHub.
    const again = loadPage(storage, { api: ok([push(1)]) });
    await submit(again, 'octo');
    assert.equal(again.requests.api.length, 1);
    assert.equal(again.error(), '');
    // A fresh search that fails to draw is not what the cache is for either: nothing was cached from the cache.
    const storage2 = memoryStorage();
    const fresh = loadPage(storage2, { api: ok([{ id: 'bad' }]) });
    await submit(fresh, 'octo');
    assert.equal(fresh.error(), 'EVENTOS NO VALIDOS');
  },

  async '429 is a rate limit too, and a later success hides the offer'() {
    let n = 0;
    const p = loadPage(memoryStorage(), { api: () => (++n === 1 ? { status: 429 } : { status: 200, body: [] }) });
    await submit(p, 'octo');
    assert.equal(p.alt.hidden, false);
    await submit(p, 'octo');
    assert.equal(p.alt.hidden, true);
    assert.equal(p.error(), '');
  },

  async 'a server error does not offer the demo'() {
    for (const status of [500, 503]) {
      const p = loadPage(memoryStorage(), { api: () => ({ status }) });
      await submit(p, 'octo');
      assert.equal(p.alt.hidden, true, String(status));
    }
  },
};

(async () => {
  let failed = 0;
  for (const [name, fn] of Object.entries(tests)) {
    try {
      await fn();
      console.log('ok   ' + name);
    } catch (e) {
      failed++;
      console.log('FAIL ' + name + '\n     ' + String(e.stack || e).split('\n').slice(0, 6).join('\n     '));
    }
  }
  if (failed) {
    console.log(failed + ' failed');
    process.exit(1);
  }
})();
