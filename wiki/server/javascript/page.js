import { MarkdownEditor } from './markdownEditor.js';

class Page extends HTMLElement {
  constructor(id) {
    super();
    this.id = id;
    this.editor = new MarkdownEditor();
    this.classNames = [];
    fetch(`/api/v1/page/${id}`)
      .then((resp) => resp.text()).then((text) => this.editor.setValue(text));
  }
  connectedCallback() {
    const shadow = this.attachShadow({mode: "open"});
    const style = document.createElement("style");
    style.textContent = `
    `;
    shadow.appendChild(style);

    const wrapper = document.createElement("div");
    wrapper.classList.add('wrapper');
    this.editor.id = 'main-editor';
    this.editor.editor.addEventListener('input', async (event) => {
      const newSource = this.editor.getEditorContent();
      console.log(newSource);
      const resp = await fetch(`/api/v1/page/${this.id}`, { method: "POST", body: newSource });
      if (!resp.ok) {
        throw new Error(`resp not ok: ${resp.status}`);
      }
    });
    wrapper.appendChild(this.editor);
    shadow.appendChild(wrapper);
  }
}

window.customElements.define("wiki-page", Page);

export { Page }
