import { Page } from './page.js';
import { Image } from './image.js';

// Utility function to debounce frequent calls
function debounce(func, wait) {
  let timeout;
  return function(...args) {
    const context = this;
    clearTimeout(timeout);
    timeout = setTimeout(() => func.apply(context, args), wait);
  };
}

// Format class identifier into a user-friendly name
function formatClassName(className) {
  // Handle known special cases
  if (className === "inaba.kiyuri.ca/2025/convind/wiki") {
    return "Wiki Links";
  }
  
  // Extract last meaningful part of the path
  const parts = className.split('/');
  if (parts.length > 0) {
    const lastPart = parts[parts.length - 1];
    // If it's a descriptive name, use it directly
    if (lastPart && !lastPart.includes('.') && lastPart.length > 1) {
      // Capitalize and add spaces to camelCase/PascalCase
      return lastPart.replace(/([A-Z])/g, ' $1').trim();
    }
  }
  
  // For domain-based class identifiers, extract the domain part
  if (className.includes('.')) {
    const domainParts = className.split('.');
    if (domainParts.length > 1) {
      return domainParts[0].charAt(0).toUpperCase() + domainParts[0].slice(1);
    }
  }
  
  // Fallback: return the original but truncated if very long
  return className.length > 30 ? className.substring(0, 27) + '...' : className;
}

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
    
    // Create debounced version of loadClasses to avoid too many refreshes
    this.debouncedLoadClasses = debounce(() => {
      this.loadClasses();
    }, 500); // 500ms debounce time
  }
  connectedCallback() {
    this.instancesWrapper = document.createElement("div");
    this.instancesWrapper.classList.add('instances-wrapper');
    this.instancesWrapper.textContent = "Loading instances…";

    const shadow = this.attachShadow({mode: "open"});
    const style = document.createElement("style");
    style.textContent = `
    .wrapper {
      display: flex;
      flex-direction: row;
    }
    @media only screen and (max-width: 900px) {
      .wrapper {
        flex-direction: column;
      }
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
            
            // Listen for revisionChanged events from Page elements
            if (this.showElem instanceof Page) {
              this.showElem.addEventListener('revisionChanged', (event) => {
                console.log('Revision changed event received:', event.detail);
                // Reload the instances when a revision changes using the debounced method
                this.debouncedLoadClasses();
              });
            }
            
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
  async makeInstanceElem(className) {
    const instanceUrl = `/api/v1/data/${this.id}/instance/${encodeURIComponent(className)}`;
    const resp = await fetch(instanceUrl);
    if (!resp.ok) return;
    if (className === "inaba.kiyuri.ca/2025/convind/wiki") {
      return this.loadWikiInstance(await resp.json());
    }
    if (resp.headers.get("Content-Type").startsWith("image/")) {
      const img = document.createElement('img');
      img.src = instanceUrl;
      return img;
    }
    if (resp.headers.get("Content-Type").startsWith("text/")) {
      return document.createTextNode(await resp.text());
    }
    if (resp.headers.get("Content-Type") === "application/json") {
      const elem = document.createElement("code");
      elem.textContent = JSON.stringify(await resp.json(), null, 1);
      return elem;
    }
    return null;
  };
  async loadInstances() {
    console.log('this.classNames', this.classNames);
    this.instancesWrapper.textContent = '';
    
    // Create a map to store elements by className
    const elementsMap = new Map();
    
    // Process all class names and create their elements
    const processPromises = this.classNames.map(async (className) => {
      const elem = await this.makeInstanceElem(className);
      if (!elem) return;
      
      const e = document.createElement("div");
      e.dataset.className = className; // Store className for sorting
      const h2 = document.createElement("h2");
      h2.textContent = formatClassName(className);
      h2.title = className; // Keep the full identifier as a tooltip
      e.appendChild(h2);
      e.appendChild(elem);
      
      // Store in map
      elementsMap.set(className, e);
    });
    
    // Wait for all elements to be created
    await Promise.all(processPromises);
    
    // Sort the class names
    const sortedClassNames = Array.from(elementsMap.keys()).sort();
    
    // Add elements to the instancesWrapper in sorted order
    sortedClassNames.forEach(className => {
      const element = elementsMap.get(className);
      if (element) {
        this.instancesWrapper.appendChild(element);
      }
    });
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
      const context = document.createTextNode(" " +page.Context);
      const li = document.createElement("li");
      li.appendChild(a);
      li.appendChild(context);
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
    document.title = data.title;
    console.log('hops', hops);
    return hops;
  }
}

window.customElements.define("wiki-data", Data);

export { Data, formatClassName }
