// Behaviour test of theme.js. Run with:
//   node internal/gallery/testdata/theme.test.js internal/gallery/theme.js
'use strict';
const fs = require('node:fs');
const vm = require('node:vm');
const assert = require('node:assert/strict');

function element(attrs) {
  const values = Object.assign({}, attrs);
  return {
    checked: false,
    hidden: true,
    listeners: {},
    getAttribute(name) { return Object.prototype.hasOwnProperty.call(values, name) ? values[name] : null; },
    setAttribute(name, value) { values[name] = String(value); },
    removeAttribute(name) { delete values[name]; },
    addEventListener(name, fn) { this.listeners[name] = fn; },
    fire(name) { if (this.listeners[name]) this.listeners[name].call(this); },
    value: values.value,
    get media() { return values.media; }
  };
}

function run(saved, broken) {
  const source = fs.readFileSync(process.argv[2] || 'theme.js', 'utf8');
  const root = element({});
  const sources = [element({ media: '(prefers-color-scheme: dark)' })];
  const metas = [element({ media: '(prefers-color-scheme: light)' }), element({ media: '(prefers-color-scheme: dark)' })];
  const radios = ['system', 'light', 'dark'].map((value) => element({ value }));
  const pick = { hidden: true, querySelectorAll() { return radios; } };
  const store = {
    data: Object.assign({}, saved),
    getItem(key) { if (broken) throw new Error('blocked'); return Object.prototype.hasOwnProperty.call(this.data, key) ? this.data[key] : null; },
    setItem(key, value) { if (broken) throw new Error('blocked'); this.data[key] = value; },
    removeItem(key) { if (broken) throw new Error('blocked'); delete this.data[key]; }
  };
  const document = {
    readyState: 'complete', documentElement: root,
    querySelectorAll(selector) { return selector.indexOf('source') >= 0 ? sources : metas; },
    getElementById() { return pick; }, addEventListener() {}
  };
  const sandbox = { console, document, window: { localStorage: store } };
  vm.runInNewContext(source, sandbox);
  return { api: sandbox.window.commitlingTheme, root, sources, metas, radios, pick, store };
}

function ok(name, fn) { fn(); console.log('ok ' + name); }

ok('elección guardada fuerza el tema', () => {
  const r = run({ 'commitling:theme': 'dark' });
  assert.equal(r.api.read(), 'dark');
  assert.equal(r.root.getAttribute('data-theme'), 'dark');
  assert.equal(r.sources[0].getAttribute('media'), 'all');
  assert.equal(r.metas[1].getAttribute('media'), 'all');
  assert.equal(r.metas[0].getAttribute('media'), 'not all');
  assert.equal(r.radios[2].checked, true);
  assert.equal(r.pick.hidden, false);
});
ok('valor inválido se ignora y se borra', () => {
  for (const value of ['<script>', 'blue', '', 'constructor', '__proto__', 'Dark', 'system']) {
    const r = run({ 'commitling:theme': value });
    assert.equal(r.api.read(), 'system');
    assert.equal(r.store.data['commitling:theme'], undefined);
    assert.equal(r.root.getAttribute('data-theme'), null);
  }
});
ok('claro, oscuro y sistema', () => {
  const r = run({});
  r.api.choose('light');
  assert.equal(r.store.data['commitling:theme'], 'light');
  assert.equal(r.sources[0].getAttribute('media'), 'not all');
  r.api.choose('dark');
  assert.equal(r.store.data['commitling:theme'], 'dark');
  r.api.choose('system');
  assert.equal(r.store.data['commitling:theme'], undefined);
  assert.equal(r.sources[0].getAttribute('media'), '(prefers-color-scheme: dark)');
  assert.equal(r.metas[0].getAttribute('media'), '(prefers-color-scheme: light)');
});
ok('almacenamiento bloqueado no rompe el cambio', () => {
  const r = run({}, true);
  r.api.choose('dark');
  assert.equal(r.root.getAttribute('data-theme'), 'dark');
  assert.equal(r.sources[0].getAttribute('media'), 'all');
  assert.deepEqual(Object.keys(r.store.data), []);
});
ok('solo se escribe la clave del tema', () => {
  const r = run({ other: 'keep' });
  r.api.choose('light');
  assert.deepEqual(r.store.data, { other: 'keep', 'commitling:theme': 'light' });
});
ok('al pulsar un radio se aplica y se guarda', () => {
  const r = run({});
  r.radios[2].fire('change');
  assert.equal(r.root.getAttribute('data-theme'), 'dark');
  assert.equal(r.store.data['commitling:theme'], 'dark');
  r.radios[0].fire('change');
  assert.equal(r.root.getAttribute('data-theme'), null);
  assert.equal(r.store.data['commitling:theme'], undefined);
});
