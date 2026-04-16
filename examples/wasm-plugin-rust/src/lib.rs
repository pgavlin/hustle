// Example hustle format plugin in Rust.
//
// Parses lines of the form:
//   LEVEL: message key=value key=value ...
//
// Build:
//   cargo build --target wasm32-unknown-unknown --release
//   cp target/wasm32-unknown-unknown/release/hustle_plugin_example.wasm plugin.wasm

use std::alloc::{alloc as global_alloc, dealloc as global_dealloc, Layout};

const FORMAT_NAME: &str = "example-rust";

/// Returns (ptr, len) of the format name string packed into a u64.
#[unsafe(no_mangle)]
pub extern "C" fn name() -> u64 {
    let ptr = FORMAT_NAME.as_ptr() as u32;
    let len = FORMAT_NAME.len() as u32;
    pack(ptr, len)
}

/// Parses a log line. Returns (ptr, len) of a JSON result packed into a u64.
#[unsafe(no_mangle)]
pub extern "C" fn parse(line_ptr: u32, line_len: u32) -> u64 {
    let line = unsafe {
        let slice = std::slice::from_raw_parts(line_ptr as *const u8, line_len as usize);
        std::str::from_utf8_unchecked(slice)
    };

    let result = parse_line(line);
    let bytes = result.as_bytes();

    let out_ptr = alloc(bytes.len() as u32);
    unsafe {
        std::ptr::copy_nonoverlapping(bytes.as_ptr(), out_ptr as *mut u8, bytes.len());
    }

    pack(out_ptr, bytes.len() as u32)
}

/// Allocates `size` bytes in WASM linear memory.
#[unsafe(no_mangle)]
pub extern "C" fn alloc(size: u32) -> u32 {
    if size == 0 {
        return 0;
    }
    let layout = Layout::from_size_align(size as usize, 1).unwrap();
    let ptr = unsafe { global_alloc(layout) };
    ptr as u32
}

/// Frees memory previously allocated by `alloc`.
#[unsafe(no_mangle)]
pub extern "C" fn dealloc(ptr: u32, size: u32) {
    if ptr == 0 || size == 0 {
        return;
    }
    let layout = Layout::from_size_align(size as usize, 1).unwrap();
    unsafe {
        global_dealloc(ptr as *mut u8, layout);
    }
}

/// Pack two u32 values into a single u64 return value.
/// Low 32 bits = ptr, high 32 bits = len.
fn pack(ptr: u32, len: u32) -> u64 {
    (ptr as u64) | ((len as u64) << 32)
}

fn parse_line(line: &str) -> String {
    // Find ":" separator
    let colon = match line.find(':') {
        Some(i) => i,
        None => return r#"{"ok":false,"error":"expected LEVEL: message"}"#.to_string(),
    };

    let level = &line[..colon];
    if !is_level(level) {
        return r#"{"ok":false,"error":"unknown level"}"#.to_string();
    }

    if colon + 2 > line.len() {
        return r#"{"ok":false,"error":"expected LEVEL: message"}"#.to_string();
    }
    let rest = &line[colon + 2..]; // skip ": "

    let (msg, attrs) = split_msg_attrs(rest);

    let mut json = String::with_capacity(128);
    json.push_str(r#"{"ok":true,"level":""#);
    json.push_str(level);
    json.push_str(r#"","msg":""#);
    json_escape_into(&mut json, msg);
    json.push('"');

    if !attrs.is_empty() {
        json.push_str(r#","attrs":{"#);
        for (i, (k, v)) in attrs.iter().enumerate() {
            if i > 0 {
                json.push(',');
            }
            json.push('"');
            json.push_str(k);
            json.push_str(r#"":""#);
            json_escape_into(&mut json, v);
            json.push('"');
        }
        json.push('}');
    }

    json.push('}');
    json
}

fn is_level(s: &str) -> bool {
    matches!(
        s,
        "TRACE" | "DEBUG" | "INFO" | "WARN" | "WARNING" | "ERROR" | "FATAL" | "PANIC"
    )
}

/// Split "message text key=val key2=val2" into message and attrs.
/// Attrs are contiguous key=value tokens at the end.
fn split_msg_attrs(s: &str) -> (&str, Vec<(&str, &str)>) {
    let tokens: Vec<&str> = s.split_whitespace().collect();

    // Find where attrs start (contiguous key=value tokens from the end)
    let mut attr_start = tokens.len();
    for i in (0..tokens.len()).rev() {
        if tokens[i].contains('=') {
            attr_start = i;
        } else {
            break;
        }
    }

    let attrs: Vec<(&str, &str)> = tokens[attr_start..]
        .iter()
        .filter_map(|token| {
            let eq = token.find('=')?;
            if eq > 0 {
                Some((&token[..eq], &token[eq + 1..]))
            } else {
                None
            }
        })
        .collect();

    // Message is everything before attrs
    let msg = if attr_start == 0 {
        ""
    } else if attr_start >= tokens.len() {
        s
    } else {
        // Find the byte offset where the attr_start-th whitespace-separated token begins
        let mut count = 0;
        let mut end = s.len();
        let mut in_space = false;
        for (i, c) in s.char_indices() {
            if c.is_whitespace() {
                if !in_space {
                    count += 1;
                    if count == attr_start {
                        end = i;
                        break;
                    }
                    in_space = true;
                }
            } else {
                in_space = false;
            }
        }
        s[..end].trim_end()
    };

    (msg, attrs)
}

fn json_escape_into(buf: &mut String, s: &str) {
    for c in s.chars() {
        match c {
            '"' => buf.push_str(r#"\""#),
            '\\' => buf.push_str(r#"\\"#),
            '\n' => buf.push_str(r#"\n"#),
            '\r' => buf.push_str(r#"\r"#),
            '\t' => buf.push_str(r#"\t"#),
            _ => buf.push(c),
        }
    }
}
