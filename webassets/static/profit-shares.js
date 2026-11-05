(() => {
  const root = document.querySelector("[data-profit-share-settings]");
  if (!root) return;
  const body = root.querySelector("tbody");
  const total = root.querySelector("[data-profit-share-total]");
  const error = root.querySelector("[data-profit-share-error]");
  const escapeHTML = value => String(value ?? "").replace(/[&<>'"]/g, char => ({"&":"&amp;","<":"&lt;",">":"&gt;","'":"&#39;",'"':"&quot;"}[char]));
  const rows = () => [...body.querySelectorAll("tr")];
  const refresh = () => {
    let sum = 0;
    rows().forEach((row, index) => {
      row.querySelector(".profit-share-seq").textContent = String(index + 1);
      sum += Number(row.querySelector("[data-profit-share-rate]").value || 0);
    });
    total.textContent = `合计 ${sum.toFixed(2)}%`;
    total.classList.toggle("profit-share-invalid", Math.abs(sum - 100) > 0.0001);
    return sum;
  };
  const addRow = (name = "", rate = "") => {
    const row = document.createElement("tr");
    row.innerHTML = `<td class="profit-share-seq"></td><td><input data-profit-share-name maxlength="64" value="${escapeHTML(name)}" required></td><td><input data-profit-share-rate type="number" min="0.01" max="100" step="0.01" value="${escapeHTML(rate)}" required></td><td><button class="danger-link" type="button" data-profit-share-remove>删除</button></td>`;
    body.appendChild(row); refresh(); row.querySelector("[data-profit-share-name]").focus();
  };
  root.querySelector("[data-profit-share-add]").addEventListener("click", () => addRow());
  root.addEventListener("input", event => { if (event.target.matches("[data-profit-share-rate]")) refresh(); });
  root.addEventListener("click", event => {
    if (!event.target.matches("[data-profit-share-remove]")) return;
    if (rows().length <= 2) { error.textContent = "至少保留两位分成参与方"; return; }
    event.target.closest("tr").remove(); error.textContent = ""; refresh();
  });
  root.querySelector("form").addEventListener("submit", async event => {
    event.preventDefault(); error.textContent = "";
    const sum = refresh();
    if (Math.abs(sum - 100) > 0.0001) { error.textContent = "所有参与方比例合计必须等于100%"; return; }
    const participants = rows().map(row => ({name: row.querySelector("[data-profit-share-name]").value.trim(), rate: row.querySelector("[data-profit-share-rate]").value}));
    if (participants.some(item => !item.name || Number(item.rate) <= 0)) { error.textContent = "请填写参与方姓名和大于0的比例"; return; }
    const submit = event.currentTarget.querySelector("button[type=submit]"); submit.disabled = true;
    try {
      const response = await fetch("/api/admin/profit-share-participants", {method:"PUT",headers:{"Content-Type":"application/json"},body:JSON.stringify({month: root.dataset.month, participants})});
      const result = await response.json().catch(() => ({}));
      if (!response.ok) throw new Error(result.error || "保存分成比例失败");
      window.location.reload();
    } catch (err) { error.textContent = err.message; submit.disabled = false; }
  });
  refresh();
})();
