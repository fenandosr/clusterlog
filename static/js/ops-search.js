(() => {
  "use strict";

  const input = document.getElementById("ops-search-input");
  const results = document.getElementById("ops-search-results");
  if (!input || !results) return;

  const indexURL = input.dataset.indexUrl;
  let documentsPromise = null;

  const normalize = (value) => String(value ?? "")
    .normalize("NFD")
    .replace(/\p{Diacritic}/gu, "")
    .toLocaleLowerCase("es")
    .replace(/\s+/g, " ")
    .trim();

  const loadDocuments = () => {
    if (documentsPromise) return documentsPromise;
    documentsPromise = fetch(indexURL, { credentials: "same-origin" })
      .then((response) => {
        if (!response.ok) throw new Error(`HTTP ${response.status}`);
        return response.json();
      })
      .then((index) => {
        const docs = index?.documentStore?.docs ?? {};
        return Object.entries(docs).map(([ref, doc]) => ({ ref, ...doc }));
      });
    return documentsPromise;
  };

  const scoreDocument = (doc, tokens, wholeQuery) => {
    const title = normalize(doc.title);
    const description = normalize(doc.description);
    const content = normalize(doc.content ?? doc.body);
    const path = normalize(doc.path ?? doc.ref);
    const fullText = `${title} ${description} ${path} ${content}`;

    if (!tokens.every((token) => fullText.includes(token))) return -1;

    let score = 0;
    if (title === wholeQuery) score += 100;
    if (title.startsWith(wholeQuery)) score += 40;
    if (title.includes(wholeQuery)) score += 25;
    if (description.includes(wholeQuery)) score += 12;

    for (const token of tokens) {
      if (title.includes(token)) score += 12;
      if (description.includes(token)) score += 6;
      if (path.includes(token)) score += 4;
      if (content.includes(token)) score += 2;
    }
    return score;
  };

  const clearResults = () => {
    results.hidden = true;
    results.replaceChildren();
  };

  const appendMessage = (title, detail) => {
    const item = document.createElement("div");
    item.className = "ops-search-result";
    const strong = document.createElement("strong");
    strong.textContent = title;
    const small = document.createElement("small");
    small.textContent = detail;
    item.append(strong, small);
    results.replaceChildren(item);
    results.hidden = false;
  };

  const renderMatches = (matches) => {
    const fragment = document.createDocumentFragment();
    for (const { doc } of matches) {
      const link = document.createElement("a");
      link.className = "ops-search-result";
      link.href = doc.permalink ?? doc.ref ?? doc.path ?? "/";

      const title = document.createElement("strong");
      title.textContent = doc.title || link.href;
      const detail = document.createElement("small");
      detail.textContent = doc.description || doc.path || "";
      link.append(title, detail);
      fragment.append(link);
    }
    results.replaceChildren(fragment);
    results.hidden = false;
  };

  let requestSerial = 0;
  const render = async (rawQuery) => {
    const serial = ++requestSerial;
    const query = normalize(rawQuery);
    const tokens = [...new Set(query.split(" ").filter((token) => token.length > 1))];
    if (query.length < 2 || tokens.length === 0) {
      clearResults();
      return;
    }

    try {
      const documents = await loadDocuments();
      if (serial !== requestSerial) return;
      const matches = documents
        .map((doc) => ({ doc, score: scoreDocument(doc, tokens, query) }))
        .filter((item) => item.score >= 0)
        .sort((a, b) => b.score - a.score || String(a.doc.title).localeCompare(String(b.doc.title), "es"))
        .slice(0, 12);

      if (matches.length === 0) {
        appendMessage("Sin resultados", "Pruebe otro término o una etiqueta más corta.");
        return;
      }
      renderMatches(matches);
    } catch (error) {
      console.warn("No se pudo cargar el índice de búsqueda", error);
      appendMessage("Búsqueda no disponible", "Compruebe que el sitio se construyó con el índice habilitado.");
    }
  };

  let debounceTimer = null;
  input.addEventListener("input", () => {
    clearTimeout(debounceTimer);
    debounceTimer = setTimeout(() => render(input.value), 100);
  });
  input.addEventListener("keydown", (event) => {
    if (event.key === "Escape") {
      input.value = "";
      clearResults();
      input.blur();
    }
  });
  document.addEventListener("keydown", (event) => {
    if (event.key === "/" && document.activeElement !== input && !event.metaKey && !event.ctrlKey && !event.altKey) {
      event.preventDefault();
      input.focus();
    }
  });
  document.addEventListener("click", (event) => {
    if (!results.contains(event.target) && event.target !== input) results.hidden = true;
  });
})();
