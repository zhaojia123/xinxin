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
    category: [{id:"public_rest",name:"正常公休"},{id:"leave",name:"请假"},{id:"holiday",name:"法定节假日休息"}],
    attendance_kind: [{id:"public_rest",name:"公休"},{id:"leave",name:"请假"},{id:"holiday",name:"法定节假日"},{id:"special_leave",name:"特殊请假补贴"}],
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
    attendance: { title: "请假、公休与节假日", fields: [
      ["employee_ids","员工（可多选）","employees_multi",true],["attendance_kind","记录类型","attendance_kind",true],["occurred_ons","发生日期（可多选）","dates_multi",true],
      ["duration_days","每个选中日期天数","number"],["subsidy_amount","补助金额","number"],["reason","原因","textarea"]
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
    if (type === "employees_multi") {
      const selected = Array.isArray(value) ? value.map(String) : (value ? [String(value)] : []);
      return `<label data-record-field="${name}"><span class="record-label-row"><span>${escapeHTML(label)}${required ? " *" : ""}</span><small>可按住 Ctrl/Command 多选</small></span><select name="${name}" multiple size="6" ${required ? "required" : ""} ${readonly ? "disabled" : ""}>${(adminOptions?.employees || []).map(item => `<option value="${escapeHTML(item.id)}" ${selected.includes(String(item.id)) ? "selected" : ""}>${escapeHTML(item.display_name || item.name)}</option>`).join("")}</select></label>`;
    }
    if (type === "dates_multi") {
      const selected = (Array.isArray(value) ? value : (value ? [value] : [])).map(String).filter(Boolean);
      const calendarMonth = (selected[0] || today()).slice(0, 7);
      return `<label class="record-dates-multi" data-record-field="${name}"><span class="record-label-row"><span>${escapeHTML(label)}${required ? " *" : ""}</span><small>可在月历中按住鼠标拖拽选择连续日期</small></span><div class="record-date-picker"><input type="date" data-date-multi-input ${readonly ? "disabled" : ""}><button type="button" class="ghost-button compact" data-date-multi-add ${readonly ? "disabled" : ""}>加入日期</button></div><div class="record-calendar" data-date-calendar data-calendar-month="${escapeHTML(calendarMonth)}"${readonly ? " data-readonly" : ""}></div><div class="record-date-list" data-date-multi-list>${selected.map(date => `<span class="record-date-chip" data-date-value="${escapeHTML(date)}">${escapeHTML(date)}${readonly ? "" : `<button type="button" data-date-multi-remove aria-label="删除${escapeHTML(date)}">×</button>`}</span>`).join("")}</div><small class="record-date-summary" data-date-multi-summary></small><input type="hidden" name="${name}" value="${escapeHTML(selected.join(","))}"></label>`;
    }
    if (options.length || ["departments","employees","positions","accounts","categories"].includes(type)) {
      const empty = required ? "请选择" : "不指定";
      const create = !readonly && ["accounts","categories"].includes(type) ? `<button type="button" class="record-inline-add" data-create-option="${type}">＋ 新建</button>` : "";
      return `<label data-record-field="${name}"><span class="record-label-row"><span>${escapeHTML(label)}${required ? " *" : ""}</span>${create}</span><select name="${name}" ${required ? "required" : ""} ${readonly ? "disabled" : ""}><option value="">${empty}</option>${options.map(item => `<option value="${escapeHTML(item.id)}" data-department-id="${escapeHTML(item.department_id || "")}" data-direction="${escapeHTML(item.direction || "")}" ${String(item.id) === String(value) ? "selected" : ""}>${escapeHTML(item.display_name || item.name)}${item.direction ? ` · ${item.direction === "income" ? "收入" : "支出"}` : ""}</option>`).join("")}</select></label>`;
    }
    const inputType = ["number","date","time","datetime-local"].includes(type) ? type : "text";
    const step = type === "number" ? (name === "duration_days" ? ' step="0.5"' : ' step="0.01"') : type === "datetime-local" ? ' step="1"' : "";
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
      const attendanceKind = overlay.querySelector('[name="attendance_kind"]')?.value || "";
      const specialLeave = attendanceKind === "special_leave";
      const daysField = overlay.querySelector('[data-record-field="duration_days"]');
      const subsidyField = overlay.querySelector('[data-record-field="subsidy_amount"]');
      if (daysField) daysField.hidden = false;
      if (subsidyField) subsidyField.hidden = !specialLeave;
      const days = daysField?.querySelector("input");
      if (days) days.required = true;
      const subsidy = subsidyField?.querySelector("input");
      if (subsidy) subsidy.required = specialLeave;
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
  const setupDateMulti = overlay => {
    const field = overlay.querySelector('[data-record-field="occurred_ons"], [data-work-log-dates]');
    if (!field) return;
    const input = field.querySelector("[data-date-multi-input]");
    const isWorkLog = field.hasAttribute("data-work-log-dates");
    const hidden = field.querySelector(isWorkLog ? 'input[name="work_dates"]' : 'input[name="occurred_ons"]');
    const list = field.querySelector("[data-date-multi-list]");
    const summary = field.querySelector("[data-date-multi-summary]");
    const calendar = field.querySelector("[data-date-calendar]");
    const editable = !calendar?.hasAttribute("data-readonly");
    const duration = overlay.querySelector(isWorkLog ? '[name="hours"]' : '[name="duration_days"]');
    let dragging = false;
    let dragStart = "";
    let dragEnd = "";
    const readDates = () => (hidden?.value || "").split(",").map(value => value.trim()).filter(Boolean);
    const parseDate = value => {
      const [year, month, day] = String(value).split("-").map(Number);
      return year && month && day ? new Date(Date.UTC(year, month - 1, day)) : null;
    };
    const formatDate = date => date ? date.toISOString().slice(0, 10) : "";
    const dateRange = (startValue, endValue) => {
      const start = parseDate(startValue);
      const end = parseDate(endValue);
      if (!start || !end) return [];
      const from = start <= end ? start : end;
      const to = start <= end ? end : start;
      const result = [];
      for (const cursor = new Date(from); cursor <= to; cursor.setUTCDate(cursor.getUTCDate() + 1)) result.push(formatDate(cursor));
      return result;
    };
    const dateFromPoint = event => document.elementFromPoint(event.clientX, event.clientY)?.closest("[data-calendar-date]")?.dataset.calendarDate || "";
    const renderCalendar = () => {
      if (!calendar) return;
      const monthValue = /^\d{4}-\d{2}$/.test(calendar.dataset.calendarMonth || "") ? calendar.dataset.calendarMonth : today().slice(0, 7);
      calendar.dataset.calendarMonth = monthValue;
      const [year, month] = monthValue.split("-").map(Number);
      const first = new Date(Date.UTC(year, month - 1, 1));
      const days = new Date(Date.UTC(year, month, 0)).getUTCDate();
      const offset = (first.getUTCDay() + 6) % 7;
      const selected = new Set(readDates());
      const preview = new Set(dragging ? dateRange(dragStart, dragEnd) : []);
      const cells = [];
      for (let i = 0; i < offset; i++) cells.push("<span class=\"record-calendar-empty\"></span>");
      for (let day = 1; day <= days; day++) {
        const date = `${year}-${String(month).padStart(2, "0")}-${String(day).padStart(2, "0")}`;
        const classes = ["record-calendar-day"];
        if (selected.has(date)) classes.push("is-selected");
        if (preview.has(date)) classes.push("is-preview");
        cells.push(`<button type="button" class="${classes.join(" ")}" data-calendar-date="${date}">${day}</button>`);
      }
      calendar.innerHTML = `<div class="record-calendar-head"><button type="button" data-calendar-prev aria-label="上一个月">‹</button><strong>${year}年${month}月</strong><button type="button" data-calendar-next aria-label="下一个月">›</button></div><div class="record-calendar-week"><span>一</span><span>二</span><span>三</span><span>四</span><span>五</span><span>六</span><span>日</span></div><div class="record-calendar-grid">${cells.join("")}</div>`;
      calendar.querySelector("[data-calendar-prev]")?.addEventListener("click", () => {
        const previous = new Date(Date.UTC(year, month - 2, 1));
        calendar.dataset.calendarMonth = `${previous.getUTCFullYear()}-${String(previous.getUTCMonth() + 1).padStart(2, "0")}`;
        renderCalendar();
      });
      calendar.querySelector("[data-calendar-next]")?.addEventListener("click", () => {
        const next = new Date(Date.UTC(year, month, 1));
        calendar.dataset.calendarMonth = `${next.getUTCFullYear()}-${String(next.getUTCMonth() + 1).padStart(2, "0")}`;
        renderCalendar();
      });
    };
    const render = () => {
      const dates = readDates().sort();
      if (hidden) hidden.value = dates.join(",");
      if (list) list.innerHTML = dates.map(date => `<span class="record-date-chip" data-date-value="${escapeHTML(date)}">${escapeHTML(date)}${editable ? `<button type="button" data-date-multi-remove aria-label="删除${escapeHTML(date)}">×</button>` : ""}</span>`).join("");
      if (summary) {
        const perDate = Number(duration?.value || 0);
        const total = dates.length * (Number.isFinite(perDate) ? perDate : 0);
        summary.textContent = dates.length
          ? (isWorkLog ? `已选 ${dates.length} 个工作日期，每日 ${perDate.toFixed(2)} 小时，本次合计 ${total.toFixed(2)} 小时` : `已选 ${dates.length} 个日期，每个日期 ${perDate.toFixed(2)} 天，本次合计 ${total.toFixed(2)} 天`)
          : (isWorkLog ? "请至少加入一个工作日期" : "请至少加入一个发生日期");
      }
      renderCalendar();
      list?.querySelectorAll("[data-date-multi-remove]").forEach(button => button.addEventListener("click", () => {
        const next = readDates().filter(date => date !== button.closest("[data-date-value]")?.dataset.dateValue);
        if (hidden) hidden.value = next.join(",");
        render();
      }));
    };
    editable && field.querySelector("[data-date-multi-add]")?.addEventListener("click", () => {
      const date = input?.value || "";
      if (!date) return;
      const dates = readDates();
      if (!dates.includes(date) && hidden) hidden.value = [...dates, date].join(",");
      if (input) input.value = "";
      render();
    });
    editable && calendar?.addEventListener("pointerdown", event => {
      const date = event.target.closest("[data-calendar-date]")?.dataset.calendarDate;
      if (!date) return;
      event.preventDefault();
      dragging = true;
      dragStart = date;
      dragEnd = date;
      renderCalendar();
    });
    editable && calendar?.addEventListener("pointermove", event => {
      if (!dragging) return;
      const date = dateFromPoint(event);
      if (date && date !== dragEnd) {
        dragEnd = date;
        renderCalendar();
      }
    });
    editable && document.addEventListener("pointerup", () => {
      if (!dragging) return;
      dragging = false;
      const next = [...new Set([...readDates(), ...dateRange(dragStart, dragEnd)])];
      if (hidden) hidden.value = next.join(",");
      dragStart = "";
      dragEnd = "";
      render();
    }, {once: false});
    duration?.addEventListener("input", render);
    render();
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
      if (kind === "attendance" && data.employee_id) {
        data.employee_ids = [data.employee_id];
        data.occurred_ons = data.occurred_on ? [data.occurred_on] : [];
        data.attendance_kind = data.record_type === "公休" ? "public_rest" : data.record_type === "节假日" ? "holiday" : data.record_type === "特殊请假" ? "special_leave" : "leave";
      }
      if (kind === "ledger" && data.payroll_batch_id) readonly = true;
      if (data.enabled === undefined) data.enabled = true;
      const ledgerProofs = kind === "ledger" ? `<section class="ledger-proof-panel"><div class="record-label-row"><strong>转账凭证</strong><small>支持 JPG、PNG、WebP，单张大小以服务端配置为准</small></div>${data.attachments?.length ? `<div class="ledger-proof-list">${data.attachments.map(item => `<a href="${escapeHTML(item.file_url)}" target="_blank" rel="noopener"><img src="${escapeHTML(item.file_url)}" alt="${escapeHTML(item.original_name || "转账凭证")}"><span>${escapeHTML(item.original_name || "查看图片")}</span></a>`).join("")}</div>` : `<p class="ledger-proof-empty">暂无转账凭证</p>`}<div class="upload-dropzone" data-ledger-proof-upload><input type="file" data-ledger-proof-files accept="image/jpeg,image/png,image/webp" multiple><span class="upload-dropzone-plus">＋</span><strong>拖入或选择转账凭证</strong><small>可上传银行回单、转账截图，支持多张图片</small><em data-ledger-proof-names>尚未选择文件</em></div><button type="button" class="ghost-button ledger-proof-upload-button" data-ledger-proof-submit ${id ? "" : "disabled"}>上传凭证图片</button>${id ? "" : `<p class="ledger-proof-empty">保存台账后会自动上传已选择的图片</p>`}</section>` : "";
      const payrollBreakdown = kind === "ledger" && data.payroll_items?.length ? `<section class="ledger-payroll-breakdown"><div class="record-label-row"><strong>工资发放明细</strong><small>每名员工对应一条工资金额</small></div><div class="ledger-payroll-items">${data.payroll_items.map(item => `<div><span>${escapeHTML(item.employee_name)}</span><strong>¥${escapeHTML(item.amount)}</strong></div>`).join("")}</div></section>` : "";
      const detailActions = readonly && id && kind === "ledger" && !data.payroll_batch_id ? `<button type="button" class="ghost-button" data-detail-edit>编辑</button><button type="button" class="primary-button danger-button" data-detail-delete>删除</button>` : "";
      closeRecordDialog();
      const overlay = document.createElement("div");
      overlay.className = "record-dialog";
      overlay.innerHTML = `<div class="record-dialog-card"><div class="record-dialog-head"><div><small>${readonly ? "记录详情" : id ? "修改记录" : "新增记录"}</small><h2>${escapeHTML(schema.title)}</h2></div><button type="button" data-close-dialog>×</button></div><form class="record-dialog-form"><div class="record-form-grid">${schema.fields.map(field => fieldHTML(field,data,readonly || (kind === "salary" && field[0] === "before_salary"))).join("")}</div>${payrollBreakdown}${ledgerProofs}<div class="record-dialog-error" role="alert"></div><div class="form-actions"><button type="button" class="ghost-button" data-close-dialog>${readonly ? "关闭" : "取消"}</button>${detailActions}${readonly ? "" : `<button type="submit" class="primary-button">保存${escapeHTML(schema.title)}</button>`}</div></form></div>`;
      document.body.appendChild(overlay);
      setupDateMulti(overlay);
      overlay.querySelectorAll("[data-close-dialog]").forEach(button => button.addEventListener("click", closeRecordDialog));
      overlay.addEventListener("click", event => { if (event.target === overlay) closeRecordDialog(); });
      overlay.querySelector("[data-detail-edit]")?.addEventListener("click", () => openRecordDialog(kind, id, {}, false));
      overlay.querySelector("[data-detail-delete]")?.addEventListener("click", async () => {
        const confirmed = await askConfirm({title: "删除台账记录", message: `确定删除“${data.summary || "这条流水"}”吗？删除后不可恢复。`});
        if (!confirmed) return;
        try {
          await fetchJSON(`${recordEndpoints.ledger}?id=${id}`, {method: "DELETE"});
          // 弹窗从当前列表打开，保存后只刷新当前列表和筛选条件。
          window.location.reload();
        } catch (error) {
          overlay.querySelector(".record-dialog-error").textContent = error.message;
        }
      });
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
      overlay.querySelector('[name="attendance_kind"]')?.addEventListener("change", () => syncDialogDependencies(overlay, kind));
      overlay.querySelector('[name="subsidy_amount"]')?.addEventListener("input", () => syncDialogDependencies(overlay, kind));
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
          const employeeIDs = kind === "attendance" ? Array.from(form.elements.employee_ids?.selectedOptions || []).map(option => Number(option.value)).filter(Boolean) : [];
          const attendanceDates = kind === "attendance" ? (String(values.occurred_ons || "").split(",").map(value => value.trim()).filter(Boolean)) : [];
          if (kind === "attendance") {
            const attendanceMap = {
              public_rest: {record_type: "公休", salary_effect: "none"},
              leave: {record_type: "请假", salary_effect: "deduct"},
              holiday: {record_type: "节假日", salary_effect: "none"},
              special_leave: {record_type: "特殊请假", salary_effect: "subsidy"}
            };
            const mapped = attendanceMap[values.attendance_kind];
            if (!mapped) throw new Error("请选择记录类型");
            values.category = "leave";
            values.record_type = mapped.record_type;
            values.salary_effect = mapped.salary_effect;
            delete values.attendance_kind;
          }
          if (kind === "attendance" && !id) {
            if (!employeeIDs.length) throw new Error("请选择至少一名员工");
            if (!attendanceDates.length) throw new Error("请至少加入一个发生日期");
            delete values.employee_ids;
            delete values.occurred_ons;
            for (const employeeID of employeeIDs) for (const occurredOn of attendanceDates) await fetchJSON(recordEndpoints[kind], {method:"POST",headers:{"Content-Type":"application/json"},body:JSON.stringify({...values, employee_id: employeeID, occurred_on: occurredOn})});
          } else {
            if (kind === "attendance") { if (employeeIDs.length !== 1) throw new Error("编辑记录时只能选择一名员工"); if (attendanceDates.length !== 1) throw new Error("编辑记录时只能保留一个发生日期"); delete values.employee_ids; delete values.occurred_ons; values.employee_id = employeeIDs[0] || 0; values.occurred_on = attendanceDates[0]; }
            const saved = await fetchJSON(recordEndpoints[kind] + (id ? `?id=${id}` : ""), {method:id ? "PUT" : "POST",headers:{"Content-Type":"application/json"},body:JSON.stringify(values)});
            if (kind === "ledger" && pendingProofs.length) await uploadLedgerProofFiles(saved.id || id, pendingProofs);
          }
          // 新增或修改后保持在当前列表，不丢失已选筛选条件。
          window.location.reload();
        } catch (error) { form.querySelector(".record-dialog-error").textContent = error.message; button.disabled = false; }
      });
    } catch (error) { showMessage(error.message); }
  };
  document.querySelectorAll("[data-record-form]").forEach(button => button.addEventListener("click", () => {
    const kind = button.dataset.recordForm;
    const defaults = {};
    if (kind === "attendance") { const category = button.dataset.category || "leave"; const recordType = button.dataset.recordType || (category === "public_rest" ? "公休" : category === "holiday" ? "节假日" : category === "leave" && button.dataset.recordType === "特殊请假" ? "特殊请假" : "请假"); const attendanceKind = recordType === "公休" ? "public_rest" : recordType === "节假日" ? "holiday" : recordType === "特殊请假" ? "special_leave" : "leave"; Object.assign(defaults,{attendance_kind:attendanceKind,occurred_on:today(),occurred_ons:[today()],duration_days:"1.00",subsidy_amount:"0"}); }
    if (kind === "ledger") Object.assign(defaults,{direction:"expense",occurred_on:today(),amount:""});
    if (kind === "change") Object.assign(defaults,{change_type:"transfer",after_status:"active",effective_on:today()});
    if (kind === "salary") Object.assign(defaults,{after_salary:"",effective_on:today()});
    if (kind === "purchase") Object.assign(defaults,{date:today(),supplier_name:"默认供货商",category:"蔬菜",quantity:"1",unit:"斤",unit_price:"0.00"});
    openRecordDialog(kind,Number(button.dataset.id || 0),defaults);
  }));
  document.querySelectorAll("[data-record-view]").forEach(button => button.addEventListener("click", () => openRecordDialog(button.dataset.recordView,Number(button.dataset.id),{},true)));
  // 列表行支持直接点击编辑，操作按钮和链接仍保持原来的独立行为。
  const enableListRowEditing = (selector, fallbackKind = "") => {
    document.querySelectorAll(selector).forEach(row => {
      if (row.dataset.listRowEditingBound) return;
      const editButton = row.querySelector("[data-record-form][data-id]");
      const viewButton = row.querySelector("[data-record-view][data-id]");
      const deleteButton = row.querySelector("[data-record-delete][data-id]");
      const editLink = row.querySelector('a[href*="/admin/employees/edit?"]');
      const trigger = editButton || viewButton || deleteButton;
      if (!trigger && !editLink) return;
      const kind = editButton?.dataset.recordForm || viewButton?.dataset.recordView || deleteButton?.dataset.recordDelete || fallbackKind;
      const id = Number(editButton?.dataset.id || viewButton?.dataset.id || deleteButton?.dataset.id || 0);
      if (!kind || (!id && !editLink)) return;
      row.dataset.listRowEditingBound = "true";
      row.classList.add("list-row-editable");
      row.title = editButton || editLink ? "点击列表内容可直接编辑" : "点击列表内容查看详情";
      row.addEventListener("click", event => {
        if (event.target.closest("button,a,input,select,textarea,[data-row-no-edit]")) return;
        if (editLink) {
          window.location.href = editLink.href;
          return;
        }
        openRecordDialog(kind, id, {}, !editButton);
      });
    });
  };
  enableListRowEditing(".purchase-card tbody tr", "purchase");
  enableListRowEditing(".ledger-table tbody tr", "ledger");
  enableListRowEditing(".table-card tbody tr", "attendance");
  enableListRowEditing(".organization-grid tbody tr");
  enableListRowEditing("[data-employee-row]");
  // 从员工列表进入新增或编辑页时，带上当前列表地址，保存和取消都会回到原列表。
  document.querySelectorAll("[data-employee-create], [data-employee-edit]").forEach(link => {
    link.addEventListener("click", () => {
      const target = new URL(link.href, window.location.origin);
      target.searchParams.set("return_to", window.location.pathname + window.location.search);
      link.href = target.pathname + target.search;
    });
  });
  document.querySelectorAll("[data-employee-inline]").forEach(input => {
    input.dataset.originalValue = input.value;
    input.addEventListener("change", async () => {
      if (input.value === input.dataset.originalValue) return;
      input.disabled = true;
      try {
        await fetchJSON(`/api/admin/employees?id=${input.dataset.id}`, {method:"PATCH", headers:{"Content-Type":"application/json"}, body:JSON.stringify({field:input.dataset.field, value:input.value})});
        input.dataset.originalValue = input.value;
        showMessage("员工资料已自动保存");
      } catch (error) {
        input.value = input.dataset.originalValue;
        showMessage(error.message);
      } finally { input.disabled = false; }
    });
  });
  document.querySelectorAll("[data-record-delete]").forEach(button => button.addEventListener("click", async () => {
    const kind = button.dataset.recordDelete;
    const endpoint = kind === "employee" ? "/api/admin/employees" : recordEndpoints[kind];
  const labels = {employee:"员工",department:"部门",position:"岗位",attendance:"请假与公休记录",ledger:"台账记录",purchase:"采购明细"};
    const confirmed = await askConfirm({title:`删除${labels[kind] || "记录"}`,message:`确定删除“${button.dataset.name || "这条记录"}”吗？有关联业务数据时，系统会阻止删除并说明原因。`});
    if (!confirmed) return;
    button.disabled = true;
    try {
      await fetchJSON(`${endpoint}?id=${button.dataset.id}`, {method:"DELETE"});
      // 详情页删除后回到列表，避免继续停留在已不存在的详情地址。
      // 列表内删除只刷新当前列表，保留当前筛选和页码；详情页才返回员工列表。
      if (button.closest("tr")) {
        window.location.reload();
        return;
      }
      window.location.href = kind === "employee" ? "/admin/employees" : window.location.pathname + window.location.search;
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
    try { await fetchJSON(`/api/admin/employees/leave?id=${button.dataset.id}`,{method:"PUT",headers:{"Content-Type":"application/json"},body:JSON.stringify({left_on:date})}); window.location.reload(); }
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
  const payrollSelectAll = document.querySelector("[data-payroll-select-all]");
  const payrollSelectedButton = document.querySelector("[data-pay-selected]");
  const payrollDeleteSelectedButton = document.querySelector("[data-payroll-delete-selected]");
  const paymentAccountOptions = () => adminOptions.accounts.map(v => `<option value="${v.id}" data-account-remark="${escapeHTML(v.remark || "")}">${escapeHTML(v.display_name || v.name)}</option>`).join("");
  const bindPaymentAccountRemark = overlay => {
    const select = overlay.querySelector('[name="account_id"]');
    const note = overlay.querySelector("[data-payment-account-remark]");
    const update = () => {
      const remark = select?.selectedOptions[0]?.dataset.accountRemark || "未填写";
      if (note) note.textContent = `账户备注：${remark}`;
    };
    select?.addEventListener("change", update);
    update();
  };
  const payrollSelections = () => Array.from(document.querySelectorAll("[data-payroll-select]:checked"));
  const syncPayrollSelections = () => {
    const checkboxes = Array.from(document.querySelectorAll("[data-payroll-select]:not(:disabled)"));
    const selected = payrollSelections();
    if (payrollSelectAll) {
      payrollSelectAll.checked = checkboxes.length > 0 && selected.length === checkboxes.length;
      payrollSelectAll.indeterminate = selected.length > 0 && selected.length < checkboxes.length;
    }
    if (payrollSelectedButton) {
      payrollSelectedButton.disabled = selected.length === 0;
      payrollSelectedButton.textContent = selected.length ? `发放所选（${selected.length}人）` : "发放所选工资";
    }
    if (payrollDeleteSelectedButton) {
      payrollDeleteSelectedButton.disabled = selected.length === 0;
      payrollDeleteSelectedButton.textContent = selected.length ? `删除所选（${selected.length}人）` : "删除所选";
    }
  };
  payrollSelectAll?.addEventListener("change", () => {
    document.querySelectorAll("[data-payroll-select]:not(:disabled)").forEach(input => { input.checked = payrollSelectAll.checked; });
    syncPayrollSelections();
  });
  document.querySelectorAll("[data-payroll-select]").forEach(input => input.addEventListener("change", syncPayrollSelections));
  syncPayrollSelections();
  const deletePayrollItems = async ids => {
    if (!ids.length) return;
    if (!window.confirm(`确定删除选中的 ${ids.length} 条工资明细吗？删除后可重新生成，已发放工资不能删除。`)) return;
    try {
      await fetchJSON("/api/admin/payroll", {method:"DELETE", headers:{"Content-Type":"application/json"}, body:JSON.stringify({item_ids:ids})});
      window.location.reload();
    } catch (error) { showMessage(error.message); }
  };
  payrollDeleteSelectedButton?.addEventListener("click", () => deletePayrollItems(payrollSelections().map(input => Number(input.dataset.itemId))));
  document.querySelectorAll("[data-delete-payroll]").forEach(button => button.addEventListener("click", () => deletePayrollItems([Number(button.dataset.itemId)])));
  payrollSelectedButton?.addEventListener("click", async event => {
    const button = event.currentTarget;
    const selected = payrollSelections();
    if (!selected.length) return;
    if (!window.confirm(`确定发放选中的 ${selected.length} 名员工工资吗？已发放员工不会重复入账。`)) return;
    try {
      await loadAdminOptions();
      if (!adminOptions.accounts.length) throw new Error("请先在台账新增一个资金账户");
      const overlay = document.createElement("div");
      overlay.className = "record-dialog";
      const paymentRemarkFields = selected.map(input => `<label>收款备注（${escapeHTML(input.dataset.employeeName || "员工")}，选填）<input type="text" data-payment-remark data-item-id="${input.dataset.itemId}" maxlength="500" placeholder="银行卡号、微信号或QQ号"></label>`).join("");
      overlay.innerHTML = `<div class="record-dialog-card"><div class="record-dialog-head"><div><small>工资管理</small><h2>批量发放所选工资</h2></div><button type="button" data-close-dialog>×</button></div><form class="record-dialog-form"><div class="notice-card"><span>核</span><div><strong>只发放当前勾选的员工</strong><p>已经发放的员工不会重复入账，每名员工会生成一条独立工资台账支出。</p></div></div><div class="record-form-grid"><label>付款账户 *<select name="account_id" required><option value="">请选择</option>${paymentAccountOptions()}</select><small class="payment-account-remark" data-payment-account-remark>账户备注：未填写</small></label><label>发放日期 *<input name="occurred_on" type="date" value="${today()}" required></label></div><div class="payment-remarks-grid">${paymentRemarkFields}</div><div class="record-dialog-error"></div><div class="form-actions"><button type="button" class="ghost-button" data-close-dialog>取消</button><button type="submit" class="primary-button">确认发放</button></div></form></div>`;
      document.body.appendChild(overlay);
      bindPaymentAccountRemark(overlay);
      overlay.querySelectorAll("[data-close-dialog]").forEach(v => v.addEventListener("click", closeRecordDialog));
      overlay.querySelector("form").addEventListener("submit", async submitEvent => {
        submitEvent.preventDefault();
        const form = submitEvent.currentTarget;
        const submit = form.querySelector("button[type=submit]");
        submit.disabled = true;
        const values = Object.fromEntries(new FormData(form).entries());
        values.batch_id = Number(button.dataset.batchId);
        values.item_ids = selected.map(input => Number(input.dataset.itemId));
        values.payment_remarks = {};
        form.querySelectorAll("[data-payment-remark]").forEach(input => { values.payment_remarks[input.dataset.itemId] = input.value.trim(); });
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
  });
  document.querySelector("[data-pay-payroll]")?.addEventListener("click", async event => {
    const button = event.currentTarget;
    try {
      await loadAdminOptions();
      if (!adminOptions.accounts.length) throw new Error("请先在台账新增一个资金账户");
      closeRecordDialog();
      const overlay=document.createElement("div");overlay.className="record-dialog";
      overlay.innerHTML=`<div class="record-dialog-card"><div class="record-dialog-head"><div><small>工资管理</small><h2>发放本月工资</h2></div><button type="button" data-close-dialog>×</button></div><form class="record-dialog-form"><div class="notice-card"><span>核</span><div><strong>发放后工资将锁定</strong><p>系统会同时在台账生成工资支出，请确认工资明细已经核对完成。</p></div></div><div class="record-form-grid"><label>付款账户 *<select name="account_id" required><option value="">请选择</option>${paymentAccountOptions()}</select><small class="payment-account-remark" data-payment-account-remark>账户备注：未填写</small></label><label>发放日期 *<input name="occurred_on" type="date" value="${today()}" required></label></div><div class="record-dialog-error"></div><div class="form-actions"><button type="button" class="ghost-button" data-close-dialog>取消</button><button type="submit" class="primary-button">确认发放</button></div></form></div>`;
      document.body.appendChild(overlay);bindPaymentAccountRemark(overlay);overlay.querySelectorAll("[data-close-dialog]").forEach(v=>v.addEventListener("click",closeRecordDialog));
      overlay.querySelector("form").addEventListener("submit",async submitEvent=>{submitEvent.preventDefault();const form=submitEvent.currentTarget;const submit=form.querySelector("button[type=submit]");submit.disabled=true;const values=Object.fromEntries(new FormData(form).entries());values.batch_id=Number(button.dataset.batchId);values.account_id=Number(values.account_id);try{await fetchJSON("/api/admin/payroll/pay",{method:"POST",headers:{"Content-Type":"application/json"},body:JSON.stringify(values)});window.location.reload()}catch(error){form.querySelector(".record-dialog-error").textContent=error.message;submit.disabled=false}});
    } catch(error){showMessage(error.message)}
  });
  document.querySelectorAll("[data-pay-payroll-item]").forEach(button => button.addEventListener("click", async () => {
    try {
      await loadAdminOptions();
      if (!adminOptions.accounts.length) throw new Error("请先在台账新增一个资金账户");
      const overlay = document.createElement("div");
      overlay.className = "record-dialog";
      overlay.innerHTML = `<div class="record-dialog-card"><div class="record-dialog-head"><div><small>工资管理</small><h2>单独发放：${escapeHTML(button.dataset.employeeName || "员工")}</h2></div><button type="button" data-close-dialog>×</button></div><form class="record-dialog-form"><div class="notice-card"><span>核</span><div><strong>只发放当前员工</strong><p>已发放的其他员工不会重复入账；当前员工发放后将锁定。</p></div></div><div class="record-form-grid"><label>付款账户 *<select name="account_id" required><option value="">请选择</option>${paymentAccountOptions()}</select><small class="payment-account-remark" data-payment-account-remark>账户备注：未填写</small></label><label>发放日期 *<input name="occurred_on" type="date" value="${today()}" required></label></div><label class="payment-remark-field">收款备注（选填）<input name="payment_remark" maxlength="500" placeholder="填写该员工的银行卡号、微信号或QQ号"></label><div class="record-dialog-error"></div><div class="form-actions"><button type="button" class="ghost-button" data-close-dialog>取消</button><button type="submit" class="primary-button">确认发放</button></div></form></div>`;
      document.body.appendChild(overlay);
      bindPaymentAccountRemark(overlay);
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

  const employeeLeftDate = document.querySelector('[data-employee-form] [name="left_on"]');
  const employeeStatusSelect = document.querySelector('[data-employee-form] [name="employment_status"]');
  employeeLeftDate?.addEventListener("change", () => {
    if (employeeLeftDate.value && employeeStatusSelect) employeeStatusSelect.value = "left";
  });

  const employeeSearch = document.querySelector("[data-employee-search]");
  const employeeDepartment = document.querySelector("[data-employee-department]");
  const employeePosition = document.querySelector("[data-employee-position]");
  const employeeStatus = document.querySelector("[data-employee-status]");
  const employeeTypeTabs = Array.from(document.querySelectorAll("[data-employee-type-tab]"));
  const employeeTable = document.querySelector("[data-employee-table]");
  const employeeListDescription = document.querySelector("[data-employee-list-description]");
  const employeeSelectAll = document.querySelector("[data-employee-select-all]");
  const employeeDeleteSelected = document.querySelector("[data-employee-delete-selected]");
  let employeeType = "";
  const visibleEmployeeSelections = () => Array.from(document.querySelectorAll("[data-employee-row]"))
    .filter(row => !row.hidden)
    .map(row => row.querySelector("[data-employee-select]"))
    .filter(Boolean);
  const selectedEmployees = () => Array.from(document.querySelectorAll("[data-employee-select]:checked"));
  const syncEmployeeSelections = () => {
    const visible = visibleEmployeeSelections();
    const selected = selectedEmployees();
    if (employeeSelectAll) {
      employeeSelectAll.checked = visible.length > 0 && visible.every(input => input.checked);
      employeeSelectAll.indeterminate = selected.length > 0 && !employeeSelectAll.checked;
    }
    if (employeeDeleteSelected) {
      employeeDeleteSelected.disabled = selected.length === 0;
      employeeDeleteSelected.textContent = selected.length ? `删除所选（${selected.length}人）` : "删除所选";
    }
  };
  const deleteEmployees = async ids => {
    if (!ids.length) return;
    if (!window.confirm(`确定删除选中的 ${ids.length} 名员工吗？删除后会隐藏档案，但不会物理清除工资、请假和台账历史。`)) return;
    try {
      await fetchJSON("/api/admin/employees", {method:"DELETE", headers:{"Content-Type":"application/json"}, body:JSON.stringify({ids})});
      window.location.reload();
    } catch (error) { showMessage(error.message); }
  };
  employeeSelectAll?.addEventListener("change", () => {
    visibleEmployeeSelections().forEach(input => { input.checked = employeeSelectAll.checked; });
    syncEmployeeSelections();
  });
  document.querySelectorAll("[data-employee-select]").forEach(input => input.addEventListener("change", syncEmployeeSelections));
  employeeDeleteSelected?.addEventListener("click", () => deleteEmployees(selectedEmployees().map(input => Number(input.value))));
  const updateEmployeeTableMode = () => {
    const temporaryMode = employeeType === "兼职";
    employeeTable?.classList.toggle("temporary-table-mode", temporaryMode);
    if (employeeListDescription) employeeListDescription.textContent = temporaryMode
      ? "临时工不设公休和当前月工资；只维护计薪方式与单价，每日工时会自动汇总至临时工工资"
      : "正式工可单独设置公休；临时工只设置计薪方式和单价，工资由每日工时自动汇总";
  };
  const filterEmployees = () => {
    const keyword = employeeSearch?.value.trim().toLowerCase() || "";
    const departmentID = employeeDepartment?.value || "";
    const positionID = employeePosition?.value || "";
    const status = employeeStatus?.value || "";
    let visibleCount = 0;
    document.querySelectorAll("[data-employee-row]").forEach((row) => {
      const matchesKeyword = !keyword || row.dataset.search.toLowerCase().includes(keyword);
      const matchesDepartment = !departmentID || row.dataset.departmentId === departmentID;
      const matchesPosition = !positionID || row.dataset.positionId === positionID;
      const matchesStatus = !status || row.dataset.status === status;
      const matchesType = !employeeType || row.dataset.employmentType === employeeType;
      row.dataset.textFiltered = String(!(matchesKeyword && matchesDepartment && matchesPosition && matchesStatus && matchesType));
      row.hidden = row.dataset.textFiltered === "true";
      if (row.hidden) {
        const checkbox = row.querySelector("[data-employee-select]");
        if (checkbox) checkbox.checked = false;
      }
      if (!row.hidden) visibleCount += 1;
    });
    const count = document.querySelector("[data-employee-count]");
    if (count) count.textContent = `${visibleCount} 人`;
    syncEmployeeSelections();
  };
  employeeSearch?.addEventListener("input", filterEmployees);
  employeeDepartment?.addEventListener("change", filterEmployees);
  employeePosition?.addEventListener("change", filterEmployees);
  employeeStatus?.addEventListener("change", filterEmployees);
  employeeTypeTabs.forEach(tab => tab.addEventListener("click", () => {
    employeeType = tab.dataset.employeeTypeTab || "";
    employeeTypeTabs.forEach(item => item.classList.toggle("active", item === tab));
    updateEmployeeTableMode();
    filterEmployees();
  }));
  syncEmployeeSelections();

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

  const payrollPayload = (row) => {
    const payload = {
      work_hours: numberValue(row, "work_hours"), base_salary: numberValue(row, "base_salary"), bonus: numberValue(row, "bonus"),
      attendance_deduction: numberValue(row, "attendance_deduction"), other_deduction: numberValue(row, "other_deduction"),
      social_security: numberValue(row, "social_security"), tax: numberValue(row, "tax")
    };
    const manual = row.querySelector('[data-field="manual_net_salary"]')?.value.trim();
    if (manual !== undefined && manual !== "") payload.manual_net_salary = Number(manual);
    return payload;
  };
  const savePayrollRow = async (row, version = Number(row.dataset.payrollVersion || 0)) => {
    const response = await fetch(`/api/admin/payroll?id=${row.dataset.id}`, {method: "PUT", headers: {"Content-Type": "application/json"}, body: JSON.stringify(payrollPayload(row))});
    const data = await response.json().catch(() => ({}));
    if (!response.ok) throw new Error(data.error || "保存失败");
    if (version === Number(row.dataset.payrollVersion || 0)) {
      row.querySelector("[data-net-salary]").textContent = data.net_salary;
      row.dataset.payrollDirty = "false";
    }
    return data;
  };
  const flushPayrollRow = (row) => {
    clearTimeout(row._payrollSaveTimer);
    row._payrollSaveTimer = null;
    const version = Number(row.dataset.payrollVersion || 0);
    const previous = row._payrollPending || Promise.resolve();
    const task = previous.catch(() => {}).then(() => savePayrollRow(row, version));
    row._payrollPending = task;
    return task;
  };
  const schedulePayrollSave = (row) => {
    clearTimeout(row._payrollSaveTimer);
    row._payrollSaveTimer = setTimeout(() => {
      row._payrollSaveTimer = null;
      flushPayrollRow(row).catch(error => showMessage(`自动保存失败：${error.message}`));
    }, 450);
  };

  const payrollRows = Array.from(document.querySelectorAll("[data-payroll-row]"));
  const payrollTable = document.querySelector("[data-payroll-table]");
  let payrollSort = {key: "", direction: 1};
  payrollTable?.querySelectorAll("[data-payroll-sort]").forEach(button => button.addEventListener("click", () => {
    const key = button.dataset.payrollSort;
    payrollSort = {key, direction: payrollSort.key === key ? -payrollSort.direction : 1};
    const body = payrollTable.querySelector("tbody");
    const rows = Array.from(body?.querySelectorAll("[data-payroll-row]") || []);
    const value = row => key === "net_salary"
      ? Number((row.querySelector("[data-net-salary]")?.textContent || "").replace(/[^0-9.-]/g, ""))
      : numberValue(row, key);
    rows.sort((left, right) => (value(left) - value(right)) * payrollSort.direction).forEach(row => body.appendChild(row));
    payrollTable.querySelectorAll("[data-payroll-sort]").forEach(item => item.classList.toggle("is-active", item === button));
  }));
  payrollRows.forEach((row) => {
    row.querySelectorAll(".money-input").forEach((input) => input.addEventListener("input", () => {
      row.dataset.payrollVersion = String(Number(row.dataset.payrollVersion || 0) + 1);
      row.dataset.payrollDirty = "true";
      renderNet(row);
      schedulePayrollSave(row);
    }));
    row.querySelector("[data-save-payroll]")?.addEventListener("click", async (event) => {
      event.preventDefault();
      try {
        await flushPayrollRow(row);
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

  const employeeReturnToList = () => {
    const target = new URLSearchParams(window.location.search).get("return_to") || "/admin/employees";
    return target.startsWith("/admin/employees") ? target : "/admin/employees";
  };
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
        window.location.href = employeeReturnToList();
      } catch (error) {
        if (saved?.id) {
          showMessage(`员工已保存，但证件图片上传失败：${error.message}`);
          setTimeout(() => { window.location.href = `/admin/employees/edit?id=${saved.id}&return_to=${encodeURIComponent(employeeReturnToList())}`; }, 700);
        } else {
          showMessage(`保存员工失败：${error.message}`);
          button.disabled = false;
        }
      }
    });
  }

  // 统一补充列表序号、查询条件和金额区间筛选，避免各页面重复实现。
  const pagePath = window.location.pathname;
  if (pagePath === "/admin/payroll") {
    const exportButton = document.querySelector("[data-payroll-export]");
    if (exportButton) {
      exportButton.addEventListener("click", (event) => {
        event.preventDefault();
        const month = exportButton.dataset.month || document.querySelector('input[type="month"]')?.value || "";
        const type = exportButton.dataset.payrollType || "full_time";
        const query = `?month=${encodeURIComponent(month)}&type=${encodeURIComponent(type)}`;
        const dirtyRows = Array.from(document.querySelectorAll("[data-payroll-row][data-payroll-dirty=\"true\"]"));
        exportButton.disabled = true;
        Promise.all(dirtyRows.map(flushPayrollRow)).then(() => {
          window.location.assign(`/admin/payroll/export${query}`);
        }).catch(error => {
          exportButton.disabled = false;
          showMessage(`导出前保存失败：${error.message}`);
        });
      });
    }
  }
  const addSequenceColumn = (table) => {
    if (!table || table.querySelector("th[data-sequence-column]")) return;
    const head = table.querySelector("thead tr");
    if (!head) return;
    const th = document.createElement("th");
    th.dataset.sequenceColumn = "true";
    th.textContent = "序号";
    head.insertBefore(th, head.firstElementChild);
    table.querySelectorAll("tbody tr").forEach((row, index) => {
      const empty = row.querySelector(".empty-cell");
      if (empty) { empty.colSpan = Number(empty.colSpan || 1) + 1; return; }
      const td = document.createElement("td");
      td.className = "mono";
      td.textContent = String(index + 1);
      row.insertBefore(td, row.firstElementChild);
    });
  };
  if (["/admin/payroll", "/admin/purchases", "/admin/salary-adjustments", "/admin/organization", "/admin/audit-logs"].includes(pagePath)) {
    document.querySelectorAll(".table-wrap table").forEach(addSequenceColumn);
  }

  const queryValue = (name) => new URLSearchParams(window.location.search).get(name) || "";
  const navigateWithQuery = (values) => {
    const url = new URL(window.location.href);
    Object.entries(values).forEach(([key, value]) => {
      if (value === undefined || value === null || String(value).trim() === "") url.searchParams.delete(key);
      else url.searchParams.set(key, String(value).trim());
    });
    url.searchParams.delete("page");
    window.location.href = `${url.pathname}?${url.searchParams.toString()}`;
  };
  const addToolbarInput = (toolbar, placeholder, type = "search") => {
    const input = document.createElement("input");
    input.type = type;
    input.placeholder = placeholder;
    input.className = "table-filter-input";
    toolbar.appendChild(input);
    return input;
  };
  const addToolbarButton = (toolbar) => {
    const button = document.createElement("button");
    button.type = "button";
    button.className = "primary-button compact";
    button.textContent = "筛选";
    toolbar.appendChild(button);
    return button;
  };
  const setupQueryToolbar = (kind) => {
    const toolbar = document.querySelector(".table-card .table-toolbar");
    if (!toolbar) return;
    const monthInput = toolbar.querySelector('input[type="month"]');
    const select = toolbar.querySelector("select");
    let search = toolbar.querySelector(".search-box input");
    if (!search) {
      const box = document.createElement("div");
      box.className = "search-box";
      box.textContent = "⌕ ";
      search = addToolbarInput(box, kind === "payroll" ? "搜索员工或编号" : "搜索员工、编号或原因");
      box.appendChild(search);
      toolbar.insertBefore(box, toolbar.lastElementChild);
    }
    if (select) {
      select.options.forEach((option, index) => { option.value = index === 0 ? "" : option.textContent.trim(); });
      if (kind === "changes") {
        ["调岗", "降职"].forEach(name => {
          if (!Array.from(select.options).some(option => option.value === name)) {
            const option = document.createElement("option");
            option.value = name;
            option.textContent = name;
            select.appendChild(option);
          }
        });
      }
      if (kind === "payroll" && !Array.from(select.options).some(option => option.value === "已确认")) {
        const option = document.createElement("option"); option.value = "已确认"; option.textContent = "已确认"; select.appendChild(option);
      }
    }
    const button = addToolbarButton(toolbar);
    if (monthInput) monthInput.value = queryValue("month");
    if (select) select.value = queryValue(kind === "payroll" ? "status" : "change_type");
    search.value = queryValue("q");
    const queryKey = kind === "payroll" ? "status" : kind === "changes" ? "change_type" : "";
    const submit = () => navigateWithQuery({month: monthInput?.value, ...(queryKey ? {[queryKey]: select?.value} : {}), q: search.value});
    button.addEventListener("click", submit);
    search.addEventListener("keydown", event => { if (event.key === "Enter") { event.preventDefault(); submit(); } });
    monthInput?.addEventListener("change", submit);
    select?.addEventListener("change", submit);
  };
  if (pagePath === "/admin/changes") setupQueryToolbar("changes");
  if (pagePath === "/admin/salary-adjustments") setupQueryToolbar("salary");
  if (pagePath === "/admin/payroll") setupQueryToolbar("payroll");
  if (pagePath === "/admin/leaves") {
    const monthInput = document.querySelector('.table-card .table-toolbar input[type="month"]');
    const form = monthInput?.closest("form");
    monthInput?.addEventListener("change", () => form?.requestSubmit());
  }
  document.querySelectorAll(".pagination-bar a").forEach(link => {
    const target = new URL(link.href, window.location.href);
    const current = new URL(window.location.href);
    current.searchParams.forEach((value, key) => { if (key !== "page") target.searchParams.set(key, value); });
    link.href = `${target.pathname}?${target.searchParams.toString()}`;
  });

  const addNumberRange = (toolbar, minPlaceholder, maxPlaceholder) => {
    const min = addToolbarInput(toolbar, minPlaceholder, "number");
    const max = addToolbarInput(toolbar, maxPlaceholder, "number");
    min.step = max.step = "0.01";
    return {min, max};
  };
  const filterRowsByAmount = (table, amountIndex, range) => {
    if (!table || !range) return;
    const min = Number(range.min.value || "-Infinity");
    const max = Number(range.max.value || "Infinity");
    table.querySelectorAll("tbody tr").forEach(row => {
      if (row.querySelector(".empty-cell")) return;
      const amount = Number((row.cells[amountIndex]?.textContent || "").replace(/[^0-9.-]/g, ""));
      row.hidden = row.dataset.textFiltered === "true" || (!Number.isNaN(amount) && (amount < min || amount > max));
    });
  };
  const setupClientTextFilter = (table, toolbar, placeholder) => {
    if (!table || !toolbar) return null;
    let input = toolbar.querySelector(".search-box input");
    if (!input) {
      const box = document.createElement("div");
      box.className = "search-box";
      box.textContent = "⌕ ";
      input = addToolbarInput(box, placeholder);
      toolbar.appendChild(box);
    }
    const apply = () => {
      const keyword = input.value.trim().toLowerCase();
      table.querySelectorAll("tbody tr").forEach(row => {
        if (row.querySelector(".empty-cell")) return;
        row.dataset.textFiltered = String(Boolean(keyword) && !row.textContent.toLowerCase().includes(keyword));
        row.hidden = row.dataset.textFiltered === "true";
      });
    };
    input.addEventListener("input", apply);
    return input;
  };
  const pageTable = document.querySelector(".table-card .table-wrap table");
  if (pagePath === "/admin/salary-adjustments" && pageTable) {
    const toolbar = document.querySelector(".table-card .table-toolbar");
    const range = addNumberRange(toolbar, "调整后最低", "调整后最高");
    const apply = () => filterRowsByAmount(pageTable, 3, range);
    range.min.addEventListener("input", apply); range.max.addEventListener("input", apply);
  }
  if (pagePath === "/admin/purchases" && pageTable) {
    const toolbar = document.querySelector(".table-card .table-toolbar");
    setupClientTextFilter(pageTable, toolbar, "搜索供货商、品类或物料");
    const range = addNumberRange(toolbar, "总价最低", "总价最高");
    const apply = () => filterRowsByAmount(pageTable, 8, range);
    range.min.addEventListener("input", apply); range.max.addEventListener("input", apply);
  }
  if (pagePath === "/admin/employees" && pageTable) {
    const toolbar = document.querySelector(".table-card .table-toolbar");
    const range = addNumberRange(toolbar, "工资最低", "工资最高");
    const apply = () => filterRowsByAmount(pageTable, 9, range);
    range.min.addEventListener("input", apply); range.max.addEventListener("input", apply);
  }
  if (pagePath === "/admin/leaves" && pageTable) {
    const toolbar = document.querySelector(".table-card .table-toolbar");
    const range = addNumberRange(toolbar, "时长最低", "时长最高");
    const apply = () => filterRowsByAmount(pageTable, 6, range);
    range.min.addEventListener("input", apply); range.max.addEventListener("input", apply);
  }
  if (pagePath === "/admin/payroll" && pageTable) {
    const toolbar = document.querySelector(".table-card .table-toolbar");
    const range = addNumberRange(toolbar, "实发最低", "实发最高");
    const payrollType = document.querySelector("[data-payroll-table]")?.dataset.payrollType;
    const apply = () => filterRowsByAmount(pageTable, payrollType === "part_time" ? 7 : 10, range);
    range.min.addEventListener("input", apply); range.max.addEventListener("input", apply);
  }
  if (pagePath === "/admin/stalls" || pagePath === "/admin/stalls/monthly") {
    const root = document.querySelector("[data-stalls-page]");
    const month = () => root?.querySelector('input[name="month"]')?.value || today().slice(0, 7);
    const closeDialog = overlay => overlay.remove();
    const openStallDialog = (data = {}, monthly = false) => {
      const edit = Boolean(data.id);
      const overlay = document.createElement("div");
      overlay.className = "record-dialog";
      overlay.innerHTML = `<div class="record-dialog-card"><div class="record-dialog-head"><div><small>${monthly ? "档口月度核算" : "档口管理"}</small><h2>${monthly ? `编辑${escapeHTML(data.month || month())}当月规则` : edit ? "编辑档口" : "新增档口"}</h2></div><button type="button" data-close-dialog>×</button></div><form class="record-dialog-form"><div class="record-form-grid">${monthly ? `<label>月份 *<input name="month" type="month" required value="${escapeHTML(data.month || month())}"></label>` : `<label>档口编号 *<input name="stall_no" required maxlength="64" value="${escapeHTML(data.stallNo || "")}" placeholder="例如：A01"></label><label>档口名称 *<input name="name" required maxlength="128" value="${escapeHTML(data.name || "")}"></label><label>负责人<input name="responsible_name" maxlength="64" value="${escapeHTML(data.responsibleName || "")}"></label><label>负责人电话（选填）<input name="responsible_mobile" maxlength="32" value="${escapeHTML(data.responsibleMobile || "")}"></label>`}<label>抽成模式<select name="allocation_mode"><option value="custom" ${data.allocationMode === "custom" ? "selected" : ""}>其他档口自定义</option><option value="profit_share" ${data.allocationMode === "profit_share" ? "selected" : ""}>甲乙扣成本后分成</option><option value="none" ${data.allocationMode === "none" ? "selected" : ""}>不参与我方抽成</option></select></label><label>月租（元）<input name="monthly_rent" type="number" min="0" step="0.01" value="${escapeHTML(data.monthlyRent || "0")}"></label><label>档口默认保底金额（元）<input name="default_guaranteed_amount" type="number" min="0" step="0.01" value="${escapeHTML(data.defaultGuaranteedAmount || "0")}"></label><label>甲分成比例（%）<input name="profit_jia_rate" type="number" min="0" max="100" step="0.01" value="${escapeHTML(data.profitJiaRate || "0")}"></label><label>乙分成比例（%）<input name="profit_yi_rate" type="number" min="0" max="100" step="0.01" value="${escapeHTML(data.profitYiRate || "0")}"></label><label>保底抽成比例（%）<input name="guaranteed_rate" type="number" min="0" max="100" step="0.01" value="${escapeHTML(data.guaranteedRate || "0")}"></label><label>营业额第一层比例（%）<input name="base_amount_rate" type="number" min="0" max="100" step="0.01" value="${escapeHTML(data.baseAmountRate || "0")}"></label><label>第一层再乘比例（%）<input name="base_amount_factor" type="number" min="0" max="100" step="0.01" value="${escapeHTML(data.baseAmountFactor || "0")}"></label><label>总营业额比例（%）<input name="total_amount_rate" type="number" min="0" max="100" step="0.01" value="${escapeHTML(data.totalAmountRate || "0")}"></label><label class="span-2">备注（选填）<textarea name="remark" maxlength="500">${escapeHTML(data.remark || "")}</textarea></label></div><div class="notice-card compact-notice"><span>i</span><div><strong>${monthly ? "这里只保存当前月份，不会修改档口总设置" : "保底金额在这里设置，按档口分别保存"}</strong><p>${monthly ? "当月规则覆盖只对当前月份生效；下个月仍按档口长期规则计算。" : "当月营业额窗口可临时覆盖保底基数；百分比留空或填0表示不启用。"}</p></div></div><div class="record-dialog-error" role="alert"></div><div class="form-actions"><button type="button" class="ghost-button" data-close-dialog>取消</button><button type="submit" class="primary-button">${monthly ? "保存当月规则" : "保存档口"}</button></div></form></div>`;
      document.body.appendChild(overlay);
      const close = () => overlay.remove();
      overlay.querySelectorAll("[data-close-dialog]").forEach(button => button.addEventListener("click", close));
      overlay.addEventListener("click", event => { if (event.target === overlay) close(); });
      overlay.querySelector("form").addEventListener("submit", async event => {
        event.preventDefault();
        const form = event.currentTarget; const submit = form.querySelector("button[type=submit]"); submit.disabled = true;
        try { const values = Object.fromEntries(new FormData(form).entries()); if (monthly) { values.stall_id = Number(data.id); await fetchJSON("/api/admin/stall-monthly-rules", {method:"POST", headers:{"Content-Type":"application/json"}, body:JSON.stringify(values)}); } else { await fetchJSON(`/api/admin/stalls${edit ? `?id=${data.id}` : ""}`, {method: edit ? "PUT" : "POST", headers: {"Content-Type": "application/json"}, body: JSON.stringify(values)}); } window.location.reload(); }
        catch (error) { form.querySelector(".record-dialog-error").textContent = error.message; submit.disabled = false; }
      });
    };
    const openTurnoverDialog = (data) => {
      const edit = Boolean(data.turnoverId && Number(data.turnoverId)); const overlay = document.createElement("div"); overlay.className = "record-dialog";
      overlay.innerHTML = `<div class="record-dialog-card"><div class="record-dialog-head"><div><small>${escapeHTML(data.stallName || "档口")}</small><h2>录入${escapeHTML(data.month || month())}营业额</h2></div><button type="button" data-close-dialog>×</button></div><form class="record-dialog-form"><div class="record-form-grid"><label>月份 *<input name="month" type="month" required value="${escapeHTML(data.month || month())}"></label><label>学校卡机营业额<input name="card_amount" type="number" min="0" step="0.01" value="${escapeHTML(data.card || "0")}"></label><label>汇来米营业额<input name="huilaimi_amount" type="number" min="0" step="0.01" value="${escapeHTML(data.huilaimi || "0")}"></label><label>当月成本<input name="cost_amount" type="number" min="0" step="0.01" value="${escapeHTML(data.cost || "0")}"></label><label>保底金额基数<input name="guaranteed_amount" type="number" min="0" step="0.01" value="${escapeHTML(data.guaranteed || "0")}"></label><label>手动我方抽成<input name="manual_own_commission" type="number" min="0" step="0.01" value="${escapeHTML(data.manualOwn || "")}" placeholder="未配规则时可填写，留空恢复算法"></label><label class="span-2">备注（选填）<textarea name="remark" maxlength="500"></textarea></label></div><div class="notice-card compact-notice"><span>算</span><div><strong>本月总营业额 = 学校卡机 + 汇来米</strong><p>康莱德只按学校卡机计算；未配置档口抽成规则时，可填写手动我方抽成，留空则继续按档口规则自动计算。</p></div></div><div class="record-dialog-error" role="alert"></div><div class="form-actions"><button type="button" class="ghost-button" data-close-dialog>取消</button><button type="submit" class="primary-button">保存营业额</button></div></form></div>`;
      document.body.appendChild(overlay); const close = () => overlay.remove(); overlay.querySelectorAll("[data-close-dialog]").forEach(button => button.addEventListener("click", close)); overlay.addEventListener("click", event => { if (event.target === overlay) close(); });
      overlay.querySelector("form").addEventListener("submit", async event => { event.preventDefault(); const form = event.currentTarget; const submit = form.querySelector("button[type=submit]"); submit.disabled = true; try { const values = Object.fromEntries(new FormData(form).entries()); values.stall_id = Number(data.stallId); await fetchJSON(`/api/admin/stall-turnovers${edit ? `?id=${data.turnoverId}` : ""}`, {method: edit ? "PUT" : "POST", headers: {"Content-Type": "application/json"}, body: JSON.stringify(values)}); window.location.reload(); } catch (error) { form.querySelector(".record-dialog-error").textContent = error.message; submit.disabled = false; } });
    };
    const openLedgerDialog = async (data) => {
      try {
        await loadAdminOptions();
        const accounts = (adminOptions.accounts || []).map(item => `<option value="${escapeHTML(item.id)}">${escapeHTML(item.display_name || item.name)}</option>`).join("");
        const overlay = document.createElement("div"); overlay.className = "record-dialog";
        overlay.innerHTML = `<div class="record-dialog-card"><div class="record-dialog-head"><div><small>档口抽成</small><h2>计入台账：${escapeHTML(data.stallName || "档口")}</h2></div><button type="button" data-close-dialog>×</button></div><form class="record-dialog-form"><div class="notice-card"><span>收</span><div><strong>本方抽成 ¥${escapeHTML(data.ownCommission || "0")}</strong><p>只将甲乙及其他自营档口的本方抽成写入台账；康莱德公司抽成不计入本方收入。</p></div></div><div class="record-form-grid"><label>入账账户 *<select name="account_id" required><option value="">请选择</option>${accounts}</select></label><label>入账日期 *<input name="occurred_on" type="date" required value="${escapeHTML(`${data.month || month()}-01`)}"></label></div><div class="record-dialog-error" role="alert"></div><div class="form-actions"><button type="button" class="ghost-button" data-close-dialog>取消</button><button type="submit" class="primary-button">确认计入台账</button></div></form></div>`;
        document.body.appendChild(overlay); const close = () => overlay.remove(); overlay.querySelectorAll("[data-close-dialog]").forEach(button => button.addEventListener("click", close)); overlay.addEventListener("click", event => { if (event.target === overlay) close(); });
        overlay.querySelector("form").addEventListener("submit", async event => { event.preventDefault(); const form = event.currentTarget; const submit = form.querySelector("button[type=submit]"); submit.disabled = true; try { const values = Object.fromEntries(new FormData(form).entries()); values.stall_id = Number(data.stallId); values.month = data.month || month(); values.account_id = Number(values.account_id); await fetchJSON("/api/admin/stall-commission-ledger", {method:"POST",headers:{"Content-Type":"application/json"},body:JSON.stringify(values)}); close(); showMessage("档口抽成已计入台账"); } catch (error) { form.querySelector(".record-dialog-error").textContent = error.message; submit.disabled = false; } });
      } catch (error) { showMessage(error.message); }
    };
    document.querySelector("[data-stall-settings]")?.addEventListener("submit", async event => {
      event.preventDefault();
      const form = event.currentTarget; const button = form.querySelector("button[type=submit]"); button.disabled = true;
      try { const values = Object.fromEntries(new FormData(form).entries()); await fetchJSON("/api/admin/stall-settings", {method:"POST",headers:{"Content-Type":"application/json"},body:JSON.stringify(values)}); showMessage("公共抽成设置已保存"); }
      catch (error) { form.querySelector("[data-stall-settings-error]").textContent = error.message; }
      finally { button.disabled = false; }
    });
    const inlineTurnoverHTML = value => `<strong>¥${value || "0.00"}</strong><small>点击修改</small>`;
    const inlineOwnCommissionHTML = item => {
      if (item.can_manual_own_commission) return `<strong class="salary">¥${item.own_commission || "0.00"}</strong><small class="cell-subtitle">${item.manual_own_commission_set ? "手动抽成，点击修改" : "未设规则，点击填写"}</small>`;
      const modeText = item.allocation_mode === "profit_share" ? "甲乙净额分成" : item.allocation_mode === "none" ? "不抽成" : "月租+自定义比例";
      return `<strong class="salary">¥${item.own_commission || "0.00"}</strong><small class="cell-subtitle">${modeText}</small>`;
    };
    const updateCalculatedRow = (row, item) => {
      const money = value => `¥${value || "0.00"}`;
      const cells = row.cells;
      cells[3].innerHTML = inlineTurnoverHTML(item.school_card_amount);
      cells[4].innerHTML = inlineTurnoverHTML(item.huilaimi_amount);
      cells[5].innerHTML = `<strong>${money(item.total_turnover)}</strong>`;
      cells[6].innerHTML = inlineTurnoverHTML(item.cost_amount);
      cells[7].textContent = money(item.net_turnover);
      cells[8].textContent = money(item.conrad_commission);
      cells[9].textContent = money(item.jia_commission);
      cells[10].textContent = money(item.yi_commission);
      cells[11].innerHTML = inlineOwnCommissionHTML(item);
      cells[12].textContent = money(item.remaining);
      const turnoverButton = cells[13].querySelector("[data-turnover-edit]");
      if (turnoverButton) {
        turnoverButton.dataset.turnoverId = item.turnover_id || "";
        turnoverButton.dataset.month = item.month || "";
        turnoverButton.dataset.card = item.school_card_amount || "0.00";
        turnoverButton.dataset.huilaimi = item.huilaimi_amount || "0.00";
        turnoverButton.dataset.cost = item.cost_amount || "0.00";
        turnoverButton.dataset.guaranteed = item.guaranteed_amount || "0.00";
        turnoverButton.dataset.manualOwn = item.manual_own_commission_set ? item.manual_own_commission || "0.00" : "";
      }
      const ledgerButton = cells[13].querySelector("[data-stall-ledger]");
      if (ledgerButton) ledgerButton.dataset.ownCommission = item.own_commission || "0.00";
    };
    document.querySelector("[data-stalls-recalculate-all]")?.addEventListener("click", () => {
      const url = new URL(window.location.href);
      url.searchParams.set("month", month());
      window.location.href = url.toString();
    });
    document.querySelector("[data-month-navigate]")?.addEventListener("change", event => {
      const value = event.currentTarget.value;
      if (!value) return;
      const url = new URL(window.location.href);
      url.searchParams.set("month", value);
      window.location.href = url.toString();
    });
    const startInlineEdit = (cell, value, save, afterSave) => {
      if (cell.dataset.inlineEditing === "true") return;
      cell.dataset.inlineEditing = "true";
      const originalHTML = cell.innerHTML;
      const input = document.createElement("input");
      input.type = "number"; input.min = "0"; input.step = "0.01";
      input.value = value === "" && cell.dataset.inlineTurnoverField === "manual_own_commission" ? "" : value || "0";
      input.className = "inline-number-input";
      cell.textContent = ""; cell.appendChild(input); input.focus(); input.select();
      let finished = false;
      const finish = async commit => {
        if (finished) return;
        finished = true;
        if (!commit) { cell.innerHTML = originalHTML; delete cell.dataset.inlineEditing; return; }
        input.disabled = true;
        try {
          await save(input.value);
          if (afterSave) await afterSave();
          else cell.innerHTML = originalHTML;
          delete cell.dataset.inlineEditing;
          showMessage("已保存并重算当前档口");
        } catch (error) {
          cell.innerHTML = originalHTML;
          delete cell.dataset.inlineEditing;
          showMessage(error.message);
        }
      };
      input.addEventListener("keydown", event => { if (event.key === "Enter") finish(true); if (event.key === "Escape") finish(false); });
      input.addEventListener("blur", () => finish(true));
    };
    document.querySelector("[data-stall-add]")?.addEventListener("click", () => openStallDialog());
    root?.addEventListener("keydown", event => {
      const cell = event.target.closest("[data-inline-turnover-field], [data-inline-master-field]");
      if (!cell || (event.key !== "Enter" && event.key !== " ")) return;
      event.preventDefault();
      cell.click();
    });
    root?.addEventListener("click", async event => {
      const row = event.target.closest("[data-stall-row]"); if (!row) return;
      const masterCell = event.target.closest("[data-inline-master-field]");
      if (masterCell && pagePath === "/admin/stalls") {
        const field = masterCell.dataset.inlineMasterField;
        const values = {stall_no: row.dataset.stallNo, name: row.dataset.name, responsible_name: row.dataset.responsibleName, responsible_mobile: row.dataset.responsibleMobile, monthly_rent: row.dataset.monthlyRent, default_guaranteed_amount: row.dataset.defaultGuaranteedAmount, allocation_mode: row.dataset.allocationMode, profit_jia_rate: row.dataset.profitJiaRate, profit_yi_rate: row.dataset.profitYiRate, guaranteed_rate: row.dataset.guaranteedRate, base_amount_rate: row.dataset.baseAmountRate, base_amount_factor: row.dataset.baseAmountFactor, total_amount_rate: row.dataset.totalAmountRate, remark: row.dataset.remark};
        startInlineEdit(masterCell, values[field], async value => { values[field] = value; await fetchJSON(`/api/admin/stalls?id=${row.dataset.id}`, {method:"PUT", headers:{"Content-Type":"application/json"}, body:JSON.stringify(values)}); }, () => {
          row.dataset[field.replace(/_([a-z])/g, (_, letter) => letter.toUpperCase())] = values[field];
          masterCell.textContent = field.endsWith("rate") || field.endsWith("factor") ? `${values[field]}%` : `¥${values[field]}`;
        });
        return;
      }
      const turnoverCell = event.target.closest("[data-inline-turnover-field]");
      if (turnoverCell && pagePath === "/admin/stalls/monthly") {
        const field = turnoverCell.dataset.inlineTurnoverField;
        const button = row.querySelector("[data-turnover-edit]");
        const values = {stall_id: Number(row.dataset.id), month: month(), card_amount: button?.dataset.card || "0", huilaimi_amount: button?.dataset.huilaimi || "0", cost_amount: button?.dataset.cost || "0", guaranteed_amount: button?.dataset.guaranteed || "0", manual_own_commission: button?.dataset.manualOwn || "", school_amount: button?.dataset.card || "0", remark: ""};
        startInlineEdit(turnoverCell, values[field], async value => { values[field] = value; if (field === "card_amount") values.school_amount = value; await fetchJSON(`/api/admin/stall-turnovers${button?.dataset.turnoverId ? `?id=${button.dataset.turnoverId}` : ""}`, {method: button?.dataset.turnoverId ? "PUT" : "POST", headers:{"Content-Type":"application/json"}, body:JSON.stringify(values)}); }, async () => {
          const data = await fetchJSON(`/api/admin/stalls?month=${encodeURIComponent(month())}&q=${encodeURIComponent(row.dataset.stallNo || "")}`);
          const item = (data.stalls || []).find(value => String(value.id) === String(row.dataset.id));
          if (!item) throw new Error("没有找到当前档口");
          updateCalculatedRow(row, item);
        });
        return;
      }
      if (event.target.closest("[data-stall-edit]")) openStallDialog(row.dataset, pagePath === "/admin/stalls/monthly");
      const calculateButton = event.target.closest("[data-stall-calculate]");
      if (calculateButton) {
        calculateButton.disabled = true;
        try {
          const data = await fetchJSON(`/api/admin/stalls?month=${encodeURIComponent(month())}&q=${encodeURIComponent(row.dataset.stallNo || "")}`);
          const item = (data.stalls || []).find(value => String(value.id) === String(row.dataset.id));
          if (!item) throw new Error("没有找到当前档口");
          updateCalculatedRow(row, item);
          showMessage(`${row.dataset.name || "当前档口"}已重新计算`);
        } catch (error) { showMessage(error.message); }
        finally { calculateButton.disabled = false; }
      }
      if (event.target.closest("[data-turnover-edit]")) openTurnoverDialog(event.target.closest("[data-turnover-edit]").dataset);
      if (event.target.closest("[data-stall-ledger]")) openLedgerDialog(event.target.closest("[data-stall-ledger]").dataset);
      const deleteButton = event.target.closest("[data-stall-delete]"); if (deleteButton && await askConfirm({title:"删除档口", message:"删除后档口及其营业额不再出现在列表中，确定继续吗？"})) { try { await fetchJSON(`/api/admin/stalls?id=${row.dataset.id}`, {method:"DELETE"}); window.location.reload(); } catch (error) { showMessage(error.message); } }
    });
  }
  // 临时工不建立员工档案，直接生成工资台账支出。
  if (pagePath === "/admin/payroll" && !document.querySelector("[data-temporary-payroll]")) {
    const actions = document.querySelector(".admin-topbar .header-actions");
    if (actions) { const button = document.createElement("button"); button.className = "ghost-button"; button.type = "button"; button.dataset.temporaryPayroll = "true"; button.textContent = "临时工发放"; actions.insertBefore(button, actions.firstElementChild); }
  }
  document.querySelector("[data-temporary-payroll]")?.addEventListener("click", async () => {
    try {
      await loadAdminOptions();
      const overlay = document.createElement("div");
      overlay.className = "record-dialog";
      const accounts = (adminOptions.accounts || []).map(item => `<option value="${escapeHTML(item.id)}">${escapeHTML(item.display_name || item.name)}</option>`).join("");
      overlay.innerHTML = `<div class="record-dialog-card"><div class="record-dialog-head"><div><small>临时工工资</small><h2>发放临时工</h2></div><button type="button" data-close-temp>×</button></div><form class="record-dialog-form"><div class="record-form-grid"><label>姓名 *<input name="name" required maxlength="64" placeholder="例如：临时帮工"></label><label>金额 *<input name="amount" required type="number" min="0.01" step="0.01"></label><label>付款账户 *<select name="account_id" required><option value="">请选择</option>${accounts}</select></label><label>发放日期 *<input name="occurred_on" required type="date" value="${today()}"></label><label class="span-2">备注（选填）<textarea name="remark" maxlength="255" placeholder="例如：10月临时工工资"></textarea></label></div><div class="record-dialog-error" role="alert"></div><div class="form-actions"><button type="button" class="ghost-button" data-close-temp>取消</button><button type="submit" class="primary-button">确认发放并入账</button></div></form></div>`;
      document.body.appendChild(overlay);
      const close = () => overlay.remove();
      overlay.querySelectorAll("[data-close-temp]").forEach(button => button.addEventListener("click", close));
      overlay.addEventListener("click", event => { if (event.target === overlay) close(); });
      overlay.querySelector("form").addEventListener("submit", async event => {
        event.preventDefault();
        const form = event.currentTarget;
        const submit = form.querySelector("button[type=submit]");
        submit.disabled = true;
        try {
          const values = Object.fromEntries(new FormData(form).entries());
          values.amount = Number(values.amount); values.account_id = Number(values.account_id);
          await fetchJSON("/api/admin/payroll/temporary-pay", {method:"POST", headers:{"Content-Type":"application/json"}, body:JSON.stringify(values)});
          close(); showMessage("临时工工资已发放并写入台账");
        } catch (error) { form.querySelector(".record-dialog-error").textContent = error.message; submit.disabled = false; }
      });
    } catch (error) { showMessage(error.message); }
  });
  if (pagePath === "/admin/organization") {
    document.querySelectorAll(".organization-grid .table-card").forEach(card => {
      const table = card.querySelector("table");
      const toolbar = document.createElement("div");
      toolbar.className = "table-toolbar compact-toolbar";
      card.insertBefore(toolbar, card.firstElementChild);
      setupClientTextFilter(table, toolbar, "搜索名称、编号或负责人");
    });
  }
  if (pagePath === "/admin/audit-logs") {
    const toolbar = document.querySelector(".table-card .table-toolbar");
    const table = document.querySelector(".table-card .table-wrap table");
    const date = toolbar?.querySelector('input[type="date"]');
    const module = toolbar?.querySelector("select");
    const search = toolbar?.querySelector(".search-box input");
    module?.options.forEach((option, index) => { option.value = index === 0 ? "" : option.textContent.trim(); });
    const apply = () => {
      const dateValue = date?.value || "";
      const moduleValue = module?.value || "";
      const keyword = (search?.value || "").trim().toLowerCase();
      table?.querySelectorAll("tbody tr").forEach(row => {
        if (row.querySelector(".empty-cell")) return;
        const text = row.textContent.toLowerCase();
        row.hidden = (dateValue && !text.includes(dateValue)) || (moduleValue && !text.includes(moduleValue)) || (keyword && !text.includes(keyword));
      });
    };
    date?.addEventListener("change", apply); module?.addEventListener("change", apply); search?.addEventListener("input", apply);
  }
  if (pagePath === "/admin/ledger" && pageTable) {
    const toolbar = document.querySelector(".table-card .table-toolbar");
    const search = toolbar.querySelector(".search-box input");
    const range = addNumberRange(toolbar, "金额最低", "金额最高");
    const apply = () => {
      const keyword = (search?.value || "").trim().toLowerCase();
      const min = Number(range.min.value || "-Infinity");
      const max = Number(range.max.value || "Infinity");
      pageTable.querySelectorAll("tbody tr").forEach(row => {
        if (row.querySelector(".empty-cell")) return;
        const text = row.textContent.toLowerCase();
        const income = Number((row.cells[7]?.textContent || "").replace(/[^0-9.-]/g, ""));
        const expense = Number((row.cells[8]?.textContent || "").replace(/[^0-9.-]/g, ""));
        const amount = Number.isNaN(income) ? expense : income;
        row.hidden = (keyword && !text.includes(keyword)) || (!Number.isNaN(amount) && (amount < min || amount > max));
      });
    };
    search?.addEventListener("input", apply); range.min.addEventListener("input", apply); range.max.addEventListener("input", apply);
  }
  if (pagePath === "/admin/work-logs") {
    const root = document.querySelector("[data-work-logs-page]");
    const form = root?.querySelector("[data-work-log-form]");
    const workLogSelectAll = root?.querySelector("[data-work-log-select-all]");
    const workLogDeleteSelected = root?.querySelector("[data-work-log-delete-selected]");
    const selectedWorkLogs = () => Array.from(root?.querySelectorAll("[data-work-log-select]:checked") || []);
    const syncWorkLogSelections = () => {
      const all = Array.from(root?.querySelectorAll("[data-work-log-select]") || []);
      const selected = selectedWorkLogs();
      if (workLogSelectAll) {
        workLogSelectAll.checked = all.length > 0 && selected.length === all.length;
        workLogSelectAll.indeterminate = selected.length > 0 && selected.length < all.length;
      }
      if (workLogDeleteSelected) {
        workLogDeleteSelected.disabled = selected.length === 0;
        workLogDeleteSelected.textContent = selected.length ? `删除所选（${selected.length}条）` : "删除所选";
      }
    };
    workLogSelectAll?.addEventListener("change", () => {
      root.querySelectorAll("[data-work-log-select]").forEach(input => { input.checked = workLogSelectAll.checked; });
      syncWorkLogSelections();
    });
    root?.querySelectorAll("[data-work-log-select]").forEach(input => input.addEventListener("change", syncWorkLogSelections));
    workLogDeleteSelected?.addEventListener("click", async () => {
      const ids = selectedWorkLogs().map(input => Number(input.value));
      if (!ids.length || !window.confirm(`删除选中的 ${ids.length} 条工时记录吗？删除后将不再计入当月工资。`)) return;
      try {
        await fetchJSON("/api/admin/work-logs", {method:"DELETE", headers:{"Content-Type":"application/json"}, body:JSON.stringify({ids})});
        window.location.reload();
      } catch (error) { showMessage(error.message); }
    });
    syncWorkLogSelections();
    let workLogSort = {key: "", direction: 1};
    root?.querySelectorAll("[data-work-log-sort]").forEach(button => button.addEventListener("click", () => {
      const key = button.dataset.workLogSort;
      workLogSort = {key, direction: workLogSort.key === key ? -workLogSort.direction : 1};
      const body = root.querySelector("tbody");
      const rows = Array.from(body?.querySelectorAll("[data-work-log-row]") || []);
      const value = row => key === "hours"
        ? Number(row.querySelector('[data-work-log-field="hours"]')?.value || 0)
        : Number((row.querySelector("[data-work-log-amount]")?.textContent || "").replace(/[^0-9.-]/g, ""));
      rows.sort((left, right) => (value(left) - value(right)) * workLogSort.direction).forEach(row => body.appendChild(row));
      root.querySelectorAll("[data-work-log-sort]").forEach(item => item.classList.toggle("is-active", item === button));
    }));
    setupDateMulti(root);
    const refreshWorkLogRate = () => {
      const option = form?.querySelector('select[name="employee_id"] option:checked');
      const rate = Number(option?.dataset.payRate || 0);
      const payBasis = option?.dataset.payBasis || "";
      const hint = form?.querySelector("[data-work-log-rate-hint]");
      if (!hint) return;
      if (rate <= 0) { hint.textContent = "选择临时工后显示对应的计算方式"; return; }
      if (payBasis === "daily") hint.textContent = `日薪 ¥${rate.toFixed(2)}；全天 ¥${rate.toFixed(2)}，半天 ¥${(rate / 2).toFixed(2)}`;
      else if (payBasis === "half_daily") hint.textContent = `半天工资 ¥${rate.toFixed(2)}；半天 ¥${rate.toFixed(2)}，全天 ¥${(rate * 2).toFixed(2)}`;
      else hint.textContent = `时薪 ¥${rate.toFixed(2)}；全天约 ¥${(rate * 8).toFixed(2)}，半天约 ¥${(rate * 4).toFixed(2)}`;
    };
    form?.querySelector('select[name="employee_id"]')?.addEventListener("change", refreshWorkLogRate);
    form?.querySelectorAll("[data-work-log-hours]").forEach(button => button.addEventListener("click", () => {
      const hours = form.querySelector('input[name="hours"]');
      hours.value = button.dataset.workLogHours;
      hours.dispatchEvent(new Event("input", {bubbles: true}));
    }));
    form?.addEventListener("submit", async event => {
      event.preventDefault();
      const submit = form.querySelector("button[type=submit]");
      submit.disabled = true;
      try {
        const values = Object.fromEntries(new FormData(form).entries());
        values.work_dates = String(values.work_dates || "").split(",").map(value => value.trim()).filter(Boolean);
        if (!values.work_dates.length) throw new Error("请至少选择一个工作日期");
        await fetchJSON("/api/admin/work-logs", {method: "POST", headers: {"Content-Type": "application/json"}, body: JSON.stringify(values)});
        window.location.reload();
      }
      catch (error) { form.querySelector(".record-dialog-error").textContent = error.message; submit.disabled = false; }
    });
    const workLogPayload = row => ({
      employee_id: Number(row.dataset.employeeId),
      work_date: row.querySelector('[data-work-log-field="work_date"]').value,
      hours: row.querySelector('[data-work-log-field="hours"]').value,
      remark: row.querySelector('[data-work-log-field="remark"]').value
    });
    const saveWorkLogRow = async row => {
      await fetchJSON(`/api/admin/work-logs?id=${row.dataset.id}`, {method: "PUT", headers: {"Content-Type": "application/json"}, body: JSON.stringify(workLogPayload(row))});
      row.dataset.workLogDirty = "false";
    };
    const flushWorkLogRow = row => {
      clearTimeout(row._workLogSaveTimer);
      row._workLogSaveTimer = null;
      const previous = row._workLogPending || Promise.resolve();
      const task = previous.catch(() => {}).then(() => saveWorkLogRow(row));
      row._workLogPending = task;
      return task;
    };
    const scheduleWorkLogSave = row => {
      clearTimeout(row._workLogSaveTimer);
      row._workLogSaveTimer = setTimeout(() => {
        row._workLogSaveTimer = null;
        flushWorkLogRow(row).catch(error => showMessage(`工时自动保存失败：${error.message}`));
      }, 450);
    };
    root?.addEventListener("input", event => {
      const row = event.target.closest("[data-work-log-row]");
      if (!row || !event.target.dataset.workLogField) return;
      if (event.target.dataset.workLogField === "hours") {
        const rate = Number((row.querySelector("[data-work-log-rate]")?.textContent || "").replace(/[^0-9.-]/g, ""));
        const hours = Number(event.target.value || 0);
        const amount = row.querySelector("[data-work-log-amount]");
        const divisor = row.dataset.payBasis === "daily" ? 8 : row.dataset.payBasis === "half_daily" ? 4 : 1;
        if (amount) amount.textContent = `¥${(rate * hours / divisor).toFixed(2)}`;
      }
      row.dataset.workLogDirty = "true";
      scheduleWorkLogSave(row);
    });
    root?.addEventListener("change", event => {
      const row = event.target.closest("[data-work-log-row]");
      if (!row || event.target.dataset.workLogField !== "work_date") return;
      row.dataset.workLogDirty = "true";
      scheduleWorkLogSave(row);
    });
    root?.addEventListener("click", async event => {
      const row = event.target.closest("[data-work-log-row]");
      if (!row) return;
      if (event.target.closest("[data-work-log-save]")) {
        try { await flushWorkLogRow(row); showMessage("工时已保存"); }
        catch (error) { showMessage(error.message); }
      }
      if (event.target.closest("[data-work-log-delete]")) {
        if (!window.confirm("删除后该天工时不再计入工资，确定继续吗？")) return;
        try { await fetchJSON(`/api/admin/work-logs?id=${row.dataset.id}`, {method: "DELETE"}); row.remove(); showMessage("工时已删除"); }
        catch (error) { showMessage(error.message); }
      }
    });
  }
  if (pagePath === "/admin/menu") {
    const root = document.querySelector("[data-menu-page]");
    const openMenuDialog = (data = {}) => {
      const edit = Boolean(data.id);
      const overlay = document.createElement("div");
      overlay.className = "record-dialog";
      overlay.innerHTML = `<div class="record-dialog-card"><div class="record-dialog-head"><div><small>菜单管理</small><h2>${edit ? "编辑菜品" : "新增菜品"}</h2></div><button type="button" data-close-dialog>×</button></div><form class="record-dialog-form"><div class="record-form-grid"><label>食堂<input name="restaurant" value="${escapeHTML(data.restaurant || "知行二餐厅")}"></label><label>窗口名称<input name="window_name" required value="${escapeHTML(data.windowName || "大餐")}"></label><label>菜品名称 *<input name="name" required value="${escapeHTML(data.name || "")}"></label><label>规格<input name="specification" value="${escapeHTML(data.specification || "")}"></label><label>价格（元）*<input name="price" required type="number" min="0" step="0.01" value="${escapeHTML(data.price || "0")}"></label><label>价格显示<input name="price_text" value="${escapeHTML(data.priceText || "")}" placeholder="免费等文字"></label><label>大餐价格区间<input name="price_range" value="${escapeHTML(data.priceRange || "")}"></label><label>价格类别<input name="price_category" value="${escapeHTML(data.priceCategory || "")}"></label><label>菜品类别<input name="item_category" value="${escapeHTML(data.itemCategory || "")}"></label><label class="span-2">备注<textarea name="remark">${escapeHTML(data.remark || "")}</textarea></label></div><div class="record-dialog-error" role="alert"></div><div class="form-actions"><button type="button" class="ghost-button" data-close-dialog>取消</button><button type="submit" class="primary-button">保存菜品</button></div></form></div>`;
      document.body.appendChild(overlay);
      const close = () => overlay.remove();
      overlay.querySelectorAll("[data-close-dialog]").forEach(button => button.addEventListener("click", close));
      overlay.addEventListener("click", event => { if (event.target === overlay) close(); });
      overlay.querySelector("form").addEventListener("submit", async event => {
        event.preventDefault();
        const form = event.currentTarget; const submit = form.querySelector("button[type=submit]"); submit.disabled = true;
        try { await fetchJSON(`/api/admin/menu-items${edit ? `?id=${data.id}` : ""}`, {method: edit ? "PUT" : "POST", headers: {"Content-Type": "application/json"}, body: JSON.stringify(Object.fromEntries(new FormData(form).entries()))}); window.location.reload(); }
        catch (error) { form.querySelector(".record-dialog-error").textContent = error.message; submit.disabled = false; }
      });
    };
    const rowValues = row => ({restaurant: row.dataset.restaurant, window_name: row.dataset.windowName, name: row.dataset.name, specification: row.dataset.specification, price: row.dataset.price, price_text: row.dataset.priceText, sales_period: row.dataset.salesPeriod, price_range: row.dataset.priceRange, price_category: row.dataset.priceCategory, item_category: row.dataset.itemCategory, remark: row.dataset.remark});
    const startMenuInlineEdit = (cell, field, row) => {
      if (cell.dataset.inlineEditing === "true") return;
      cell.dataset.inlineEditing = "true";
      const dataKey = ({price_category:"priceCategory", item_category:"itemCategory"})[field] || field;
      const input = document.createElement("input"); input.type = field === "price" ? "number" : "text"; input.step = "0.01"; input.min = "0"; input.value = row.dataset[dataKey] || ""; input.className = "inline-number-input";
      cell.textContent = ""; cell.appendChild(input); input.focus(); input.select();
      let finished = false;
      const finish = async commit => { if (finished) return; finished = true; if (!commit) { window.location.reload(); return; } input.disabled = true; const values = rowValues(row); values[field] = input.value; try { await fetchJSON(`/api/admin/menu-items?id=${row.dataset.id}`, {method:"PUT", headers:{"Content-Type":"application/json"}, body:JSON.stringify(values)}); window.location.reload(); } catch (error) { showMessage(error.message); window.location.reload(); } };
      input.addEventListener("keydown", event => { if (event.key === "Enter") finish(true); if (event.key === "Escape") finish(false); }); input.addEventListener("blur", () => finish(true));
    };
    root?.querySelector("[data-menu-add]")?.addEventListener("click", () => openMenuDialog());
    root?.addEventListener("click", async event => {
      const row = event.target.closest("[data-menu-row]"); if (!row) return;
      const cell = event.target.closest("[data-inline-menu-field]");
      if (cell) { startMenuInlineEdit(cell, cell.dataset.inlineMenuField, row); return; }
      if (event.target.closest("[data-menu-edit]")) { openMenuDialog(row.dataset); return; }
      const deleteButton = event.target.closest("[data-menu-delete]");
      if (deleteButton && await askConfirm({title:"删除菜品", message:"删除后该菜品不再出现在菜单中，确定继续吗？"})) { try { await fetchJSON(`/api/admin/menu-items?id=${row.dataset.id}`, {method:"DELETE"}); window.location.reload(); } catch (error) { showMessage(error.message); } }
    });
  }
});
