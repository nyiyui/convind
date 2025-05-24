import page from 'page';
import { navigateNewPage } from './wiki.js';
import { Ribbon } from './ribbon.js';
import { Data } from './data.js';
import { PageList } from './page-list.js';

const main = document.querySelector('body > main');

page('/', () => page.redirect('/page-list'));
page('/ribbon', showRibbon);
page('/data/:id', showData);
page('/page-list', showPageList);
page('/page/new', () => navigateNewPage());
page(showNotFound);
page();

function showRibbon(ctx, next) {
  document.title = "ribbon";
  main.textContent = '';

  const params = new URLSearchParams(ctx.querystring);
  main.appendChild(new Ribbon(params.getAll("id")));
}

function showData(ctx, next) {
  document.title = "data";
  main.textContent = '';

  const id = ctx.params.id;
  main.appendChild(new Data(id));
}

function showPageList(ctx, next) {
  document.title = "page list";
  main.textContent = '';

  main.appendChild(new PageList());
}

function showNotFound(ctx, next) {
  main.textContent = `route ${ctx.path} not found`;
}
