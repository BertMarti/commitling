// Small enhancements of the page: the «Copiar» buttons of the code blocks and
// the fallback of the live creature. Kept in its own file, not inline (no
// <script> blocks, no onerror attributes), so the page can use a strict
// Content-Security-Policy.
(function () {
  'use strict';

  // If the live creature cannot be loaded, say so instead of a broken image.
  // The image may have failed before this deferred script runs.
  var live = document.getElementById('live-img');
  if (live) {
    var liveFailed = function () {
      var box = live.closest('.live');
      var missing = box && box.querySelector('.missing');
      if (missing) missing.hidden = false;
      var pic = live.closest('picture');
      if (pic) pic.hidden = true;
    };
    live.addEventListener('error', liveFailed);
    if (live.complete && live.naturalWidth === 0 && live.currentSrc) liveFailed();
  }

  document.querySelectorAll('.copy').forEach(function (b) {
    var box = b.closest('.code');
    var msg = box.querySelector('.copy-msg');
    var code = document.getElementById(b.dataset.copy);
    function say(text) { msg.textContent = text; }
    function select() {
      var range = document.createRange();
      range.selectNodeContents(code);
      var sel = window.getSelection();
      sel.removeAllRanges();
      sel.addRange(range);
    }
    function done() {
      b.textContent = 'Copiado';
      say('Copiado al portapapeles.');
      setTimeout(function () { b.textContent = 'Copiar'; say(''); }, 2000);
    }
    function failed() {
      select();
      say('No se pudo copiar automáticamente. El texto está seleccionado: cópialo con Ctrl+C (o Cmd+C).');
    }
    b.hidden = false;
    b.addEventListener('click', function () {
      var text = code.textContent;
      if (navigator.clipboard && navigator.clipboard.writeText) {
        navigator.clipboard.writeText(text).then(done, failed);
        return;
      }
      select();
      try { document.execCommand('copy') ? done() : failed(); } catch (e) { failed(); }
    });
  });

  // Let visitors shorten the gallery without changing the no-script view.
  var speciesPick = document.getElementById('species-pick');
  if (speciesPick) {
    var speciesMsg = document.getElementById('species-pick-msg');
    var speciesRadios = speciesPick.querySelectorAll('input[name="gal-species"]');
    var speciesBlocks = document.querySelectorAll('#galeria [data-species]');
    var speciesNames = {};
    speciesRadios.forEach(function (radio) {
      var label = document.querySelector('label[for="' + radio.id + '"]');
      speciesNames[radio.value] = label ? label.textContent : radio.value;
    });
    // The closed list is the values the radios had when the page loaded.
    function validSpecies(value) {
      return typeof value === 'string' && Object.prototype.hasOwnProperty.call(speciesNames, value) ? value : 'all';
    }
    function filterSpecies(value, announce) {
      value = validSpecies(value);
      speciesBlocks.forEach(function (block) {
        block.hidden = value !== 'all' && block.getAttribute('data-species') !== value;
      });
      if (announce && speciesMsg) speciesMsg.textContent = value === 'all' ? 'Se muestran todas las especies.' : 'Se muestra solo: ' + speciesNames[value] + ' (sus fases, ánimos, accesorios y tarjeta compacta).';
    }
    function chooseSpecies(value) {
      value = validSpecies(value);
      speciesRadios.forEach(function (radio) { radio.checked = radio.value === value; });
      filterSpecies(value, true);
    }
    speciesRadios.forEach(function (radio) {
      radio.addEventListener('change', function () { chooseSpecies(radio.value); });
    });
    speciesPick.hidden = false;
    if (speciesMsg) speciesMsg.hidden = false;
    document.querySelectorAll('nav.jump a[href^="#especie-"]').forEach(function (link) {
      link.addEventListener('click', function () {
        var id = link.getAttribute('href').slice('#especie-'.length);
        var block = document.getElementById('especie-' + id);
        if (block && block.closest('.species-block').hidden) chooseSpecies(id);
      });
    });
    function followHash() {
      var hash = window.location.hash;
      var match = /^#(?:especie|accesorios)-([A-Za-z0-9-]+)$/.exec(hash);
      if (!match) return;
      var target = document.getElementById(hash.slice(1));
      var block = target && target.closest('.species-block');
      if (block && block.hidden) chooseSpecies(match[1]);
    }
    window.addEventListener('hashchange', followHash);
  }
})();
