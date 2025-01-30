const std = @import("std");

pub fn build(b: *std.Build) void {
    const target = b.standardTargetOptions(.{});
    const optimize = b.standardOptimizeOption(.{});

    // Command-line version
    const cli_exe = b.addExecutable(.{
        .name = "dns_speed_test",
        .root_source_file = .{ .cwd_relative = "dns_speed_test.zig" },
        .target = target,
        .optimize = optimize,
    });
    b.installArtifact(cli_exe);

    // GUI version
    const gui_exe = b.addExecutable(.{
        .name = "dns_speed_test_gui",
        .root_source_file = .{ .cwd_relative = "dns_speed_test_gui.zig" },
        .target = target,
        .optimize = optimize,
    });

    // Add SDL2 dependency
    const vcpkg_root = "D:/hacks/zigDnsTest/vcpkg";
    const sdl_include_path = vcpkg_root ++ "/installed/x64-windows/include";
    const sdl_lib_path = vcpkg_root ++ "/installed/x64-windows/lib";

    gui_exe.addIncludePath(.{ .cwd_relative = sdl_include_path });
    gui_exe.addLibraryPath(.{ .cwd_relative = sdl_lib_path });
    
    gui_exe.linkSystemLibrary("SDL2");
    gui_exe.linkSystemLibrary("SDL2_ttf");
    gui_exe.linkSystemLibrary("c");

    // Add Windows-specific libraries
    if (target.result.os.tag == .windows) {
        gui_exe.linkSystemLibrary("gdi32");
        gui_exe.linkSystemLibrary("user32");
        gui_exe.linkSystemLibrary("winmm");
        gui_exe.linkSystemLibrary("imm32");
        gui_exe.linkSystemLibrary("ole32");
        gui_exe.linkSystemLibrary("oleaut32");
        gui_exe.linkSystemLibrary("version");
        gui_exe.linkSystemLibrary("uuid");
        gui_exe.linkSystemLibrary("shell32");
        gui_exe.linkSystemLibrary("setupapi");
    }

    b.installArtifact(gui_exe);

    // Copy SDL2.dll and SDL2_ttf.dll to the output directory
    _ = b.addInstallBinFile(
        .{ .cwd_relative = vcpkg_root ++ "/installed/x64-windows/bin/SDL2.dll" },
        "SDL2.dll",
    );
    _ = b.addInstallBinFile(
        .{ .cwd_relative = vcpkg_root ++ "/installed/x64-windows/bin/SDL2_ttf.dll" },
        "SDL2_ttf.dll",
    );
} 