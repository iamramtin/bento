// JSON Utilities
function isValidJSON(str) {
  try {
    JSON.parse(str);
    return true;
  } catch (e) {
    return false;
  }
}

function formatJSON(jsonString) {
  try {
    const parsed = JSON.parse(jsonString);
    return JSON.stringify(parsed, null, 2);
  } catch (e) {
    return jsonString;
  }
}

function minifyJSON(jsonString) {
  try {
    const parsed = JSON.parse(jsonString);
    return JSON.stringify(parsed);
  } catch (e) {
    return jsonString;
  }
}

// Bloblang Utilities
function formatBloblang(mappingString) {
  const lines = mappingString.split("\n");
  const formatted = lines
    .map((line) => {
      const trimmed = line.trim();
      if (trimmed === "" || trimmed.startsWith("#")) return trimmed;

      return trimmed
        .replace(/\s*=\s*/g, " = ")
        .replace(/\s*\.\s*/g, ".")
        .replace(/\s*\(\s*/g, "(")
        .replace(/\s*\)\s*/g, ")");
    })
    .join("\n");

  return formatted;
}

// Syntax Highlighting
function syntaxHighlightJSON(json) {
  if (typeof json !== "string") {
    json = JSON.stringify(json, null, 2);
  }

  json = json
    .replace(/&/g, "&amp;")
    .replace(/</g, "&lt;")
    .replace(/>/g, "&gt;");

  return json.replace(
    /("(\\u[a-zA-Z0-9]{4}|\\[^u]|[^\\"])*"(\s*:)?|\b(true|false|null)\b|-?\d+(?:\.\d*)?(?:[eE][+\-]?\d+)?)/g,
    function (match) {
      let cls = "json-number";
      if (/^"/.test(match)) {
        if (/:$/.test(match)) {
          cls = "json-key";
        } else {
          cls = "json-string";
        }
      } else if (/true|false/.test(match)) {
        cls = "json-boolean";
      } else if (/null/.test(match)) {
        cls = "json-null";
      }
      return `<span class="${cls}">${match}</span>`;
    }
  );
}

// Linting
function lintJSON(jsonString) {
  try {
    JSON.parse(jsonString);
    return { valid: true, message: "Valid JSON" };
  } catch (e) {
    return { valid: false, message: `Invalid JSON: ${e.message}` };
  }
}

function lintBloblang(mappingString) {
  const lines = mappingString.split("\n");
  const errors = [];

  for (let i = 0; i < lines.length; i++) {
    const line = lines[i].trim();
    if (line === "" || line.startsWith("#")) continue;

    if (line.includes("=") && !line.match(/^(root|let)\./)) {
      if (!line.match(/^\w+\s*=/) && !line.match(/^root\./)) {
        errors.push(
          `Line ${
            i + 1
          }: Assignments should start with 'root.' or variable name`
        );
      }
    }
  }

  if (errors.length > 0) {
    return { valid: false, message: errors[0] };
  }

  return { valid: true, message: "Valid Syntax" };
}

function updateInputLinter(input) {
  const lint = lintJSON(input);
  const indicator = document.getElementById("inputLint");
  if (indicator) {
    indicator.textContent = lint.message;
    indicator.className = `lint-indicator ${lint.valid ? "valid" : "invalid"}`;
  }
}

function updateMappingLinter(mapping, errorMessage = null) {
  const lint = errorMessage
    ? { valid: false, message: "Invalid Syntax" }
    : lintBloblang(mapping);

  const indicator = document.getElementById("mappingLint");
  if (indicator) {
    indicator.textContent = lint.message;
    indicator.className = `lint-indicator ${lint.valid ? "valid" : "invalid"}`;
  }
}

function updateOutputLinter(output) {
  const indicator = document.getElementById("outputLint");
  const formatBtn = document.getElementById("formatOutputBtn");
  const minifyBtn = document.getElementById("minifyOutputBtn");

  if (!indicator) return;

  if (output === "Ready to execute your first mapping...") {
    indicator.textContent = "Ready";
    indicator.className = "lint-indicator";
    if (formatBtn) formatBtn.disabled = true;
    if (minifyBtn) minifyBtn.disabled = true;
    return;
  }

  const isJSON = isValidJSON(output);
  if (isJSON) {
    indicator.textContent = "Valid JSON";
    indicator.className = "lint-indicator valid";
    if (formatBtn) formatBtn.disabled = false;
    if (minifyBtn) minifyBtn.disabled = false;
  } else {
    indicator.textContent = "Text Output";
    indicator.className = "lint-indicator warning";
    if (formatBtn) formatBtn.disabled = true;
    if (minifyBtn) minifyBtn.disabled = true;
  }
}

// File Operations
async function copyToClipboard(text, successMessage = "Copied to clipboard!") {
  try {
    await navigator.clipboard.writeText(text);
    if (window.playground && window.playground.ui) {
      window.playground.ui.showNotification(successMessage, "success");
    }
  } catch (err) {
    console.error("Failed to copy:", err);
    if (window.playground && window.playground.ui) {
      window.playground.ui.showNotification(
        "Failed to copy to clipboard",
        "error"
      );
    }
  }
}

function downloadFile(content, filename, contentType = "text/plain") {
  const blob = new Blob([content], { type: contentType });
  const url = window.URL.createObjectURL(blob);
  const link = document.createElement("a");
  link.href = url;
  link.download = filename;
  link.click();
  window.URL.revokeObjectURL(url);

  if (window.playground && window.playground.ui) {
    window.playground.ui.showNotification(`Downloaded ${filename}`, "success");
  }
}

// Error Message Creation
function createErrorMessage(title, message, details = null) {
  return `
    <div class="error-message">
      <div class="error-title">${title}</div>
      <div>${message}</div>
      ${details ? `<div class="error-details">${details}</div>` : ""}
    </div>
  `;
}
