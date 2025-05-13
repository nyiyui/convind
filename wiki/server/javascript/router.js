import page from 'page';
import { navigateNewPage } from './wiki.js';
import { Page } from './page.js';
import { PageList } from './page-list.js';

const main = document.querySelector('body > main');
const newPageButton = document.getElementById('button-page-new');

newPageButton.addEventListener('click', navigateNewPage);

page('/page/:id', showPage);
page('/page-list', showPageList);
page(showNotFound);
page();

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
