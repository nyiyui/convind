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
      .then((pageEntries) => {
        const pageEntries2 = pageEntries  
          .filter((pageEntry) => pageEntry.Data.Revisions.length !== 0)
          .filter((pageEntry) => pageEntry.Data.MIMEType === "text/markdown")
          .sort((a, b) => latestCreationTime(b.Data) - latestCreationTime(a.Data))
        pageEntries2
          .forEach((pageEntry) => {
            const li = document.createElement("li");
            const a = document.createElement("a");
            a.href = `/data/${pageEntry.Data.ID}`;
            a.textContent = pageEntry.LatestRevisionTitle
            a.textContent = a.textContent;
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
