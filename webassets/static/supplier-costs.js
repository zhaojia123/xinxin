(() => {
  const root = document.querySelector("[data-supplier-costs]");
  if (!root) return;
  const toast = message => {
    const target = document.querySelector(".demo-toast");
    if (!target) return;
    target.textContent = message;
    target.classList.add("show");
    window.setTimeout(() => target.classList.remove("show"), 2200);
  };
  const escapeHTML = value => String(value ?? "").replace(/[&<>'\"]/g, char => ({"&":"&amp;","<":"&lt;",">":"&gt;","'":"&#39;",'"':"&quot;"}[char]));
  const request = async (url, options = {}) => {
    const response = await fetch(url, options);
    const body = await response.json().catch(() => ({}));
    if (!response.ok) throw new Error(body.error || "操作失败，请稍后重试");
    return body;
  };
  const stalls = [...document.querySelectorAll("#supplier-cost-stall-options option")].map(option => ({id: option.value, name: option.textContent}));
  const suppliers = [...document.querySelectorAll("#supplier-cost-options option")].map(option => option.value);
  const options = selected => `<option value="">请选择</option>${stalls.map(stall => `<option value="${escapeHTML(stall.id)}" ${String(stall.id) === String(selected) ? "selected" : ""}>${escapeHTML(stall.name)}</option>`).join("")}`;
  const supplierOptions = selected => `<option value="">请选择供货商</option>${suppliers.map(name => `<option value="${escapeHTML(name)}" ${name === selected ? "selected" : ""}>${escapeHTML(name)}</option>`).join("")}`;
  const openDialog = (data = {}) => {
    const editing = Boolean(data.id);
    const overlay = document.createElement("div");
    overlay.className = "record-dialog";
    overlay.innerHTML = `<div class="record-dialog-card"><div class="record-dialog-head"><div><small>供货商对账</small><h2>${editing ? "编辑对账" : "新增对账"}</h2></div><button type="button" data-close>×</button></div><form class="record-dialog-form"><div class="record-form-grid"><label>月份 *<input name="month" type="month" required value="${escapeHTML(data.month || root.dataset.month)}"></label><label>档口 *<select name="stall_id" required>${options(data.stallId)}</select></label><label>供货商 *<select name="supplier_name" required>${supplierOptions(data.supplierName || "")}</select></label><label>食材费用（元）<input name="ingredient_amount" type="number" min="0" step="0.01" value="${escapeHTML(data.ingredientAmount || "0")}"></label><label>米油费用（元）<input name="rice_oil_amount" type="number" min="0" step="0.01" value="${escapeHTML(data.riceOilAmount || "0")}"></label><label>备注（选填）<input name="remark" maxlength="255" value="${escapeHTML(data.remark || "")}"></label></div><div class="notice-card compact-notice"><span>算</span><div><strong>食材费用 + 米油费用 = 档口成本</strong><p>保存后会直接联动到档口成本、月度核算和月度统一入账。</p></div></div><div class="record-dialog-error" role="alert"></div><div class="form-actions"><button type="button" class="ghost-button" data-close>取消</button><button type="submit" class="primary-button">保存对账</button></div></form></div>`;
    document.body.appendChild(overlay);
    const close = () => overlay.remove();
    overlay.querySelectorAll("[data-close]").forEach(button => button.addEventListener("click", close));
    overlay.addEventListener("click", event => { if (event.target === overlay) close(); });
    overlay.querySelector("form").addEventListener("submit", async event => {
      event.preventDefault();
      const form = event.currentTarget;
      const button = form.querySelector("button[type=submit]");
      button.disabled = true;
      try {
        const values = Object.fromEntries(new FormData(form).entries());
        values.stall_id = Number(values.stall_id);
        values.cost_type = "ingredient";
        values.all_self_operated = false;
        await request(`/api/admin/stall-costs${editing ? `?id=${data.id}` : ""}`, {method: editing ? "PUT" : "POST", headers: {"Content-Type":"application/json"}, body: JSON.stringify(values)});
        close(); window.location.reload();
      } catch (error) { form.querySelector(".record-dialog-error").textContent = error.message; button.disabled = false; }
    });
  };
  document.querySelector("[data-supplier-cost-add]")?.addEventListener("click", () => openDialog());
  root.addEventListener("click", async event => {
    const row = event.target.closest("[data-supplier-cost-row]");
    if (!row) return;
    if (event.target.closest("[data-supplier-cost-edit]")) {
      openDialog({id: row.dataset.id, stallId: row.dataset.stallId, month: row.dataset.month, supplierName: row.dataset.supplierName, ingredientAmount: row.dataset.ingredientAmount, riceOilAmount: row.dataset.riceOilAmount, remark: row.dataset.remark});
      return;
    }
    if (event.target.closest("[data-supplier-cost-delete]")) {
      if (!window.confirm("确定删除这条供货商对账吗？")) return;
      try { await request(`/api/admin/stall-costs?id=${row.dataset.id}`, {method: "DELETE"}); window.location.reload(); } catch (error) { toast(error.message); }
    }
  });
})();
