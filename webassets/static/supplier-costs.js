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
  const amount = value => Number(value || 0);
  const money = value => `¥${amount(value).toFixed(2)}`;

  // 保存成功后只更新当前格与相关汇总，不再整页刷新横向大表。
  const setMoney = (element, value) => { if (element) element.textContent = money(value); };
  const updateCellAndTotals = (cell, nextAmount) => {
    const previous = amount(cell.dataset.totalAmount);
    const next = amount(nextAmount);
    const difference = next - previous;
    const riceOilOnly = cell.dataset.riceOilOnly === "true";
    cell.dataset.totalAmount = String(next);
    cell.innerHTML = next > 0 ? `<strong>${money(next)}</strong><small>点击修改</small>` : `<span>—</span><small>点击录入</small>`;
    if (riceOilOnly || difference === 0) return;
    const rowTotal = cell.closest("tr")?.querySelector("td.supplier-matrix-total strong");
    if (rowTotal) setMoney(rowTotal, amount(rowTotal.textContent.replace(/[^0-9.-]/g, "")) + difference);
    const footer = root.querySelector("tfoot tr");
    const columnTotal = footer?.cells[cell.cellIndex]?.querySelector("strong");
    if (columnTotal) setMoney(columnTotal, amount(columnTotal.textContent.replace(/[^0-9.-]/g, "")) + difference);
    const grandTotal = footer?.querySelector("td.supplier-matrix-total strong");
    if (grandTotal) setMoney(grandTotal, amount(grandTotal.textContent.replace(/[^0-9.-]/g, "")) + difference);
  };

  const openCostDialog = cell => {
    if (cell.dataset.readOnly === "true") {
      toast("该格已入账或包含多条历史明细，请到档口成本页处理");
      return;
    }
    const supplierName = cell.dataset.supplierName;
    const selector = `th[data-stall-id="${cell.dataset.stallId}"]`;
    const stallName = root.querySelector(selector)?.textContent || "当前档口";
    const overlay = document.createElement("div");
    overlay.className = "record-dialog";
    const riceOilOnly = cell.dataset.riceOilOnly === "true";
    const costLabel = riceOilOnly ? "米油费用（元）" : "食材费用（元）";
    overlay.innerHTML = `<div class="record-dialog-card supplier-matrix-dialog"><div class="record-dialog-head"><div><small>供货商档口对账</small><h2>${escapeHTML(supplierName)} · ${escapeHTML(stallName)}</h2></div><button type="button" data-close>×</button></div><form class="record-dialog-form"><div class="record-form-grid"><label>${costLabel}<input name="amount" type="number" min="0" step="0.01" value="${escapeHTML(cell.dataset.totalAmount || "0")}"></label></div><div class="notice-card compact-notice"><span>算</span><div><strong>金额会自动汇总到对应档口成本</strong><p>普通供货商计入食材费用，“米油”行计入米油费用；填写 0 后保存会清空这条未入账数据。</p></div></div><div class="record-dialog-error" role="alert"></div><div class="form-actions"><button type="button" class="ghost-button" data-close>取消</button><button type="submit" class="primary-button">保存金额</button></div></form></div>`;
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
        values.month = root.dataset.month;
        values.stall_id = Number(cell.dataset.stallId);
        values.supplier_name = supplierName;
        values.ingredient_amount = riceOilOnly ? "0" : values.amount;
        values.rice_oil_amount = riceOilOnly ? values.amount : "0";
        delete values.amount;
        await request("/api/admin/supplier-cost-matrix", {method: "PUT", headers: {"Content-Type":"application/json"}, body: JSON.stringify(values)});
        updateCellAndTotals(cell, values[riceOilOnly ? "rice_oil_amount" : "ingredient_amount"]);
        close();
        toast("已保存，档口成本汇总已同步更新");
      } catch (error) {
        form.querySelector(".record-dialog-error").textContent = error.message;
        button.disabled = false;
      }
    });
  };

  root.addEventListener("click", event => {
    const cell = event.target.closest("[data-supplier-matrix-cell]");
    if (cell) openCostDialog(cell);
  });
})();
