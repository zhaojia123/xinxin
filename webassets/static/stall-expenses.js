document.addEventListener("DOMContentLoaded", () => {
  const root = document.querySelector("[data-stall-monthly-expenses]");
  if (!root) return;

  const form = root.querySelector("[data-stall-monthly-expense-form]");
  const errorBox = root.querySelector("[data-stall-monthly-expense-error]");
  const toast = document.querySelector(".demo-toast");
  let autoSaveTimer;
  let saving = false;
  let timer;
  const showMessage = (message) => {
    if (!toast) return;
    toast.textContent = message;
    toast.classList.add("show");
    clearTimeout(timer);
    timer = setTimeout(() => toast.classList.remove("show"), 2600);
  };
  const requestJSON = async (url, options = {}) => {
    const response = await fetch(url, options);
    const data = await response.json().catch(() => ({}));
    if (!response.ok) throw new Error(data.error || "操作失败，请稍后重试");
    return data;
  };
  const escapeHTML = (value) => String(value ?? "").replace(/[&<>'"]/g, (char) => ({"&":"&amp;","<":"&lt;",">":"&gt;","'":"&#39;",'"':"&quot;"})[char]);

  // 金额输入完成或离开输入框后直接保存，避免再额外点击保存按钮。
  form?.querySelectorAll('input[name="water_amount"], input[name="electricity_amount"]').forEach((input) => {
    const saveWhenReady = () => {
      clearTimeout(autoSaveTimer);
      if (input.value === "") return;
      autoSaveTimer = setTimeout(() => form.requestSubmit(), 800);
    };
    input.addEventListener("input", () => {
      saveWhenReady();
    });
    input.addEventListener("change", () => {
      clearTimeout(autoSaveTimer);
      form.requestSubmit();
    });
  });

  form?.addEventListener("submit", async (event) => {
    event.preventDefault();
    if (saving) return;
    saving = true;
    errorBox.textContent = "";
    const submitButton = form.querySelector('[type="submit"]');
    const values = Object.fromEntries(new FormData(form));
    submitButton.disabled = true;
    try {
      await requestJSON("/api/admin/stall-monthly-expenses", {method: "POST", headers: {"Content-Type": "application/json"}, body: JSON.stringify(values)});
      showMessage("当月经营支出已保存。");
      window.location.reload();
    } catch (error) {
      errorBox.textContent = error.message;
    } finally {
      submitButton.disabled = false;
      saving = false;
    }
  });

  root.querySelector("[data-post-monthly-expenses]")?.addEventListener("click", async (event) => {
    const button = event.currentTarget;
    errorBox.textContent = "";
    button.disabled = true;
    try {
      const overlay = document.createElement("div");
      overlay.className = "record-prompt";
      overlay.innerHTML = `<form class="record-prompt-card"><h3>确认 ${escapeHTML(root.dataset.month)} 月核算入账</h3><p>学校卡机、汇来米分别记为收入；水费、电费、康莱德卡机抽点、全部未入账档口成本和非自营档口未分配余额分别记为支出。流水会记入当前核算月份，日期可改为该月内任一天；无需填写对方账户。</p><label>入账日期<input name="occurred_on" type="date" required value="${escapeHTML(root.dataset.month)}-01"></label><div class="form-actions"><button class="ghost-button" type="button" data-close>取消</button><button class="primary-button" type="submit">确认入账</button></div><p class="record-inline-error" role="alert"></p></form>`;
      const close = () => overlay.remove();
      overlay.querySelector("[data-close]").addEventListener("click", close);
      overlay.addEventListener("click", (clickEvent) => { if (clickEvent.target === overlay) close(); });
      overlay.querySelector("form").addEventListener("submit", async (submitEvent) => {
        submitEvent.preventDefault();
        const submitButton = submitEvent.currentTarget.querySelector('[type="submit"]');
        const dialogError = submitEvent.currentTarget.querySelector(".record-inline-error");
        submitButton.disabled = true;
        try {
          const values = Object.fromEntries(new FormData(submitEvent.currentTarget));
          values.month = root.dataset.month;
          await requestJSON("/api/admin/stall-monthly-expenses/ledger", {method: "POST", headers: {"Content-Type": "application/json"}, body: JSON.stringify(values)});
          showMessage("月度营业收入和经营支出已计入台账。");
          window.location.reload();
        } catch (error) {
          dialogError.textContent = error.message;
          submitButton.disabled = false;
        }
      });
      document.body.append(overlay);
    } catch (error) {
      errorBox.textContent = error.message;
      button.disabled = false;
    }
  });

  root.querySelector("[data-revoke-monthly-expenses]")?.addEventListener("click", async (event) => {
    const button = event.currentTarget;
    if (!window.confirm(`确定撤销 ${root.dataset.month} 月核算入账吗？自动生成的收入、支出流水会作废保留审计记录，档口成本恢复为未入账，可修改后重新确认。`)) return;
    errorBox.textContent = "";
    button.disabled = true;
    try {
      await requestJSON("/api/admin/stall-monthly-expenses/ledger/revoke", {method: "POST", headers: {"Content-Type": "application/json"}, body: JSON.stringify({month: root.dataset.month})});
      showMessage("本月核算入账已撤销，可修改后重新确认。");
      window.location.reload();
    } catch (error) {
      errorBox.textContent = error.message;
      button.disabled = false;
    }
  });
});
