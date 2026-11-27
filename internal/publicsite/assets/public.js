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

  if (document.readyState === "loading") {
    document.addEventListener("DOMContentLoaded", renderMath);
  } else {
    renderMath();
  }

  document.body.addEventListener("htmx:afterSwap", renderMath);
})();
