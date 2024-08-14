local nk = require("layerg")

--[[
Called when the matchmaker has found a match for some set of users.

Context represents information about the match and server, for information purposes. Format:
{
  env = {}, -- key-value data set in the runtime.env server configuration.
  execution_mode = "Matchmaker",
}

Matchmaker users will contain a table array of the users that have matched together and their properties. Format:
{
  {
    presence: {
      user_id: "user unique ID",
      session_id: "session ID of the user's current connection",
      username: "user's unique username",
      node: "name of the LayerG node the user is connected to"
    },
    properties: {
      foo: "any properties the client set when it started its matchmaking process",
      baz: 1.5
    }
  },
  ...
}

Expected to return an authoritative match ID for a match ready to receive these users, or `nil` if the match should
proceed through the peer-to-peer relayed mode.
--]]
local function matchmaker_matched(context, matchmaker_users)
  if #matchmaker_users ~= 2 then
    return nil
  end

  if matchmaker_users[1].properties["mode"] ~= "authoritative" then
    return nil
  end
  if matchmaker_users[2].properties["mode"] ~= "authoritative" then
    return nil
  end

  return nk.match_create("match", {debug = true, expected_users = matchmaker_users})
end
nk.register_matchmaker_matched(matchmaker_matched)
