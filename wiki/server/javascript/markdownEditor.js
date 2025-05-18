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
        node.destination = "/data/" + node.destination.slice(10)
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
      height: 100vh;
      position: relative;
    }
    .viewer {
      flex: 1;
      overflow: auto;
      padding-right: 1rem;
    }
    .editor {
      position: sticky;
      top: 0;
      flex: 1;
      height: 100vh;
      overflow: auto;
      font-family: monospace;
      box-shadow: -2px 0 5px rgba(0,0,0,0.1);
    }
    .viewer img {
      max-width: min(100%, max(70%, 512px));
    }
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
    this.editor.addEventListener("click", this.onEditorClick.bind(this));
    this.editor.addEventListener("keyup", this.onEditorClick.bind(this));
    wrapper.appendChild(this.editor);
    shadow.appendChild(wrapper);
  }
  refreshEditorContent() {
    this.editor.textContent = '';
    const ol = document.createElement('ol');
    this.editor.appendChild(ol);
    this.source.split('\n').forEach((line) => {
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
    this.render();
    
    // Trigger onEditorClick to find and scroll to the corresponding element in the viewer
    this.onEditorClick(event);
  }
  onPaste(event) {
    // Process only file items (direct files, not HTML)
    const hasFiles = Array.from(event.clipboardData.items).some(item => 
      item.kind === "file" && item.type.startsWith("image/"));
      
    if (hasFiles) {
      // If we have files, only process those and ignore text/html
      event.preventDefault();
      
      // Process all items
      Array.from(event.clipboardData.items).forEach(async (item) => {
        if (item.kind === "file") {
          const file = item.getAsFile();
          if (!file) return;
          
          if (file.type.startsWith("image/")) {
            try {
              const resp = await fetch('/api/v1/data/new', {
                method: "POST",
                body: file,
                headers: {
                  "Content-Type": file.type,
                },
              });
              
              if (!resp.ok) throw new Error(`HTTP error! status: ${resp.status}`);
              
              // Get ID from the response URL
              const url = new URL(resp.url);
              const urlParts = url.pathname.split('/');
              const id = urlParts[urlParts.length - 1]; // Get the last part of the path
              
              // Insert markdown for image
              const textToAdd = `![](/api/v1/data/${id})`;
              
              // Insert at cursor position
              const selection = window.getSelection();
              if (!selection.rangeCount) return;
              selection.deleteFromDocument();
              selection.getRangeAt(0).insertNode(document.createTextNode(textToAdd));
              selection.collapseToEnd();
              
              this.onChange(null);
              this.editor.dispatchEvent(new Event("input"));
            } catch (error) {
              console.error("Failed to upload image:", error);
            }
          } else {
            // Non-image file
            try {
              const resp = await fetch('/api/v1/data/new', {
                method: "POST",
                body: file,
                headers: {
                  "Content-Type": file.type || "application/octet-stream",
                },
              });
              
              if (!resp.ok) throw new Error(`HTTP error! status: ${resp.status}`);
              
              // Get ID from the response URL
              const url = new URL(resp.url);
              const urlParts = url.pathname.split('/');
              const id = urlParts[urlParts.length - 1]; // Get the last part of the path
              
              // Insert plain link for non-image files
              const textToAdd = `[](/api/v1/data/${id})`;
              
              const selection = window.getSelection();
              if (!selection.rangeCount) return;
              selection.deleteFromDocument();
              selection.getRangeAt(0).insertNode(document.createTextNode(textToAdd));
              selection.collapseToEnd();
              
              this.onChange(null);
              this.editor.dispatchEvent(new Event("input"));
            } catch (error) {
              console.error("Failed to upload file:", error);
            }
          }
        }
      });
      return;
    }
    
    // If no files are present, let the default paste behavior handle text content
    // This will fall through to the browser's default handling of text pasting
  }
  render() {
    const parsed = this.parser.parse(this.source)
    fixCorvindLinks(parsed.walker());
    const result = this.renderer.render(parsed);
    this.viewer.innerHTML = result;
    
    // Add links around images that point to /data/<id>
    this.wrapImagesWithLinks();
  }
  
  wrapImagesWithLinks() {
    // Find all images in the viewer that have src starting with /api/v1/data/
    const images = Array.from(this.viewer.querySelectorAll('img[src^="/api/v1/data/"]'));
    
    // Process each image
    images.forEach(img => {
      const src = img.getAttribute('src');
      const id = src.split('/').pop(); // Get the ID from the end of the URL
      
      // Only process images that aren't already wrapped in a link
      if (img.parentNode.tagName !== 'A') {
        // Create a link element that points to the data
        const link = document.createElement('a');
        link.href = `/data/${id}`;
        link.target = '_blank'; // Open in new tab
        
        // Replace the image with the link containing the image
        img.parentNode.insertBefore(link, img);
        link.appendChild(img);
      }
    });
  }
  onClick(event) {
    if (event.target == "A") return;
    const sourcepos = event.target.dataset.sourcepos;
    if (!sourcepos) return; // give up
    const [startPos, endPos] = sourcepos.split("-");
    const [startLine, startCol] = startPos.split(":").map(parseInt);
    this.editor.firstChild.children[startLine-1].focus();
    //this.editor.setSelectionRange(startIndex, endIndex+1);
  }
  onEditorClick(event) {
    // Get the current line number in the editor
    const selection = window.getSelection();
    if (!selection.rangeCount) return;
    
    // Find which list item contains the cursor
    let currentNode = selection.anchorNode;
    
    // Navigate up to find the LI element
    while (currentNode && currentNode.nodeName !== 'LI') {
      currentNode = currentNode.parentNode;
    }
    
    if (!currentNode) return;
    
    // Get the line number (zero-based index in the OL)
    const lineNumber = Array.from(currentNode.parentNode.children).indexOf(currentNode) + 1;
    
    // Find an element in the viewer with a sourcepos attribute starting with this line number
    const sourceposPattern = `${lineNumber}:`;
    
    // Look for elements with matching sourcepos in the viewer
    const elements = Array.from(this.viewer.querySelectorAll('[data-sourcepos]'));
    
    // Find elements whose sourcepos starts with our line number
    const matchingElements = elements.filter(el => {
      const sourcepos = el.dataset.sourcepos;
      if (!sourcepos) return false;
      
      const [startPos, _] = sourcepos.split('-');
      return startPos.startsWith(sourceposPattern);
    });
    
    // If we found a matching element, scroll to it
    if (matchingElements.length > 0) {
      // Sort by specificity (most specific match first)
      matchingElements.sort((a, b) => {
        const aPos = a.dataset.sourcepos.split('-')[0];
        const bPos = b.dataset.sourcepos.split('-')[0];
        return bPos.length - aPos.length;
      });
      
      const targetElement = matchingElements[0];
      this.scrollElementIntoView(targetElement);
    }
  }
  scrollElementIntoView(element) {
    if (!element) return;
    
    // Scroll the element into view with smooth behavior
    element.scrollIntoView({
      behavior: 'smooth',
      block: 'center'
    });
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
