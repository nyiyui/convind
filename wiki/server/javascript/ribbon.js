import { Page } from './page.js';

class Ribbon extends HTMLElement {
  constructor(ids) {
    super();
    this.ids = ids;
    this.pages = null;
  }
  connectedCallback() {
    const shadow = this.attachShadow({mode: "open"});
    const style = document.createElement("style");
    style.textContent = `
    .wrapper {
      display: flex;
      flex-direction: row;
    }
    .wiki-page {
      box-shadow: 0 0 8px black;
      padding: 8px;
      margin: 8px;
    }
    `;
    shadow.appendChild(style);

    const wrapper = document.createElement("div");
    wrapper.classList.add('wrapper');
    this.pages = this.ids.map((id) => new Page(id));
    this.pages.forEach((page) => {
      const w = document.createElement('div');
      w.classList.add('wiki-page');
      w.appendChild(page);
      wrapper.appendChild(w);
    });
    shadow.appendChild(wrapper);
  }
}

window.customElements.define("wiki-ribbon", Ribbon);

export { Ribbon }
