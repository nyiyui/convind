import { MarkdownEditor } from './markdownEditor.js';

class Page extends HTMLElement {
  constructor(id) {
    super();
    this.id = id;
    this.editor = new MarkdownEditor();
    this.hops = document.createElement("div");
    this.hop1 = document.createElement("ul");
    this.hop2 = document.createElement("ul");
    this.instancesWrapper = document.createElement("div");
    this.classNames = [];
    fetch(`/api/v1/page/${id}`)
      .then((resp) => resp.text()).then((text) => this.editor.setValue(text));
    fetch(`/api/v1/data/${id}/instances`)
      .then((resp) => resp.json()).then((classNames) => {
        console.log('classNames', classNames);
        this.classNames = classNames;
        this.loadInstances();
      });
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
      console.log(newSource);
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

    this.hops.appendChild(hop1Wrapper);
    this.hops.appendChild(hop2Wrapper);
    wrapper.appendChild(this.instancesWrapper);
    shadow.appendChild(wrapper);
  }
  async loadInstances() {
    console.log('this.classNames', this.classNames);
    let elems = await Promise.all(this.classNames.map(async (className) => {
      const resp = await fetch(`/api/v1/data/${this.id}/instance/${encodeURIComponent(className)}`);
      if (!resp.ok) return;
      if (resp.headers.get("Content-Type").startsWith("text/")) {
        return { className, elem: document.createTextNode(await resp.text()) }
      }
      if (resp.headers.get("Content-Type") === "application/json") {
        const elem = document.createElement("code");
        elem.textContent = JSON.stringify(await resp.json(), null, 1);
        return { className, elem };
      }
      return null;
    }));
    elems = elems.filter((entry) => !!entry);
    this.instancesWrapper.textContent = '';
    elems.forEach(({ className, elem }) => {
      console.log(elem);
      const e = document.createElement("div");
      const h2 = document.createElement("h2");
      h2.textContent = className;
      e.appendChild(h2);
      e.appendChild(elem);
      this.instancesWrapper.appendChild(e);
    })
  }
}

window.customElements.define("wiki-page", Page);

export { Page }
