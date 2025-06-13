class BloblangPlayground {
  constructor() {
    this.state = {
      wasmReady: false,
      isExecuting: false,
      executionTimeout: null,
      inputFormatMode: "format",
      outputFormatMode: "minify",
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
      // Initialize modules
      this.wasm = new WasmManager();
      this.editor = new EditorManager();
      this.ui = new UIManager();

      // Load WASM
      await this.wasm.load();
      this.state.wasmReady = true;

      // Setup ACE and fallback editors
      this.editor.init({
        onInputChange: () => this.onEditorChange("input"),
        onMappingChange: () => this.onEditorChange("mapping"),
      });

      // Setup UI
      this.ui.init();
      this.bindEvents();

      // Initial execution
      this.updateLinters();
      this.execute();
      this.hideLoading();
    } catch (error) {
      this.handleError(error);
    }
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
      "save-output": () => this.saveOutput(),
      "format-mapping": () => this.formatMapping(),
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

  execute() {
    if (!this.state.wasmReady || this.state.isExecuting) return;

    this.state.isExecuting = true;

    try {
      const input = this.editor.getInput();
      const mapping = this.editor.getMapping();
      const response = this.wasm.execute(input, mapping);

      this.handleExecutionResult(response);
    } catch (error) {
      this.handleExecutionError(error);
    } finally {
      this.state.isExecuting = false;
    }
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
      this.handleMappingError(response.mapping_error);
      mappingError = response.mapping_error;
    } else if (response.input_error && response.input_error.length > 0) {
      this.handleInputError(response.input_error);
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

  handleMappingError(error) {
    this.elements.mappingPanel.classList.add("error");
    this.elements.outputArea.classList.add("error");
    this.elements.outputArea.innerHTML = createErrorMessage(
      "Mapping Error",
      "There is an error in your Bloblang mapping:",
      error
    );
    this.ui.updateStatus("mappingStatus", "error", "Error");
    this.ui.updateStatus("outputStatus", "error", "Mapping Error");
    // updateMappingLinter(this.editor.getMapping(), error);
  }

  handleInputError(error) {
    this.elements.inputPanel.classList.add("error");
    this.elements.outputArea.classList.add("error");
    this.elements.outputArea.innerHTML = createErrorMessage(
      "Input Error",
      "There is an error parsing your input JSON:",
      error
    );
    this.ui.updateStatus("inputStatus", "error", "Error");
    this.ui.updateStatus("outputStatus", "error", "Input Error");
  }

  handleExecutionError(error) {
    console.error("Execution error:", error);
    this.elements.outputArea.classList.add("error");
    this.elements.outputArea.innerHTML = createErrorMessage(
      "Execution Error",
      error.message
    );
    this.ui.updateStatus("outputStatus", "error", "Execution Error");
  }

  // Format actions
  toggleFormatInput() {
    if (this.state.inputFormatMode === "format") {
      this.formatInput();
      this.state.inputFormatMode = "minify";
      this.elements.toggleFormatInputBtn.textContent = "Minify";
    } else {
      this.minifyInput();
      this.state.inputFormatMode = "format";
      this.elements.toggleFormatInputBtn.textContent = "Format";
    }
  }

  toggleFormatOutput() {
    if (this.state.outputFormatMode === "format") {
      this.formatOutput();
      this.state.outputFormatMode = "minify";
      this.elements.toggleFormatOutputBtn.textContent = "Minify";
    } else {
      this.minifyOutput();
      this.state.outputFormatMode = "format";
      this.elements.toggleFormatOutputBtn.textContent = "Format";
    }
  }

  formatInput() {
    const formatted = formatJSON(this.editor.getInput());
    this.editor.setInput(formatted);
  }

  minifyInput() {
    const minified = minifyJSON(this.editor.getInput());
    this.editor.setInput(minified);
  }

  formatOutput() {
    const output = this.elements.outputArea.textContent;
    if (isValidJSON(output)) {
      const formatted = formatJSON(output);
      const highlighted = syntaxHighlightJSON(formatted);
      this.elements.outputArea.innerHTML = highlighted;
      this.elements.outputArea.classList.add("json-formatted");
    } else {
      this.ui.showNotification("Output is not valid JSON", "error");
    }
  }

  minifyOutput() {
    const output = this.elements.outputArea.textContent;
    if (isValidJSON(output)) {
      const minified = minifyJSON(output);
      const highlighted = syntaxHighlightJSON(minified);
      this.elements.outputArea.innerHTML = highlighted;
      this.elements.outputArea.classList.add("json-formatted");
    } else {
      this.ui.showNotification("Output is not valid JSON", "error");
    }
  }

  formatMapping() {
    const formatted = formatBloblang(this.editor.getMapping());
    this.editor.setMapping(formatted);
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

  saveOutput() {
    const output = this.elements.outputArea.textContent;
    const timestamp = new Date()
      .toISOString()
      .slice(0, 19)
      .replace(/[:.]/g, "-");

    let formattedOutput = output;
    let extension = "txt";

    if (isValidJSON(output)) {
      formattedOutput = formatJSON(output);
      extension = "json";
    }

    downloadFile(formattedOutput, `bloblang-output-${timestamp}.${extension}`);
  }

  updateLinters(mappingError = null) {
    updateInputLinter(this.editor.getInput());
    updateMappingLinter(this.editor.getMapping(), mappingError);
    updateOutputLinter(this.elements.outputArea.textContent);
  }

  hideLoading() {
    this.elements.loadingOverlay.classList.add("hidden");
  }

  handleError(error) {
    console.error("Application error:", error);
    this.elements.loadingOverlay.innerHTML = `
      <div style="color: var(--bento-error); text-align: center;">
        <h3>Failed to Load Playground</h3>
        <p>${error.message}</p>
      </div>
    `;
  }
}

// Initialize when DOM is ready
document.addEventListener("DOMContentLoaded", () => {
  window.playground = new BloblangPlayground();
});
