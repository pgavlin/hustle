// Example hustle format plugin in JavaScript, compiled with Javy.
//
// Parses lines of the form:
//   LEVEL: message key=value key=value ...
//
// Build:
//   javy build -J javy-stream-io -o plugin.wasm plugin.js

const LEVELS = new Set([
  "TRACE", "DEBUG", "INFO", "WARN", "WARNING", "ERROR", "FATAL", "PANIC",
]);

function parseLine(line) {
  const colonIdx = line.indexOf(":");
  if (colonIdx < 0 || colonIdx + 2 > line.length) {
    return { ok: false, error: "expected LEVEL: message" };
  }

  const level = line.substring(0, colonIdx);
  if (!LEVELS.has(level)) {
    return { ok: false, error: "unknown level" };
  }

  const rest = line.substring(colonIdx + 2);
  const tokens = rest.split(/\s+/).filter((t) => t.length > 0);

  let attrStart = tokens.length;
  for (let i = tokens.length - 1; i >= 0; i--) {
    if (tokens[i].includes("=")) {
      attrStart = i;
    } else {
      break;
    }
  }

  const msg = tokens.slice(0, attrStart).join(" ");
  const attrs = {};
  for (let i = attrStart; i < tokens.length; i++) {
    const eq = tokens[i].indexOf("=");
    if (eq > 0) {
      attrs[tokens[i].substring(0, eq)] = tokens[i].substring(eq + 1);
    }
  }

  const result = { ok: true, level, msg };
  if (Object.keys(attrs).length > 0) {
    result.attrs = attrs;
  }
  return result;
}

// Read all of stdin using Javy.IO.readSync.
function readStdin() {
  const chunks = [];
  const buf = new Uint8Array(4096);
  while (true) {
    const n = Javy.IO.readSync(0, buf);
    if (n <= 0) break;
    chunks.push(buf.slice(0, n));
  }
  let total = 0;
  for (const c of chunks) total += c.length;
  const result = new Uint8Array(total);
  let offset = 0;
  for (const c of chunks) {
    result.set(c, offset);
    offset += c.length;
  }
  // Decode bytes to string (ASCII-safe for log lines)
  let s = "";
  for (let i = 0; i < result.length; i++) {
    s += String.fromCharCode(result[i]);
  }
  return s;
}

function writeStdout(s) {
  const buf = new Uint8Array(s.length);
  for (let i = 0; i < s.length; i++) {
    buf[i] = s.charCodeAt(i);
  }
  Javy.IO.writeSync(1, buf);
}

const input = readStdin().trimEnd();
const result = parseLine(input);
writeStdout(JSON.stringify(result));
