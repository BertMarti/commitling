// Behaviour test of install.js (the workflow generator of «Instálalo en tu
// perfil»), run by Go (install_test.go) or by hand:
//
//   node internal/gallery/testdata/install.test.js internal/gallery/install.js [--dump]
//
// With --dump it prints, as JSON, what build() gives for every option, and Go
// compares it with generate.Install. No dependencies: node:fs, node:vm, node:assert.
'use strict';
const fs = require('node:fs');
const vm = require('node:vm');
const assert = require('node:assert/strict');

(function main() {
const source = fs.readFileSync(process.argv[2] || 'install.js', 'utf8');
const sandbox = { console, encodeURIComponent, window: {} };
vm.runInNewContext(source, sandbox);
const api = sandbox.window.commitlingInstall;
assert.ok(api);

const users = ['', 'octocat', '007', 'a-b-9'];
const species = ['moss', 'mushroom'];
const themes = ['light', 'dark', 'both'];
const sizes = ['full', 'compact'];

if (process.argv[3] === '--dump') {
  const out = [];
  for (const user of users) for (const specie of species) for (const theme of themes) for (const size of sizes) {
    const result = api.build({ user, species: specie, theme, size });
    out.push({ user, species: specie, theme, size, workflow: result.workflow, readme: result.readme, files: result.files });
  }
  // No process.exit(): on a pipe, stdout is asynchronous and exiting at
  // once cuts the JSON (it happened in the CI). The scenarios below are skipped.
  process.stdout.write(JSON.stringify(out));
  return;
}

function ok(name, fn) { fn(); console.log('ok ' + name); }

ok('workflow por defecto', () => {
  const r = api.build({});
  assert.equal(r.readme, '![commitling](./commitling.svg)');
  assert.deepEqual(Array.from(r.files), ['commitling.svg']);
  assert.match(r.workflow, /^name: commitling/m);
  assert.ok(r.workflow.includes('out: commitling.svg'));
});
ok('parámetros seguros y usuario entrecomillado', () => {
  const w = api.build({ user: '007', species: 'mushroom', theme: 'dark', size: 'compact' }).workflow;
  assert.ok(w.includes('user: "007"') && w.includes('species: mushroom') && w.includes('theme: dark') && w.includes('size: compact'));
  assert.ok(!w.includes('@main') && !w.includes('${{'));
});
ok('tema automático', () => {
  const r = api.build({ user: 'octocat', theme: 'both' });
  assert.equal(r.files.length, 2);
  assert.ok(r.workflow.includes('git add commitling.svg commitling-dark.svg'));
  assert.ok(r.readme.includes('<picture>'));
});
ok('picture solo automático', () => assert.ok(!api.build({ theme: 'dark' }).readme.includes('<picture>')));
ok('normalización', () => {
  for (const [input, expected] of [['@x', 'x'], ['  x  ', 'x'], ['https://github.com/x', 'x'], ['https://github.com/x/', 'x'],
    ['github.com/x?tab=repositories', 'x'], ['http://www.github.com/x#readme', 'x'], ['https://gitlab.com/x', 'https://gitlab.com/x']]) {
    assert.equal(api.normalizeUser(input), expected);
  }
});
ok('validación', () => {
  for (const bad of ['-x', 'a b', 'a"b', 'x'.repeat(40), 'x\nrun: evil']) assert.equal(api.validLogin(bad), false, bad);
});
ok('usuario inválido no entra', () => assert.ok(!api.build({ user: 'x\nrun: evil' }).workflow.includes('run: evil')));
ok('enlaces codificados', () => {
  const l = api.links('a-b');
  assert.equal(l.newRepo, 'https://github.com/new?name=a-b&visibility=public');
  assert.ok(l.newFile('a b').includes('value=a%20b'));
  assert.equal(l.profile, 'https://github.com/a-b');
});
ok('estructura YAML', () => {
  const w = api.build({}).workflow;
  assert.ok(!w.includes('\t'));
  for (const line of w.split('\n')) if (line.trim()) assert.equal((line.match(/^ */) || [''])[0].length % 2, 0, line);
  assert.deepEqual(Array.from(w.split('\n')).filter((x) => /^[a-z]+:/.test(x)).map((x) => x.split(':')[0]), ['name', 'on', 'permissions', 'jobs']);
});
ok('API disponible sin DOM', () => assert.equal(typeof api.build, 'function'));
ok('sin almacenamiento', () => assert.ok(!source.includes('localStorage') && !source.includes('sessionStorage') && !source.includes('document.cookie')));

ok('todas las combinaciones usan @v1 y estructura válida', () => {
  for (const user of users) for (const sp of species) for (const theme of themes) for (const size of sizes) {
    const w = api.build({ user, species: sp, theme, size }).workflow;
    assert.equal(w.split('uses: BertMarti/commitling@v1').length - 1, theme === 'both' ? 2 : 1);
    assert.ok(!w.includes('@main') && !w.includes('${{') && !w.includes('\t'));
    if (user) assert.ok(w.includes('user: "' + user + '"\n'));
    assert.equal(w.split('\n').filter((x) => /^[a-z]+:/.test(x)).map((x) => x.split(':')[0]).join(), 'name,on,permissions,jobs');
  }
});

// A fake page with the elements of the section and of the generator form.
function page() {
  const els = {};
  const listeners = {};
  function el(id, extra) {
    const attrs = {};
    const e = Object.assign({
      id, hidden: false, textContent: '', value: '',
      setAttribute(k, v) { attrs[k] = String(v); },
      getAttribute(k) { return k in attrs ? attrs[k] : null; },
      removeAttribute(k) { delete attrs[k]; },
      addEventListener(type, fn) { (listeners[id + ':' + type] = listeners[id + ':' + type] || []).push(fn); },
    }, extra);
    els[id] = e;
    return e;
  }
  for (const id of ['ins-user', 'ins-error', 'wf', 'rd', 'ins-repo', 'ins-nolinks', 'ins-new-repo', 'ins-new-file',
    'ins-actions', 'ins-edit-readme', 'ins-profile', 'gen-user']) el(id);
  el('ins-form', { hidden: true });
  el('gen-form');
  const insLinks = [1, 2, 3, 4, 5, 6].map((i) => el('link' + i, { hidden: true }));
  // Radios of one name behave as a group: checking one unchecks the rest.
  const radio = (name, value, checked) => {
    const r = { name, value, on: checked };
    Object.defineProperty(r, 'checked', {
      get() { return r.on; },
      set(v) { if (v) radios.filter((o) => o.name === name).forEach((o) => { o.on = false; }); r.on = !!v; },
    });
    return r;
  };
  const radios = [
    radio('ins-species', 'moss', true), radio('ins-species', 'mushroom', false),
    radio('ins-theme', 'light', true), radio('ins-theme', 'dark', false), radio('ins-theme', 'both', false),
    radio('ins-size', 'full', true), radio('ins-size', 'compact', false),
    radio('species', 'moss', false), radio('species', 'mushroom', true),
    radio('theme', 'light', false), radio('theme', 'dark', true),
    radio('size', 'full', false), radio('size', 'compact', true),
  ];
  const stored = [];
  const document = {
    getElementById: (id) => els[id] || null,
    querySelectorAll(sel) {
      if (sel === '.ins-link') return insLinks;
      const m = sel.match(/^input\[name="([^"]+)"\]$/);
      if (m) return radios.filter((r) => r.name === m[1]);
      throw new Error('unexpected selector ' + sel);
    },
    querySelector(sel) {
      let m = sel.match(/^input\[name="([^"]+)"\]:checked$/);
      if (m) return radios.find((r) => r.name === m[1] && r.checked) || null;
      m = sel.match(/^input\[name="([^"]+)"\]\[value="([^"]*)"\]$/);
      if (m) return radios.find((r) => r.name === m[1] && r.value === m[2]) || null;
      throw new Error('unexpected selector ' + sel);
    },
    get cookie() { stored.push('cookie'); return ''; },
  };
  const window = {};
  const trap = { get() { stored.push('storage'); return null; } };
  Object.defineProperty(window, 'localStorage', trap);
  Object.defineProperty(window, 'sessionStorage', trap);
  vm.runInNewContext(source, { console, encodeURIComponent, window, document });
  const fire = (id, type) => (listeners[id + ':' + type] || []).forEach((fn) => fn({ preventDefault() {} }));
  const type = (v) => { els['ins-user'].value = v; fire('ins-form', 'input'); };
  return { els, insLinks, radios, fire, type, stored, listeners, api: window.commitlingInstall };
}

ok('la página muestra el formulario y el workflow base', () => {
  const p = page();
  assert.equal(p.els['ins-form'].hidden, false);
  assert.equal(p.els.wf.textContent, p.api.build({}).workflow);
  assert.equal(p.els.rd.textContent, '![commitling](./commitling.svg)');
  assert.equal(p.els['ins-repo'].textContent, 'tu-usuario/tu-usuario');
  assert.ok(p.insLinks.every((l) => l.hidden));
  assert.equal(p.els['ins-nolinks'].hidden, false);
  assert.equal(p.els['ins-error'].textContent, '');
});

ok('usuario no válido: error, aria-invalid, sin enlaces y sin usuario en el YAML', () => {
  const p = page();
  p.type('a b"\nrun: evil');
  assert.notEqual(p.els['ins-error'].textContent, '');
  assert.equal(p.els['ins-user'].getAttribute('aria-invalid'), 'true');
  assert.ok(p.insLinks.every((l) => l.hidden));
  assert.ok(!p.els.wf.textContent.includes('user:') && !p.els.wf.textContent.includes('evil'));
});

ok('usuario válido (pegado como enlace): enlaces visibles y correctos', () => {
  const p = page();
  p.type('nope bad');
  p.type(' https://github.com/Octo-Cat/ ');
  assert.equal(p.els['ins-error'].textContent, '');
  assert.equal(p.els['ins-user'].getAttribute('aria-invalid'), null);
  assert.ok(p.insLinks.every((l) => !l.hidden));
  assert.equal(p.els['ins-nolinks'].hidden, true);
  assert.equal(p.els['ins-repo'].textContent, 'Octo-Cat/Octo-Cat');
  assert.ok(p.els.wf.textContent.includes('user: "Octo-Cat"\n'));
  assert.equal(p.els['ins-new-repo'].getAttribute('href'), 'https://github.com/new?name=Octo-Cat&visibility=public');
  assert.equal(p.els['ins-actions'].getAttribute('href'), 'https://github.com/Octo-Cat/Octo-Cat/actions/workflows/commitling.yml');
  assert.equal(p.els['ins-edit-readme'].getAttribute('href'), 'https://github.com/Octo-Cat/Octo-Cat/edit/main/README.md');
  assert.equal(p.els['ins-profile'].getAttribute('href'), 'https://github.com/Octo-Cat');
  const file = p.els['ins-new-file'].getAttribute('href');
  assert.ok(file.startsWith('https://github.com/Octo-Cat/Octo-Cat/new/main?filename=.github%2Fworkflows%2Fcommitling.yml&value='));
  assert.equal(decodeURIComponent(file.split('&value=')[1]), p.els.wf.textContent);
});

ok('tema automático: dos pasos y <picture> en el README', () => {
  const p = page();
  p.radios.find((r) => r.name === 'ins-theme' && r.value === 'light').checked = false;
  p.radios.find((r) => r.name === 'ins-theme' && r.value === 'both').checked = true;
  p.fire('ins-form', 'change');
  assert.ok(p.els.wf.textContent.includes('git add commitling.svg commitling-dark.svg'));
  assert.ok(p.els.rd.textContent.startsWith('<picture>'));
});

ok('el generador de arriba pasa usuario y opciones si aquí está vacío', () => {
  const p = page();
  p.els['gen-user'].value = '@octocat';
  p.fire('gen-form', 'submit');
  assert.equal(p.els['ins-user'].value, 'octocat');
  const w = p.els.wf.textContent;
  assert.ok(w.includes('user: "octocat"') && w.includes('species: mushroom') && w.includes('theme: dark') && w.includes('size: compact'));
  p.els['gen-user'].value = 'otra';
  p.fire('gen-form', 'submit');
  assert.equal(p.els['ins-user'].value, 'octocat', 'lo que la persona ya escribió no se pisa');
  const q = page();
  q.els['gen-user'].value = 'mal nombre';
  q.fire('gen-form', 'submit');
  assert.equal(q.els['ins-user'].value, '', 'un nombre no válido no se copia');
});

ok('Enter en el formulario no lo envía (nada de ?user= en la URL)', () => {
  const p = page();
  p.type('octocat');
  let prevented = 0;
  (p.listeners['ins-form:submit'] || []).forEach((fn) => fn({ preventDefault() { prevented++; } }));
  assert.ok(prevented >= 1, 'submit sin preventDefault');
  assert.equal(p.els['ins-user'].value, 'octocat');
});

ok('solo valores permitidos de especie, tema y tamaño', () => {
  const base = api.build({ user: 'octocat' }).workflow;
  assert.equal(api.build({ user: 'octocat', species: 'dragon\nrun: evil', theme: 'sepia', size: 'xl' }).workflow, base);
  assert.equal(api.build({ user: 'octocat', species: ' MUSHROOM ', theme: 'Dark', size: 'COMPACT' }).workflow,
    api.build({ user: 'octocat', species: 'mushroom', theme: 'dark', size: 'compact' }).workflow);
});

ok('no toca almacenamiento ni cookies', () => {
  const p = page();
  p.type('octocat');
  assert.deepEqual(Array.from(p.stored), []);
});
})();
