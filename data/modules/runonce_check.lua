local nk = require("layerg")

--[[
  Test run_once function calls at server startup.
--]]

nk.run_once(function(context)
  assert(context.execution_mode, "run_once")
--   nk.match_create("match", {debug = true, label = "{\"foo\":123}"})
end)

nk.run_once(function(context)
  error("Should not be executed.")
end)

local function rpc_signal(context, payload)
  local matches = nk.match_list(1, true)
  if #matches < 1 then
    error("no matches")
  end
  return nk.match_signal(matches[1].match_id, payload)
end
nk.register_rpc(rpc_signal, "rpc_signal")
