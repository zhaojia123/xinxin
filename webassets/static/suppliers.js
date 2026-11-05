(() => {
  const root = document.querySelector("[data-suppliers]");
  if (!root) return;
  // 供货商页未加载公共脚本时，仍保持两小时无操作自动退出。
  const idleTimeout = 2 * 60 * 60 * 1000;
  let idleTimer;
  const logout = async () => {
    try { await fetch("/admin/logout", {method: "POST", credentials: "same-origin"}); } catch (_) {}
    window.location.assign("/admin/login");
  };
  const resetIdleTimer = () => {
    window.clearTimeout(idleTimer);
    idleTimer = window.setTimeout(logout, idleTimeout);
  };
  ["pointerdown", "keydown", "scroll", "touchstart", "focus"].forEach(eventName => window.addEventListener(eventName, resetIdleTimer, {passive: true}));
  resetIdleTimer();
  document.querySelector("[data-admin-logout]")?.addEventListener("click", logout);
  const escapeHTML = value => String(value ?? "").replace(/[&<>'\"]/g, char => ({"&":"&amp;","<":"&lt;",">":"&gt;","'":"&#39;",'"':"&quot;"}[char]));
  const toast = message => { const node = document.querySelector(".demo-toast"); if (!node) return; node.textContent = message; node.classList.add("show"); window.setTimeout(() => node.classList.remove("show"), 2200); };
  const request = async (url, options = {}) => { const response = await fetch(url, options); const body = await response.json().catch(() => ({})); if (!response.ok) throw new Error(body.error || "操作失败，请稍后重试"); return body; };
  const openDialog = (data = {}) => {
    const editing = Boolean(data.id); const overlay = document.createElement("div"); overlay.className = "record-dialog";
    overlay.innerHTML = `<div class="record-dialog-card"><div class="record-dialog-head"><div><small>供货商管理</small><h2>${editing ? "编辑供货商" : "新增供货商"}</h2></div><button type="button" data-close>×</button></div><form class="record-dialog-form"><div class="record-form-grid"><label>供货商名称 *<input name="name" maxlength="64" required value="${escapeHTML(data.name || "")}"></label><label>联系人<input name="contact" maxlength="64" value="${escapeHTML(data.contact || "")}"></label><label>联系电话<input name="mobile" maxlength="32" value="${escapeHTML(data.mobile || "")}"></label><label>备注<input name="remark" maxlength="255" value="${escapeHTML(data.remark || "")}"></label></div><div class="record-dialog-error" role="alert"></div><div class="form-actions"><button type="button" class="ghost-button" data-close>取消</button><button type="submit" class="primary-button">保存供货商</button></div></form></div>`;
    document.body.appendChild(overlay); const close = () => overlay.remove(); overlay.querySelectorAll("[data-close]").forEach(button => button.addEventListener("click", close)); overlay.addEventListener("click", event => { if (event.target === overlay) close(); });
    overlay.querySelector("form").addEventListener("submit", async event => { event.preventDefault(); const form = event.currentTarget; const button = form.querySelector("button[type=submit]"); button.disabled = true; try { await request(`/api/admin/suppliers${editing ? `?id=${data.id}` : ""}`, {method: editing ? "PUT" : "POST", headers:{"Content-Type":"application/json"}, body:JSON.stringify(Object.fromEntries(new FormData(form).entries()))}); close(); window.location.reload(); } catch (error) { form.querySelector(".record-dialog-error").textContent = error.message; button.disabled = false; } });
  };
  document.querySelector("[data-supplier-add]")?.addEventListener("click", () => openDialog());
  root.addEventListener("click", async event => { const row = event.target.closest("[data-supplier-row]"); if (!row) return; if (event.target.closest("[data-supplier-edit]")) { openDialog({id:row.dataset.id,name:row.dataset.name,contact:row.dataset.contact,mobile:row.dataset.mobile,remark:row.dataset.remark}); return; } if (event.target.closest("[data-supplier-delete]")) { if (!window.confirm(`确定停用“${row.dataset.name}”吗？`)) return; try { await request(`/api/admin/suppliers?id=${row.dataset.id}`, {method:"DELETE"}); window.location.reload(); } catch (error) { toast(error.message); } } });
})();
