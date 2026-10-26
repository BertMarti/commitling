// The web theme selector. It runs before the stylesheet to avoid a flash.
(function () {
  'use strict';

  var key = 'commitling:theme';
  var darkMedia = '(prefers-color-scheme: dark)';

  function storage() {
    try { return window.localStorage; } catch (_) { return null; }
  }

  function read() {
    var store = storage();
    if (!store) return 'system';
    try {
      var value = store.getItem(key);
      // A closed list: anything else (another site of the shared origin,
      // an old value, a tampered one) is ignored and removed.
      if (value === 'light' || value === 'dark') return value;
      if (value !== null) store.removeItem(key);
    } catch (_) {}
    return 'system';
  }

  function apply(value) {
    if (value !== 'light' && value !== 'dark') value = 'system';
    if (value === 'system') document.documentElement.removeAttribute('data-theme');
    else document.documentElement.setAttribute('data-theme', value);

    var sources = document.querySelectorAll('picture source[media]');
    for (var i = 0; i < sources.length; i++) {
      var source = sources[i];
      var original = source.getAttribute('data-media');
      if (original === null) {
        original = source.getAttribute('media');
        source.setAttribute('data-media', original);
      }
      source.setAttribute('media', value === 'system' ? original :
        (original === darkMedia ? (value === 'dark' ? 'all' : 'not all') : original));
    }

    var metas = document.querySelectorAll('meta[name="theme-color"][media]');
    for (var j = 0; j < metas.length; j++) {
      var meta = metas[j];
      var metaOriginal = meta.getAttribute('data-media');
      if (metaOriginal === null) {
        metaOriginal = meta.getAttribute('media');
        meta.setAttribute('data-media', metaOriginal);
      }
      if (value === 'system') meta.setAttribute('media', metaOriginal);
      else {
        var isDark = metaOriginal === darkMedia;
        var isLight = metaOriginal === '(prefers-color-scheme: light)';
        var selected = (isDark && value === 'dark') || (isLight && value === 'light');
        meta.setAttribute('media', selected ? 'all' : 'not all');
      }
    }
    return value;
  }

  function choose(value) {
    if (value !== 'system' && value !== 'light' && value !== 'dark') return read();
    apply(value);
    var store = storage();
    if (store) {
      try {
        if (value === 'system') store.removeItem(key);
        else store.setItem(key, value);
      } catch (_) {}
    }
    return value;
  }

  var initial = read();
  apply(initial);

  function init() {
    apply(read());
    var pick = document.getElementById('theme-pick');
    if (!pick) return;
    pick.hidden = false;
    var radios = pick.querySelectorAll('input[name="site-theme"]');
    for (var i = 0; i < radios.length; i++) {
      radios[i].checked = radios[i].value === read();
      radios[i].addEventListener('change', function () { choose(this.value); });
    }
  }
  if (document.readyState === 'loading') document.addEventListener('DOMContentLoaded', init);
  else init();

  window.commitlingTheme = { read: read, choose: choose, apply: apply };
}());
