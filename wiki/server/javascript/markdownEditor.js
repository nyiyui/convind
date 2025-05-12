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

class MarkdownEditor extends HTMLElement {
  constructor() {
    super();
    this.source = '# Title\nbody text';
    this.source += '\n\na'.repeat(30);
    this.parser = new Parser();
    this.renderer = new HtmlRenderer({sourcepos: true});
    this.viewer = null;
    this.editor = null;
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
    /* TODO: where to place editor on tall screens? */
    `;
    shadow.appendChild(style);

    const wrapper = document.createElement("div");
    wrapper.classList.add('wrapper');
    this.viewer = document.createElement("div");
    this.viewer.classList.add('viewer');
    this.viewer.addEventListener("click", this.onClick.bind(this));
    this.render();
    wrapper.appendChild(this.viewer);
    this.editor = document.createElement("textarea");
    this.editor.classList.add('editor');
    this.editor.value = this.source;
    this.editor.addEventListener("input", this.onChange.bind(this));
    wrapper.appendChild(this.editor);
    shadow.appendChild(wrapper);
  }
  onChange(event) {
    this.source = this.editor.value;
    this.render();
  }
  render() {
    const parsed = this.parser.parse(this.source)
    const result = this.renderer.render(parsed);
    this.viewer.innerHTML = result;
  }
  onClick(event) {
    const sourcepos = event.target.dataset.sourcepos;
    console.log(sourcepos);
    const [startIndex, endIndex] = sourcepos.split("-").map((pos) => indexOfPos(pos, this.source));
    this.editor.focus();
    this.editor.setSelectionRange(startIndex, endIndex+1);
  }
}

window.customElements.define("markdown-editor", MarkdownEditor);
