class UIManager {
  constructor() {
    this.isResizing = false;
    this.container = document.getElementById("container");
    this.horizontalResizer = document.getElementById("horizontalResizer");
  }

  init() {
    this.setupResizer();
  }

  setupResizer() {
    let startY = 0;

    this.horizontalResizer.addEventListener("mousedown", (e) => {
      this.isResizing = true;
      startY = e.clientY;
      document.addEventListener("mousemove", doResize);
      document.addEventListener("mouseup", stopResize);
      document.body.style.cursor = "row-resize";
      document.body.style.userSelect = "none";
    });

    const doResize = (e) => {
      if (!this.isResizing) return;

      const deltaY = e.clientY - startY;
      const containerHeight = this.container.clientHeight - 24;
      const resizerHeight = 8;
      const minPanelHeight = 150;

      const totalHeight = containerHeight - resizerHeight;
      const currentTopHeight = totalHeight / 2 + deltaY;
      const currentBottomHeight = totalHeight - currentTopHeight;

      if (
        currentTopHeight >= minPanelHeight &&
        currentBottomHeight >= minPanelHeight
      ) {
        const topFr = currentTopHeight / totalHeight;
        const bottomFr = currentBottomHeight / totalHeight;
        this.container.style.gridTemplateRows = `${topFr}fr auto ${bottomFr}fr`;
      }
    };

    const stopResize = () => {
      this.isResizing = false;
      document.removeEventListener("mousemove", doResize);
      document.removeEventListener("mouseup", stopResize);
      document.body.style.cursor = "";
      document.body.style.userSelect = "";
    };
  }

  showNotification(message, type = "info") {
    const notification = document.createElement("div");
    notification.className = `notification ${type}`;
    notification.textContent = message;

    const colors = {
      success: { bg: "#E8F5E8", color: "#2E7D32", border: "#2E7D32" },
      error: { bg: "#FFEBEE", color: "#D32F2F", border: "#D32F2F" },
      info: { bg: "#FDE5D8", color: "#553630", border: "#EB8788" },
    };

    const style = colors[type] || colors.info;

    notification.style.cssText = `
      position: fixed;
      top: 20px;
      right: 20px;
      background: ${style.bg};
      color: ${style.color};
      padding: 12px 16px;
      border-radius: 6px;
      border: 1px solid ${style.border};
      font-family: 'IBM Plex Sans', sans-serif;
      font-size: 13px;
      font-weight: 500;
      z-index: 1000;
      opacity: 0;
      transform: translateX(100%);
      transition: all 0.3s ease;
    `;

    document.body.appendChild(notification);

    // Animate in
    setTimeout(() => {
      notification.style.opacity = "1";
      notification.style.transform = "translateX(0)";
    }, 10);

    // Animate out and remove
    setTimeout(() => {
      notification.style.opacity = "0";
      notification.style.transform = "translateX(100%)";
      setTimeout(() => {
        if (notification.parentNode) {
          document.body.removeChild(notification);
        }
      }, 300);
    }, 3000);
  }

  updateStatus(elementId, status, message) {
    const badge = document.getElementById(elementId);

    if (badge) {
      badge.textContent = message;
      badge.className = `status-badge show ${status}`;

      setTimeout(() => {
        if (!badge.classList.contains("error")) {
          badge.classList.remove("show");
        }
      }, 2000);
    }
  }
}
