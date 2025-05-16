import { Page } from './page.js';
import { Image } from './image.js';

class Data extends HTMLElement {
  constructor(id) {
    super();
    this.id = id;
    this.showElem = null;
    this.constructors = [
      ['text/markdown', (id) => new Page(id)],
      [/.*/, (id) => new Image(id)],
    ];
    this.classNames = null;
    this.instancesWrapper = null;
  }
  connectedCallback() {
    this.instancesWrapper = document.createElement("div");
    this.instancesWrapper.classList.add('instances-wrapper');

    const shadow = this.attachShadow({mode: "open"});
    const style = document.createElement("style");
    style.textContent = `
    .wrapper {
      display: flex;
      flex-direction: row;
    }
    .wrapper > :not(.instances-wrapper) {
      flex: 2;
    }
    .instances-wrapper {
      flex: 1;
    }
    `;
    shadow.appendChild(style);

    const wrapper = document.createElement("div");
    wrapper.classList.add('wrapper');
    fetch(`/api/v1/data/${this.id}`)
      .then((resp) => {
        const mimeType = resp.headers.get('Content-Type');
        for (let [pattern, make] of this.constructors) {
          if (pattern === mimeType || (pattern.test && pattern.test(mimeType))) {
            this.showElem = make(this.id);
            wrapper.insertBefore(this.showElem, wrapper.firstChild);
            break;
          }
        }
      })
    wrapper.appendChild(this.instancesWrapper);
    shadow.appendChild(wrapper);

    this.loadClasses();
  }
  loadClasses() {
    fetch(`/api/v1/data/${this.id}/instances`)
      .then((resp) => resp.json()).then((classNames) => {
        console.log('classNames', classNames);
        this.classNames = classNames;
        this.loadInstances();
      });
  }
  async loadInstances() {
    console.log('this.classNames', this.classNames);
    let elems = await Promise.all(this.classNames.map(async (className) => {
      const instanceUrl = `/api/v1/data/${this.id}/instance/${encodeURIComponent(className)}`;
      const resp = await fetch(instanceUrl);
      if (!resp.ok) return;
      if (className === "inaba.kiyuri.ca/2025/convind/wiki") {
        return { className, elem: this.loadWikiInstance(await resp.json()) };
      }
      if (resp.headers.get("Content-Type").startsWith("image/")) {
        const img = document.createElement('img');
        img.src = instanceUrl;
        return { className, elem: img }
      }
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
      const e = document.createElement("div");
      const h2 = document.createElement("h2");
      h2.textContent = className;
      e.appendChild(h2);
      e.appendChild(elem);
      this.instancesWrapper.appendChild(e);
    })
  }
  loadWikiInstance(data) {
    const hops = document.createElement("div");
    const hop1 = document.createElement("ul");
    const hop2 = document.createElement("ul");
    hops.classList.add("hops");

    const hop1Wrapper = document.createElement("div");
    hop1Wrapper.appendChild(document.createElement("h2"));
    hop1Wrapper.firstChild.textContent = "1 hop";
    hop1Wrapper.appendChild(hop1);

    const hop2Wrapper = document.createElement("div");
    hop2Wrapper.appendChild(document.createElement("h2"));
    hop2Wrapper.firstChild.textContent = "2 hop";
    hop2Wrapper.appendChild(hop2);

    hops.appendChild(hop1Wrapper);
    hops.appendChild(hop2Wrapper);

    const markdownOnly = (page) => page.MIMEType === 'text/markdown';
    data["1"].filter(markdownOnly).forEach((page) => {
      if (page.ID == this.id) return;
      const a = document.createElement("a");
      a.href = `/data/${page.ID}`;
      a.textContent = page.Title ? page.Title : page.ID;
      const li = document.createElement("li");
      li.appendChild(a);
      hop1.appendChild(li);
    });
    data["2"].filter(markdownOnly).forEach((page) => {
      if (page.ID == this.id) return;
      const a = document.createElement("a");
      console.log('page', page);
      a.href = `/data/${page.ID}`;
      a.textContent = page.Title ? page.Title : page.ID;
      const li = document.createElement("li");
      li.appendChild(a);
      hop2.appendChild(li);
    });
    console.log('hops', hops);
    return hops;
  }
}

window.customElements.define("wiki-data", Data);

export { Data }
