(function () {
  'use strict';

  var SETTINGS_URL = '/api/v1/settings/public';
  var TIMEOUT_MS = 3000;

  var root = document.documentElement;
  root.classList.add('js');

  var reduceMotion = window.matchMedia('(prefers-reduced-motion: reduce)');
  var coarsePointer = window.matchMedia('(pointer: coarse)');

  function queryAll(selector) {
    return Array.prototype.slice.call(document.querySelectorAll(selector));
  }

  function setText(selector, value) {
    var text = typeof value === 'string' ? value.trim() : '';
    if (!text) return;
    queryAll(selector).forEach(function (node) {
      node.textContent = text;
    });
  }

  function formatAmount(value) {
    var amount = Number(value);
    if (!isFinite(amount)) return '';
    return Number.isInteger(amount) ? String(amount) : amount.toFixed(2);
  }

  function applyDocLinks(docUrl) {
    var url = typeof docUrl === 'string' ? docUrl.trim() : '';
    if (!url) return;
    queryAll('[data-doc-link]').forEach(function (node) {
      node.setAttribute('href', url);
      node.hidden = false;
    });
  }

  function applyAuthLink(registrationEnabled) {
    var node = document.querySelector('[data-auth-link]');
    if (!node) return;
    if (registrationEnabled === true) {
      node.setAttribute('href', '/register');
      node.textContent = '注册';
      return;
    }
    node.setAttribute('href', '/login');
    node.textContent = '登录';
  }

  function applyLogo(logo) {
    var src = typeof logo === 'string' ? logo.trim() : '';
    if (!src) return;
    queryAll('[data-site-logo]').forEach(function (node) {
      node.setAttribute('src', src);
    });
  }

  function applySettings(cfg) {
    if (!cfg || typeof cfg !== 'object') return;

    var siteName = typeof cfg.site_name === 'string' ? cfg.site_name.trim() : '';
    if (siteName) document.title = siteName;

    setText('[data-site-name]', cfg.site_name);
    setText('[data-subtitle]', cfg.site_subtitle);
    applyLogo(cfg.site_logo);
    applyDocLinks(cfg.doc_url);
    applyAuthLink(cfg.registration_enabled);

    var contact = typeof cfg.contact_info === 'string' ? cfg.contact_info.trim() : '';
    if (contact) {
      contactValue = contact;
      queryAll('[data-contact]').forEach(function (node) {
        node.hidden = false;
      });
    }
  }

  function loadSettings() {
    var controller = typeof AbortController === 'function' ? new AbortController() : null;
    var timer = window.setTimeout(function () {
      if (controller) controller.abort();
    }, TIMEOUT_MS);

    return fetch(SETTINGS_URL, {
      credentials: 'omit',
      cache: 'no-store',
      signal: controller ? controller.signal : undefined
    })
      .then(function (response) {
        if (!response.ok) throw new Error('settings status ' + response.status);
        return response.json();
      })
      .then(function (body) {
        return body && body.data ? body.data : null;
      })
      .then(function (data) {
        window.clearTimeout(timer);
        applySettings(data);
      })
      .catch(function () {
        window.clearTimeout(timer);
        // 公开设置不可用时保持静态兜底：站名、副标题与 logo 使用 HTML 默认值。
      });
  }

  var contactValue = '';
  var toastTimer = 0;

  function showToast(message) {
    var toast = document.querySelector('.toast');
    if (!toast) return;
    toast.textContent = message;
    toast.classList.add('on');
    window.clearTimeout(toastTimer);
    toastTimer = window.setTimeout(function () {
      toast.classList.remove('on');
    }, 2600);
  }

  function copyContact() {
    if (!contactValue) return;
    var done = function () {
      showToast('客服群号已复制：' + contactValue);
    };
    var fallback = function () {
      showToast('客服群：' + contactValue);
    };
    if (navigator.clipboard && navigator.clipboard.writeText) {
      navigator.clipboard.writeText(contactValue).then(done, fallback);
      return;
    }
    fallback();
  }

  function setYear() {
    var year = '';
    try {
      year = new Intl.DateTimeFormat('zh-CN', { timeZone: 'Asia/Shanghai', year: 'numeric' }).format(new Date());
    } catch (err) {
      year = String(new Date().getFullYear());
    }
    setText('[data-year]', year);
  }

  function watchReveal() {
    var targets = queryAll('.reveal, .models');
    if (!targets.length) return;

    if (!('IntersectionObserver' in window) || reduceMotion.matches) {
      targets.forEach(function (node) {
        node.classList.add('in');
      });
      return;
    }

    var observer = new IntersectionObserver(
      function (entries) {
        entries.forEach(function (entry) {
          if (!entry.isIntersecting) return;
          entry.target.classList.add('in');
          observer.unobserve(entry.target);
        });
      },
      { rootMargin: '0px 0px -12% 0px', threshold: 0.12 }
    );

    targets.forEach(function (node) {
      observer.observe(node);
    });
  }

  function watchPointer() {
    var spot = document.querySelector('.spot');
    if (!spot || reduceMotion.matches || coarsePointer.matches) return;

    var frame = 0;
    var x = 50;
    var y = 26;

    var paint = function () {
      frame = 0;
      spot.style.setProperty('--mx', x.toFixed(2) + '%');
      spot.style.setProperty('--my', y.toFixed(2) + '%');
    };

    window.addEventListener(
      'pointermove',
      function (event) {
        x = (event.clientX / Math.max(window.innerWidth, 1)) * 100;
        y = (event.clientY / Math.max(window.innerHeight, 1)) * 100;
        if (!frame) frame = window.requestAnimationFrame(paint);
      },
      { passive: true }
    );

    window.addEventListener(
      'pointerleave',
      function () {
        x = 50;
        y = 26;
        if (!frame) frame = window.requestAnimationFrame(paint);
      },
      { passive: true }
    );
  }

  var contactButton = document.querySelector('[data-contact]');
  if (contactButton) contactButton.addEventListener('click', copyContact);

  setYear();
  watchReveal();
  watchPointer();
  loadSettings();
})();
