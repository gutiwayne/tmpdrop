// English / Spanish switch. The page is written in English; in Spanish the DOM is
// translated on the fly (text, placeholder, title, aria-label), including what is
// drawn later (MutationObserver), toasts and confirm() dialogs. File names are left
// alone. The choice is kept in localStorage; by default, the browser language.
(() => {
  "use strict";
  const KEY = "tmpdrop.lang";
  let lang;
  try { lang = localStorage.getItem(KEY); } catch (_) {}
  if (lang !== "es" && lang !== "en") lang = /^es\b/i.test(navigator.language || "") ? "es" : "en";

  const btn = document.getElementById("lang-btn");
  if (btn) {
    btn.textContent = "🌐 " + lang.toUpperCase();
    btn.addEventListener("click", () => {
      try { localStorage.setItem(KEY, lang === "en" ? "es" : "en"); } catch (_) {}
      location.reload();
    });
  }
  if (lang === "en") return;

  const D = {
    "tmpdrop — temporary file sharing": "tmpdrop — compartir archivos temporales",
    "Temporary file sharing": "Compartir archivos temporales",
    "Upload a file, share the link, it’s gone when the timer runs out.": "Sube un archivo, comparte el enlace y desaparece cuando se acaba el tiempo.",
    "Language / Idioma": "Idioma / Language", "Upload a file": "Subir un archivo", "Choose a file": "Elige un archivo",
    "or drag it here": "o arrástralo aquí", "Expiration": "Caducidad", "Server default": "Por defecto del servidor",
    "1 hour": "1 hora", "1 day": "1 día", "3 days": "3 días", "7 days": "7 días", "Downloads": "Descargas",
    "Leave empty for unlimited downloads": "Vacío = descargas ilimitadas", "Password": "Contraseña", "optional": "opcional",
    "Strip GPS/EXIF": "Quitar GPS/EXIF", "Upload": "Subir", "Uploading…": "Subiendo…", "Share link": "Enlace para compartir",
    "Copy": "Copiar", "Copy the share link": "Copiar el enlace", "Shared files": "Archivos compartidos", "Files": "Archivos",
    "No files yet. Share something above.": "Aún no hay archivos. Comparte algo arriba.",
    "tmpdrop — files are stored locally and removed automatically.": "tmpdrop — los archivos se guardan en local y se borran solos.",
    "Source code": "Código fuente", "Password required": "Contraseña necesaria", "Cancel": "Cancelar", "Unlock": "Desbloquear",
    "Name": "Nombre", "Size": "Tamaño", "Expires": "Caduca", "Actions": "Acciones", "Open": "Abrir", "Download": "Descargar",
    "Convert": "Convertir", "Convert ▾": "Convertir ▾", "Download ▾": "Descargar ▾", "Converting…": "Convirtiendo…", "Delete": "Borrar", "Deleted": "Borrado",
    "never": "nunca", "expired": "caducado", "Link copied": "Enlace copiado",
    "Could not copy — select the link above": "No se pudo copiar — selecciona el enlace de arriba",
    "Could not copy — the link is selected": "No se pudo copiar — el enlace está seleccionado",
    "Incorrect password": "Contraseña incorrecta", "Upload failed: network error": "Error al subir: fallo de red",
    // Server errors
    "a conversion is already running for this file": "ya hay una conversión en marcha para este archivo",
    "client file limit reached": "límite de archivos alcanzado", "client quota exceeded": "cuota superada",
    "conversion service is not configured": "el servicio de conversión no está configurado", "delete failed": "no se pudo borrar",
    "download limit reached": "límite de descargas alcanzado", "empty file": "archivo vacío",
    "file exceeds the size limit": "el archivo supera el tamaño máximo", "file expired": "archivo caducado",
    "file not found": "archivo no encontrado", "incorrect password": "contraseña incorrecta", "internal error": "error interno",
    "invalid file name": "nombre de archivo no válido", "invalid max_downloads": "número de descargas no válido",
    "invalid multipart form": "formulario no válido", "invalid request body": "petición no válida",
    "missing file field": "falta el archivo", "no conversion for this file": "no hay conversión para este archivo",
    "not found": "no encontrado", "password required": "contraseña necesaria", "storage error": "error de almacenamiento",
    "storage is full": "el almacenamiento está lleno", "too many attempts": "demasiados intentos",
    "too many password attempts": "demasiados intentos de contraseña", "unknown converter": "conversor desconocido",
    "unsupported target format": "formato de destino no soportado", "upload failed": "error al subir", "upload too large": "subida demasiado grande",
  };
  const t = (s) => D[s] || s;
  // Texts with names, numbers or server messages inside.
  const P = [
    [/^(\d+) files? · (.+) of (.+)$/, (m, a, b, c) => a + (a === "1" ? " archivo · " : " archivos · ") + b + " de " + c],
    [/^Could not load files: (.+)$/, (m, a) => "No se pudieron cargar los archivos: " + t(a)],
    [/^Delete "(.+)"\?$/, (m, a) => "¿Borrar «" + a + "»?"],
    [/^Delete failed: (.+)$/, (m, a) => "No se pudo borrar: " + t(a)],
    [/^Could not open: (.+)$/, (m, a) => "No se pudo abrir: " + t(a)],
    [/^Converting to (\.\S+)…$/, (m, a) => "Convirtiendo a " + a + "…"],
    [/^Convert failed: (.+)$/, (m, a) => "Error al convertir: " + t(a)],
    [/^Converted to (\.\S+) — it is now in the Download menu$/, (m, a) => "Convertido a " + a + " — ya está en el menú Descargar"],
    [/^Conversion failed: (.+)$/, (m, a) => "La conversión falló: " + t(a)],
    [/^Convert status error: (.+)$/, (m, a) => "Error al consultar la conversión: " + t(a)],
    [/^Upload failed: (.+)$/, (m, a) => "Error al subir: " + t(a)],
  ];
  function tr(s) {
    if (D[s]) return D[s];
    for (const [re, f] of P) if (re.test(s)) return s.replace(re, f);
    return s;
  }
  // File names, and anything marked data-no-i18n, are not translated.
  const SKIP = ".fname, #selected-file, #pw-file-name, #share-link, [data-no-i18n], script, style";
  function text(node) {
    const raw = node.nodeValue, s = raw.trim();
    if (!s || !/[A-Za-z]/.test(s)) return;
    const p = node.parentElement; if (p && p.closest(SKIP)) return;
    const out = tr(s); if (out !== s) node.nodeValue = raw.replace(s, out);
  }
  const ATTRS = ["placeholder", "title", "aria-label"];
  function attrs(el) {
    if (el.closest(SKIP)) return;
    for (const a of ATTRS) { const v = el.getAttribute(a); if (v) { const o = tr(v); if (o !== v) el.setAttribute(a, o); } }
  }
  // "files · used of max" is one string split by <strong>: translate the whole line.
  function usage() {
    const u = document.getElementById("usage"); if (!u) return;
    const m = /^(\d+) files? · $/.exec(u.firstChild && u.firstChild.nodeValue || "");
    if (m) u.firstChild.nodeValue = m[1] + (m[1] === "1" ? " archivo · " : " archivos · ");
    const last = u.lastChild;
    if (last && last.nodeType === 3 && /^ of /.test(last.nodeValue)) last.nodeValue = last.nodeValue.replace(/^ of /, " de ");
  }
  function walk(root) {
    if (root.nodeType === 3) { text(root); return; }
    if (root.nodeType !== 1 || root.matches("script, style")) return;
    attrs(root);
    root.querySelectorAll("[placeholder], [title], [aria-label]").forEach(attrs);
    const w = document.createTreeWalker(root, NodeFilter.SHOW_TEXT);
    for (let n; (n = w.nextNode());) text(n);
  }
  document.documentElement.lang = "es";
  document.title = t(document.title);
  walk(document.body);
  new MutationObserver((muts) => {
    for (const m of muts) {
      if (m.type === "characterData") text(m.target);
      else if (m.type === "attributes") attrs(m.target);
      else m.addedNodes.forEach(walk);
    }
    usage();
  }).observe(document.body, { subtree: true, childList: true, characterData: true, attributes: true, attributeFilter: ATTRS });
  const _confirm = window.confirm.bind(window);
  window.confirm = (msg) => _confirm(tr(String(msg)));
})();
