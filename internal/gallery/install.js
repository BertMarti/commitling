// Workflow generator of «Instálalo en tu perfil». It builds in the browser,
// with no wasm, no network and no storage, the same text as generate.Install
// (a Go test compares both for every option) and fills the steps with direct
// links to GitHub. No HTML is built from what the person types: only
// textContent and validated logins in href.
(function (root) {
  'use strict';

  var BaseWorkflow = 'name: commitling\n\n' +
    'on:\n' +
    '  schedule:\n' +
    '    - cron: "23 5 * * *" # cada día a las 05:23 UTC\n' +
    '  workflow_dispatch:\n\n' +
    'permissions:\n' +
    '  contents: write\n\n' +
    'jobs:\n' +
    '  commitling:\n' +
    '    runs-on: ubuntu-latest\n' +
    '    steps:\n' +
    '      - uses: actions/checkout@v7\n' +
    '      - uses: BertMarti/commitling@v1\n' +
    '        with:\n' +
    '          out: commitling.svg\n' +
    '      - name: Guardar la criatura si ha cambiado\n' +
    '        run: |\n' +
    '          git add commitling.svg\n' +
    '          if git diff --cached --quiet; then\n' +
    '            echo "Sin cambios."\n' +
    '            exit 0\n' +
    '          fi\n' +
    '          git config user.name "github-actions[bot]"\n' +
    '          git config user.email "41898282+github-actions[bot]@users.noreply.github.com"\n' +
    '          git commit -m "chore: actualiza commitling"\n' +
    '          if ! git push; then\n' +
    '            echo "::error title=commitling::No se pudo guardar la criatura en el repositorio. Comprueba que este archivo tiene «permissions: contents: write» y, si la rama está protegida, deja que github-actions[bot] suba cambios."\n' +
    '            exit 1\n' +
    '          fi\n';

  var ReadmeSnippet = '![commitling](./commitling.svg)';
  var ReadmePicture = '<picture>\n' +
    '  <source media="(prefers-color-scheme: dark)" srcset="./commitling-dark.svg">\n' +
    '  <img alt="commitling" src="./commitling.svg">\n' +
    '</picture>';

  function validLogin(s) {
    return typeof s === 'string' && s.length >= 1 && s.length <= 39 &&
      s.charAt(0) !== '-' && !/[^A-Za-z0-9-]/.test(s);
  }

  function normalizeUser(s) {
    s = typeof s === 'string' ? s.trim() : '';
    if (s.charAt(0) === '@') s = s.slice(1);
    // A pasted profile link: https://github.com/<login>, with or without the
    // scheme, a trailing slash, a path, a query or a fragment.
    var match = s.match(/^(?:https?:\/\/)?(?:www\.)?github\.com\/([^/?#]+)(?:[/?#].*)?$/i);
    return match ? match[1] : s;
  }

  // Only the canonical values of the radios are accepted (as Go checks them
  // with SpeciesByName, ThemeByName and SizeByName); anything else is the
  // default, so nothing unexpected can reach the YAML.
  var ALLOWED = { species: ['moss', 'mushroom'], theme: ['light', 'dark', 'both'], size: ['full', 'compact'] };
  function option(kind, value) {
    var v = typeof value === 'string' ? value.trim().toLowerCase() : '';
    return ALLOWED[kind].indexOf(v) >= 0 ? v : ALLOWED[kind][0];
  }

  function build(input) {
    input = input || {};
    var user = normalizeUser(input.user);
    if (!validLogin(user)) user = '';
    var species = option('species', input.species);
    var theme = option('theme', input.theme);
    var size = option('size', input.size);
    var both = theme === 'both';
    var files = both ? ['commitling.svg', 'commitling-dark.svg'] : ['commitling.svg'];

    function step(stepTheme, out) {
      var text = '      - uses: BertMarti/commitling@v1\n        with:\n';
      if (user) text += '          user: "' + user + '"\n';
      if (species !== 'moss') text += '          species: mushroom\n';
      if (stepTheme === 'dark') text += '          theme: dark\n';
      if (size === 'compact') text += '          size: compact\n';
      return text + '          out: ' + out + '\n';
    }

    var steps = both ? step('light', files[0]) + step('dark', files[1]) : step(theme, files[0]);
    var workflow = BaseWorkflow.replace(
      '      - uses: BertMarti/commitling@v1\n        with:\n          out: commitling.svg\n', steps
    ).replace('git add commitling.svg\n', 'git add ' + files.join(' ') + '\n');
    return { workflow: workflow, readme: both ? ReadmePicture : ReadmeSnippet, files: files };
  }

  function links(login) {
    var encoded = encodeURIComponent(login);
    var repo = 'https://github.com/' + encoded + '/' + encoded;
    return {
      newRepo: 'https://github.com/new?name=' + encoded + '&visibility=public',
      newFile: function (workflow) {
        return repo + '/new/main?filename=.github%2Fworkflows%2Fcommitling.yml&value=' + encodeURIComponent(workflow);
      },
      actions: repo + '/actions/workflows/commitling.yml',
      editReadme: repo + '/edit/main/README.md',
      profile: 'https://github.com/' + encoded
    };
  }

  root.commitlingInstall = { build: build, normalizeUser: normalizeUser, validLogin: validLogin, links: links };

  if (typeof document === 'undefined') return;
  var form = document.getElementById('ins-form');
  if (!form) return;
  form.hidden = false;
  // The form only feeds the code blocks: Enter in the user field must not
  // send it (that would reload the page with ?user=… in the URL and history).
  form.addEventListener('submit', function (e) { e.preventDefault(); });
  var userInput = document.getElementById('ins-user');
  var error = document.getElementById('ins-error');
  var workflowEl = document.getElementById('wf');
  var readmeEl = document.getElementById('rd');
  var repoEl = document.getElementById('ins-repo');
  var linkParagraphs = document.querySelectorAll('.ins-link');
  var noLinks = document.getElementById('ins-nolinks');

  function checked(name, fallback) {
    var radios = document.querySelectorAll('input[name="' + name + '"]');
    for (var i = 0; i < radios.length; i++) if (radios[i].checked) return radios[i].value;
    return fallback;
  }

  function recalculate() {
    var login = normalizeUser(userInput.value);
    var valid = login === '' || validLogin(login);
    var result = build({ user: valid ? login : '', species: checked('ins-species', 'moss'), theme: checked('ins-theme', 'light'), size: checked('ins-size', 'full') });
    workflowEl.textContent = result.workflow;
    readmeEl.textContent = result.readme;
    repoEl.textContent = login && valid ? login + '/' + login : 'tu-usuario/tu-usuario';
    if (!valid) {
      error.textContent = 'Escribe un usuario de GitHub válido: entre 1 y 39 letras, números o guiones, sin guion inicial.';
      userInput.setAttribute('aria-invalid', 'true');
    } else {
      error.textContent = '';
      userInput.removeAttribute('aria-invalid');
    }
    var show = valid && login !== '';
    for (var i = 0; i < linkParagraphs.length; i++) linkParagraphs[i].hidden = !show;
    noLinks.hidden = show;
    if (show) {
      var urls = links(login);
      document.getElementById('ins-new-repo').setAttribute('href', urls.newRepo);
      document.getElementById('ins-new-file').setAttribute('href', urls.newFile(result.workflow));
      document.getElementById('ins-actions').setAttribute('href', urls.actions);
      document.getElementById('ins-edit-readme').setAttribute('href', urls.editReadme);
      document.getElementById('ins-profile').setAttribute('href', urls.profile);
    }
  }

  form.addEventListener('input', recalculate);
  form.addEventListener('change', recalculate);
  recalculate();
  var genForm = document.getElementById('gen-form');
  if (genForm) genForm.addEventListener('submit', function () {
    var genUser = document.getElementById('gen-user');
    var login = normalizeUser(genUser && genUser.value);
    // Only a valid name is carried over: the generator shows its own error.
    if (!userInput.value && validLogin(login)) {
      userInput.value = login;
      var names = [['species', 'ins-species'], ['theme', 'ins-theme'], ['size', 'ins-size']];
      for (var i = 0; i < names.length; i++) {
        var source = document.querySelector('input[name="' + names[i][0] + '"]:checked');
        var target = document.querySelector('input[name="' + names[i][1] + '"][value="' + (source ? source.value : '') + '"]');
        if (target) target.checked = true;
      }
      recalculate();
    }
  });
})(typeof window !== 'undefined' ? window : this);
