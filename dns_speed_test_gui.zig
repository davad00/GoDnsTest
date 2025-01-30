const std = @import("std");
const sdl = @cImport({
    @cInclude("SDL2/SDL.h");
    @cInclude("SDL2/SDL_ttf.h");
});

const net = std.net;
const time = std.time;
const mem = std.mem;
const Thread = std.Thread;

const DnsProvider = struct {
    name: []const u8,
    ip: []const u8,
    avg_time: u64,
    success_rate: u32,
    timeouts: u32,
    failures: u32,
    is_testing: bool,
};

const WINDOW_WIDTH = 1200;
const WINDOW_HEIGHT = 800;
const PADDING = 20;
const BAR_HEIGHT = 30;
const MAX_LATENCY = 500; // ms
const NUM_TESTS = 5;
const TIMEOUT_MS = 1000;

var dns_providers = [_]DnsProvider{
    .{ .name = "Google DNS", .ip = "8.8.8.8", .avg_time = 0, .success_rate = 0, .timeouts = 0, .failures = 0, .is_testing = false },
    .{ .name = "Google DNS 2", .ip = "8.8.4.4", .avg_time = 0, .success_rate = 0, .timeouts = 0, .failures = 0, .is_testing = false },
    .{ .name = "Cloudflare", .ip = "1.1.1.1", .avg_time = 0, .success_rate = 0, .timeouts = 0, .failures = 0, .is_testing = false },
    .{ .name = "Cloudflare 2", .ip = "1.0.0.1", .avg_time = 0, .success_rate = 0, .timeouts = 0, .failures = 0, .is_testing = false },
    .{ .name = "Quad9", .ip = "9.9.9.9", .avg_time = 0, .success_rate = 0, .timeouts = 0, .failures = 0, .is_testing = false },
    .{ .name = "Quad9 2", .ip = "149.112.112.112", .avg_time = 0, .success_rate = 0, .timeouts = 0, .failures = 0, .is_testing = false },
};

const TEST_DOMAINS = [_][]const u8{
    "google.com",
    "facebook.com",
    "amazon.com",
    "microsoft.com",
    "github.com",
};

const TestResult = struct {
    success_count: u32,
    total_time: u64,
    timeouts: u32,
    failures: u32,
};

pub fn main() !void {
    if (sdl.SDL_Init(sdl.SDL_INIT_VIDEO) < 0) {
        return error.SDLInitializationFailed;
    }
    defer sdl.SDL_Quit();

    if (sdl.TTF_Init() < 0) {
        return error.TTFInitializationFailed;
    }
    defer sdl.TTF_Quit();

    const font = sdl.TTF_OpenFont("C:/Windows/Fonts/arial.ttf", 16) orelse {
        return error.FontLoadFailed;
    };
    defer sdl.TTF_CloseFont(font);

    const window = sdl.SDL_CreateWindow(
        "DNS Speed Test",
        sdl.SDL_WINDOWPOS_CENTERED,
        sdl.SDL_WINDOWPOS_CENTERED,
        WINDOW_WIDTH,
        WINDOW_HEIGHT,
        sdl.SDL_WINDOW_SHOWN,
    ) orelse {
        return error.SDLWindowCreationFailed;
    };
    defer sdl.SDL_DestroyWindow(window);

    const renderer = sdl.SDL_CreateRenderer(window, -1, sdl.SDL_RENDERER_ACCELERATED) orelse {
        return error.SDLRendererCreationFailed;
    };
    defer sdl.SDL_DestroyRenderer(renderer);

    // Start testing thread
    const thread = try Thread.spawn(.{}, testAllProviders, .{});
    defer thread.join();

    mainLoop(renderer, font) catch |err| {
        std.debug.print("Error in main loop: {}\n", .{err});
    };
}

fn mainLoop(renderer: *sdl.SDL_Renderer, font: *sdl.TTF_Font) !void {
    var quit = false;
    while (!quit) {
        var event: sdl.SDL_Event = undefined;
        while (sdl.SDL_PollEvent(&event) != 0) {
            switch (event.type) {
                sdl.SDL_QUIT => {
                    quit = true;
                },
                else => {},
            }
        }

        try render(renderer, font);
        sdl.SDL_Delay(16); // ~60 FPS
    }
}

fn renderText(renderer: *sdl.SDL_Renderer, font: *sdl.TTF_Font, text: [*c]const u8, x: c_int, y: c_int, color: sdl.SDL_Color) !void {
    const surface = sdl.TTF_RenderText_Blended(font, text, color) orelse {
        return error.TextRenderFailed;
    };
    defer sdl.SDL_FreeSurface(surface);

    const texture = sdl.SDL_CreateTextureFromSurface(renderer, surface) orelse {
        return error.TextureCreationFailed;
    };
    defer sdl.SDL_DestroyTexture(texture);

    var rect = sdl.SDL_Rect{
        .x = x,
        .y = y,
        .w = surface.*.w,
        .h = surface.*.h,
    };

    _ = sdl.SDL_RenderCopy(renderer, texture, null, &rect);
}

fn render(renderer: *sdl.SDL_Renderer, font: *sdl.TTF_Font) !void {
    // Clear screen
    _ = sdl.SDL_SetRenderDrawColor(renderer, 240, 240, 240, 255);
    _ = sdl.SDL_RenderClear(renderer);

    // Draw title
    _ = sdl.SDL_SetRenderDrawColor(renderer, 0, 0, 0, 255);
    const title_y = PADDING;
    var rect = sdl.SDL_Rect{
        .x = PADDING,
        .y = title_y,
        .w = WINDOW_WIDTH - 2 * PADDING,
        .h = BAR_HEIGHT,
    };
    _ = sdl.SDL_RenderFillRect(renderer, &rect);

    // Draw title text
    try renderText(renderer, font, "DNS Speed Test", PADDING + 10, title_y + 5, .{ .r = 255, .g = 255, .b = 255, .a = 255 });

    // Draw provider bars
    var y: i32 = title_y + BAR_HEIGHT + PADDING;
    for (dns_providers) |provider| {
        // Provider name background
        _ = sdl.SDL_SetRenderDrawColor(renderer, 200, 200, 200, 255);
        rect.y = y;
        rect.w = 200;
        _ = sdl.SDL_RenderFillRect(renderer, &rect);

        // Draw provider name
        try renderText(renderer, font, provider.name.ptr, PADDING + 10, y + 5, .{ .r = 0, .g = 0, .b = 0, .a = 255 });

        // Latency bar background
        _ = sdl.SDL_SetRenderDrawColor(renderer, 230, 230, 230, 255);
        rect.x = PADDING + 220;
        rect.w = WINDOW_WIDTH - 2 * PADDING - 220;
        _ = sdl.SDL_RenderFillRect(renderer, &rect);

        if (provider.avg_time > 0) {
            // Latency bar
            const bar_width = @as(i32, @intCast(@min(provider.avg_time / 1_000_000, MAX_LATENCY) * @as(u64, @intCast(rect.w)) / MAX_LATENCY));
            _ = sdl.SDL_SetRenderDrawColor(renderer, 0, 150, 0, 255);
            rect.w = bar_width;
            _ = sdl.SDL_RenderFillRect(renderer, &rect);

            // Draw latency text
            var text_buf: [64]u8 = undefined;
            const text = try std.fmt.bufPrintZ(&text_buf, "{d}ms ({d}% success)", .{ provider.avg_time / 1_000_000, provider.success_rate });
            try renderText(renderer, font, text.ptr, rect.x + bar_width + 10, y + 5, .{ .r = 0, .g = 0, .b = 0, .a = 255 });

            // Testing indicator
            if (provider.is_testing) {
                _ = sdl.SDL_SetRenderDrawColor(renderer, 255, 165, 0, 255);
                const indicator_rect = sdl.SDL_Rect{
                    .x = rect.x + rect.w - 10,
                    .y = rect.y + 5,
                    .w = 5,
                    .h = rect.h - 10,
                };
                _ = sdl.SDL_RenderFillRect(renderer, &indicator_rect);
            }
        } else if (provider.is_testing) {
            // Progress bar for testing
            _ = sdl.SDL_SetRenderDrawColor(renderer, 255, 165, 0, 255);
            rect.w = 100;
            _ = sdl.SDL_RenderFillRect(renderer, &rect);
            try renderText(renderer, font, "Testing...", rect.x + rect.w + 10, y + 5, .{ .r = 0, .g = 0, .b = 0, .a = 255 });
        }

        rect.x = PADDING;
        y += BAR_HEIGHT + 10;
    }

    sdl.SDL_RenderPresent(renderer);
}

fn testAllProviders() !void {
    while (true) {
        for (&dns_providers) |*provider| {
            provider.is_testing = true;
            const result = try testProvider(provider);
            provider.avg_time = if (result.success_count > 0) result.total_time / result.success_count else 0;
            const total_tests = @as(u32, TEST_DOMAINS.len) * NUM_TESTS;
            provider.success_rate = @as(u32, @intCast((result.success_count * 100) / total_tests));
            provider.timeouts = result.timeouts;
            provider.failures = result.failures;
            provider.is_testing = false;
        }
        std.time.sleep(5 * std.time.ns_per_s); // Wait 5 seconds before next round
    }
}

fn testProvider(provider: *DnsProvider) !TestResult {
    var result = TestResult{
        .success_count = 0,
        .total_time = 0,
        .timeouts = 0,
        .failures = 0,
    };

    for (TEST_DOMAINS) |domain| {
        const domain_result = try testDomain(provider.ip, domain);
        result.success_count += domain_result.success_count;
        result.total_time += domain_result.total_time;
        result.timeouts += domain_result.timeouts;
        result.failures += domain_result.failures;
    }

    return result;
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

    var query_buffer: [512]u8 = undefined;
    var pos: usize = 0;

    // DNS header
    query_buffer[pos] = 0x12; pos += 1;
    query_buffer[pos] = 0x34; pos += 1;
    query_buffer[pos] = 0x01; pos += 1;
    query_buffer[pos] = 0x00; pos += 1;
    query_buffer[pos] = 0x00; pos += 1;
    query_buffer[pos] = 0x01; pos += 1;
    query_buffer[pos] = 0x00; pos += 1;
    query_buffer[pos] = 0x00; pos += 1;
    query_buffer[pos] = 0x00; pos += 1;
    query_buffer[pos] = 0x00; pos += 1;
    query_buffer[pos] = 0x00; pos += 1;
    query_buffer[pos] = 0x00; pos += 1;

    // Encode domain name
    var start: usize = 0;
    for (domain, 0..) |char, i| {
        if (char == '.') {
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

    // Query type and class
    query_buffer[pos] = 0x00; pos += 1;
    query_buffer[pos] = 0x01; pos += 1;
    query_buffer[pos] = 0x00; pos += 1;
    query_buffer[pos] = 0x01; pos += 1;

    // Send query
    const query_len = pos;
    try stream.writer().writeByte(@as(u8, @intCast(query_len >> 8)));
    try stream.writer().writeByte(@as(u8, @intCast(query_len & 0xFF)));
    try stream.writer().writeAll(query_buffer[0..query_len]);

    // Read response
    const response_len_high = try stream.reader().readByte();
    const response_len_low = try stream.reader().readByte();
    const response_len = (@as(usize, response_len_high) << 8) | response_len_low;

    var response_buffer: [512]u8 = undefined;
    if (response_len > response_buffer.len) return error.ResponseTooLarge;
    _ = try stream.reader().readAll(response_buffer[0..response_len]);
} 