// Somatrack — shared site behaviour
(function () {
  var doc = document.documentElement;
  doc.classList.add('js');

  // Sticky header shadow
  var header = document.querySelector('.site-header');
  var onScroll = function () { header && header.classList.toggle('is-scrolled', window.scrollY > 8); };
  window.addEventListener('scroll', onScroll, { passive: true });
  onScroll();

  // Mobile menu
  var toggle = document.querySelector('.nav-toggle');
  if (toggle) {
    toggle.addEventListener('click', function () {
      var open = document.body.classList.toggle('nav-open');
      toggle.setAttribute('aria-expanded', open ? 'true' : 'false');
    });
    document.querySelectorAll('.nav-menu a').forEach(function (a) {
      a.addEventListener('click', function () {
        document.body.classList.remove('nav-open');
        toggle.setAttribute('aria-expanded', 'false');
      });
    });
  }

  // Flip cards: tap / click / keyboard toggles (hover handles desktop)
  document.querySelectorAll('.flip').forEach(function (card) {
    var flip = function () {
      var flipped = card.classList.toggle('is-flipped');
      card.setAttribute('aria-pressed', flipped ? 'true' : 'false');
    };
    card.addEventListener('click', flip);
    card.addEventListener('keydown', function (e) {
      if (e.key === 'Enter' || e.key === ' ') { e.preventDefault(); flip(); }
    });
  });

  // Module filter chips
  document.querySelectorAll('.chips').forEach(function (group) {
    var grid = group.parentElement.querySelector('.module-grid');
    group.addEventListener('click', function (e) {
      var chip = e.target.closest('.chip');
      if (!chip || !grid) return;
      var f = chip.getAttribute('data-filter');
      group.querySelectorAll('.chip').forEach(function (c) {
        var on = c === chip;
        c.classList.toggle('is-active', on);
        c.setAttribute('aria-pressed', on ? 'true' : 'false');
      });
      grid.querySelectorAll('.tile').forEach(function (t) {
        var show = f === 'all' || t.getAttribute('data-cat') === f;
        t.classList.toggle('is-hidden', !show);
        t.classList.remove('is-entering');
        if (show) { void t.offsetWidth; t.classList.add('is-entering', 'is-in'); }
      });
    });
  });

  // Cursor spotlight on tiles
  document.querySelectorAll('.tile').forEach(function (tile) {
    tile.addEventListener('pointermove', function (e) {
      var r = tile.getBoundingClientRect();
      tile.style.setProperty('--mx', (e.clientX - r.left) + 'px');
      tile.style.setProperty('--my', (e.clientY - r.top) + 'px');
    });
  });

  // Stacking deck: index each card so they stack with an offset
  document.querySelectorAll('.deck').forEach(function (deck) {
    deck.querySelectorAll('.deck-card').forEach(function (card, i) { card.style.setProperty('--i', i); });
  });

  // Reveal on scroll
  var targets = document.querySelectorAll('.reveal, .phone');
  if (!('IntersectionObserver' in window)) {
    targets.forEach(function (el) { el.classList.add('is-in'); });
    return;
  }
  var io = new IntersectionObserver(function (entries) {
    entries.forEach(function (entry) {
      if (entry.isIntersecting) { entry.target.classList.add('is-in'); io.unobserve(entry.target); }
    });
  }, { threshold: 0.12, rootMargin: '0px 0px -40px 0px' });
  targets.forEach(function (el) { io.observe(el); });
})();
