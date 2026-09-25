/* Draws the integration visual inside every .st-visual (port of the React Integration component).
   data-center: "logo" for the SomaTrack logo, otherwise a lucide icon name.
   data-icons:  comma-separated lucide icon names, up to six. */
(function () {
  var W = 564, H = 410;
  // Slot positions and connector paths from the original component (centre at 282, 205).
  var SLOTS = {
    tl: { x: 110, y: 90,  d: "M 270 205 V 105 Q 270 90 255 90 H 110" },
    tr: { x: 360, y: 70,  d: "M 294 205 V 85 Q 294 70 309 70 H 360" },
    ml: { x: 160, y: 205, d: "M 250 205 H 160" },
    mr: { x: 480, y: 205, d: "M 314 205 H 480" },
    b:  { x: 282, y: 360, d: "M 282 205 V 360" },
    br: { x: 460, y: 340, d: "M 314 215 V 325 Q 314 340 329 340 H 460" }
  };
  var ORDER = {
    1: ["mr"],
    2: ["ml", "mr"],
    3: ["tl", "mr", "b"],
    4: ["tl", "tr", "ml", "br"],
    5: ["tl", "tr", "ml", "mr", "br"],
    6: ["tl", "tr", "ml", "mr", "b", "br"]
  };
  var SVG_NS = "http://www.w3.org/2000/svg";

  function build(visual) {
    if (visual.dataset.built) return;
    visual.dataset.built = "1";
    var icons = (visual.dataset.icons || "").split(",").map(function (s) { return s.trim(); }).filter(Boolean).slice(0, 6);
    var slots = ORDER[icons.length] || [];

    var stage = document.createElement("div");
    stage.className = "st-stage";

    var svg = document.createElementNS(SVG_NS, "svg");
    svg.setAttribute("class", "st-lines");
    svg.setAttribute("viewBox", "0 0 " + W + " " + H);
    svg.setAttribute("preserveAspectRatio", "none");
    svg.setAttribute("aria-hidden", "true");
    slots.forEach(function (key) {
      ["st-line-base", "st-line-flow"].forEach(function (cls) {
        var p = document.createElementNS(SVG_NS, "path");
        p.setAttribute("d", SLOTS[key].d);
        p.setAttribute("class", cls);
        p.setAttribute("vector-effect", "non-scaling-stroke");
        if (cls === "st-line-flow") p.style.animationDelay = (-Math.random() * 4).toFixed(2) + "s";
        svg.appendChild(p);
      });
    });
    stage.appendChild(svg);

    var center = document.createElement("div");
    var inner = document.createElement("div");
    inner.className = "st-center-inner";
    if (visual.dataset.center === "logo") {
      center.className = "st-center st-center--logo";
      var img = document.createElement("img");
      img.src = "images/favicon.png";
      img.alt = "SomaTrack";
      inner.appendChild(img);
    } else {
      center.className = "st-center";
      inner.innerHTML = '<i data-lucide="' + (visual.dataset.center || "sparkles") + '"></i>';
    }
    center.appendChild(inner);
    stage.appendChild(center);

    icons.forEach(function (name, i) {
      var s = SLOTS[slots[i]];
      var tile = document.createElement("div");
      tile.className = "st-tile";
      tile.style.left = (s.x / W * 100) + "%";
      tile.style.top = (s.y / H * 100) + "%";
      tile.style.setProperty("--d", (0.1 + i * 0.1).toFixed(2) + "s");
      tile.innerHTML = '<i data-lucide="' + name + '"></i>';
      stage.appendChild(tile);
    });

    visual.appendChild(stage);
    visual.setAttribute("aria-hidden", "true");
  }

  function init() {
    var visuals = document.querySelectorAll(".st-visual");
    visuals.forEach(build);
    if (window.lucide) window.lucide.createIcons();

    // whileInView: tiles pop in once the card scrolls into view.
    if (!("IntersectionObserver" in window)) {
      visuals.forEach(function (v) { v.classList.add("is-in"); });
      return;
    }
    var io = new IntersectionObserver(function (entries) {
      entries.forEach(function (e) {
        if (e.isIntersecting) { e.target.classList.add("is-in"); io.unobserve(e.target); }
      });
    }, { threshold: 0.25 });
    visuals.forEach(function (v) { io.observe(v); });
  }

  if (document.readyState === "loading") document.addEventListener("DOMContentLoaded", init);
  else init();
})();
