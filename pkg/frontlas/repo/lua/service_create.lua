local service_key = KEYS[1]
local frontier_id = KEYS[2]
local service_alive_key = KEYS[3]
local frontier_key = KEYS[4]

local service_data = ARGV[1]
local service_data_expiration = ARGV[2]
local service_alive = ARGV[3]

-- service side
redis.call("HSET", service_key, frontier_id, service_data)
redis.call("EXPIRE", service_key, tonumber(service_data_expiration))
redis.call("SET", service_alive_key, 1)
redis.call("EXPIRE", service_alive_key, tonumber(service_alive))
-- frontier side
redis.call("HINCRBY", frontier_key, "service_count", 1)
return 0