const revisionIdElem = document.getElementById("revision-id");
function getId() {
  const params = new URLSearchParams(window.location.search);
  return params.get("id");
}

async function navigateNewPage() {
  const resp = await fetch("/api/v1/page/new", { method: "POST", body: "" });
  if (!resp.ok) {
    throw new Error(`resp not ok: ${resp.status}`);
  }
  if (!resp.redirected) {
    throw new Error("should be redirected");
  }
  const id = (new URL(resp.url)).pathname.split('/').slice(-1);
  window.location.search = `?id=${id}`;
}

async function onInput(event) {
  const newSource = mainEditor.editor.value;
  const resp = await fetch(`/api/v1/page/${getId()}`, { method: "POST", body: newSource });
  if (!resp.ok) {
    throw new Error(`resp not ok: ${resp.status}`);
  }
  revisionIdElem.textContent = resp.headers.get("Revision-ID");
}

const id = getId();
const mainEditor = document.getElementById("main-editor");
const resp = await fetch(`/api/v1/page/${encodeURIComponent(id)}`);
revisionIdElem.textContent = resp.headers.get("Revision-ID");
const source = await resp.text();
mainEditor.setValue(source);

document.getElementById("button-page-new").addEventListener("click", navigateNewPage);
mainEditor.editor.addEventListener("input", onInput);
