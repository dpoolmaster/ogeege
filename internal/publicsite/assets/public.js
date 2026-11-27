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

  function focusAfterSwap(event) {
    var target = event.detail && event.detail.target;
    if (!target) {
      return;
    }
    var focusTarget = target.querySelector("[data-autofocus]");
    if (focusTarget) {
      focusTarget.focus();
    }
  }

  function init() {
    renderMath();
  }

  if (document.readyState === "loading") {
    document.addEventListener("DOMContentLoaded", init);
  } else {
    init();
  }

  document.body.addEventListener("htmx:afterSwap", function (event) {
    init();
    focusAfterSwap(event);
  });
})();
