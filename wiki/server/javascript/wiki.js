import page from 'page';


async function navigateNewPage() {
  const resp = await fetch("/api/v1/page/new", { method: "POST", body: "" });
  if (!resp.ok) {
    throw new Error(`resp not ok: ${resp.status}`);
  }
  if (!resp.redirected) {
    throw new Error("should be redirected");
  }
  const id = (new URL(resp.url)).pathname.split('/').slice(-1);
  page(`/data/${id}`);
}

export { navigateNewPage };
