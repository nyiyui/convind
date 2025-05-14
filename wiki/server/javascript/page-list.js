function latestCreationTime(data) {
  const ts = data.Revisions
    .map((revision) => (new Date(revision.CreationTime)).getTime());
  return Math.max(...ts);
}

class PageList extends HTMLElement {
  constructor(id) {
    super();
  }
  connectedCallback() {
    const shadow = this.attachShadow({mode: "open"});
    const style = document.createElement("style");
    style.textContent = `
    `;
    shadow.appendChild(style);

    const wrapper = document.createElement("div");
    wrapper.classList.add('wrapper');
    const ul = document.createElement("ul");
    fetch('/api/v1/pages')
      .then((resp) => resp.json())
      .then((pages) => {
        pages  
          .filter((page) => page.Revisions.length !== 0)
          .filter((page) => page.MIMEType === "text/markdown")
          .sort((a, b) => { latestCreationTime(b) - latestCreationTime(a); })
          .forEach((page) => {
            const li = document.createElement("li");
            const a = document.createElement("a");
            a.href = `/page/${page.ID}`;
            a.textContent = page.ID;
            li.appendChild(a);
            ul.appendChild(li);
          });
      });
    wrapper.appendChild(ul);
    shadow.appendChild(wrapper);
  }
}

window.customElements.define("wiki-page-list", PageList);

export { PageList }
