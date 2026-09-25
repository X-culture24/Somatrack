/* Turns every .st-grid with three or more .st-card items into the feature carousel
   (port of the FeatureCarousel React component). Load after st-cards.js. */
(function () {
  var AUTO_PLAY_INTERVAL = 3000;
  var ITEM_HEIGHT = 65;
  var MIN_CARDS = 3;

  function wrap(min, max, v) {
    var range = max - min;
    return ((((v - min) % range) + range) % range) + min;
  }

  function setup(grid) {
    var cards = Array.prototype.slice.call(grid.children).filter(function (el) {
      return el.classList.contains("st-card");
    });
    var n = cards.length;
    if (n < MIN_CARDS) return;

    var root = document.createElement("div");
    root.className = "st-carousel";
    var rail = document.createElement("div");
    rail.className = "st-car-rail";
    var chips = document.createElement("div");
    chips.className = "st-car-chips";
    rail.appendChild(chips);
    var stageWrap = document.createElement("div");
    stageWrap.className = "st-car-stage-wrap";
    var stage = document.createElement("div");
    stage.className = "st-car-stage";
    stageWrap.appendChild(stage);
    root.appendChild(rail);
    root.appendChild(stageWrap);

    var step = 0, paused = false, visible = false, timer = null;
    var rows = [];

    cards.forEach(function (c, i) {
      var title = (c.querySelector("h3") || {}).textContent || "Item " + (i + 1);
      var visual = c.querySelector(".st-visual");
      var icon = visual && visual.dataset.center && visual.dataset.center !== "logo" ? visual.dataset.center : "sparkles";

      var row = document.createElement("div");
      row.className = "st-chip-row";
      var chip = document.createElement("button");
      chip.type = "button";
      chip.className = "st-chip";
      chip.setAttribute("aria-label", "Show " + title);
      chip.innerHTML = '<i data-lucide="' + icon + '"></i><span></span>';
      chip.querySelector("span").textContent = title;
      chip.addEventListener("click", function () { go(i); });
      chip.addEventListener("mouseenter", function () { paused = true; });
      chip.addEventListener("mouseleave", function () { paused = false; });
      chip.addEventListener("focus", function () { paused = true; });
      chip.addEventListener("blur", function () { paused = false; });
      row.appendChild(chip);
      chips.appendChild(row);
      rows.push({ row: row, chip: chip });

      var count = document.createElement("span");
      count.className = "st-car-count";
      count.textContent = String(i + 1).padStart(2, "0") + " / " + String(n).padStart(2, "0");
      if (visual) visual.appendChild(count); else c.insertBefore(count, c.firstChild);
      c.addEventListener("mouseenter", function () { paused = true; });
      c.addEventListener("mouseleave", function () { paused = false; });
      stage.appendChild(c);
    });

    grid.parentNode.replaceChild(root, grid);
    if (window.lucide) window.lucide.createIcons({ nameAttr: "data-lucide" });

    function current() { return ((step % n) + n) % n; }

    function status(i) {
      var d = i - current();
      if (d > n / 2) d -= n;
      if (d < -n / 2) d += n;
      if (d === 0) return "active";
      if (d === -1) return "prev";
      if (d === 1) return "next";
      return "hidden";
    }

    function render() {
      var cur = current();
      rows.forEach(function (r, i) {
        var dist = wrap(-(n / 2), n / 2, i - cur);
        r.row.style.transform = "translateY(" + dist * ITEM_HEIGHT + "px)";
        r.row.style.opacity = Math.max(0, 1 - Math.abs(dist) * 0.25);
        r.chip.classList.toggle("is-active", i === cur);
        r.chip.tabIndex = Math.abs(dist) <= 3 ? 0 : -1;
      });
      cards.forEach(function (c, i) {
        var st = status(i);
        c.dataset.status = st;
        c.setAttribute("aria-hidden", st === "active" ? "false" : "true");
        c.querySelectorAll("a, button").forEach(function (el) { el.tabIndex = st === "active" ? 0 : -1; });
      });
      // Make sure the card's visual animates when it comes to the front.
      var v = cards[cur].querySelector(".st-visual");
      if (v) v.classList.add("is-in");
    }

    function go(index) {
      var diff = (index - current() + n) % n;
      if (diff > 0) { step += diff; render(); restart(); }
    }

    function sizeStage() {
      // Cards are stacked absolutely, so the stage takes the tallest card's height.
      var h = 0;
      cards.forEach(function (c) { h = Math.max(h, c.offsetHeight); });
      stage.style.height = h + "px";
    }

    function tick() { if (!paused && visible) { step += 1; render(); } }
    function restart() {
      clearInterval(timer);
      if (!window.matchMedia("(prefers-reduced-motion: reduce)").matches) timer = setInterval(tick, AUTO_PLAY_INTERVAL);
    }

    render();
    sizeStage();
    window.addEventListener("resize", sizeStage);
    window.addEventListener("load", sizeStage);
    if ("IntersectionObserver" in window) {
      new IntersectionObserver(function (es) { visible = es[0].isIntersecting; }, { threshold: 0.3 }).observe(root);
    } else {
      visible = true;
    }
    restart();
  }

  function init() {
    document.querySelectorAll(".st-grid").forEach(setup);
  }

  if (document.readyState === "loading") document.addEventListener("DOMContentLoaded", init);
  else init();
})();
