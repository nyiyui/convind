import { MarkdownEditor } from './markdownEditor.js';

class Page extends HTMLElement {
  constructor(id) {
    super();
    this.id = id;
    this.editor = new MarkdownEditor();
    this.classNames = [];
    this.progressIndicator = null;
    fetch(`/api/v1/page/${id}`)
      .then((resp) => resp.text()).then((text) => this.editor.setValue(text));
  }
  progressSetIndeterminate() {
    this.progressIndicator.style.visibility = "visible";
    this.progressIndicator.removeAttribute("value");
  }
  progressSetDone() {
    this.progressIndicator.value = this.progressIndicator.max;
    this.progressIndicator.style.visibility = "hidden";
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
      let done = false;
      setTimeout(() => {
        if (!done) this.progressSetIndeterminate();
      }, 100);
      const newSource = this.editor.getEditorContent();
      const resp = await fetch(`/api/v1/page/${this.id}`, { method: "POST", body: newSource });
      done = true;
      this.progressSetDone();
      if (!resp.ok) {
        throw new Error(`resp not ok: ${resp.status}`);
      }
    });
    this.progressIndicator = document.createElement("progress");
    this.progressIndicator.max = 100;
    this.progressSetDone();
    wrapper.appendChild(this.progressIndicator);
    wrapper.appendChild(this.editor);
    shadow.appendChild(wrapper);
  }
}

window.customElements.define("wiki-page", Page);

export { Page }
