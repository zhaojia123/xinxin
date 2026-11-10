(() => {
  const money = value => `¥${Number(value || 0).toFixed(2)}`;
  const numberValue = value => {
    const parsed = Number.parseFloat(String(value ?? "").replace(/[¥,]/g, ""));
    return Number.isFinite(parsed) ? parsed : 0;
  };
  const request = async (url, options = {}) => {
    const response = await fetch(url, options);
    const body = await response.json().catch(() => ({}));
    if (!response.ok) throw new Error(body.error || "操作失败，请稍后重试");
    return body;
  };

  const renderTrigger = (trigger, amount) => {
    trigger.dataset.amount = amount.toFixed(2);
    trigger.innerHTML = `${money(amount)}<small>点击修改</small>`;
  };

  const updateRowTotal = (trigger, amount) => {
    const total = numberValue(trigger.dataset.baseAmount) + numberValue(trigger.dataset.riceOilAmount) + amount;
    const totalNode = trigger.closest("tr")?.querySelector("[data-stall-cost-total]");
    if (totalNode) totalNode.textContent = money(total);
  };

  const restore = (trigger, amount) => {
    trigger.disabled = false;
    trigger.classList.remove("is-saving");
    renderTrigger(trigger, amount);
  };

  const beginInlineEdit = trigger => {
    if (trigger.dataset.editing === "true" || trigger.classList.contains("is-saving")) return;
    const previousAmount = numberValue(trigger.dataset.amount);
    trigger.dataset.editing = "true";
    trigger.innerHTML = "";
    const input = document.createElement("input");
    input.className = "inline-number-input stall-adjustment-input";
    input.type = "number";
    input.min = "0";
    input.step = "0.01";
    input.value = previousAmount.toFixed(2);
    input.setAttribute("aria-label", "特殊调整金额");
    trigger.appendChild(input);
    input.focus();
    input.select();

    let finished = false;
    const finish = async shouldSave => {
      if (finished) return;
      finished = true;
      if (shouldSave && !input.validity.valid) {
        restore(trigger, previousAmount);
        window.alert("请输入正确的特殊调整金额");
        return;
      }
      const nextAmount = numberValue(input.value);
      delete trigger.dataset.editing;
      if (!shouldSave || nextAmount === previousAmount) {
        restore(trigger, previousAmount);
        return;
      }
      if (nextAmount < 0) {
        restore(trigger, previousAmount);
        window.alert("特殊调整金额不能小于0");
        return;
      }
      trigger.disabled = true;
      trigger.classList.add("is-saving");
      try {
        await request("/api/admin/stall-costs/special-adjustment", {
          method: "PUT",
          headers: {"Content-Type": "application/json"},
          body: JSON.stringify({
            stall_id: Number(trigger.dataset.stallId),
            month: trigger.dataset.month,
            amount: nextAmount.toFixed(2),
          }),
        });
        renderTrigger(trigger, nextAmount);
        updateRowTotal(trigger, nextAmount);
      } catch (error) {
        restore(trigger, previousAmount);
        window.alert(error.message);
      } finally {
        trigger.disabled = false;
        trigger.classList.remove("is-saving");
      }
    };
    input.addEventListener("keydown", event => {
      if (event.key === "Enter") {
        event.preventDefault();
        finish(true);
      }
      if (event.key === "Escape") {
        event.preventDefault();
        finish(false);
      }
    });
    input.addEventListener("blur", () => finish(true));
  };

  document.addEventListener("click", event => {
    const trigger = event.target.closest("[data-stall-adjustment]");
    if (trigger && event.target.tagName !== "INPUT") beginInlineEdit(trigger);
  });
  document.addEventListener("keydown", event => {
    const trigger = event.target.closest("[data-stall-adjustment]");
    if (!trigger || event.target.tagName === "INPUT" || (event.key !== "Enter" && event.key !== " ")) return;
    event.preventDefault();
    beginInlineEdit(trigger);
  });
})();
