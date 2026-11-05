(() => {
  const escapeHTML = value => String(value ?? "").replace(/[&<>'"]/g, char => ({"&":"&amp;","<":"&lt;",">":"&gt;","'":"&#39;",'"':"&quot;"}[char]));
  const money = value => `¥${Number(value || 0).toFixed(2)}`;
  const request = async (url, options = {}) => {
    const response = await fetch(url, options);
    const body = await response.json().catch(() => ({}));
    if (!response.ok) throw new Error(body.error || "操作失败，请稍后重试");
    return body;
  };

  const openEditor = ({ title, entry = {}, onSaved }) => {
    const overlay = document.createElement("div");
    overlay.className = "record-dialog";
    overlay.innerHTML = `<div class="record-dialog-card"><div class="record-dialog-head"><div><small>档口成本</small><h2>${escapeHTML(title)}</h2></div><button type="button" data-close>×</button></div><form class="record-dialog-form"><div class="record-form-grid"><label>调整金额（元）<input name="amount" type="number" min="0.01" step="0.01" required value="${escapeHTML(entry.ingredient_amount || "")}"></label><label>成本类型<select name="cost_type"><option value="other" ${entry.cost_type !== "labor" ? "selected" : ""}>其他成本</option><option value="labor" ${entry.cost_type === "labor" ? "selected" : ""}>人工成本</option></select></label><label class="span-2">备注（必填）<textarea name="remark" maxlength="255" required placeholder="例如：临时补货、特殊支出">${escapeHTML(entry.remark || "")}</textarea></label></div><div class="notice-card compact-notice"><span>调</span><div><strong>不会写入供货商对账表</strong><p>调整金额单独保存为成本明细，并自动计入该档口当月成本合计。</p></div></div><div class="record-dialog-error" role="alert"></div><div class="form-actions"><button type="button" class="ghost-button" data-close>取消</button><button type="submit" class="primary-button">保存调整</button></div></form></div>`;
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
        await onSaved(Object.fromEntries(new FormData(form).entries()));
        close();
      } catch (error) {
        form.querySelector(".record-dialog-error").textContent = error.message;
        button.disabled = false;
      }
    });
  };

  const openAdjustments = async trigger => {
    const month = trigger.dataset.month;
    const stallID = Number(trigger.dataset.stallId);
    const stallName = trigger.dataset.stallName;
    const overlay = document.createElement("div");
    overlay.className = "record-dialog";
    overlay.innerHTML = `<div class="record-dialog-card adjustment-list-dialog"><div class="record-dialog-head"><div><small>档口成本</small><h2>${escapeHTML(stallName)} · 特殊调整明细</h2></div><button type="button" data-close>×</button></div><div class="adjustment-list-loading">正在读取调整明细…</div></div>`;
    document.body.appendChild(overlay);
    const close = () => overlay.remove();
    overlay.querySelector("[data-close]").addEventListener("click", close);
    overlay.addEventListener("click", event => { if (event.target === overlay) close(); });
    const reloadPage = () => window.location.reload();
    const load = async () => {
      const data = await request(`/api/admin/stall-costs?month=${encodeURIComponent(month)}`);
      const entries = (data.items || []).filter(item => Number(item.stall_id) === stallID && !item.supplier_name);
      const total = entries.reduce((sum, item) => sum + Number(item.total_amount || 0), 0);
      const content = `<div class="adjustment-list-summary"><strong>本月特殊调整：${money(total)}</strong><button class="primary-button compact" type="button" data-adjustment-add>＋ 新增调整</button></div><div class="table-wrap"><table><thead><tr><th>类型</th><th>金额</th><th>备注</th><th>状态</th><th>操作</th></tr></thead><tbody>${entries.length ? entries.map(item => `<tr><td>${escapeHTML(item.cost_type_label || "其他成本")}</td><td><strong>${money(item.total_amount)}</strong></td><td>${escapeHTML(item.remark || "—")}</td><td>${item.ledger_posted ? "已入账" : "未入账"}</td><td>${item.ledger_posted ? "—" : `<button type="button" data-adjustment-edit="${item.id}">编辑</button><button type="button" class="danger-link" data-adjustment-delete="${item.id}">删除</button>`}</td></tr>`).join("") : `<tr><td colspan="5" class="empty-cell">暂无特殊调整，可点击上方新增。</td></tr>`}</tbody></table></div>`;
      overlay.querySelector(".adjustment-list-loading").outerHTML = content;
      overlay.querySelector("[data-adjustment-add]").addEventListener("click", () => openEditor({title: `${stallName} · 新增特殊调整`, onSaved: async values => { await request("/api/admin/stall-costs", {method: "POST", headers: {"Content-Type": "application/json"}, body: JSON.stringify({stall_id: stallID, month, cost_type: values.cost_type, ingredient_amount: values.amount, rice_oil_amount: "0", remark: values.remark})}); reloadPage(); }}));
      overlay.querySelectorAll("[data-adjustment-edit]").forEach(button => button.addEventListener("click", () => {
        const entry = entries.find(item => Number(item.id) === Number(button.dataset.adjustmentEdit));
        openEditor({title: `${stallName} · 编辑特殊调整`, entry, onSaved: async values => { await request(`/api/admin/stall-costs?id=${entry.id}`, {method: "PUT", headers: {"Content-Type": "application/json"}, body: JSON.stringify({stall_id: stallID, month, cost_type: values.cost_type, ingredient_amount: values.amount, rice_oil_amount: "0", supplier_name: "", remark: values.remark})}); reloadPage(); }});
      }));
      overlay.querySelectorAll("[data-adjustment-delete]").forEach(button => button.addEventListener("click", async () => {
        if (!window.confirm("确定删除这条特殊调整吗？")) return;
        try { await request(`/api/admin/stall-costs?id=${button.dataset.adjustmentDelete}`, {method: "DELETE"}); reloadPage(); } catch (error) { window.alert(error.message); }
      }));
    };
    try { await load(); } catch (error) { overlay.querySelector(".adjustment-list-loading").textContent = error.message; }
  };

  document.addEventListener("click", event => {
    const trigger = event.target.closest("[data-stall-adjustment]");
    if (trigger) openAdjustments(trigger);
  });
})();
