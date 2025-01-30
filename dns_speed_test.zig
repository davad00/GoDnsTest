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
    .{ .name = "Cloudflare", .ip = "1.1.1.1" },
    .{ .name = "OpenDNS", .ip = "208.67.222.222" },
    .{ .name = "Quad9", .ip = "9.9.9.9" },
    .{ .name = "AdGuard DNS", .ip = "94.140.14.14" },
    .{ .name = "CleanBrowsing", .ip = "185.228.168.168" },
};

const TEST_DOMAINS = [_][]const u8{
    "google.com",
    "facebook.com",
    "amazon.com",
    "microsoft.com",
    "github.com",
};

const NUM_TESTS = 3;

pub fn main() !void {
    const stdout = std.io.getStdOut().writer();

    try stdout.writeAll("\n=== DNS Provider Speed Test ===\n\n");

    for (DNS_PROVIDERS) |provider| {
        var total_time: u64 = 0;
        var success_count: u32 = 0;

        try stdout.print("\nTesting {s} ({s}):\n", .{ provider.name, provider.ip });
        try stdout.writeAll("----------------------------------------\n");

        for (TEST_DOMAINS) |domain| {
            var test_times: u64 = 0;
            var test_successes: u32 = 0;

            for (0..NUM_TESTS) |_| {
                const start_time = time.nanoTimestamp();
                if (try testDnsServer(provider.ip)) {
                    const query_time = @as(u64, @intCast(time.nanoTimestamp() - start_time));
                    test_times += query_time;
                    test_successes += 1;
                }
            }

            if (test_successes > 0) {
                const avg_time = test_times / test_successes;
                try stdout.print("{s}: {d}ms\n", .{ domain, avg_time / 1_000_000 });
                total_time += test_times;
                success_count += test_successes;
            } else {
                try stdout.print("{s}: Failed\n", .{domain});
            }
        }

        if (success_count > 0) {
            const avg_total = total_time / success_count;
            try stdout.print("\nAverage response time: {d}ms\n", .{avg_total / 1_000_000});
        } else {
            try stdout.writeAll("\nAll queries failed\n");
        }
    }
}

fn testDnsServer(dns_ip: []const u8) !bool {
    const dns_addr = try net.Address.parseIp4(dns_ip, 53);
    var stream = try net.tcpConnectToAddress(dns_addr);
    defer stream.close();
    return true;
}
