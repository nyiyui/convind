import page from 'page';
import { navigateNewPage } from './wiki.js';
import { Ribbon } from './ribbon.js';
import { Page } from './page.js';
import { PageList } from './page-list.js';

const main = document.querySelector('body > main');
const newPageButton = document.getElementById('button-page-new');

newPageButton.addEventListener('click', navigateNewPage);

page('/ribbon', showRibbon);
page('/page/:id', showPage);
page('/page-list', showPageList);
page(showNotFound);
page();

function showRibbon(ctx, next) {
  main.textContent = '';

  const params = new URLSearchParams(ctx.querystring);
  main.appendChild(new Ribbon(params.getAll("id")));
}

function showPage(ctx, next) {
  main.textContent = '';

  const id = ctx.params.id;
  main.appendChild(new Page(id));
}

function showPageList(ctx, next) {
  main.textContent = '';

  main.appendChild(new PageList());
}

function showNotFound(ctx, next) {
  main.textContent = `route ${ctx.path} not found`;
}
