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

  const today = () => new Date().toLocaleDateString("sv-SE");
  const escapeHTML = (value) => String(value ?? "").replace(/[&<>'"]/g, (char) => ({"&":"&amp;","<":"&lt;",">":"&gt;","'":"&#39;",'"':"&quot;"})[char]);
  const recordEndpoints = {
    department: "/api/admin/departments", position: "/api/admin/positions", attendance: "/api/admin/attendance",
    change: "/api/admin/employment-changes", salary: "/api/admin/salary-adjustments", ledger: "/api/admin/ledger"
  };
  const choices = {
    direction: [{id:"income",name:"收入"},{id:"expense",name:"支出"}],
    category: [{id:"leave",name:"请假"},{id:"exception",name:"迟到或早退等异常"}],
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
      ["start_time","开始或发生时间","time"],["end_time","结束时间","time"],["duration_days","请假天数","number"],["duration_minutes","异常分钟数","number"],["reason","原因","textarea"]
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
      return `<label data-record-field="${name}"><span class="record-label-row"><span>${escapeHTML(label)}${required ? " *" : ""}</span>${create}</span><select name="${name}" ${required ? "required" : ""} ${readonly ? "disabled" : ""}><option value="">${empty}</option>${options.map(item => `<option value="${escapeHTML(item.id)}" data-department-id="${escapeHTML(item.department_id || "")}" data-direction="${escapeHTML(item.direction || "")}" ${String(item.id) === String(value) ? "selected" : ""}>${escapeHTML(item.name)}${item.direction ? ` · ${item.direction === "income" ? "收入" : "支出"}` : ""}</option>`).join("")}</select></label>`;
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
      if (daysField) daysField.hidden = !leave;
      if (minutesField) minutesField.hidden = leave;
      const days = daysField?.querySelector("input");
      const minutes = minutesField?.querySelector("input");
      if (days) days.required = leave;
      if (minutes) minutes.required = !leave;
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
  const openRecordDialog = async (kind, id = 0, defaults = {}, readonly = false) => {
    try {
      await loadAdminOptions();
      const schema = schemas[kind];
      let data = {...defaults};
      if (id) data = await fetchJSON(`${recordEndpoints[kind]}?id=${id}`);
      if (kind === "ledger" && data.payroll_batch_id) readonly = true;
      if (data.enabled === undefined) data.enabled = true;
      closeRecordDialog();
      const overlay = document.createElement("div");
      overlay.className = "record-dialog";
      overlay.innerHTML = `<div class="record-dialog-card"><div class="record-dialog-head"><div><small>${readonly ? "记录详情" : id ? "修改记录" : "新增记录"}</small><h2>${escapeHTML(schema.title)}</h2></div><button type="button" data-close-dialog>×</button></div><form class="record-dialog-form"><div class="record-form-grid">${schema.fields.map(field => fieldHTML(field,data,readonly || (kind === "salary" && field[0] === "before_salary"))).join("")}</div><div class="record-dialog-error" role="alert"></div><div class="form-actions"><button type="button" class="ghost-button" data-close-dialog>${readonly ? "关闭" : "取消"}</button>${readonly ? "" : `<button type="submit" class="primary-button">保存${escapeHTML(schema.title)}</button>`}</div></form></div>`;
      document.body.appendChild(overlay);
      overlay.querySelectorAll("[data-close-dialog]").forEach(button => button.addEventListener("click", closeRecordDialog));
      overlay.addEventListener("click", event => { if (event.target === overlay) closeRecordDialog(); });
      overlay.querySelectorAll("[data-create-option]").forEach(button => button.addEventListener("click", async event => {
        event.preventDefault();
        const kind = button.dataset.createOption;
        const label = kind === "accounts" ? "资金账户" : "收支分类";
        const name = await askValue({title:`新建${label}`,label:`${label}名称`});
        if (!name) return;
        button.disabled = true;
        try {
          const direction = kind === "categories" ? overlay.querySelector('[name="direction"]')?.value || "" : "";
          const created = await fetchJSON("/api/admin/options", {method:"POST",headers:{"Content-Type":"application/json"},body:JSON.stringify({kind,name,direction})});
          const select = button.closest("label").querySelector("select");
          const option = document.createElement("option");
          option.value = String(created.id);
          option.textContent = created.name;
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
        const values = Object.fromEntries(new FormData(form).entries());
        schema.fields.forEach(([name,,type]) => {
          if (type === "checkbox") values[name] = form.elements[name].checked;
          if (["departments","employees","positions","accounts","categories"].includes(type) || name.endsWith("_id") || name === "sort_order" || name === "duration_minutes") values[name] = Number(values[name] || 0);
        });
        button.disabled = true;
        try {
          await fetchJSON(recordEndpoints[kind] + (id ? `?id=${id}` : ""), {method:id ? "PUT" : "POST",headers:{"Content-Type":"application/json"},body:JSON.stringify(values)});
          window.location.href = window.location.pathname + window.location.search;
        } catch (error) { form.querySelector(".record-dialog-error").textContent = error.message; button.disabled = false; }
      });
    } catch (error) { showMessage(error.message); }
  };
  document.querySelectorAll("[data-record-form]").forEach(button => button.addEventListener("click", () => {
    const kind = button.dataset.recordForm;
    const defaults = {};
    if (kind === "attendance") Object.assign(defaults,{category:button.dataset.category || "leave",occurred_on:today(),duration_days:"1.00",duration_minutes:0});
    if (kind === "ledger") Object.assign(defaults,{direction:"expense",occurred_on:today(),amount:""});
    if (kind === "change") Object.assign(defaults,{change_type:"transfer",after_status:"active",effective_on:today()});
    if (kind === "salary") Object.assign(defaults,{after_salary:"",effective_on:today()});
    openRecordDialog(kind,Number(button.dataset.id || 0),defaults);
  }));
  document.querySelectorAll("[data-record-view]").forEach(button => button.addEventListener("click", () => openRecordDialog(button.dataset.recordView,Number(button.dataset.id),{},true)));
  document.querySelectorAll("[data-record-delete]").forEach(button => button.addEventListener("click", async () => {
    const kind = button.dataset.recordDelete;
    const endpoint = kind === "employee" ? "/api/admin/employees" : recordEndpoints[kind];
    const labels = {employee:"员工",department:"部门",position:"岗位",attendance:"请假或异常记录",ledger:"台账记录"};
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
  document.querySelector("[data-pay-payroll]")?.addEventListener("click", async event => {
    const button = event.currentTarget;
    try {
      await loadAdminOptions();
      if (!adminOptions.accounts.length) throw new Error("请先在台账新增一个资金账户");
      closeRecordDialog();
      const overlay=document.createElement("div");overlay.className="record-dialog";
      overlay.innerHTML=`<div class="record-dialog-card"><div class="record-dialog-head"><div><small>工资管理</small><h2>发放本月工资</h2></div><button type="button" data-close-dialog>×</button></div><form class="record-dialog-form"><div class="notice-card"><span>核</span><div><strong>发放后工资将锁定</strong><p>系统会同时在台账生成工资支出，请确认工资明细已经核对完成。</p></div></div><div class="record-form-grid"><label>付款账户 *<select name="account_id" required><option value="">请选择</option>${adminOptions.accounts.map(v=>`<option value="${v.id}">${escapeHTML(v.name)}</option>`).join("")}</select></label><label>发放日期 *<input name="occurred_on" type="date" value="${today()}" required></label></div><div class="record-dialog-error"></div><div class="form-actions"><button type="button" class="ghost-button" data-close-dialog>取消</button><button type="submit" class="primary-button">确认发放</button></div></form></div>`;
      document.body.appendChild(overlay);overlay.querySelectorAll("[data-close-dialog]").forEach(v=>v.addEventListener("click",closeRecordDialog));
      overlay.querySelector("form").addEventListener("submit",async submitEvent=>{submitEvent.preventDefault();const form=submitEvent.currentTarget;const submit=form.querySelector("button[type=submit]");submit.disabled=true;const values=Object.fromEntries(new FormData(form).entries());values.batch_id=Number(button.dataset.batchId);values.account_id=Number(values.account_id);try{await fetchJSON("/api/admin/payroll/pay",{method:"POST",headers:{"Content-Type":"application/json"},body:JSON.stringify(values)});window.location.reload()}catch(error){form.querySelector(".record-dialog-error").textContent=error.message;submit.disabled=false}});
    } catch(error){showMessage(error.message)}
  });

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
  const employeeStatus = document.querySelector("[data-employee-status]");
  const filterEmployees = () => {
    const keyword = employeeSearch?.value.trim().toLowerCase() || "";
    const departmentID = employeeDepartment?.value || "";
    const status = employeeStatus?.value || "";
    document.querySelectorAll("[data-employee-row]").forEach((row) => {
      const matchesKeyword = !keyword || row.dataset.search.toLowerCase().includes(keyword);
      const matchesDepartment = !departmentID || row.dataset.departmentId === departmentID;
      const matchesStatus = !status || row.dataset.status === status;
      row.hidden = !(matchesKeyword && matchesDepartment && matchesStatus);
    });
  };
  employeeSearch?.addEventListener("input", filterEmployees);
  employeeDepartment?.addEventListener("change", filterEmployees);
  employeeStatus?.addEventListener("change", filterEmployees);

  const numberValue = (row, name) => Number(row.querySelector(`[data-field="${name}"]`)?.value || 0);
  const calculateNet = (row) => numberValue(row, "base_salary") + numberValue(row, "bonus")
    - numberValue(row, "attendance_deduction") - numberValue(row, "other_deduction")
    - numberValue(row, "social_security") - numberValue(row, "tax");
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
});
