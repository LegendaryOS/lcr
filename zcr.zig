const std = @import("std");
const fs = std.fs;
const mem = std.mem;
const process = std.process;
const Allocator = std.mem.Allocator;

// ANSI color codes for CLI output
const Color = struct {
    const reset = "\x1b[0m";
    const red = "\x1b[31m";
    const green = "\x1b[32m";
    const yellow = "\x1b[33m";
    const blue = "\x1b[34m";
    const cyan = "\x1b[36m";
};

// Log file path
const log_file_path = "/tmp/zcr.logs";

fn logMessage(comptime fmt: []const u8, args: anytype) !void {
    // Try to open the file for writing, create it if it doesn't exist
    var file = fs.cwd().openFile(log_file_path, .{ .mode = .write_only }) catch |err| switch (err) {
        error.FileNotFound => try fs.cwd().createFile(log_file_path, .{ .truncate = false }),
        else => return err,
    };
        defer file.close();

        try file.seekFromEnd(0); // Move to end of file for appending
        const writer = file.writer();
        const timestamp = std.time.milliTimestamp();
        try writer.print("[{}] ", .{timestamp});
        try writer.print(fmt, args);
        try writer.writeByte('\n');
}

pub fn main() !void {
    var arena = std.heap.ArenaAllocator.init(std.heap.page_allocator);
    defer arena.deinit();
    const allocator = arena.allocator();
    try logMessage("Starting zcr execution", .{});
    const args = try process.argsAlloc(allocator);
    defer process.argsFree(allocator, args);

    if (args.len < 2) {
        try printHelp();
        try logMessage("No command provided, showing help", .{});
        return;
    }

    const command = args[1];
    try logMessage("Received command: {s}", .{command});

    if (mem.eql(u8, command, "install")) {
        if (args.len != 3) {
            try std.io.getStdErr().writer().print("{s}Usage: zcr install <package>{s}\n", .{ Color.red, Color.reset });
            try logMessage("Invalid install command: expected 3 arguments, got {d}", .{args.len});
            return;
        }
        try installPackage(allocator, args[2]);
    } else if (mem.eql(u8, command, "find")) {
        if (args.len != 3) {
            try std.io.getStdErr().writer().print("{s}Usage: zcr find <package>{s}\n", .{ Color.red, Color.reset });
            try logMessage("Invalid find command: expected 3 arguments, got {d}", .{args.len});
            return;
        }
        try findPackage(allocator, args[2]);
    } else if (mem.eql(u8, command, "remove")) {
        if (args.len != 3) {
            try std.io.getStdErr().writer().print("{s}Usage: zcr remove <package>{s}\n", .{ Color.red, Color.reset });
            try logMessage("Invalid remove command: expected 3 arguments, got {d}", .{args.len});
            return;
        }
        try removePackage(allocator, args[2]);
    } else if (mem.eql(u8, command, "update")) {
        if (args.len != 3) {
            try std.io.getStdErr().writer().print("{s}Usage: zcr update <package>{s}\n", .{ Color.red, Color.reset });
            try logMessage("Invalid update command: expected 3 arguments, got {d}", .{args.len});
            return;
        }
        try updatePackage(allocator, args[2]);
    } else if (mem.eql(u8, command, "update-all")) {
        try updateAllPackages(allocator);
    } else if (mem.eql(u8, command, "autoremove")) {
        try autoremove();
    } else if (mem.eql(u8, command, "?") or mem.eql(u8, command, "help")) {
        try printHelp();
        try logMessage("Displayed help message", .{});
    } else if (mem.eql(u8, command, "how-to-add")) {
        try printHowToAdd();
        try logMessage("Displayed how-to-add instructions", .{});
    } else {
        try std.io.getStdErr().writer().print("{s}Unknown command: {s}{s}\n", .{ Color.red, command, Color.reset });
        try logMessage("Unknown command: {s}", .{command});
        try printHelp();
    }
}

fn printHelp() !void {
    const help_text =
    \\{s}zcr - Zenit Linux Package Manager{s}
    \\Usage:
    \\ {s}zcr install <package>{s} - Install a package
    \\ {s}zcr find <package>{s} - Search for a package
    \\ {s}zcr remove <package>{s} - Remove a package
    \\ {s}zcr update <package>{s} - Update a specific package
    \\ {s}zcr update-all{s} - Update all installed packages
    \\ {s}zcr autoremove{s} - Remove temporary files and logs
    \\ {s}zcr help{s} - Show this help message
    \\ {s}zcr how-to-add{s} - Instructions for adding new repositories
    \\
    ;
    try std.io.getStdOut().writer().print(help_text, .{ Color.cyan, Color.reset, Color.green, Color.reset, Color.green, Color.reset, Color.green, Color.reset, Color.green, Color.reset, Color.green, Color.reset, Color.green, Color.reset, Color.green, Color.reset, Color.green, Color.reset });
}

fn printHowToAdd() !void {
    const how_to_add_text =
    \\{s}How to Add a Repository to zcr{s}
    \\You can contribute your own repository to zcr by submitting it to:
    \\ - {s}Issues: https://github.com/Zenit-Linux/zcr/issues{s}
    \\ - {s}Discussions: https://github.com/Zenit-Linux/zcr/discussions{s}
    \\Please read the documentation for more details:
    \\ - {s}https://github.com/Zenit-Linux/zcr/blob/main/README.md{s}
    \\
    ;
    try std.io.getStdOut().writer().print(how_to_add_text, .{ Color.cyan, Color.reset, Color.blue, Color.reset, Color.blue, Color.reset, Color.blue, Color.reset });
}

fn fetchRepoList(allocator: Allocator) ![]u8 {
    const repo_url = "https://raw.githubusercontent.com/Zenit-Linux/zcr/main/library/repo-list.zcr";
    try std.io.getStdOut().writer().print("{s}Fetching repository list...{s}\n", .{ Color.yellow, Color.reset });
    try logMessage("Fetching repo list from {s}", .{repo_url});
    var child = std.process.Child.init(&[_][]const u8{ "curl", "-s", repo_url }, allocator);
    child.stdout_behavior = .Pipe;
    try child.spawn();
    var stdout = std.ArrayList(u8).init(allocator);
    defer stdout.deinit();
    try child.stdout.?.reader().readAllArrayList(&stdout, 1024 * 1024);
    const status = try child.wait();
    if (status != .Exited or status.Exited != 0) {
        try std.io.getStdErr().writer().print("{s}Failed to fetch repo list{s}\n", .{ Color.red, Color.reset });
        try logMessage("Failed to fetch repo list, exit status: {}", .{status});
        return error.CurlFailed;
    }
    try fs.cwd().makePath("/tmp");
    try fs.cwd().writeFile(.{ .sub_path = "/tmp/repo-list.zcr", .data = stdout.items });
    try logMessage("Saved repo list to /tmp/repo-list.zcr", .{});
    return try stdout.toOwnedSlice();
}

fn parseRepoList(allocator: Allocator, content: []const u8, package: []const u8) !?[]u8 {
    var lines = mem.splitSequence(u8, content, "\n");
    while (lines.next()) |line| {
        if (line.len == 0 or line[0] == '#') continue;
        var parts = mem.splitSequence(u8, line, " -> ");
        const pkg = parts.next() orelse continue;
        const repo = parts.next() orelse continue;
        if (mem.eql(u8, mem.trim(u8, pkg, " "), package)) {
            const repo_url = try allocator.dupe(u8, mem.trim(u8, repo, " "));
            try logMessage("Found package {s} with repo {s}", .{ package, repo_url });
            return repo_url;
        }
    }
    try logMessage("Package {s} not found in repo list", .{package});
    return null;
}

fn installPackage(allocator: Allocator, package: []const u8) !void {
    try logMessage("Starting installation of package {s}", .{package});
    const repo_list = try fetchRepoList(allocator);
    defer allocator.free(repo_list);
    const repo_url = try parseRepoList(allocator, repo_list, package) orelse {
        try std.io.getStdErr().writer().print("{s}Package '{s}' not found in repository list{s}\n", .{ Color.red, package, Color.reset });
        return;
    };
    defer allocator.free(repo_url);
    try std.io.getStdOut().writer().print("{s}Installing package '{s}' from {s}{s}\n", .{ Color.green, package, repo_url, Color.reset });
    try logMessage("Cloning package {s} from {s}", .{ package, repo_url });
    const install_dir = try std.fmt.allocPrint(allocator, "/usr/lib/zcr/{s}", .{ package });
    defer allocator.free(install_dir);
    try fs.cwd().makePath(install_dir);
    try logMessage("Created install directory {s}", .{install_dir});
    var child = std.process.Child.init(&[_][]const u8{ "git", "clone", repo_url, install_dir }, allocator);
    try child.spawn();
    const status = try child.wait();
    if (status != .Exited or status.Exited != 0) {
        try std.io.getStdErr().writer().print("{s}Failed to clone repository{s}\n", .{ Color.red, Color.reset });
        try logMessage("Failed to clone repository for {s}, exit status: {}", .{ package, status });
        return;
    }
    try logMessage("Successfully cloned {s} to {s}", .{ package, install_dir });
    const unpack_script = try std.fmt.allocPrint(allocator, "{s}/zcr-build-files/unpack.sh", .{ install_dir });
    defer allocator.free(unpack_script);
    var script_exists = true;
    fs.cwd().access(unpack_script, .{ .mode = .read_only }) catch |err| {
        switch (err) {
            error.FileNotFound => {
                script_exists = false;
            },
            else => return err,
        }
    };
    if (!script_exists) {
        try std.io.getStdOut().writer().print("{s}No unpack.sh found, package cloned to {s}{s}\n", .{ Color.yellow, install_dir, Color.reset });
        try logMessage("No unpack.sh found for {s}, package cloned to {s}", .{ package, install_dir });
        return;
    }
    try std.io.getStdOut().writer().print("{s}Executing unpack.sh for {s}{s}\n", .{ Color.yellow, package, Color.reset });
    try logMessage("Executing unpack.sh for {s}", .{package});
    var chmod_child = std.process.Child.init(&[_][]const u8{ "sudo", "chmod", "+x", unpack_script }, allocator);
    try chmod_child.spawn();
    const chmod_status = try chmod_child.wait();
    if (chmod_status != .Exited or chmod_status.Exited != 0) {
        try std.io.getStdErr().writer().print("{s}Failed to make unpack.sh executable{s}\n", .{ Color.red, Color.reset });
        try logMessage("Failed to chmod unpack.sh for {s}, exit status: {}", .{ package, chmod_status });
        return;
    }
    var exec_child = std.process.Child.init(&[_][]const u8{ "sudo", "sh", unpack_script }, allocator);
    try exec_child.spawn();
    const exec_status = try exec_child.wait();
    if (exec_status != .Exited or exec_status.Exited != 0) {
        try std.io.getStdErr().writer().print("{s}Failed to execute unpack.sh{s}\n", .{ Color.red, Color.reset });
        try logMessage("Failed to execute unpack.sh for {s}, exit status: {}", .{ package, exec_status });
    } else {
        try std.io.getStdOut().writer().print("{s}Package '{s}' installed successfully{s}\n", .{ Color.green, package, Color.reset });
        try logMessage("Package {s} installed successfully", .{package});
    }
}

fn findPackage(allocator: Allocator, package: []const u8) !void {
    try logMessage("Searching for package {s}", .{package});
    const repo_list = try fetchRepoList(allocator);
    defer allocator.free(repo_list);
    const repo_url = try parseRepoList(allocator, repo_list, package) orelse {
        try std.io.getStdOut().writer().print("{s}Package '{s}' not found{s}\n", .{ Color.red, package, Color.reset });
        return;
    };
    defer allocator.free(repo_url);
    try std.io.getStdOut().writer().print("{s}Found package '{s}' at {s}{s}\n", .{ Color.green, package, repo_url, Color.reset });
    try logMessage("Found package {s} at {s}", .{ package, repo_url });
}

fn removePackage(allocator: Allocator, package: []const u8) !void {
    try logMessage("Starting removal of package {s}", .{package});
    const install_dir = try std.fmt.allocPrint(allocator, "/usr/lib/zcr/{s}", .{ package });
    defer allocator.free(install_dir);
    const remove_script = try std.fmt.allocPrint(allocator, "{s}/zcr-build-files/remove.sh", .{ install_dir });
    defer allocator.free(remove_script);
    var script_exists = true;
    fs.cwd().access(remove_script, .{ .mode = .read_only }) catch |err| {
        switch (err) {
            error.FileNotFound => {
                script_exists = false;
                try std.io.getStdOut().writer().print("{s}No remove.sh found, proceeding to delete package directory{s}\n", .{ Color.yellow, Color.reset });
                try logMessage("No remove.sh found for {s}, proceeding to delete directory", .{package});
            },
            else => return err,
        }
    };
    if (script_exists) {
        try std.io.getStdOut().writer().print("{s}Executing remove.sh for {s}{s}\n", .{ Color.yellow, package, Color.reset });
        try logMessage("Executing remove.sh for {s}", .{package});
        var exec_child = std.process.Child.init(&[_][]const u8{ "sudo", "sh", remove_script }, allocator);
        try exec_child.spawn();
        const exec_status = try exec_child.wait();
        if (exec_status != .Exited or exec_status.Exited != 0) {
            try std.io.getStdErr().writer().print("{s}Failed to execute remove.sh{s}\n", .{ Color.red, Color.reset });
            try logMessage("Failed to execute remove.sh for {s}, exit status: {}", .{ package, exec_status });
        } else {
            try logMessage("Successfully executed remove.sh for {s}", .{package});
        }
    }
    try fs.cwd().deleteTree(install_dir);
    try std.io.getStdOut().writer().print("{s}Package '{s}' removed successfully{s}\n", .{ Color.green, package, Color.reset });
    try logMessage("Package {s} removed successfully", .{package});
}

fn updatePackage(allocator: Allocator, package: []const u8) !void {
    try logMessage("Starting update of package {s}", .{package});
    const install_dir = try std.fmt.allocPrint(allocator, "/usr/lib/zcr/{s}", .{ package });
    defer allocator.free(install_dir);
    fs.cwd().access(install_dir, .{ .mode = .read_only }) catch |err| switch (err) {
        error.FileNotFound => {
            try std.io.getStdErr().writer().print("{s}Package '{s}' is not installed{s}\n", .{ Color.red, package, Color.reset });
            try logMessage("Package {s} is not installed, cannot update", .{package});
            return;
        },
        else => return err,
    };
        try std.io.getStdOut().writer().print("{s}Updating package '{s}'{s}\n", .{ Color.yellow, package, Color.reset });
        try logMessage("Removing old package directory {s} for update", .{install_dir});
        try fs.cwd().deleteTree(install_dir);
        try installPackage(allocator, package);
}

fn updateAllPackages(allocator: Allocator) !void {
    try logMessage("Starting update-all operation", .{});
    const repo_list = try fetchRepoList(allocator);
    defer allocator.free(repo_list);
    var lines = mem.splitSequence(u8, repo_list, "\n");
    try std.io.getStdOut().writer().print("{s}Updating all packages...{s}\n", .{ Color.yellow, Color.reset });
    while (lines.next()) |line| {
        if (line.len == 0 or line[0] == '#') continue;
        var parts = mem.splitSequence(u8, line, " -> ");
        const pkg = parts.next() orelse continue;
        try updatePackage(allocator, mem.trim(u8, pkg, " "));
    }
    try std.io.getStdOut().writer().print("{s}All packages updated{s}\n", .{ Color.green, Color.reset });
    try logMessage("All packages updated successfully", .{});
}

fn autoremove() !void {
    try std.io.getStdOut().writer().print("{s}Cleaning up temporary files...{s}\n", .{ Color.yellow, Color.reset });
    try logMessage("Starting autoremove operation", .{});
    const temp_files = [_][]const u8{ "/tmp/repo-list.zcr" };
    for (temp_files) |file| {
        fs.cwd().access(file, .{ .mode = .read_only }) catch |err| switch (err) {
            error.FileNotFound => {
                try logMessage("Temporary file {s} not found, skipping", .{file});
                continue;
            },
            else => return err,
        };
            try fs.cwd().deleteFile(file);
            try std.io.getStdOut().writer().print("{s}Removed {s}{s}\n", .{ Color.green, file, Color.reset });
            try logMessage("Removed temporary file {s}", .{file});
    }
    try std.io.getStdOut().writer().print("{s}Cleanup completed{s}\n", .{ Color.green, Color.reset });
    try logMessage("Autoremove completed", .{});
}
