import { MarkdownEditor } from './markdownEditor.js';

function formatTime(t, { alwaysAbsolute }) {
  const f = new Intl.DateTimeFormat("en-CA", {
    weekday: "short",
    year: "numeric",
    month: "short",
    day: "2-digit",
    hour: "2-digit",
    minute: "2-digit",
    hour12: false,
  });

  if (alwaysAbsolute) {
    return f.format(t);
  }

  const delta = Math.floor((Date.now() - t.getTime()) / 1000);
  if (delta < 60) {
    return "less than a minute ago";
  } else if (delta < 2*3600) {
    if (1 === Math.floor(delta/60)) {
      return "a minute ago";
    }
    return `${Math.floor(delta/60)} minutes ago`;
  } else if (delta < 24*3600) {
    return `${Math.floor(delta/3600)} hours and ${Math.floor(delta/60) % 60} minutes ago`;
  } else {
    return f.format(t);
  }
}

class Page extends HTMLElement {
  constructor(id) {
    super();
    this.id = id;
    this.editor = new MarkdownEditor();
    this.classNames = [];
    this.progressIndicator = null;
    this.latestRevisionIndicator = null;
  }
  progressSetIndeterminate() {
    this.progressIndicator.style.visibility = "visible";
    this.progressIndicator.removeAttribute("value");
  }
  progressSetDone() {
    this.progressIndicator.value = this.progressIndicator.max;
    this.progressIndicator.style.visibility = "hidden";
  }
  async loadSource() {
    const resp = await fetch(`/api/v1/page/${this.id}`);
    resp.text().then((text) => this.editor.setValue(text));
    const lastModified = new Date(Date.parse(resp.headers.get('Last-Modified')));
    this.latestRevisionIndicator.textContent = formatTime(lastModified, { alwaysAbsolute: false });
    this.latestRevisionIndicatorPrint.textContent = formatTime(lastModified, { alwaysAbsolute: true });
    
    // Get the revision ID from the Revision-ID header
    const revisionId = resp.headers.get('Revision-ID');
    
    // Emit revisionChanged event with the loaded revision ID
    this.dispatchEvent(new CustomEvent('revisionChanged', {
      bubbles: true,
      composed: true, // This allows the event to cross shadow DOM boundaries
      detail: {
        id: this.id,
        revisionId: revisionId,
        timestamp: lastModified
      }
    }));
  }
  connectedCallback() {
    this.loadSource();
    const shadow = this.attachShadow({mode: "open"});
    const style = document.createElement("style");
    style.textContent = `
    .print-show {
      display: none;
    }
    @media print {
      .print-hide {
        display: none;
      }
      .print-show {
        display: initial;
      }
    }
    `;
    shadow.appendChild(style);

    const wrapper = document.createElement("div");
    wrapper.classList.add('wrapper');
    this.editor.id = 'main-editor';
    
    // Listen for title changes from the editor
    this.editor.addEventListener('titlechange', (event) => {
      const title = event.detail.title;
      // Update the document title (just the page title)
      document.title = title || 'no title';
    });
    
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
      
      // Get the revision ID from the Revision-ID header
      const revisionId = resp.headers.get('Revision-ID');
      
      // Emit revisionChanged event with the new revision ID
      this.dispatchEvent(new CustomEvent('revisionChanged', {
        bubbles: true,
        composed: true, // This allows the event to cross shadow DOM boundaries
        detail: {
          id: this.id,
          revisionId: revisionId,
          timestamp: new Date()
        }
      }));
    });
    this.progressIndicator = document.createElement("progress");
    this.progressIndicator.max = 100;
    this.progressSetDone();
    this.latestRevisionIndicator = document.createElement("span");
    this.latestRevisionIndicator.classList.add('print-hide');
    this.latestRevisionIndicatorPrint = document.createElement("span");
    this.latestRevisionIndicatorPrint.classList.add('print-show');
    wrapper.appendChild(this.latestRevisionIndicator);
    wrapper.appendChild(this.latestRevisionIndicatorPrint);
    wrapper.appendChild(this.progressIndicator);
    wrapper.appendChild(this.editor);
    shadow.appendChild(wrapper);
  }
}

window.customElements.define("wiki-page", Page);

export { Page }
