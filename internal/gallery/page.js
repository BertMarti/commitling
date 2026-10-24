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
})();
