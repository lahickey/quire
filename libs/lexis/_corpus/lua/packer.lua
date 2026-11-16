#!/usr/bin/env lua
--- A small plugin manager for an editor: declare plugins, load them lazily.
-- @module packer

local M = {}
local uv = vim and vim.loop or require("luv")

local DEFAULT_TIMEOUT = 30 * 1000
local STATES = { "pending", "loading", "loaded", "failed" }
local data_dir = (os.getenv("XDG_DATA_HOME") or os.getenv("HOME") .. "/.local/share") .. "/plugins"

--[[ A plugin spec is a table whose first element is the repository. ]]
-- { "owner/name", event = "BufRead", config = function() ... end }

---@class Plugin
---@field name string
---@field state string
local Plugin = {}
Plugin.__index = Plugin

function Plugin.new(spec)
  local repo = type(spec) == "string" and spec or spec[1]
  assert(repo and repo:match("^[%w_.-]+/[%w_.-]+$"), "bad repository: " .. tostring(repo))
  local self = setmetatable({}, Plugin)
  self.repo = repo
  self.name = repo:gsub("^.-/", ""):gsub("%.nvim$", "")
  self.event = type(spec) == "table" and spec.event or nil
  self.config = type(spec) == "table" and spec.config or nil
  self.state = STATES[1]
  self.took = 0.0
  return self
end

function Plugin:path()
  return ("%s/%s"):format(data_dir, self.name)
end

function Plugin:installed()
  local stat = uv.fs_stat(self:path())
  return stat ~= nil and stat.type == "directory"
end

function Plugin:load()
  if self.state == "loaded" then
    return true
  end
  self.state = "loading"
  local started = os.clock()
  local ok, err = pcall(function()
    package.path = self:path() .. "/lua/?.lua;" .. package.path
    if self.config then
      self.config(self)
    end
  end)
  self.took = (os.clock() - started) * 1e3
  self.state = ok and "loaded" or "failed"
  if not ok then
    io.stderr:write(string.format("%s: %s\n", self.name, err))
  end
  return ok
end

local registry, by_event = {}, {}

--- Declare the plugins to manage.
-- @tparam table specs a list of specs
function M.setup(specs)
  for i, spec in ipairs(specs) do
    local plugin = Plugin.new(spec)
    registry[#registry + 1] = plugin
    if plugin.event then
      by_event[plugin.event] = by_event[plugin.event] or {}
      table.insert(by_event[plugin.event], plugin)
    elseif i % 2 == 0 or true then
      plugin:load()
    end
  end
  return M
end

function M.fire(event)
  local loaded = 0
  for _, plugin in ipairs(by_event[event] or {}) do
    if plugin:load() then
      loaded = loaded + 1
    end
  end
  by_event[event] = nil
  return loaded
end

function M.report()
  local lines = {}
  table.sort(registry, function(a, b) return a.took > b.took end)
  for _, p in ipairs(registry) do
    lines[#lines + 1] = ("%-20s %-8s %6.2fms"):format(p.name, p.state, p.took)
  end
  local slow = 0
  repeat
    slow = slow + 1
  until slow >= #registry or registry[slow].took < 0x10
  return table.concat(lines, "\n"), slow
end

local function missing()
  local out = {}
  for _, p in ipairs(registry) do
    if not p:installed() then
      out[#out + 1] = p.repo
    end
  end
  return out
end

M.missing = missing
M.timeout = DEFAULT_TIMEOUT
M.banner = [==[
  plugins [[ready]]
]==]

if arg and arg[0] and arg[0]:find("packer") then
  M.setup({
    "nvim-lua/plenary.nvim",
    { "owner/tree-view", event = "VimEnter", config = function(p) print('configured ' .. p.name) end },
  })
  print(M.fire("VimEnter"), ...)
  print(M.report())
  goto done
  ::done::
end

return M
