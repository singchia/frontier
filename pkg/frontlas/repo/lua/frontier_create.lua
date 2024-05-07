local frontier_key = KEYS[1]
local frontier_alive_key = KEYS[2]
local sb_addr_key = KEYS[3]
local eb_addr_key = KEYS[4]
local edge_count_key = KEYS[5]
local service_count_key = KEYS[6]

local frontier_alive = ARGV[1]
local sb_addr = ARGV[2]
local eb_addr = ARGV[3]
local edge_count = ARGV[4]
local service_count = ARGV[5]

-- if the frontier is alive, reject it
local ret = redis.call("SETNX", frontier_alive_key, 1)
if ret ~= 1 then
    return 0
end

redis.call("EXPIRE", frontier_alive_key, tonumber(frontier_alive))
redis.call("HSET", frontier_key, sb_addr_key, sb_addr, eb_addr_key, eb_addr, edge_count_key, tonumber(edge_count), service_count_key, tonumber(service_count))
return 1