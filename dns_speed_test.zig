const std = @import("std");
const net = std.net;
const time = std.time;
const mem = std.mem;

const DnsProvider = struct {
    name: []const u8,
    ip: []const u8,
};

const DNS_PROVIDERS = [_]DnsProvider{
    .{ .name = "Google DNS", .ip = "8.8.8.8" },
    .{ .name = "Google DNS 2", .ip = "8.8.4.4" },
    .{ .name = "Cloudflare", .ip = "1.1.1.1" },
    .{ .name = "Cloudflare 2", .ip = "1.0.0.1" },
    .{ .name = "OpenDNS", .ip = "208.67.222.222" },
    .{ .name = "OpenDNS 2", .ip = "208.67.220.220" },
    .{ .name = "Quad9", .ip = "9.9.9.9" },
    .{ .name = "Quad9 2", .ip = "149.112.112.112" },
    .{ .name = "AdGuard DNS", .ip = "94.140.14.14" },
    .{ .name = "AdGuard DNS 2", .ip = "94.140.15.15" },
    .{ .name = "CleanBrowsing", .ip = "185.228.168.168" },
    .{ .name = "CleanBrowsing 2", .ip = "185.228.169.168" },
    .{ .name = "Level3", .ip = "4.2.2.1" },
    .{ .name = "Level3 2", .ip = "4.2.2.2" },
    .{ .name = "Comodo", .ip = "8.26.56.26" },
    .{ .name = "Comodo 2", .ip = "8.20.247.20" },
};

const TEST_DOMAINS = [_][]const u8{
    "google.com",
    "facebook.com",
    "amazon.com",
    "microsoft.com",
    "github.com",
    "netflix.com",
    "apple.com",
    "youtube.com",
    "twitter.com",
    "instagram.com",
    "linkedin.com",
    "reddit.com",
    "twitch.tv",
    "wikipedia.org",
    "yahoo.com",
};

const NUM_TESTS = 5;  // Increased from 3 to 5 tests per domain
const TIMEOUT_MS = 1000;  // 1 second timeout

const TestResult = struct {
    success_count: u32,
    total_time: u64,
    timeouts: u32,
    failures: u32,
};

pub fn main() !void {
    const stdout = std.io.getStdOut().writer();

    try stdout.writeAll("\n=== DNS Provider Speed Test ===\n\n");
    try stdout.print("Testing {d} domains with {d} iterations each\n", .{ TEST_DOMAINS.len, NUM_TESTS });
    try stdout.print("Timeout set to {d}ms\n\n", .{TIMEOUT_MS});

    for (DNS_PROVIDERS) |provider| {
        var result = TestResult{
            .success_count = 0,
            .total_time = 0,
            .timeouts = 0,
            .failures = 0,
        };

        try stdout.print("\nTesting {s} ({s}):\n", .{ provider.name, provider.ip });
        try stdout.writeAll("----------------------------------------\n");

        for (TEST_DOMAINS) |domain| {
            const domain_result = try testDomain(provider.ip, domain);

            if (domain_result.success_count > 0) {
                const avg_time = domain_result.total_time / domain_result.success_count;
                try stdout.print("{s}: {d}ms", .{ domain, avg_time / 1_000_000 });
                if (domain_result.timeouts > 0) {
                    try stdout.print(" (timeouts: {d})", .{domain_result.timeouts});
                }
                if (domain_result.failures > 0) {
                    try stdout.print(" (failures: {d})", .{domain_result.failures});
                }
                try stdout.writeAll("\n");

                result.total_time += domain_result.total_time;
                result.success_count += domain_result.success_count;
            } else {
                try stdout.print("{s}: Failed", .{domain});
                if (domain_result.timeouts > 0) {
                    try stdout.print(" (timeouts: {d})", .{domain_result.timeouts});
                }
                if (domain_result.failures > 0) {
                    try stdout.print(" (failures: {d})", .{domain_result.failures});
                }
                try stdout.writeAll("\n");
            }
            result.timeouts += domain_result.timeouts;
            result.failures += domain_result.failures;
        }

        if (result.success_count > 0) {
            const avg_total = result.total_time / result.success_count;
            try stdout.print("\nResults for {s}:\n", .{provider.name});
            try stdout.print("  Average response time: {d}ms\n", .{avg_total / 1_000_000});
            try stdout.print("  Success rate: {d}%\n", .{(result.success_count * 100) / (TEST_DOMAINS.len * NUM_TESTS)});
            try stdout.print("  Total timeouts: {d}\n", .{result.timeouts});
            try stdout.print("  Total failures: {d}\n", .{result.failures});
        } else {
            try stdout.writeAll("\nAll queries failed\n");
        }
    }
}

fn testDomain(dns_ip: []const u8, domain: []const u8) !TestResult {
    var result = TestResult{
        .success_count = 0,
        .total_time = 0,
        .timeouts = 0,
        .failures = 0,
    };

    for (0..NUM_TESTS) |_| {
        const start_time = time.nanoTimestamp();
        testDnsServer(dns_ip, domain) catch |err| {
            switch (err) {
                error.ConnectionTimedOut => {
                    result.timeouts += 1;
                    continue;
                },
                else => {
                    result.failures += 1;
                    continue;
                },
            }
        };
        const query_time = @as(u64, @intCast(time.nanoTimestamp() - start_time));
        result.total_time += query_time;
        result.success_count += 1;
    }

    return result;
}

fn testDnsServer(dns_ip: []const u8, domain: []const u8) !void {
    const dns_addr = try net.Address.parseIp4(dns_ip, 53);
    var stream = try net.tcpConnectToAddress(dns_addr);
    defer stream.close();

    // Create a simple DNS query for the domain
    var query_buffer: [512]u8 = undefined;
    var pos: usize = 0;

    // DNS header
    query_buffer[pos] = 0x12; pos += 1; // Transaction ID (random)
    query_buffer[pos] = 0x34; pos += 1;
    query_buffer[pos] = 0x01; pos += 1; // Flags
    query_buffer[pos] = 0x00; pos += 1;
    query_buffer[pos] = 0x00; pos += 1; // Questions
    query_buffer[pos] = 0x01; pos += 1;
    query_buffer[pos] = 0x00; pos += 1; // Answer RRs
    query_buffer[pos] = 0x00; pos += 1;
    query_buffer[pos] = 0x00; pos += 1; // Authority RRs
    query_buffer[pos] = 0x00; pos += 1;
    query_buffer[pos] = 0x00; pos += 1; // Additional RRs
    query_buffer[pos] = 0x00; pos += 1;

    // Encode domain name
    var start: usize = 0;
    for (domain, 0..) |c, i| {
        if (c == '.') {
            const len = i - start;
            query_buffer[pos] = @as(u8, @intCast(len));
            pos += 1;
            @memcpy(query_buffer[pos..pos+len], domain[start..i]);
            pos += len;
            start = i + 1;
        }
    }
    const len = domain.len - start;
    query_buffer[pos] = @as(u8, @intCast(len));
    pos += 1;
    @memcpy(query_buffer[pos..pos+len], domain[start..]);
    pos += len;
    query_buffer[pos] = 0;
    pos += 1;

    // Query type (A record) and class (IN)
    query_buffer[pos] = 0x00; pos += 1;
    query_buffer[pos] = 0x01; pos += 1;
    query_buffer[pos] = 0x00; pos += 1;
    query_buffer[pos] = 0x01; pos += 1;

    // Send query with length prefix (TCP DNS requires 2-byte length prefix)
    const query_len = pos;
    try stream.writer().writeByte(@as(u8, @intCast(query_len >> 8)));
    try stream.writer().writeByte(@as(u8, @intCast(query_len & 0xFF)));
    try stream.writer().writeAll(query_buffer[0..query_len]);

    // Read response length (2 bytes)
    const response_len_high = try stream.reader().readByte();
    const response_len_low = try stream.reader().readByte();
    const response_len = (@as(usize, response_len_high) << 8) | response_len_low;

    // Read response
    var response_buffer: [512]u8 = undefined;
    if (response_len > response_buffer.len) return error.ResponseTooLarge;
    _ = try stream.reader().readAll(response_buffer[0..response_len]);
}
