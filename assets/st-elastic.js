/* Turns every .st-grid with three or more .st-card items into the elastic gallery
   (port of the ElasticGallery React component). Load after st-cards.js. */
(function () {
  var MIN_CARDS = 3;
  var DESKTOP = window.matchMedia("(min-width: 768px)");

  function setup(grid) {
    var cards = Array.prototype.slice.call(grid.children).filter(function (el) {
      return el.classList.contains("st-card");
    });
    var n = cards.length;
    if (n < MIN_CARDS) return;

    var root = document.createElement("div");
    root.className = "st-elastic";
    var panels = [];
    var active = Math.floor((n - 1) / 2); // open the middle panel first, like the original

    cards.forEach(function (card, i) {
      var title = (card.querySelector("h3") || {}).textContent || "Item " + (i + 1);
      var visual = card.querySelector(".st-visual");
      var icon = visual && visual.dataset.center && visual.dataset.center !== "logo" ? visual.dataset.center : "sparkles";

      var panel = document.createElement("div");
      panel.className = "st-el-panel";
      panel.tabIndex = 0;
      panel.setAttribute("role", "button");
      panel.setAttribute("aria-label", title);

      var cover = document.createElement("div");
      cover.className = "st-el-cover";
      cover.innerHTML = '<span class="st-el-num"></span><span class="st-el-title"></span><span class="st-el-icon"><i data-lucide="' + icon + '"></i></span>';
      cover.querySelector(".st-el-num").textContent = String(i + 1).padStart(2, "0");
      cover.querySelector(".st-el-title").textContent = title;

      panel.appendChild(card);
      panel.appendChild(cover);
      root.appendChild(panel);
      panels.push(panel);

      function open() { if (active !== i) { active = i; render(); } }
      panel.addEventListener("mouseenter", function () { if (DESKTOP.matches) open(); });
      panel.addEventListener("click", open);
      panel.addEventListener("focus", open);
      panel.addEventListener("keydown", function (e) {
        if (e.key === "ArrowRight" || e.key === "ArrowDown") { e.preventDefault(); panels[(i + 1) % n].focus(); }
        if (e.key === "ArrowLeft" || e.key === "ArrowUp") { e.preventDefault(); panels[(i - 1 + n) % n].focus(); }
      });
    });

    grid.parentNode.replaceChild(root, grid);
    if (window.lucide) window.lucide.createIcons();

    function render() {
      panels.forEach(function (p, i) {
        var on = i === active;
        p.classList.toggle("is-active", on);
        p.setAttribute("aria-expanded", on ? "true" : "false");
        p.querySelectorAll(".st-card a").forEach(function (a) { a.tabIndex = on ? 0 : -1; });
      });
      var v = cards[active].querySelector(".st-visual");
      if (v) v.classList.add("is-in");
      if (!DESKTOP.matches) sizeMobile();
    }

    function sizeDesktop() {
      // The active panel gets 4 of (n + 3) flex parts; cards keep that width so text never reflows.
      var gap = 14;
      var w = (root.clientWidth - gap * (n - 1)) * 4 / (n + 3);
      root.style.setProperty("--st-el-w", Math.round(w) + "px");
      var h = 0;
      cards.forEach(function (c) {
        c.style.height = "auto";
        c.style.width = Math.round(w) + "px";
        var body = c.querySelector(".st-card-body");
        // A capped visual plus the body gives the height the card needs.
        h = Math.max(h, Math.round(Math.min(w * 0.5, 280)) + (body ? body.scrollHeight : 0));
        c.style.height = "";
        c.style.width = "";
      });
      root.style.setProperty("--st-el-h", Math.max(h, 400) + "px");
    }

    function sizeMobile() {
      var p = panels[active];
      var card = cards[active];
      p.style.setProperty("--st-el-open", card.scrollHeight + "px");
    }

    function layout() {
      if (DESKTOP.matches) sizeDesktop(); else sizeMobile();
    }

    render();
    layout();
    window.addEventListener("resize", layout);
    window.addEventListener("load", layout);
  }

  function init() {
    document.querySelectorAll(".st-grid").forEach(setup);
  }

  if (document.readyState === "loading") document.addEventListener("DOMContentLoaded", init);
  else init();
})();
