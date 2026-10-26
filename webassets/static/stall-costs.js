(() => {
  const roots = [...document.querySelectorAll("[data-stall-costs], [data-stall-costs-other]")];
  if (!roots.length) return;
  const root = roots[0];

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
  const costOptionsNode = document.querySelector("#stall-cost-options");
  const stalls = [...(costOptionsNode?.options || [])].map(option => ({id:option.value, name:option.textContent, selfOperated:option.dataset.selfOperated === "true"}));
  const month = root.dataset.month;
  const typeOptions = [{value:"ingredient", label:"食材成本"}, {value:"water", label:"水费"}, {value:"electricity", label:"电费"}, {value:"labor", label:"人工"}, {value:"other", label:"其他"}];
  const selectOptions = (items, selected) => items.map(item => `<option value="${escapeHTML(item.value || item.id)}" ${String(item.value || item.id) === String(selected) ? "selected" : ""}>${escapeHTML(item.label || item.name)}</option>`).join("");

  const stallOptions = selected => `<option value="">请选择</option><optgroup label="自营档口"><option value="__all_self__" ${dataAllSelected(selected) ? "selected" : ""}>全部自营档口（水费/电费）</option>${selectOptions(stalls.filter(item => item.selfOperated), selected)}</optgroup><optgroup label="非自营档口">${selectOptions(stalls.filter(item => !item.selfOperated), selected)}</optgroup>`;
  const dataAllSelected = selected => String(selected) === "__all_self__";
  const openDialog = (data = {}) => {
    const editing = Boolean(data.id);
    const overlay = document.createElement("div");
    overlay.className = "record-dialog";
    const selectedStall = data.allSelfOperated ? "__all_self__" : data.stallId;
    overlay.innerHTML = `<div class="record-dialog-card"><div class="record-dialog-head"><div><small>档口成本</small><h2>${editing ? "编辑成本" : "新增成本"}</h2></div><button type="button" data-close>×</button></div><form class="record-dialog-form"><div class="record-form-grid"><label>成本月份 *<input name="month" type="month" required value="${escapeHTML(data.month || month)}"></label><label>档口 *<select name="stall_id" required>${stallOptions(selectedStall)}</select></label><label>成本类型 *<select name="cost_type">${selectOptions(typeOptions, data.costType || "other")}</select></label><label>金额（元）*<input name="amount" type="number" min="0" step="0.01" required value="${escapeHTML(data.amount || "")}"></label><label class="span-2">备注（选填）<textarea name="remark" maxlength="255" placeholder="例如：9月食材采购">${escapeHTML(data.remark || "")}</textarea></label></div><div class="notice-card compact-notice"><span>本</span><div><strong>自营与非自营分别记录</strong><p>选择“全部自营档口”时，仅允许水费或电费，并作为一笔总额参与甲乙利润计算；非自营成本不参与该计算。</p></div></div><div class="record-dialog-error" role="alert"></div><div class="form-actions"><button type="button" class="ghost-button" data-close>取消</button><button type="submit" class="primary-button">保存成本</button></div></form></div>`;
    document.body.appendChild(overlay);
    const close = () => overlay.remove();
    overlay.querySelectorAll("[data-close]").forEach(button => button.addEventListener("click", close));
    overlay.addEventListener("click", event => { if (event.target === overlay) close(); });
    const stallSelect = overlay.querySelector('[name="stall_id"]');
    const costTypeSelect = overlay.querySelector('[name="cost_type"]');
    const syncAllSelfTypes = () => {
      const allSelf = stallSelect.value === "__all_self__";
      [...costTypeSelect.options].forEach(option => { option.disabled = allSelf && option.value !== "water" && option.value !== "electricity"; });
      if (allSelf && costTypeSelect.value !== "water" && costTypeSelect.value !== "electricity") costTypeSelect.value = "water";
    };
    stallSelect.addEventListener("change", syncAllSelfTypes);
    syncAllSelfTypes();
    overlay.querySelector("form").addEventListener("submit", async event => {
      event.preventDefault();
      const form = event.currentTarget;
      const button = form.querySelector("button[type=submit]");
      button.disabled = true;
      try {
        const values = Object.fromEntries(new FormData(form).entries());
        values.all_self_operated = values.stall_id === "__all_self__";
        values.stall_id = values.all_self_operated ? 0 : Number(values.stall_id);
        await request(`/api/admin/stall-costs${editing ? `?id=${data.id}` : ""}`, {method: editing ? "PUT" : "POST", headers:{"Content-Type":"application/json"}, body:JSON.stringify(values)});
        close();
        window.location.reload();
      } catch (error) {
        form.querySelector(".record-dialog-error").textContent = error.message;
        button.disabled = false;
      }
    });
  };

  const saveRow = async (row, values) => request(`/api/admin/stall-costs?id=${row.dataset.id}`, {method:"PUT", headers:{"Content-Type":"application/json"}, body:JSON.stringify(values)});
  const inlineEdit = (cell, row) => {
    if (row.dataset.ledgerPosted === "true" || cell.dataset.editing === "true") return;
    const field = cell.dataset.stallCostField;
    const original = cell.innerHTML;
    const values = {stall_id:Number(row.dataset.stallId), month:row.dataset.month, cost_type:row.dataset.costType, amount:row.dataset.amount, remark:row.dataset.remark, all_self_operated:row.dataset.allSelfOperated === "true"};
    let input;
    if (field === "cost_type") {
      input = document.createElement("select");
      input.innerHTML = selectOptions(typeOptions, values.cost_type);
    } else if (field === "amount") {
      input = document.createElement("input");
      input.type = "number"; input.min = "0"; input.step = "0.01"; input.value = values.amount;
    } else {
      input = document.createElement("input"); input.type = "text"; input.maxLength = 255; input.value = values.remark;
    }
    input.className = "inline-number-input";
    cell.dataset.editing = "true";
    cell.textContent = ""; cell.appendChild(input); input.focus(); if (input.select) input.select();
    let done = false;
    const finish = async commit => {
      if (done) return;
      done = true;
      if (!commit) { cell.innerHTML = original; delete cell.dataset.editing; return; }
      input.disabled = true;
      try {
        values[field] = input.value;
        await saveRow(row, values);
        window.location.reload();
      } catch (error) { cell.innerHTML = original; delete cell.dataset.editing; toast(error.message); }
    };
    input.addEventListener("keydown", event => { if (event.key === "Enter") finish(true); if (event.key === "Escape") finish(false); });
    input.addEventListener("blur", () => finish(true));
    if (input.tagName === "SELECT") input.addEventListener("change", () => finish(true));
  };

  const openLedgerDialog = async row => {
    try {
      const options = await request("/api/admin/options");
      const accounts = (options.accounts || []).map(account => `<option value="${escapeHTML(account.id)}">${escapeHTML(account.display_name || account.name)}</option>`).join("");
      if (!accounts) throw new Error("请先在台账流水中新增资金账户");
      const overlay = document.createElement("div");
      overlay.className = "record-dialog";
      overlay.innerHTML = `<div class="record-dialog-card"><div class="record-dialog-head"><div><small>档口成本</small><h2>计入台账支出</h2></div><button type="button" data-close>×</button></div><form class="record-dialog-form"><div class="notice-card compact-notice"><span>支</span><div><strong>¥${escapeHTML(row.dataset.amount)}</strong><p>入账后该成本会锁定，不能再修改或删除。</p></div></div><div class="record-form-grid"><label>付款账户 *<select name="account_id" required><option value="">请选择</option>${accounts}</select></label><label>入账日期 *<input name="occurred_on" type="date" required value="${escapeHTML(`${row.dataset.month}-01`)}"></label></div><div class="record-dialog-error" role="alert"></div><div class="form-actions"><button type="button" class="ghost-button" data-close>取消</button><button type="submit" class="primary-button">确认入账</button></div></form></div>`;
      document.body.appendChild(overlay);
      const close = () => overlay.remove();
      overlay.querySelectorAll("[data-close]").forEach(button => button.addEventListener("click", close));
      overlay.addEventListener("click", event => { if (event.target === overlay) close(); });
      overlay.querySelector("form").addEventListener("submit", async event => {
        event.preventDefault();
        const form = event.currentTarget;
        const button = form.querySelector("button[type=submit]"); button.disabled = true;
        try {
          const values = Object.fromEntries(new FormData(form).entries());
          values.cost_id = Number(row.dataset.id); values.account_id = Number(values.account_id);
          await request("/api/admin/stall-costs/ledger", {method:"POST", headers:{"Content-Type":"application/json"}, body:JSON.stringify(values)});
          close(); window.location.reload();
        } catch (error) { form.querySelector(".record-dialog-error").textContent = error.message; button.disabled = false; }
      });
    } catch (error) { toast(error.message); }
  };

  document.querySelector("[data-stall-cost-add]")?.addEventListener("click", () => openDialog());
  roots.forEach(rootItem => rootItem.addEventListener("click", async event => {
    const row = event.target.closest("[data-stall-cost-row]");
    if (!row) return;
    const field = event.target.closest("[data-stall-cost-field]");
    if (field) { inlineEdit(field, row); return; }
    if (event.target.closest("[data-stall-cost-edit]")) {
      openDialog({id:row.dataset.id, stallId:row.dataset.stallId, month:row.dataset.month, costType:row.dataset.costType, amount:row.dataset.amount, remark:row.dataset.remark, allSelfOperated:row.dataset.allSelfOperated === "true"});
      return;
    }
    if (event.target.closest("[data-stall-cost-ledger]")) {
      openLedgerDialog(row);
      return;
    }
    if (event.target.closest("[data-stall-cost-delete]")) {
      if (!window.confirm("确定删除这条档口成本吗？")) return;
      try { await request(`/api/admin/stall-costs?id=${row.dataset.id}`, {method:"DELETE"}); window.location.reload(); } catch (error) { toast(error.message); }
    }
  }));
})();
