// Example hustle format plugin in Zig.
//
// Parses lines of the form:
//   LEVEL: message key=value key=value ...
//
// Build:
//   zig build-lib -target wasm32-freestanding -dynamic -O ReleaseSmall plugin.zig

const std = @import("std");

const format_name: []const u8 = "example-zig";
const wasm_alloc = std.heap.wasm_allocator;

fn pack(ptr: [*]const u8, len: usize) u64 {
    return @as(u64, @intFromPtr(ptr)) | (@as(u64, @intCast(len)) << 32);
}

// --- Exported ABI functions ---

export fn name() u64 {
    return pack(format_name.ptr, format_name.len);
}

export fn parse(line_ptr: [*]const u8, line_len: u32) u64 {
    const line = line_ptr[0..line_len];
    var buf: [4096]u8 = undefined;
    const result = parseLine(line, &buf);
    // Copy result to allocated memory
    const out = wasm_alloc.alloc(u8, result.len) catch {
        const err = "{\"ok\":false,\"error\":\"alloc failed\"}";
        return pack(err.ptr, err.len);
    };
    @memcpy(out, result);
    return pack(out.ptr, out.len);
}

export fn alloc(size: u32) usize {
    const slice = wasm_alloc.alloc(u8, size) catch return 0;
    return @intFromPtr(slice.ptr);
}

export fn dealloc(ptr: usize, size: u32) void {
    if (ptr == 0) return;
    const p: [*]u8 = @ptrFromInt(ptr);
    wasm_alloc.free(p[0..size]);
}

// --- JSON buffer writer ---

const JsonBuf = struct {
    buf: []u8,
    pos: usize = 0,

    fn write(self: *JsonBuf, s: []const u8) void {
        const end = @min(self.pos + s.len, self.buf.len);
        const n = end - self.pos;
        @memcpy(self.buf[self.pos..end], s[0..n]);
        self.pos = end;
    }

    fn writeByte(self: *JsonBuf, c: u8) void {
        if (self.pos < self.buf.len) {
            self.buf[self.pos] = c;
            self.pos += 1;
        }
    }

    fn writeEscaped(self: *JsonBuf, s: []const u8) void {
        for (s) |c| {
            switch (c) {
                '"' => self.write("\\\""),
                '\\' => self.write("\\\\"),
                '\n' => self.write("\\n"),
                '\r' => self.write("\\r"),
                '\t' => self.write("\\t"),
                else => self.writeByte(c),
            }
        }
    }

    fn written(self: *const JsonBuf) []const u8 {
        return self.buf[0..self.pos];
    }
};

// --- Parser ---

const levels = [_][]const u8{
    "TRACE", "DEBUG", "INFO", "WARN", "WARNING", "ERROR", "FATAL", "PANIC",
};

fn isLevel(s: []const u8) bool {
    for (levels) |l| {
        if (std.mem.eql(u8, s, l)) return true;
    }
    return false;
}

fn parseLine(line: []const u8, buf: *[4096]u8) []const u8 {
    const colon = std.mem.indexOfScalar(u8, line, ':') orelse {
        return "{\"ok\":false,\"error\":\"expected LEVEL: message\"}";
    };

    const level = line[0..colon];
    if (!isLevel(level)) {
        return "{\"ok\":false,\"error\":\"unknown level\"}";
    }

    if (colon + 2 > line.len) {
        return "{\"ok\":false,\"error\":\"expected LEVEL: message\"}";
    }
    const rest = line[colon + 2 ..];

    // Tokenize
    var tokens: [64][]const u8 = undefined;
    var token_count: usize = 0;
    var iter = std.mem.tokenizeScalar(u8, rest, ' ');
    while (iter.next()) |tok| {
        if (token_count < tokens.len) {
            tokens[token_count] = tok;
            token_count += 1;
        }
    }

    // Find where attrs start (scan backwards for tokens containing '=')
    var attr_start = token_count;
    var i = token_count;
    while (i > 0) {
        i -= 1;
        if (std.mem.indexOfScalar(u8, tokens[i], '=') != null) {
            attr_start = i;
        } else {
            break;
        }
    }

    // Build JSON
    var j = JsonBuf{ .buf = buf };

    j.write("{\"ok\":true,\"level\":\"");
    j.write(level);
    j.write("\",\"msg\":\"");

    for (0..attr_start) |idx| {
        if (idx > 0) j.writeByte(' ');
        j.writeEscaped(tokens[idx]);
    }
    j.writeByte('"');

    if (attr_start < token_count) {
        j.write(",\"attrs\":{");
        var first = true;
        for (attr_start..token_count) |idx| {
            const tok = tokens[idx];
            if (std.mem.indexOfScalar(u8, tok, '=')) |eq| {
                if (eq == 0) continue;
                if (!first) j.writeByte(',');
                first = false;
                j.writeByte('"');
                j.write(tok[0..eq]);
                j.write("\":\"");
                j.writeEscaped(tok[eq + 1 ..]);
                j.writeByte('"');
            }
        }
        j.writeByte('}');
    }

    j.writeByte('}');
    return j.written();
}
