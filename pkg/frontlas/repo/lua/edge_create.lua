local edge_key = KEYS[1]
local edge_alive_key = KEYS[2]
local frontier_key = KEYS[3]
local edge = ARGV[1]
local edge_alive_expiration = ARGV[2]

-- set edge
redis.call("SET", edge_key, edge)

-- set edge_alive
redis.call("SET", edge_alive_key, 1)
redis.call("EXPIRE", edge_alive_key, tonumber(edge_alive_expiration))

-- hash increment edge_count
redis.call("HINCRBY", frontier_key, "edge_count", 1)
return 1