(function () {
  "use strict";

  function renderMath() {
    if (typeof window.renderMathInElement !== "function") {
      return;
    }
    window.renderMathInElement(document.body, {
      delimiters: [
        { left: "$$", right: "$$", display: true },
        { left: "$", right: "$", display: false }
      ],
      throwOnError: false
    });
  }

  function initDemoProgress() {
    document.querySelectorAll("[data-demo-form]").forEach(function (form) {
      if (form.dataset.demoProgressReady === "true") {
        return;
      }
      form.dataset.demoProgressReady = "true";

      var progress = form.querySelector(".v-progress progress");
      var counter = form.querySelector(".v-progress .v-nums");
      if (!progress || !counter) {
        return;
      }

      function update() {
        var total = form.querySelectorAll("[data-demo-task]").length;
        var filled = 0;
        form.querySelectorAll("[data-demo-task] input[type=text]").forEach(function (input) {
          if (input.value.trim() !== "") {
            filled += 1;
          }
        });
        progress.value = String(filled);
        counter.textContent = filled + " из " + total;
      }

      form.addEventListener("input", update);
      update();
    });
  }

  function init() {
    renderMath();
    initDemoProgress();
  }

  if (document.readyState === "loading") {
    document.addEventListener("DOMContentLoaded", init);
  } else {
    init();
  }

  document.body.addEventListener("htmx:afterSwap", init);
})();
