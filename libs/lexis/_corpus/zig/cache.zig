//! A fixed-capacity LRU cache keyed by string.
const std = @import("std");
const mem = std.mem;
const testing = std.testing;
const Allocator = mem.Allocator;

pub const max_key_len: usize = 64;
const empty: u32 = 0xFFFF_FFFF;

pub const Error = error{ KeyTooLong, OutOfMemory };

pub fn Cache(comptime V: type) type {
    return struct {
        const Self = @This();

        const Slot = struct {
            key: [max_key_len]u8 = undefined,
            key_len: u8 = 0,
            value: V = undefined,
            prev: u32 = empty,
            next: u32 = empty,

            fn keySlice(self: *const Slot) []const u8 {
                return self.key[0..self.key_len];
            }
        };

        allocator: Allocator,
        slots: []Slot,
        used: u32 = 0,
        head: u32 = empty,
        tail: u32 = empty,
        hits: u64 = 0,
        misses: u64 = 0,

        pub fn init(allocator: Allocator, capacity: u32) Error!Self {
            const slots = try allocator.alloc(Slot, capacity);
            for (slots) |*slot| slot.* = .{};
            return .{ .allocator = allocator, .slots = slots };
        }

        pub fn deinit(self: *Self) void {
            self.allocator.free(self.slots);
            self.* = undefined;
        }

        fn find(self: *const Self, key: []const u8) ?u32 {
            var i = self.head;
            while (i != empty) : (i = self.slots[i].next) {
                if (mem.eql(u8, self.slots[i].keySlice(), key)) return i;
            }
            return null;
        }

        /// Unlink a slot from the list and put it at the front.
        fn touch(self: *Self, i: u32) void {
            const slot = &self.slots[i];
            if (self.head == i) return;
            if (slot.prev != empty) self.slots[slot.prev].next = slot.next;
            if (slot.next != empty) self.slots[slot.next].prev = slot.prev;
            if (self.tail == i) self.tail = slot.prev;
            slot.prev = empty;
            slot.next = self.head;
            if (self.head != empty) self.slots[self.head].prev = i;
            self.head = i;
            if (self.tail == empty) self.tail = i;
        }

        pub fn get(self: *Self, key: []const u8) ?V {
            if (self.find(key)) |i| {
                self.hits += 1;
                self.touch(i);
                return self.slots[i].value;
            }
            self.misses += 1;
            return null;
        }

        pub fn put(self: *Self, key: []const u8, value: V) Error!void {
            if (key.len > max_key_len) return error.KeyTooLong;
            const i = self.find(key) orelse blk: {
                if (self.used < self.slots.len) {
                    defer self.used += 1;
                    break :blk self.used;
                }
                break :blk self.tail;
            };
            const slot = &self.slots[i];
            @memcpy(slot.key[0..key.len], key);
            slot.key_len = @intCast(key.len);
            slot.value = value;
            self.touch(i);
        }

        pub fn hitRate(self: Self) f64 {
            const total = self.hits + self.misses;
            if (total == 0) return 0.0;
            return @as(f64, @floatFromInt(self.hits)) / @as(f64, @floatFromInt(total));
        }
    };
}

pub fn main() !void {
    var gpa = std.heap.GeneralPurposeAllocator(.{}){};
    defer _ = gpa.deinit();

    var cache = try Cache(i32).init(gpa.allocator(), 2);
    defer cache.deinit();

    const pairs = [_]struct { []const u8, i32 }{ .{ "one", 1 }, .{ "two", 2 }, .{ "three", 3 } };
    inline for (pairs) |pair| try cache.put(pair[0], pair[1]);

    const stdout = std.io.getStdOut().writer();
    for ([_][]const u8{ "one", "two", "three" }) |key| {
        if (cache.get(key)) |value| {
            try stdout.print("{s} = {d}\n", .{ key, value });
        } else {
            try stdout.print("{s} was evicted\n", .{key});
        }
    }
    try stdout.print("hit rate {d:.2}, tab {c}, {s}\n", .{ cache.hitRate(), '\t', "done \"quoted\"" });

    const text =
        \\multi-line
        \\string literal
    ;
    switch (text.len % 3) {
        0 => {},
        1, 2 => |rest| std.debug.assert(rest < 3),
        else => unreachable,
    }
}

test "the oldest key goes first" {
    var cache = try Cache(u8).init(testing.allocator, 2);
    defer cache.deinit();
    try cache.put("a", 1);
    try cache.put("b", 2);
    _ = cache.get("a");
    try cache.put("c", 3);
    try testing.expectEqual(@as(?u8, null), cache.get("b"));
    try testing.expectError(error.KeyTooLong, cache.put("x" ** 65, 0));
}
