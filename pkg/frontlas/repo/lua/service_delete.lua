local service_key = KEYS[1]
local frontier_id = KEYS[2]
local service_alive_key = KEYS[3]
local frontier_key_prefix = KEYS[4]

-- decrement the frontier_count in frontier
local frontier_key = frontier_key_prefix .. tostring(frontier_id)
redis.call("HINCRBY", frontier_key, "service_count", -1)

-- remove service side frontier
redis.call("HDEL", service_key, frontier_id)

-- remove frontier alive
local ret = redis.call("HLEN", service_key)
if ret ~= 0 then
    return 0
end
-- service offline all frontiers
local ret = redis.call("DEL", service_alive_key)
return ret