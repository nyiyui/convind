import { MarkdownEditor } from './markdownEditor.js';

class Page extends HTMLElement {
  constructor(id) {
    super();
    this.id = id;
    this.editor = new MarkdownEditor();
    this.hops = document.createElement("div");
    this.hop1 = document.createElement("ul");
    this.hop2 = document.createElement("ul");
    this.hop1Back = document.createElement("ul");
    fetch(`/api/v1/page/${id}`)
      .then((resp) => resp.text()).then((text) => this.editor.setValue(text));
    fetch(`/api/v1/page/${id}/hop`).then((resp) => resp.json()).then((data) => {
      data["1"].forEach((page) => {
        if (page.ID == this.id) return;
        const a = document.createElement("a");
        a.href = `/page/${page.ID}`;
        a.textContent = page.Title ? page.Title : page.ID;
        const li = document.createElement("li");
        li.appendChild(a);
        this.hop1.appendChild(li);
      });
      data["2"].forEach((page) => {
        if (page.ID == this.id) return;
        const a = document.createElement("a");
        a.href = `/page/${page.ID}`;
        a.textContent = page.Title ? page.Title : page.ID;
        const li = document.createElement("li");
        li.appendChild(a);
        this.hop2.appendChild(li);
      });
      data["-1"].forEach((page) => {
        if (page.ID == this.id) return;
        const a = document.createElement("a");
        a.href = `/page/${page.ID}`;
        a.textContent = page.Title ? page.Title : page.ID;
        const li = document.createElement("li");
        li.appendChild(a);
        this.hop1Back.appendChild(li);
      });
    });
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
      const resp = await fetch(`/api/v1/page/${this.id}`, { method: "POST", body: newSource });
      if (!resp.ok) {
        throw new Error(`resp not ok: ${resp.status}`);
      }
    });
    wrapper.appendChild(this.editor);
    this.hops.classList.add("hops");
    wrapper.appendChild(this.hops);

    const hop1Wrapper = document.createElement("div");
    hop1Wrapper.appendChild(document.createElement("h2"));
    hop1Wrapper.firstChild.textContent = "1 hop";
    hop1Wrapper.appendChild(this.hop1);

    const hop2Wrapper = document.createElement("div");
    hop2Wrapper.appendChild(document.createElement("h2"));
    hop2Wrapper.firstChild.textContent = "2 hop";
    hop2Wrapper.appendChild(this.hop2);

    const hop1BackWrapper = document.createElement("div");
    hop1BackWrapper.appendChild(document.createElement("h2"));
    hop1BackWrapper.firstChild.textContent = "backlinks";
    hop1BackWrapper.appendChild(this.hop1Back);

    this.hops.appendChild(hop1Wrapper);
    this.hops.appendChild(hop2Wrapper);
    this.hops.appendChild(hop1BackWrapper);
    shadow.appendChild(wrapper);
  }
}

window.customElements.define("wiki-page", Page);

export { Page }
