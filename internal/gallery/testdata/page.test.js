// Behaviour test of page.js. Run with:
//   node internal/gallery/testdata/page.test.js internal/gallery/page.js
'use strict';
const fs = require('node:fs');
const vm = require('node:vm');
const assert = require('node:assert/strict');

function node(attrs, parent) {
  const values = Object.assign({}, attrs);
  return {
    id: values.id || '', value: values.value, textContent: values.text || '', hidden: false,
    checked: Boolean(values.checked), parent, listeners: {},
    getAttribute(name) { return Object.prototype.hasOwnProperty.call(values, name) ? values[name] : null; },
    setAttribute(name, value) { values[name] = String(value); },
    addEventListener(name, fn) { this.listeners[name] = fn; },
    fire(name, event) { if (this.listeners[name]) this.listeners[name].call(this, event || {}); },
    closest(selector) { return selector === '.species-block' ? this.parent : null; }
  };
}

function run(withPicker) {
  const source = fs.readFileSync(process.argv[2] || 'page.js', 'utf8');
  const blocks = [];
  const targets = {};
  const links = [];
  const radios = [];
  const labels = {};
  const picker = withPicker ? node({ id: 'species-pick' }) : null;
  const message = node({ id: 'species-pick-msg' });
  if (withPicker) {
    ['all', 'moss', 'mushroom', 'cactus', 'pumpkin'].forEach((value, i) => {
      const radio = node({ id: 'gal-sp-' + value, value, checked: i === 0 }, picker);
      radios.push(radio);
      labels[radio.id] = node({ text: value === 'all' ? 'Todas' : ({ moss: 'Brote de musgo', mushroom: 'Hongo', cactus: 'Cactus', pumpkin: 'Calabaza' })[value] });
    });
    ['moss', 'mushroom', 'cactus', 'pumpkin'].forEach((value) => {
      const block = node({ 'data-species': value });
      blocks.push(block);
      const stage = node({ id: 'especie-' + value }, block);
      const accessories = node({ id: 'accesorios-' + value }, block);
      targets[stage.id] = stage;
      targets[accessories.id] = accessories;
      links.push(node({ href: '#especie-' + value }));
    });
    // The compact badges of each species are a second block per species.
    ['moss', 'mushroom', 'cactus', 'pumpkin'].forEach((value) => blocks.push(node({ 'data-species': value })));
    picker.querySelectorAll = () => radios;
  }
  const document = {
    getElementById(id) { return id === 'species-pick' ? picker : id === 'species-pick-msg' ? message : targets[id] || null; },
    querySelectorAll(selector) {
      if (selector === '.copy') return [];
      if (selector === '#galeria [data-species]') return blocks;
      if (selector === 'nav.jump a[href^="#especie-"]') return links;
      return [];
    },
    querySelector(selector) {
      const match = /^label\[for="([^"]+)"\]$/.exec(selector);
      return match ? labels[match[1]] : null;
    }
  };
  const window = { location: { hash: '' }, listeners: {}, addEventListener(name, fn) { this.listeners[name] = fn; } };
  vm.runInNewContext(source, { document, window, navigator: {}, setTimeout, console });
  return { picker, radios, blocks, message, links, targets, window };
}

function ok(name, fn) { fn(); console.log('ok ' + name); }

ok('sin interacción todo visible', () => {
  const r = run(true);
  assert.equal(r.picker.hidden, false);
  assert.equal(r.message.textContent, '');
  assert.ok(r.blocks.every((block) => !block.hidden));
});
ok('elegir una especie oculta las demás', () => {
  const r = run(true);
  r.radios[2].fire('change');
  assert.ok(r.blocks.every((block) => block.hidden === (block.getAttribute('data-species') !== 'mushroom')));
  assert.equal(r.blocks.filter((block) => !block.hidden).length, 2);
  assert.match(r.message.textContent, /^Se muestra solo: Hongo/);
});
ok('volver a todas lo enseña todo', () => {
  const r = run(true); r.radios[1].fire('change'); r.radios[0].fire('change');
  assert.ok(r.blocks.every((block) => !block.hidden));
  assert.equal(r.message.textContent, 'Se muestran todas las especies.');
});
ok('valor fuera de la lista equivale a todas', () => {
  const r = run(true); r.radios[1].value = 'not-a-species'; r.radios[1].fire('change');
  assert.ok(r.blocks.every((block) => !block.hidden));
  assert.equal(r.radios[0].checked, true);
});
ok('clic en Calabaza selecciona sin cancelar el ancla', () => {
  const r = run(true); r.radios[2].fire('change'); let prevented = false;
  r.links[3].fire('click', { preventDefault() { prevented = true; } });
  assert.equal(prevented, false); assert.equal(r.radios[4].checked, true); assert.equal(r.blocks[3].hidden, false);
});
ok('hashchange selecciona accesorios ocultos', () => {
  const r = run(true); r.radios[2].fire('change'); r.window.location.hash = '#accesorios-pumpkin'; r.window.listeners.hashchange();
  assert.equal(r.radios[4].checked, true); assert.equal(r.blocks[3].hidden, false);
});
ok('sin selector no falla', () => { run(false); });
