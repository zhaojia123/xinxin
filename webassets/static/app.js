document.addEventListener("DOMContentLoaded", () => {
  const toast = document.querySelector(".demo-toast");
  let timer;
  const showMessage = (message) => {
    if (!toast) return;
    toast.textContent = message;
    toast.classList.add("show");
    clearTimeout(timer);
    timer = setTimeout(() => toast.classList.remove("show"), 2600);
  };

  document.querySelectorAll("[data-demo-action]").forEach((button) => {
    button.addEventListener("click", () => showMessage(`${button.dataset.demoAction}：该文件功能暂未启用。`));
  });

  const escapeHTML = (value) => String(value ?? "").replace(/[&<>'"]/g, (char) => ({"&":"&amp;","<":"&lt;",">":"&gt;","'":"&#39;",'"':"&quot;"})[char]);
  const miniUsersRoot = document.querySelector("[data-mini-users]");
  if (miniUsersRoot) {
    const miniUsersBody = miniUsersRoot.querySelector("[data-mini-body]");
    const miniUsersHead = miniUsersRoot.querySelector("[data-mini-head]");
    const miniUsersError = miniUsersRoot.querySelector("[data-mini-error]");
    const miniUsersCount = miniUsersRoot.querySelector("[data-mini-count]");
    const hasPermission = (user, key, action) => Boolean(user.permissions?.[key]?.[action]);
    const renderMiniUsers = ({users, modules}) => {
      miniUsersHead.innerHTML = `<th>用户</th><th>状态</th>${modules.map(module => `<th>${escapeHTML(module.name)}<small class="table-subtext">查看 · 新增 · 编辑 · 删除</small></th>`).join("")}<th>操作</th>`;
      miniUsersCount.textContent = `${users.length} 位`;
      miniUsersBody.innerHTML = users.length ? users.map(user => `<tr data-mini-user="${user.id}"><td><input class="mini-user-name" data-mini-name maxlength="64" value="${escapeHTML(user.display_name || "")}" placeholder="填写备注名称"><small class="table-subtext">${escapeHTML(user.openid)}</small></td><td><label class="record-check"><input type="checkbox" data-mini-enabled ${user.enabled ? "checked" : ""}>启用</label></td>${modules.map(module => `<td><div class="mini-actions">${["view","create","edit","delete"].map(action => `<label title="${action === "view" ? "查看" : action === "create" ? "新增" : action === "edit" ? "编辑" : "删除"}"><input type="checkbox" data-mini-module="${escapeHTML(module.key)}" data-mini-action="${action}" ${hasPermission(user, module.key, action) ? "checked" : ""}></label>`).join("")}</div></td>`).join("")}<td><button class="primary-button compact" data-mini-save>保存</button></td></tr>`).join("") : `<tr><td colspan="${modules.length + 3}" class="empty-cell">暂无小程序用户</td></tr>`;
    };
    const loadMiniUsers = async () => {
      miniUsersError.textContent = "";
      try {
        const response = await fetch("/api/admin/mini-users");
        const data = await response.json().catch(() => ({}));
        if (!response.ok) throw new Error(data.error || "小程序用户读取失败");
        renderMiniUsers(data);
      } catch (error) { miniUsersError.textContent = error.message; }
    };
    miniUsersRoot.querySelector("[data-mini-refresh]")?.addEventListener("click", loadMiniUsers);
    miniUsersBody.addEventListener("click", async event => {
      const button = event.target.closest("[data-mini-save]");
      if (!button) return;
      const row = button.closest("[data-mini-user]");
      const permissions = {};
      row.querySelectorAll("[data-mini-module]").forEach(input => {
        const key = input.dataset.miniModule;
        if (!permissions[key]) permissions[key] = {};
        permissions[key][input.dataset.miniAction] = input.checked;
      });
      button.disabled = true;
      try {
        const response = await fetch(`/api/admin/mini-users?id=${row.dataset.miniUser}`, {method: "PUT", headers: {"Content-Type": "application/json"}, body: JSON.stringify({display_name: row.querySelector("[data-mini-name]").value, enabled: row.querySelector("[data-mini-enabled]").checked, permissions})});
        const data = await response.json().catch(() => ({}));
        if (!response.ok) throw new Error(data.error || "权限保存失败");
        showMessage("小程序权限已保存，请让用户重新登录。");
      } catch (error) { miniUsersError.textContent = error.message; }
      finally { button.disabled = false; }
    });
    loadMiniUsers();
  }

  const today = () => new Date().toLocaleDateString("sv-SE");
  const recordEndpoints = {
    department: "/api/admin/departments", position: "/api/admin/positions", attendance: "/api/admin/attendance",
    change: "/api/admin/employment-changes", salary: "/api/admin/salary-adjustments", ledger: "/api/admin/ledger", purchase: "/api/admin/purchases"
  };
  const choices = {
    direction: [{id:"income",name:"收入"},{id:"expense",name:"支出"}],
    category: [{id:"leave",name:"请假"},{id:"exception",name:"迟到或早退等异常"}],
    salary_effect: [{id:"none",name:"不影响工资"},{id:"deduct",name:"按天扣款"},{id:"subsidy",name:"发放补助"},{id:"deduct_and_subsidy",name:"扣款并发补助"}],
    purchase_category: [{id:"蔬菜",name:"蔬菜"},{id:"调料",name:"调料"},{id:"肉类",name:"肉类"},{id:"水产",name:"水产"},{id:"水果",name:"水果"},{id:"粮油",name:"粮油"},{id:"其他",name:"其他"}],
    status: [{id:"probation",name:"试用期"},{id:"active",name:"在职"},{id:"left",name:"已离职"}],
    change_type: [{id:"hire",name:"入职"},{id:"regularize",name:"转正"},{id:"transfer",name:"调岗"},{id:"promotion",name:"晋升"},{id:"demotion",name:"降职"},{id:"leave",name:"离职"}]
  };
  const schemas = {
    department: { title: "部门", fields: [
      ["department_no","部门编号","text",true],["name","部门名称","text",true],["parent_id","上级部门","departments"],["manager_employee_id","负责人","employees"],
      ["sort_order","排序值","number"],["enabled","启用","checkbox"],["remark","备注","textarea"]
    ]},
    position: { title: "岗位", fields: [
      ["position_no","岗位编号","text",true],["name","岗位名称","text",true],["department_id","所属部门","departments",true],["level_name","职级","text"],
      ["min_salary","薪资下限","number"],["max_salary","薪资上限","number"],["sort_order","排序值","number"],["enabled","启用","checkbox"],["remark","备注","textarea"]
    ]},
    attendance: { title: "请假与异常", fields: [
      ["employee_id","员工","employees",true],["category","记录类别","category",true],["record_type","具体类型","text",true],["occurred_on","发生日期","date",true],
      ["start_time","开始或发生时间","time"],["end_time","结束时间","time"],["duration_days","请假/异常天数","number"],["duration_minutes","异常分钟数","number"],["salary_effect","工资影响","salary_effect",true],["subsidy_amount","补助金额","number"],["reason","原因","textarea"]
    ]},
    change: { title: "人事异动", fields: [
      ["employee_id","员工","employees",true],["change_type","异动类型","change_type",true],["after_department_id","异动后部门","departments"],["after_position_id","异动后岗位","positions"],
      ["after_status","异动后状态","status",true],["effective_on","生效日期","date",true],["reason","原因","textarea"]
    ]},
    salary: { title: "调薪", fields: [
      ["employee_id","员工","employees",true],["before_salary","调整前工资","number"],["after_salary","调整后工资","number",true],["effective_on","生效日期","date",true],["reason","原因","textarea"]
    ]},
    ledger: { title: "台账收支", fields: [
      ["direction","收支方向","direction",true],["amount","金额","number",true],["occurred_on","发生日期","date",true],["account_id","资金账户","accounts",true],
      ["category_id","收支分类","categories"],["department_id","归属部门","departments"],["summary","摘要","text",true],["counterparty","对方名称","text"],["voucher_no","凭证号","text"],["remark","备注","textarea"]
    ]},
    purchase: { title: "供货采购", fields: [
      ["date","采购日期","date",true],["supplier_name","供货商","text",true],["category","采购品类","purchase_category",true],["product_name","菜品或物料","text",true],
      ["quantity","数量","number",true],["unit","单位","text",true],["unit_price","单价（元）","number",true],["remark","特殊需求/备注","textarea"]
    ]}
  };
  let adminOptions;
  const fetchJSON = async (url, options = {}) => {
    const response = await fetch(url, options);
    const data = await response.json().catch(() => ({}));
    if (!response.ok) throw new Error(data.error || "操作失败，请稍后重试");
    return data;
  };
  const loadAdminOptions = async () => adminOptions || (adminOptions = await fetchJSON("/api/admin/options"));
  const askValue = ({ title, label, type = "text", value = "" }) => new Promise(resolve => {
    const overlay = document.createElement("div");
    overlay.className = "record-prompt";
    overlay.innerHTML = `<form class="record-prompt-card"><h3>${escapeHTML(title)}</h3><label>${escapeHTML(label)}<input name="value" type="${type}" value="${escapeHTML(value)}" required autofocus></label><div class="form-actions"><button type="button" class="ghost-button" data-prompt-cancel>取消</button><button type="submit" class="primary-button">确定</button></div></form>`;
    let finished = false;
    const finish = result => {
      if (finished) return;
      finished = true;
      overlay.remove();
      resolve(result);
    };
    document.body.appendChild(overlay);
    overlay.querySelector("input").focus();
    overlay.querySelector("[data-prompt-cancel]").addEventListener("click", () => finish(""));
    overlay.addEventListener("click", event => { if (event.target === overlay) finish(""); });
    overlay.querySelector("form").addEventListener("submit", event => {
      event.preventDefault();
      finish(event.currentTarget.elements.value.value.trim());
    });
  });
  const askAccount = () => new Promise(resolve => {
    const overlay = document.createElement("div");
    overlay.className = "record-prompt";
    overlay.innerHTML = `<form class="record-prompt-card"><h3>新建资金账户</h3><label>账户名称<input name="name" maxlength="64" required autofocus placeholder="例如：建设银行工资卡"></label><label>账户类型<select name="account_type"><option value="bank">银行卡</option><option value="wechat">微信</option><option value="alipay">支付宝</option><option value="qq">QQ</option><option value="cash">现金</option><option value="other">其他</option></select></label><label>备注（选填）<input name="remark" maxlength="255" placeholder="例如：QQ号、微信号、卡号后四位"></label><div class="form-actions"><button type="button" class="ghost-button" data-prompt-cancel>取消</button><button type="submit" class="primary-button">确定</button></div></form>`;
    let finished = false;
    const finish = result => { if (finished) return; finished = true; overlay.remove(); resolve(result); };
    document.body.appendChild(overlay);
    overlay.querySelector("input[name=name]").focus();
    overlay.querySelector("[data-prompt-cancel]").addEventListener("click", () => finish(null));
    overlay.addEventListener("click", event => { if (event.target === overlay) finish(null); });
    overlay.querySelector("form").addEventListener("submit", event => { event.preventDefault(); const form = event.currentTarget; finish({name: form.elements.name.value.trim(), account_type: form.elements.account_type.value, remark: form.elements.remark.value.trim()}); });
  });
  const askConfirm = ({ title, message, confirmText = "确认删除" }) => new Promise(resolve => {
    const overlay = document.createElement("div");
    overlay.className = "record-prompt";
    overlay.innerHTML = `<div class="record-prompt-card"><h3>${escapeHTML(title)}</h3><p class="record-prompt-message">${escapeHTML(message)}</p><div class="form-actions"><button type="button" class="ghost-button" data-confirm-cancel>取消</button><button type="button" class="primary-button danger-button" data-confirm-submit>${escapeHTML(confirmText)}</button></div></div>`;
    let finished = false;
    const finish = result => {
      if (finished) return;
      finished = true;
      overlay.remove();
      resolve(result);
    };
    document.body.appendChild(overlay);
    overlay.querySelector("[data-confirm-cancel]").addEventListener("click", () => finish(false));
    overlay.querySelector("[data-confirm-submit]").addEventListener("click", () => finish(true));
    overlay.addEventListener("click", event => { if (event.target === overlay) finish(false); });
  });
  const fieldOptions = (type) => choices[type] || adminOptions?.[type] || [];
  const fieldHTML = (field, data, readonly) => {
    const [name,label,type,required] = field;
    const value = data[name] ?? "";
    if (type === "checkbox") return `<label class="record-check" data-record-field="${name}"><input name="${name}" type="checkbox" ${value ? "checked" : ""} ${readonly ? "disabled" : ""}> ${escapeHTML(label)}</label>`;
    if (type === "textarea") return `<label data-record-field="${name}">${escapeHTML(label)}${required ? " *" : ""}<textarea name="${name}" maxlength="500" ${readonly ? "disabled" : ""}>${escapeHTML(value)}</textarea></label>`;
    const options = fieldOptions(type);
    if (options.length || ["departments","employees","positions","accounts","categories"].includes(type)) {
      const empty = required ? "请选择" : "不指定";
      const create = !readonly && ["accounts","categories"].includes(type) ? `<button type="button" class="record-inline-add" data-create-option="${type}">＋ 新建</button>` : "";
      return `<label data-record-field="${name}"><span class="record-label-row"><span>${escapeHTML(label)}${required ? " *" : ""}</span>${create}</span><select name="${name}" ${required ? "required" : ""} ${readonly ? "disabled" : ""}><option value="">${empty}</option>${options.map(item => `<option value="${escapeHTML(item.id)}" data-department-id="${escapeHTML(item.department_id || "")}" data-direction="${escapeHTML(item.direction || "")}" ${String(item.id) === String(value) ? "selected" : ""}>${escapeHTML(item.display_name || item.name)}${item.direction ? ` · ${item.direction === "income" ? "收入" : "支出"}` : ""}</option>`).join("")}</select></label>`;
    }
    const inputType = ["number","date","time"].includes(type) ? type : "text";
    const step = type === "number" ? ' step="0.01"' : "";
    return `<label data-record-field="${name}">${escapeHTML(label)}${required ? " *" : ""}<input name="${name}" type="${inputType}" value="${escapeHTML(value)}" ${required ? "required" : ""}${step} ${readonly ? "disabled" : ""}></label>`;
  };
  const filterDialogOptions = (select, key, value) => {
    if (!select) return;
    Array.from(select.options).forEach((option, index) => {
      if (index === 0) return;
      const related = option.dataset[key] || "";
      option.hidden = Boolean(value && related && related !== value);
      option.disabled = option.hidden;
    });
    if (select.selectedOptions[0]?.hidden) select.value = "";
  };
  const syncDialogDependencies = (overlay, kind) => {
    if (kind === "attendance") {
      const leave = overlay.querySelector('[name="category"]')?.value === "leave";
      const daysField = overlay.querySelector('[data-record-field="duration_days"]');
      const minutesField = overlay.querySelector('[data-record-field="duration_minutes"]');
      const effect = overlay.querySelector('[name="salary_effect"]')?.value || "none";
      const dayBased = leave || effect === "deduct_and_subsidy";
      const subsidyField = overlay.querySelector('[data-record-field="subsidy_amount"]');
      if (daysField) daysField.hidden = !dayBased;
      if (minutesField) minutesField.hidden = leave || dayBased;
      if (subsidyField) subsidyField.hidden = effect !== "subsidy" && effect !== "deduct_and_subsidy";
      const days = daysField?.querySelector("input");
      const minutes = minutesField?.querySelector("input");
      if (days) days.required = dayBased;
      if (minutes) minutes.required = !leave && !dayBased;
    }
    if (kind === "change") {
      const changeType = overlay.querySelector('[name="change_type"]')?.value;
      const status = overlay.querySelector('[name="after_status"]');
      if (status && changeType === "leave") status.value = "left";
      if (status && changeType === "regularize") status.value = "active";
      filterDialogOptions(overlay.querySelector('[name="after_position_id"]'), "departmentId", overlay.querySelector('[name="after_department_id"]')?.value || "");
    }
    if (kind === "ledger") {
      filterDialogOptions(overlay.querySelector('[name="category_id"]'), "direction", overlay.querySelector('[name="direction"]')?.value || "");
    }
  };
  const closeRecordDialog = () => document.querySelector(".record-dialog")?.remove();
  const uploadLedgerProofFiles = async (ledgerID, files) => {
    for (const file of files) {
      const form = new FormData();
      form.append("ledger_id", String(ledgerID));
      form.append("proof", file);
      const response = await fetch("/api/admin/upload/ledger-proof", {method: "POST", body: form});
      const result = await response.json().catch(() => ({}));
      if (!response.ok) throw new Error(result.error || "上传转账凭证失败");
    }
  };
  const openRecordDialog = async (kind, id = 0, defaults = {}, readonly = false) => {
    try {
      await loadAdminOptions();
      const schema = schemas[kind];
      let data = {...defaults};
      if (id) data = await fetchJSON(`${recordEndpoints[kind]}?id=${id}`);
      if (kind === "ledger" && data.payroll_batch_id) readonly = true;
      if (data.enabled === undefined) data.enabled = true;
      const ledgerProofs = kind === "ledger" ? `<section class="ledger-proof-panel"><div class="record-label-row"><strong>转账凭证</strong><small>支持 JPG、PNG、WebP，单张大小以服务端配置为准</small></div>${data.attachments?.length ? `<div class="ledger-proof-list">${data.attachments.map(item => `<a href="${escapeHTML(item.file_url)}" target="_blank" rel="noopener"><img src="${escapeHTML(item.file_url)}" alt="${escapeHTML(item.original_name || "转账凭证")}"><span>${escapeHTML(item.original_name || "查看图片")}</span></a>`).join("")}</div>` : `<p class="ledger-proof-empty">暂无转账凭证</p>`}<div class="upload-dropzone" data-ledger-proof-upload><input type="file" data-ledger-proof-files accept="image/jpeg,image/png,image/webp" multiple><span class="upload-dropzone-plus">＋</span><strong>拖入或选择转账凭证</strong><small>可上传银行回单、转账截图，支持多张图片</small><em data-ledger-proof-names>尚未选择文件</em></div><button type="button" class="ghost-button ledger-proof-upload-button" data-ledger-proof-submit ${id ? "" : "disabled"}>上传凭证图片</button>${id ? "" : `<p class="ledger-proof-empty">保存台账后会自动上传已选择的图片</p>`}</section>` : "";
      closeRecordDialog();
      const overlay = document.createElement("div");
      overlay.className = "record-dialog";
      overlay.innerHTML = `<div class="record-dialog-card"><div class="record-dialog-head"><div><small>${readonly ? "记录详情" : id ? "修改记录" : "新增记录"}</small><h2>${escapeHTML(schema.title)}</h2></div><button type="button" data-close-dialog>×</button></div><form class="record-dialog-form"><div class="record-form-grid">${schema.fields.map(field => fieldHTML(field,data,readonly || (kind === "salary" && field[0] === "before_salary"))).join("")}</div>${ledgerProofs}<div class="record-dialog-error" role="alert"></div><div class="form-actions"><button type="button" class="ghost-button" data-close-dialog>${readonly ? "关闭" : "取消"}</button>${readonly ? "" : `<button type="submit" class="primary-button">保存${escapeHTML(schema.title)}</button>`}</div></form></div>`;
      document.body.appendChild(overlay);
      overlay.querySelectorAll("[data-close-dialog]").forEach(button => button.addEventListener("click", closeRecordDialog));
      overlay.addEventListener("click", event => { if (event.target === overlay) closeRecordDialog(); });
      const proofDropzone = overlay.querySelector("[data-ledger-proof-upload]");
      const proofInput = overlay.querySelector("[data-ledger-proof-files]");
      const proofButton = overlay.querySelector("[data-ledger-proof-submit]");
      proofDropzone?.addEventListener("click", event => { if (event.target !== proofInput) proofInput?.click(); });
      proofInput?.addEventListener("change", () => { overlay.querySelector("[data-ledger-proof-names]").textContent = proofInput.files.length ? Array.from(proofInput.files).map(file => file.name).join("、") : "尚未选择文件"; if (proofButton && id) proofButton.disabled = !proofInput.files.length; });
      proofButton?.addEventListener("click", async () => {
        if (!id || !proofInput?.files.length) return;
        proofButton.disabled = true;
        try {
          await uploadLedgerProofFiles(id, Array.from(proofInput.files));
          showMessage("转账凭证上传成功。");
          await openRecordDialog("ledger", id, {}, true);
        } catch (error) {
          showMessage(`转账凭证上传失败：${error.message}`);
          proofButton.disabled = false;
        }
      });
      overlay.querySelectorAll("[data-create-option]").forEach(button => button.addEventListener("click", async event => {
        event.preventDefault();
        const kind = button.dataset.createOption;
        const label = kind === "accounts" ? "资金账户" : "收支分类";
        const account = kind === "accounts" ? await askAccount() : null;
        const name = account?.name || (kind === "accounts" ? "" : await askValue({title:`新建${label}`,label:`${label}名称`}));
        if (!name) return;
        button.disabled = true;
        try {
          const direction = kind === "categories" ? overlay.querySelector('[name="direction"]')?.value || "" : "";
          const created = await fetchJSON("/api/admin/options", {method:"POST",headers:{"Content-Type":"application/json"},body:JSON.stringify({kind,name,direction,account_type:account?.account_type || "",remark:account?.remark || ""})});
          const select = button.closest("label").querySelector("select");
          const option = document.createElement("option");
          option.value = String(created.id);
          option.textContent = created.display_name || created.name;
          option.dataset.direction = created.direction || "";
          select.appendChild(option);
          select.value = option.value;
          adminOptions[kind].push(created);
          syncDialogDependencies(overlay, kind === "categories" ? "ledger" : kind);
          showMessage(`${label}已新建`);
        } catch (error) {
          overlay.querySelector(".record-dialog-error").textContent = error.message;
        } finally {
          button.disabled = false;
        }
      }));
      syncDialogDependencies(overlay, kind);
      overlay.querySelector('[name="category"]')?.addEventListener("change", () => syncDialogDependencies(overlay, kind));
      overlay.querySelector('[name="salary_effect"]')?.addEventListener("change", () => syncDialogDependencies(overlay, kind));
      overlay.querySelector('[name="change_type"]')?.addEventListener("change", () => syncDialogDependencies(overlay, kind));
      overlay.querySelector('[name="after_department_id"]')?.addEventListener("change", () => syncDialogDependencies(overlay, kind));
      overlay.querySelector('[name="direction"]')?.addEventListener("change", () => syncDialogDependencies(overlay, kind));
      if (readonly) return;
      const employeeSelect = overlay.querySelector('[name="employee_id"]');
      employeeSelect?.addEventListener("change", () => {
        const employee = adminOptions.employees.find(item => String(item.id) === employeeSelect.value);
        if (!employee) return;
        if (kind === "change") {
          overlay.querySelector('[name="after_department_id"]').value = employee.department_id || "";
          overlay.querySelector('[name="after_position_id"]').value = employee.position_id || "";
          overlay.querySelector('[name="after_status"]').value = employee.status || "active";
          syncDialogDependencies(overlay, kind);
        }
        if (kind === "salary") {
          overlay.querySelector('[name="before_salary"]').value = employee.salary || "0.00";
          if (!overlay.querySelector('[name="after_salary"]').value) overlay.querySelector('[name="after_salary"]').value = employee.salary || "0.00";
        }
      });
      overlay.querySelector("form").addEventListener("submit", async event => {
        event.preventDefault();
        const form = event.currentTarget;
        const button = form.querySelector("button[type=submit]");
        const pendingProofs = Array.from(form.querySelector("[data-ledger-proof-files]")?.files || []);
        const values = Object.fromEntries(new FormData(form).entries());
        delete values.proof;
        schema.fields.forEach(([name,,type]) => {
          if (type === "checkbox") values[name] = form.elements[name].checked;
          if (["departments","employees","positions","accounts","categories"].includes(type) || name.endsWith("_id") || name === "sort_order" || name === "duration_minutes") values[name] = Number(values[name] || 0);
        });
        button.disabled = true;
        try {
          const saved = await fetchJSON(recordEndpoints[kind] + (id ? `?id=${id}` : ""), {method:id ? "PUT" : "POST",headers:{"Content-Type":"application/json"},body:JSON.stringify(values)});
          if (kind === "ledger" && pendingProofs.length) await uploadLedgerProofFiles(saved.id || id, pendingProofs);
          window.location.href = window.location.pathname + window.location.search;
        } catch (error) { form.querySelector(".record-dialog-error").textContent = error.message; button.disabled = false; }
      });
    } catch (error) { showMessage(error.message); }
  };
  document.querySelectorAll("[data-record-form]").forEach(button => button.addEventListener("click", () => {
    const kind = button.dataset.recordForm;
    const defaults = {};
    if (kind === "attendance") { const category = button.dataset.category || "leave"; Object.assign(defaults,{category,occurred_on:today(),duration_days:"1.00",duration_minutes:0,salary_effect:category === "leave" ? "deduct" : "none",subsidy_amount:"0"}); }
    if (kind === "ledger") Object.assign(defaults,{direction:"expense",occurred_on:today(),amount:""});
    if (kind === "change") Object.assign(defaults,{change_type:"transfer",after_status:"active",effective_on:today()});
    if (kind === "salary") Object.assign(defaults,{after_salary:"",effective_on:today()});
    if (kind === "purchase") Object.assign(defaults,{date:today(),supplier_name:"默认供货商",category:"蔬菜",quantity:"1",unit:"斤",unit_price:"0.00"});
    openRecordDialog(kind,Number(button.dataset.id || 0),defaults);
  }));
  document.querySelectorAll("[data-record-view]").forEach(button => button.addEventListener("click", () => openRecordDialog(button.dataset.recordView,Number(button.dataset.id),{},true)));
  document.querySelectorAll("[data-record-delete]").forEach(button => button.addEventListener("click", async () => {
    const kind = button.dataset.recordDelete;
    const endpoint = kind === "employee" ? "/api/admin/employees" : recordEndpoints[kind];
    const labels = {employee:"员工",department:"部门",position:"岗位",attendance:"请假或异常记录",ledger:"台账记录",purchase:"采购明细"};
    const confirmed = await askConfirm({title:`删除${labels[kind] || "记录"}`,message:`确定删除“${button.dataset.name || "这条记录"}”吗？有关联业务数据时，系统会阻止删除并说明原因。`});
    if (!confirmed) return;
    button.disabled = true;
    try {
      await fetchJSON(`${endpoint}?id=${button.dataset.id}`, {method:"DELETE"});
      window.location.href = window.location.pathname + window.location.search;
    } catch (error) {
      showMessage(error.message);
      button.disabled = false;
    }
  }));
  document.querySelectorAll("[data-record-status]").forEach(button => button.addEventListener("click", async () => {
    const label = button.dataset.recordStatus === "approved" ? "通过" : "驳回";
    if (!window.confirm(`确定${label}这条申请吗？`)) return;
    button.disabled = true;
    try { await fetchJSON(`/api/admin/attendance/status?id=${button.dataset.id}`,{method:"PUT",headers:{"Content-Type":"application/json"},body:JSON.stringify({status:button.dataset.recordStatus})}); window.location.reload(); }
    catch(error){showMessage(error.message);button.disabled=false;}
  }));
  document.querySelectorAll("[data-employee-leave]").forEach(button => button.addEventListener("click", async () => {
    const date = await askValue({title:`办理 ${button.dataset.name} 离职`,label:"离职日期",type:"date",value:today()});
    if (!date) return;
    button.disabled=true;
    try { await fetchJSON(`/api/admin/employees/leave?id=${button.dataset.id}`,{method:"PUT",headers:{"Content-Type":"application/json"},body:JSON.stringify({left_on:date})}); window.location.href="/admin/employees"; }
    catch(error){showMessage(error.message);button.disabled=false;}
  }));
  document.querySelector("[data-confirm-payroll]")?.addEventListener("click", async event => {
    const button = event.currentTarget;
    if (!window.confirm("确定批量确认本月工资吗？确认后仍可在发放前修正金额。")) return;
    button.disabled = true;
    try { await fetchJSON("/api/admin/payroll/confirm",{method:"POST",headers:{"Content-Type":"application/json"},body:JSON.stringify({batch_id:Number(button.dataset.batchId)})}); window.location.reload(); }
    catch(error){showMessage(error.message);button.disabled=false;}
  });
  document.querySelectorAll("[data-confirm-payroll-item]").forEach(button => button.addEventListener("click", async () => {
    if (!window.confirm(`确定确认${button.dataset.employeeName || "该员工"}的工资吗？确认后仍可在发放前修改金额。`)) return;
    button.disabled = true;
    try {
      await fetchJSON("/api/admin/payroll/confirm", {method:"POST", headers:{"Content-Type":"application/json"}, body:JSON.stringify({item_id:Number(button.dataset.itemId)})});
      window.location.reload();
    } catch (error) {
      showMessage(error.message);
      button.disabled = false;
    }
  }));
  document.querySelector("[data-pay-payroll]")?.addEventListener("click", async event => {
    const button = event.currentTarget;
    try {
      await loadAdminOptions();
      if (!adminOptions.accounts.length) throw new Error("请先在台账新增一个资金账户");
      closeRecordDialog();
      const overlay=document.createElement("div");overlay.className="record-dialog";
      overlay.innerHTML=`<div class="record-dialog-card"><div class="record-dialog-head"><div><small>工资管理</small><h2>发放本月工资</h2></div><button type="button" data-close-dialog>×</button></div><form class="record-dialog-form"><div class="notice-card"><span>核</span><div><strong>发放后工资将锁定</strong><p>系统会同时在台账生成工资支出，请确认工资明细已经核对完成。</p></div></div><div class="record-form-grid"><label>付款账户 *<select name="account_id" required><option value="">请选择</option>${adminOptions.accounts.map(v=>`<option value="${v.id}">${escapeHTML(v.display_name || v.name)}</option>`).join("")}</select></label><label>发放日期 *<input name="occurred_on" type="date" value="${today()}" required></label></div><div class="record-dialog-error"></div><div class="form-actions"><button type="button" class="ghost-button" data-close-dialog>取消</button><button type="submit" class="primary-button">确认发放</button></div></form></div>`;
      document.body.appendChild(overlay);overlay.querySelectorAll("[data-close-dialog]").forEach(v=>v.addEventListener("click",closeRecordDialog));
      overlay.querySelector("form").addEventListener("submit",async submitEvent=>{submitEvent.preventDefault();const form=submitEvent.currentTarget;const submit=form.querySelector("button[type=submit]");submit.disabled=true;const values=Object.fromEntries(new FormData(form).entries());values.batch_id=Number(button.dataset.batchId);values.account_id=Number(values.account_id);try{await fetchJSON("/api/admin/payroll/pay",{method:"POST",headers:{"Content-Type":"application/json"},body:JSON.stringify(values)});window.location.reload()}catch(error){form.querySelector(".record-dialog-error").textContent=error.message;submit.disabled=false}});
    } catch(error){showMessage(error.message)}
  });
  document.querySelectorAll("[data-pay-payroll-item]").forEach(button => button.addEventListener("click", async () => {
    try {
      await loadAdminOptions();
      if (!adminOptions.accounts.length) throw new Error("请先在台账新增一个资金账户");
      const overlay = document.createElement("div");
      overlay.className = "record-dialog";
      overlay.innerHTML = `<div class="record-dialog-card"><div class="record-dialog-head"><div><small>工资管理</small><h2>单独发放：${escapeHTML(button.dataset.employeeName || "员工")}</h2></div><button type="button" data-close-dialog>×</button></div><form class="record-dialog-form"><div class="notice-card"><span>核</span><div><strong>只发放当前员工</strong><p>已发放的其他员工不会重复入账；当前员工发放后将锁定。</p></div></div><div class="record-form-grid"><label>付款账户 *<select name="account_id" required><option value="">请选择</option>${adminOptions.accounts.map(v => `<option value="${v.id}">${escapeHTML(v.display_name || v.name)}</option>`).join("")}</select></label><label>发放日期 *<input name="occurred_on" type="date" value="${today()}" required></label></div><div class="record-dialog-error"></div><div class="form-actions"><button type="button" class="ghost-button" data-close-dialog>取消</button><button type="submit" class="primary-button">确认发放</button></div></form></div>`;
      document.body.appendChild(overlay);
      overlay.querySelectorAll("[data-close-dialog]").forEach(v => v.addEventListener("click", closeRecordDialog));
      overlay.querySelector("form").addEventListener("submit", async submitEvent => {
        submitEvent.preventDefault();
        const form = submitEvent.currentTarget;
        const submit = form.querySelector("button[type=submit]");
        submit.disabled = true;
        const values = Object.fromEntries(new FormData(form).entries());
        values.item_id = Number(button.dataset.itemId);
        values.account_id = Number(values.account_id);
        try {
          await fetchJSON("/api/admin/payroll/pay", {method: "POST", headers: {"Content-Type": "application/json"}, body: JSON.stringify(values)});
          window.location.reload();
        } catch (error) {
          form.querySelector(".record-dialog-error").textContent = error.message;
          submit.disabled = false;
        }
      });
    } catch (error) {
      showMessage(error.message);
    }
  }));

  const departmentSelect = document.querySelector("[data-department-select]");
  const positionSelect = document.querySelector("[data-position-select]");
  const filterPositions = () => {
    if (!departmentSelect || !positionSelect) return;
    const departmentID = departmentSelect.value;
    Array.from(positionSelect.options).forEach((option) => {
      option.hidden = Boolean(option.dataset.departmentId && option.dataset.departmentId !== departmentID);
    });
    if (positionSelect.selectedOptions[0]?.hidden) positionSelect.value = "";
  };
  departmentSelect?.addEventListener("change", filterPositions);
  filterPositions();

  const employeeSearch = document.querySelector("[data-employee-search]");
  const employeeDepartment = document.querySelector("[data-employee-department]");
  const employeePosition = document.querySelector("[data-employee-position]");
  const employeeStatus = document.querySelector("[data-employee-status]");
  const filterEmployees = () => {
    const keyword = employeeSearch?.value.trim().toLowerCase() || "";
    const departmentID = employeeDepartment?.value || "";
    const positionID = employeePosition?.value || "";
    const status = employeeStatus?.value || "";
    document.querySelectorAll("[data-employee-row]").forEach((row) => {
      const matchesKeyword = !keyword || row.dataset.search.toLowerCase().includes(keyword);
      const matchesDepartment = !departmentID || row.dataset.departmentId === departmentID;
      const matchesPosition = !positionID || row.dataset.positionId === positionID;
      const matchesStatus = !status || row.dataset.status === status;
      row.hidden = !(matchesKeyword && matchesDepartment && matchesPosition && matchesStatus);
    });
  };
  employeeSearch?.addEventListener("input", filterEmployees);
  employeeDepartment?.addEventListener("change", filterEmployees);
  employeePosition?.addEventListener("change", filterEmployees);
  employeeStatus?.addEventListener("change", filterEmployees);

  const numberValue = (row, name) => Number(row.querySelector(`[data-field="${name}"]`)?.value || 0);
  const calculateNet = (row) => {
    const manual = row.querySelector('[data-field="manual_net_salary"]')?.value.trim();
    if (manual !== undefined && manual !== "") return Number(manual || 0);
    return numberValue(row, "base_salary") + numberValue(row, "bonus")
    - numberValue(row, "attendance_deduction") - numberValue(row, "other_deduction")
    - numberValue(row, "social_security") - numberValue(row, "tax");
  };
  const renderNet = (row) => {
    const target = row.querySelector("[data-net-salary]");
    if (target) target.textContent = `¥${calculateNet(row).toFixed(2)}`;
  };

  document.querySelectorAll("[data-payroll-row]").forEach((row) => {
    row.querySelectorAll(".money-input").forEach((input) => input.addEventListener("input", () => renderNet(row)));
    row.querySelector("[data-save-payroll]")?.addEventListener("click", async () => {
      const payload = {
        base_salary: numberValue(row, "base_salary"), bonus: numberValue(row, "bonus"),
        attendance_deduction: numberValue(row, "attendance_deduction"), other_deduction: numberValue(row, "other_deduction"),
        social_security: numberValue(row, "social_security"), tax: numberValue(row, "tax")
      };
      const manual = row.querySelector('[data-field="manual_net_salary"]')?.value.trim();
      if (manual !== undefined && manual !== "") payload.manual_net_salary = Number(manual);
      try {
        const response = await fetch(`/api/admin/payroll?id=${row.dataset.id}`, {method: "PUT", headers: {"Content-Type": "application/json"}, body: JSON.stringify(payload)});
        const data = await response.json();
        if (!response.ok) throw new Error(data.error || "保存失败");
        row.querySelector("[data-net-salary]").textContent = data.net_salary;
        showMessage("工资明细已保存，实发工资已由服务端重新计算。");
      } catch (error) { showMessage(`保存失败：${error.message}`); }
    });
  });

  document.querySelector("#generate-payroll")?.addEventListener("click", async (event) => {
    const button = event.currentTarget;
    button.disabled = true;
    try {
      const response = await fetch("/api/admin/payroll/generate", {method: "POST", headers: {"Content-Type": "application/json"}, body: JSON.stringify({month: button.dataset.month})});
      const data = await response.json();
      if (!response.ok) throw new Error(data.error || "生成失败");
      window.location.reload();
    } catch (error) { showMessage(`生成工资失败：${error.message}`); button.disabled = false; }
  });

  const payrollPicker = document.querySelector("[data-payroll-picker]");
  const payrollPickerToggle = payrollPicker?.querySelector("[data-payroll-picker-toggle]");
  const payrollPickerPanel = payrollPicker?.querySelector("[data-payroll-picker-panel]");
  const payrollEmployeeSearch = payrollPicker?.querySelector("[data-payroll-employee-search]");
  const payrollEmployeeOptions = payrollPicker ? Array.from(payrollPicker.querySelectorAll("[data-payroll-employee]")) : [];
  const refreshPayrollPicker = () => {
    const keyword = payrollEmployeeSearch?.value.trim().toLowerCase() || "";
    payrollEmployeeOptions.forEach(input => {
      const option = input.closest(".employee-picker-option");
      option.hidden = Boolean(keyword) && !option.textContent.toLowerCase().includes(keyword);
    });
    const selected = payrollEmployeeOptions.filter(input => input.checked);
    if (payrollPickerToggle) payrollPickerToggle.textContent = selected.length ? `已选 ${selected.length} 名员工` : "选择员工（可选）";
  };
  payrollPickerToggle?.addEventListener("click", () => {
    payrollPickerPanel.hidden = !payrollPickerPanel.hidden;
    if (!payrollPickerPanel.hidden) payrollEmployeeSearch?.focus();
  });
  payrollEmployeeSearch?.addEventListener("input", refreshPayrollPicker);
  payrollEmployeeOptions.forEach(input => input.addEventListener("change", refreshPayrollPicker));
  document.addEventListener("click", event => {
    if (payrollPicker && !payrollPicker.contains(event.target)) payrollPickerPanel.hidden = true;
  });
  document.querySelector("[data-generate-payroll-employee]")?.addEventListener("click", async (event) => {
    const button = event.currentTarget;
    const employeeIDs = payrollEmployeeOptions.filter(input => input.checked).map(input => Number(input.value));
    if (!employeeIDs.length) {
      showMessage("请先选择员工");
      return;
    }
    button.disabled = true;
    try {
      const response = await fetch("/api/admin/payroll/generate", {
        method: "POST",
        headers: {"Content-Type": "application/json"},
        body: JSON.stringify({month: button.dataset.month, employee_ids: employeeIDs})
      });
      const data = await response.json();
      if (!response.ok) throw new Error(data.error || "生成失败");
      window.location.reload();
    } catch (error) {
      showMessage(`单独生成工资失败：${error.message}`);
      button.disabled = false;
    }
  });

  document.querySelector("[data-health-upload]")?.addEventListener("submit", async (event) => {
    event.preventDefault();
    const form = event.currentTarget;
    const button = form.querySelector("button[type=submit]");
    button.disabled = true;
    try {
      const response = await fetch("/api/admin/upload/health-certificate", {method: "POST", body: new FormData(form)});
      const data = await response.json();
      if (!response.ok) throw new Error(data.error || "上传失败");
      showMessage("健康证上传成功，证件日期和到期提醒已更新。");
      setTimeout(() => { window.location.href = "/admin/employees"; }, 500);
    } catch (error) {
      showMessage(`健康证上传失败：${error.message}`);
      button.disabled = false;
    }
  });

  document.querySelector("[data-employee-document-upload]")?.addEventListener("submit", async (event) => {
    event.preventDefault();
    const form = event.currentTarget;
    const button = form.querySelector("button[type=submit]");
    button.disabled = true;
    try {
      const response = await fetch("/api/admin/upload/employee-document", {method: "POST", body: new FormData(form)});
      const data = await response.json().catch(() => ({}));
      if (!response.ok) throw new Error(data.error || "上传失败");
      showMessage("员工证件图片上传成功。");
      setTimeout(() => { window.location.reload(); }, 500);
    } catch (error) {
      showMessage(`员工证件上传失败：${error.message}`);
      button.disabled = false;
    }
  });

  document.querySelectorAll("[data-employee-form-upload]").forEach(dropzone => {
    const input = dropzone.querySelector("[data-employee-document-file]");
    const button = dropzone.parentElement.querySelector("[data-employee-document-submit]");
    const name = dropzone.querySelector("[data-employee-document-name]");
    input?.addEventListener("change", () => {
      name.textContent = input.files.length ? input.files[0].name : "选择图片";
      if (button) button.disabled = !input.files.length;
    });
    dropzone.addEventListener("click", event => { if (event.target !== input) input?.click(); });
    button?.addEventListener("click", async () => {
      if (!input.files.length) return;
      button.disabled = true;
      const form = new FormData();
      form.append("employee_id", button.dataset.employeeId);
      form.append("attachment_type", button.dataset.kind);
      form.append("document", input.files[0]);
      try {
        const response = await fetch("/api/admin/upload/employee-document", {method: "POST", body: form});
        const data = await response.json().catch(() => ({}));
        if (!response.ok) throw new Error(data.error || "上传失败");
        showMessage("员工证件图片上传成功。");
        input.value = "";
        name.textContent = "选择图片";
      } catch (error) {
        showMessage(`员工证件上传失败：${error.message}`);
        button.disabled = false;
      }
    });
  });

  const employeeCreateForm = document.querySelector("[data-employee-form]");
  const employeeCreateUploads = document.querySelectorAll("[data-employee-create-upload]");
  if (employeeCreateForm && employeeCreateUploads.length) {
    employeeCreateUploads.forEach(dropzone => {
      const input = dropzone.querySelector("[data-employee-create-file]");
      const name = dropzone.querySelector("[data-employee-create-name]");
      input?.addEventListener("change", () => { name.textContent = input.files.length ? input.files[0].name : "选择图片"; });
      dropzone.addEventListener("click", event => { if (event.target !== input) input?.click(); });
    });
    employeeCreateForm.addEventListener("submit", async event => {
      const files = Array.from(employeeCreateForm.querySelectorAll("[data-employee-create-file]"))
        .filter(input => input.files.length)
        .map(input => ({kind: input.dataset.kind, file: input.files[0]}));
      if (!files.length) return;
      event.preventDefault();
      const button = employeeCreateForm.querySelector("button[type=submit]");
      button.disabled = true;
      const values = Object.fromEntries(new FormData(employeeCreateForm).entries());
      delete values.document;
      delete values.attachment_type;
      ["department_id", "position_id"].forEach(key => { values[key] = Number(values[key] || 0); });
      let saved;
      try {
        saved = await fetchJSON("/api/admin/employees", {method: "POST", headers: {"Content-Type": "application/json"}, body: JSON.stringify(values)});
        for (const item of files) {
          const form = new FormData();
          form.append("employee_id", String(saved.id));
          form.append("attachment_type", item.kind);
          form.append("document", item.file);
          const response = await fetch("/api/admin/upload/employee-document", {method: "POST", body: form});
          const data = await response.json().catch(() => ({}));
          if (!response.ok) throw new Error(data.error || "证件图片上传失败");
        }
        window.location.href = "/admin/employees";
      } catch (error) {
        if (saved?.id) {
          showMessage(`员工已保存，但证件图片上传失败：${error.message}`);
          setTimeout(() => { window.location.href = `/admin/employees/edit?id=${saved.id}`; }, 700);
        } else {
          showMessage(`保存员工失败：${error.message}`);
          button.disabled = false;
        }
      }
    });
  }
});
