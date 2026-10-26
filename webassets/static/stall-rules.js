(() => {
  const applyFixedProfitShare = root => {
    root.querySelectorAll(".record-dialog").forEach(dialog => {
      const jia = dialog.querySelector('input[name="profit_jia_rate"]');
      const yi = dialog.querySelector('input[name="profit_yi_rate"]');
      if (!jia || !yi || dialog.querySelector("[data-fixed-profit-share]")) return;
      const jiaField = jia.closest("label");
      const yiField = yi.closest("label");
      if (!jiaField || !yiField) return;
      const fixed = document.createElement("div");
      fixed.className = "fixed-rule-field";
      fixed.dataset.fixedProfitShare = "true";
      fixed.innerHTML = "<span>甲乙净额分成（固定）</span><strong>甲 90% · 乙 10%</strong>";
      jiaField.replaceWith(fixed);
      yiField.remove();
    });
  };

  // 档口弹窗由主脚本动态生成，监听新增节点后移除不可编辑的固定比例输入框。
  const observer = new MutationObserver(() => applyFixedProfitShare(document));
  observer.observe(document.documentElement, {childList: true, subtree: true});
  applyFixedProfitShare(document);
})();
