import { Parser, HtmlRenderer } from 'commonmark';


function indexOfPos(pos, source) {
  const [line, col] = pos.split(':').map((x) => parseInt(x));
  let index = 0;
  const lines = source.split('\n');
  for (let i = 0; i < line-1; i ++) {
    index += lines[i].length + 1;
  }
  return index+(col-1);
}

function fixCorvindLinks(walker) {
  for (let entry = walker.next(); entry != null; entry = walker.next()) {
    const { node } = entry;
    if (node.type == "link") {
      if (node.destination.startsWith("convind://")) {
        node.destination = "/page/" + node.destination.slice(10)
      }
    }
  }
}

class MarkdownEditor extends HTMLElement {
  constructor() {
    super();
    this.source = '';
    this.parser = new Parser();
    this.renderer = new HtmlRenderer({
      sourcepos: true,
      softbreak: "<br />",
    });
    this.viewer = document.createElement("div");
    this.editor = document.createElement("div");
    this.editor.contentEditable = true;
  }
  connectedCallback() {
    const shadow = this.attachShadow({mode: "open"});
    const style = document.createElement("style");
    style.textContent = `
    .wrapper {
      display: flex;
    }
    .viewer, .editor {
      flex: 1;
      overflow: scroll;
    }
    .editor {
      font-family: monospace;
    }
    /* TODO: where to place editor on tall screens? */
    `;
    shadow.appendChild(style);

    const wrapper = document.createElement("div");
    wrapper.classList.add('wrapper');
    this.viewer.classList.add('viewer');
    this.viewer.addEventListener("click", this.onClick.bind(this));
    this.render();
    wrapper.appendChild(this.viewer);
    this.editor.classList.add('editor');
    this.refreshEditorContent();
    this.editor.addEventListener("input", this.onChange.bind(this));
    this.editor.addEventListener("paste", this.onPaste.bind(this));
    wrapper.appendChild(this.editor);
    shadow.appendChild(wrapper);
  }
  refreshEditorContent() {
    this.editor.textContent = '';
    const ol = document.createElement('ol');
    this.editor.appendChild(ol);
    this.source.split('\n').forEach((line) => {
      console.log('line', line);
      const elem = document.createElement('li');
      elem.textContent = line;
      ol.appendChild(elem);
    });
  }
  getEditorContent() {
    return Array.from(this.editor.querySelectorAll('ol > *'))
      .map((elem) => elem.textContent)
      .join('\n')
  }
  setValue(s) {
    this.source = s;
    this.refreshEditorContent();
    this.render();
  }
  onChange(event) {
    this.source = this.getEditorContent();
    console.log(this.source);
    this.render();
  }
  onPaste(event) {
    console.log(event.clipboardData);
    console.log(event.clipboardData.files);
    console.log(event.clipboardData.files[0]);
    Array.from(event.clipboardData.items).forEach(async (item) => {
      let textToAdd = '';
      if (item.kind === "file") {
        const resp = await fetch('/api/v1/data/new', {
          method: "POST",
          body: item.getAsFile(),
          headers: {
            "Content-Type": item.type,
          },
        });
        if (!resp.ok) throw new Error(resp.status);
        const id = (new URL(resp.url)).pathname.split('/').slice(-1);
        const isImage = item.type.startsWith("image/");
        textToAdd = (isImage ? '!' : '') + `[](/api/v1/data/${id})`;
      } else if (item.kind === "string") {
        textToAdd += await (new Promise((resolve, reject) => {
          item.getAsString(resolve);
        }));
      }
      console.log(textToAdd);
      const selection = window.getSelection();
      if (!selection.rangeCount) return;
      selection.deleteFromDocument();
      selection.getRangeAt(0)
        .insertNode(document.createTextNode(textToAdd));
      selection.collapseToEnd();
    });
    event.preventDefault();
    this.onChange(null);
  }
  render() {
    const parsed = this.parser.parse(this.source)
    fixCorvindLinks(parsed.walker());
    const result = this.renderer.render(parsed);
    console.log(result);
    this.viewer.innerHTML = result;
  }
  onClick(event) {
    if (event.target == "A") return;
    const sourcepos = event.target.dataset.sourcepos;
    if (!sourcepos) return; // give up
    const [startPos, endPos] = sourcepos.split("-");
    const [startLine, startCol] = startPos.split(":").map(parseInt);
    this.editor.firstChild.children[startLine-1].focus();
    console.log(this.editor.firstChild.children[startLine-1]);
    //this.editor.setSelectionRange(startIndex, endIndex+1);
  }
  onPotentialAutocomplete() {
    const selection = window.getSelection();
    if (!selection.rangeCount) return;
    const range = selection.getRangeAt(0);
    if (range.startContainer !== range.endContainer) return;
    if (range.startOffset !== range.endOffset) return;
    const node = range.startContainer;
    // TODO: detect: (`|` represents cursor)
    //       - by title - once at `[title to search for]|`, search pages by title (and replace with autolink)
    //       - insert new page - once at `[blah](convind://new)|`, make new page and replace `convind://new` link
  }
}

window.customElements.define("markdown-editor", MarkdownEditor);

export { MarkdownEditor }
