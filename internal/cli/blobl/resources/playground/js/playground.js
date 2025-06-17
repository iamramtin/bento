class BloblangPlayground {
  constructor() {
    this.state = {
      wasmAvailable: false,
      executionMode: "wasm", // "wasm" or "server"
      isExecuting: false,
      executionTimeout: null,
      inputFormatMode: "format", // "format" or "minify"
      outputFormatMode: "minify", // "format" or "minify"
    };

    this.elements = {
      loadingOverlay: document.getElementById("loadingOverlay"),
      outputArea: document.getElementById("output"),
      inputPanel: document.getElementById("inputPanel"),
      mappingPanel: document.getElementById("mappingPanel"),
      inputFileInput: document.getElementById("inputFileInput"),
      mappingFileInput: document.getElementById("mappingFileInput"),
      toggleFormatInputBtn: document.getElementById("toggleFormatInputBtn"),
      toggleFormatOutputBtn: document.getElementById("toggleFormatOutputBtn"),
    };
    this.init();
  }

  async init() {
    try {
      const params = new URLSearchParams(window.location.search);
      const forceServerMode = params.get("mode") === "server";

      this.editor = new EditorManager();
      this.ui = new UIManager();

      if (!forceServerMode) {
        this.wasm = new WasmManager();
        if (this.wasm.go) {
          await this.wasm.load();
          this.state.wasmAvailable = this.wasm.available;
          this.state.executionMode = this.state.wasmAvailable
            ? "wasm"
            : "server";
        } else {
          this.state.executionMode = "server";
          this.state.wasmAvailable = false;
        }
      } else {
        this.state.executionMode = "server";
        this.state.wasmAvailable = false;
      }

      // Setup ACE and fallback editors
      this.editor.init({
        onInputChange: () => this.onEditorChange("input"),
        onMappingChange: () => this.onEditorChange("mapping"),
      });

      // Setup UI
      this.ui.init();
      this.bindEvents();

      this.updateLinters();
      this.execute();
      this.hideLoading();
    } catch (error) {
      this.handleInitError(error);
    }
  }

  async execute() {
    if (this.state.isExecuting) return;
    this.state.isExecuting = true;

    try {
      const input = this.editor.getInput();
      const mapping = this.editor.getMapping();

      let response;
      switch (this.state.executionMode) {
        case "wasm":
          try {
            response = this.wasm.execute(input, mapping);
          } catch (error) {
            this.handleError(
              "WASM Error",
              "An error occurred while executing with WASM",
              error.message,
              null,
              "outputStatus",
              "WASM Error"
            );
            return;
          }
          break;
        case "server":
          try {
            response = await this.executeHttp(input, mapping);
          } catch (error) {
            this.handleError(
              "Server Error",
              "An error occurred while communicating with the server",
              error.message,
              null,
              "outputStatus",
              "Server Error"
            );
            return;
          }
          break;
        default:
          throw new Error("Unknown execution mode");
      }

      this.handleExecutionResult(response);
    } catch (error) {
      this.handleError(
        "Connection Error",
        "Ensure Bloblang server is running and try again",
        error.message,
        null,
        "outputStatus",
        "Connection Error"
      );
    } finally {
      this.state.isExecuting = false;
    }
  }

  async executeHttp(input, mapping) {
    const request = new Request("/execute", {
      method: "POST",
      headers: {
        "Content-Type": "application/json",
      },
      body: JSON.stringify({
        input: input,
        mapping: mapping,
      }),
    });

    const response = await fetch(request);

    if (response.status === 200) {
      return await response.json();
    } else {
      throw new Error("Server error: " + response.status);
    }
  }

  debouncedExecute(type) {
    if (this.state.executionTimeout) {
      clearTimeout(this.state.executionTimeout);
    }

    if (type === "input") {
      this.ui.updateStatus("inputStatus", "executing", "Processing...");
    } else if (type === "mapping") {
      this.ui.updateStatus("mappingStatus", "executing", "Processing...");
    }

    this.state.executionTimeout = setTimeout(() => {
      this.execute();
    }, 300);
  }

  bindEvents() {
    // Button clicks
    document.addEventListener("click", (e) => {
      const action = e.target.dataset.action;
      if (action) this.handleAction(action, e.target);
    });

    // File inputs
    this.elements.inputFileInput.addEventListener("change", (e) =>
      this.handleFileLoad(e, "input")
    );
    this.elements.mappingFileInput.addEventListener("change", (e) =>
      this.handleFileLoad(e, "mapping")
    );
  }

  handleAction(action, element) {
    const actions = {
      "copy-input": () => copyToClipboard(this.editor.getInput()),
      "copy-mapping": () => copyToClipboard(this.editor.getMapping()),
      "copy-output": () =>
        copyToClipboard(this.elements.outputArea.textContent),
      "load-input": () => this.elements.inputFileInput.click(),
      "load-mapping": () => this.elements.mappingFileInput.click(),
      "save-output": () => saveOutput(),
      "format-mapping": () => formatMapping(),
      "toggle-format-input": () => this.toggleFormatInput(),
      "toggle-format-output": () => this.toggleFormatOutput(),
    };

    if (actions[action]) {
      actions[action]();
    }
  }

  onEditorChange(type) {
    this.updateLinters();
    this.debouncedExecute(type);
  }

  handleExecutionResult(response) {
    // Reset error states
    this.elements.inputPanel.classList.remove("error");
    this.elements.mappingPanel.classList.remove("error");
    this.elements.outputArea.classList.remove(
      "error",
      "success",
      "json-formatted"
    );

    let mappingError = null;

    if (response.result && response.result.length > 0) {
      this.handleSuccess(response.result);
    } else if (response.mapping_error && response.mapping_error.length > 0) {
      this.handleError(
        "Input Error",
        "There was an error parsing your input JSON",
        response.mapping_error,
        "inputPanel",
        "inputStatus",
        "Input Error"
      );
    } else if (response.parse_error && response.parse_error.length > 0) {
      this.handleError(
        "Mapping Error",
        "There is an error in your Bloblang mapping",
        response.parse_error,
        "mappingPanel",
        "mappingStatus",
        "Mapping Error"
      );
    }

    this.updateLinters(mappingError);
  }

  handleSuccess(result) {
    this.elements.outputArea.classList.add("success");
    this.ui.updateStatus("outputStatus", "success", "Success");

    if (isValidJSON(result)) {
      const formatted = formatJSON(result);
      const highlighted = syntaxHighlightJSON(formatted);
      this.elements.outputArea.innerHTML = highlighted;
      this.elements.outputArea.classList.add("json-formatted");
    } else {
      this.elements.outputArea.textContent = result.trim();
    }
  }

  handleError(title, message, error, errorPanelClass, statusKey, statusLabel) {
    if (errorPanelClass && this.elements[errorPanelClass]) {
      this.elements[errorPanelClass].classList.add("error");
    }
    this.elements.outputArea.classList.add("error");

    this.elements.outputArea.innerHTML = createErrorMessage(
      title,
      message,
      error
    );

    this.ui.updateStatus(statusKey, "error", "Error");
    this.ui.updateStatus("outputStatus", "error", statusLabel);
  }

  // Format actions
  toggleFormatInput() {
    if (this.state.inputFormatMode === "format") {
      formatInput();
      this.state.inputFormatMode = "minify";
      this.elements.toggleFormatInputBtn.textContent = "Minify";
    } else {
      minifyInput();
      this.state.inputFormatMode = "format";
      this.elements.toggleFormatInputBtn.textContent = "Format";
    }
  }

  toggleFormatOutput() {
    if (this.state.outputFormatMode === "format") {
      formatOutput();
      this.state.outputFormatMode = "minify";
      this.elements.toggleFormatOutputBtn.textContent = "Minify";
    } else {
      minifyOutput();
      this.state.outputFormatMode = "format";
      this.elements.toggleFormatOutputBtn.textContent = "Format";
    }
  }

  // File operations
  handleFileLoad(event, type) {
    const file = event.target.files[0];
    if (!file) return;

    const reader = new FileReader();
    reader.onload = (e) => {
      const content = e.target.result;
      if (type === "input") {
        this.editor.setInput(content);
      } else {
        this.editor.setMapping(content);
      }
      this.ui.showNotification(`Loaded ${file.name}`, "success");
      this.execute();
    };
    reader.readAsText(file);
  }

  updateLinters(mappingError = null) {
    updateInputLinter(this.editor.getInput());
    updateMappingLinter(this.editor.getMapping(), mappingError);
    updateOutputLinter(this.elements.outputArea.textContent);
  }

  hideLoading() {
    this.elements.loadingOverlay.classList.add("hidden");
  }

  handleInitError(error) {
    console.error("Application error:", error);
    this.elements.loadingOverlay.innerHTML = `
      <div style="color: var(--bento-error); text-align: center;">
        <h3>Failed to Load Playground</h3>
      </div>
    `;
  }
}

// Initialize when DOM is ready
document.addEventListener("DOMContentLoaded", () => {
  window.playground = new BloblangPlayground();
});
