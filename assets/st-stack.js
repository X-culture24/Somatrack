/* Stacked-section scrolling: every section sticks by its bottom edge once it has scrolled fully
   into view (top = viewport height - section height), holds there for an extra stretch of
   scrolling, and only then does the next section slide up over it. */
(function () {
  var SECTIONS = [
    'section[data-framer-name][style*="width:100%"]',
    'section[data-framer-name="Hero Section"]',
    'section[data-framer-name="Hero"][id]',
    'section[data-framer-name="News"]',
    'section[data-framer-name="Key reports"]',
    'section[name="CTA"]',
    'section#cta',
    'footer[data-framer-name]'
  ].join(",");
  var REDUCED = window.matchMedia("(prefers-reduced-motion: reduce)");
  // How long each section stays fully in view before the next arrives, as a share of the viewport.
  var HOLD_DESKTOP = 0.7;
  var HOLD_PHONE = 0.5;
  var blocks = [];
  var holds = [];

  // Sections tagged with the same data-st-group are wrapped into one block, so related
  // sections (e.g. the last content section and the CTA band) travel together.
  function mergeGroups() {
    var groups = {};
    document.querySelectorAll("[data-st-group]").forEach(function (el) {
      var g = el.getAttribute("data-st-group");
      (groups[g] = groups[g] || []).push(el);
    });
    Object.keys(groups).forEach(function (g) {
      var members = groups[g];
      // The group lives where its first member lives.
      var home = blockFor(members[0]).parentElement;
      var local = [], outside = [];
      members.forEach(function (m) {
        if (home.contains(m)) {
          var t = m;
          while (t.parentElement !== home) t = t.parentElement;
          if (local.indexOf(t) === -1) local.push(t);
        } else {
          // A member elsewhere in the page (e.g. a CTA band outside the main wrapper):
          // take its own branch, stopping before any ancestor shared with the group.
          var o = m;
          while (o.parentElement && !o.parentElement.contains(home)) o = o.parentElement;
          if (outside.indexOf(o) === -1) outside.push(o);
        }
      });
      var kids = Array.prototype.slice.call(home.children);
      var first = Math.min.apply(null, local.map(function (t) { return kids.indexOf(t); }));
      var last = Math.max.apply(null, local.map(function (t) { return kids.indexOf(t); }));
      var wrap = document.createElement("div");
      wrap.className = "st-merge";
      home.insertBefore(wrap, kids[first]);
      for (var i = first; i <= last; i++) wrap.appendChild(kids[i]);
      outside.forEach(function (o) { wrap.appendChild(o); });
    });
  }

  // The sticky element must be a real box whose parent holds the other sections,
  // so climb out of single-child wrappers but never into a display:contents one.
  function blockFor(el) {
    var merged = el.closest(".st-merge");
    if (merged) return merged;
    var b = el;
    while (b.parentElement && b.parentElement !== document.body &&
           b.parentElement.children.length === 1 &&
           getComputedStyle(b.parentElement).display !== "contents") {
      b = b.parentElement;
    }
    return b;
  }

  function isShown(el) {
    return el.offsetParent !== null && getComputedStyle(el).display !== "none";
  }

  function collect() {
    blocks.forEach(function (b) {
      b.classList.remove("st-block", "st-block--first", "st-block--last");
      b.style.position = "";
      b.style.top = "";
      b.style.removeProperty("--st-cover");
    });
    holds.forEach(function (h) { h.remove(); });
    holds = [];
    var seen = [];
    document.querySelectorAll(SECTIONS).forEach(function (s) {
      if (!isShown(s)) return;
      // Skip sections nested inside another matched section (e.g. Framer's inner "Hero").
      var outer = s.parentElement && s.parentElement.closest(SECTIONS);
      if (outer && isShown(outer)) return;
      var b = blockFor(s);
      if (seen.indexOf(b) === -1) seen.push(b);
    });
    seen.sort(function (a, b) {
      return a.compareDocumentPosition(b) & Node.DOCUMENT_POSITION_FOLLOWING ? -1 : 1;
    });
    blocks = seen;
    blocks.forEach(function (b, i) {
      b.classList.add("st-block");
      if (i === 0) b.classList.add("st-block--first");
      if (i === blocks.length - 1) b.classList.add("st-block--last");
      var a = b.parentElement;
      while (a && a !== document.body) { a.classList.add("st-stack-host"); a = a.parentElement; }
      // A spacer after each block (in the same parent, so the sticky range extends) is the hold.
      if (i < blocks.length - 1 && !REDUCED.matches) {
        var hold = document.createElement("div");
        hold.className = "st-hold";
        hold.setAttribute("aria-hidden", "true");
        b.after(hold);
        holds.push(hold);
      }
    });
    document.querySelectorAll("nav, header").forEach(function (n) {
      var a = n;
      while (a && a !== document.body) {
        if (getComputedStyle(a).position === "fixed") { a.classList.add("st-over"); break; }
        a = a.parentElement;
      }
    });
  }

  function layout() {
    var vh = window.innerHeight;
    var hold = Math.round(vh * (window.innerWidth < 810 ? HOLD_PHONE : HOLD_DESKTOP));
    holds.forEach(function (h) { h.style.height = hold + "px"; });
    blocks.forEach(function (b, i) {
      if (i === blocks.length - 1 || REDUCED.matches) { b.style.position = "relative"; b.style.top = ""; return; }
      b.style.position = "sticky";
      // The first block starts at the top of the page, so a positive offset would push it down.
      var top = vh - b.offsetHeight;
      b.style.top = (i === 0 ? Math.min(0, top) : top) + "px";
    });
    shade();
  }

  // As the next block slides over, the covered one recedes: it shrinks toward the top of the
  // screen and dims, so it reads as a card going behind rather than content being cut off.
  function shade() {
    var vh = window.innerHeight;
    for (var i = 0; i < blocks.length - 1; i++) {
      var b = blocks[i];
      var nextTop = blocks[i + 1].getBoundingClientRect().top;
      var covered = Math.max(0, Math.min(1, (vh - nextTop) / vh));
      b.style.setProperty("--st-cover", (covered * 0.45).toFixed(3));
      if (covered > 0 && !REDUCED.matches) {
        var originY = -b.getBoundingClientRect().top; // viewport top, in the block's own coordinates
        b.style.transformOrigin = "50% " + originY.toFixed(0) + "px";
        b.style.transform = "scale(" + (1 - covered * 0.08).toFixed(4) + ")";
        b.classList.add("st-block--receding");
      } else if (b.style.transform) {
        b.style.transform = "";
        b.style.transformOrigin = "";
        b.classList.remove("st-block--receding");
      }
    }
  }

  function init() {
    mergeGroups();
    collect();
    layout();
    var ticking = false;
    window.addEventListener("scroll", function () {
      if (ticking) return;
      ticking = true;
      requestAnimationFrame(function () { shade(); ticking = false; });
    }, { passive: true });
    var t;
    window.addEventListener("resize", function () {
      clearTimeout(t);
      t = setTimeout(function () { collect(); layout(); }, 120);
    });
    window.addEventListener("load", layout);
    // Galleries and carousels change height after they build; keep offsets in step.
    if ("ResizeObserver" in window) {
      var ro = new ResizeObserver(function () { layout(); });
      blocks.forEach(function (b) { ro.observe(b); });
    }
  }

  if (document.readyState === "loading") document.addEventListener("DOMContentLoaded", init);
  else init();
})();
