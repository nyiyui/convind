import page from 'page';
import { navigateNewPage } from './wiki.js';

const main = document.querySelector('body > main');
const newPageButton = document.getElementById('button-page-new');

newPageButton.addEventListener('click', navigateNewPage);

page('/page/:id', showPage);
page(showNotFound);
page();

function showPage(ctx, next) {
  main.textContent = '';

  const id = ctx.params.id;
  fetch(`/api/v1/page/${encodeURIComponent(id)}`).then((resp) => resp.text()).then((text) => editor.setValue(text));

  let editor = document.createElement('markdown-editor');
  editor.id = 'main-editor';
  main.appendChild(editor);
  editor.editor.addEventListener('input', async (event) => {
    const newSource = editor.editor.value;
    const resp = await fetch(`/api/v1/page/${id}`, { method: "POST", body: newSource });
    if (!resp.ok) {
      throw new Error(`resp not ok: ${resp.status}`);
    }
  });
}

function showNotFound(ctx, next) {
  main.textContent = `route ${ctx.path} not found`;
}
