class Image extends HTMLElement {
  constructor(id) {
    super();
    this.id = id;
  }
  connectedCallback() {
    const shadow = this.attachShadow({mode: "open"});
    const style = document.createElement("style");
    style.textContent = `
    img {
      max-width: 600px;
      max-height: 600px;
    }
    `;
    shadow.appendChild(style);

    const wrapper = document.createElement("div");
    wrapper.classList.add('wrapper');
    const e = document.createElement('img');
    e.src = `/api/v1/data/${this.id}`;
    wrapper.appendChild(e);
    shadow.appendChild(wrapper);
  }
}

window.customElements.define("wiki-image", Image);

export { Image }

