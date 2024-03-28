local frontier_key = KEYS[1]
local frontier_alive_key = KEYS[2]
local frontier_key_prefix = KEYS[3]

-- get frontier and it's frontier_id
local frontier = redis.call("GET", frontier_key)
if frontier then
    local value = cjson.decode(frontier)
    local frontier_id = value['frontier_id']
    if frontier_id then
        -- decrement the frontier_count in frontier
        local frontier_key = frontier_key_prefix .. tostring(frontier_id)
        redis.call("HINCRBY", frontier_key, "frontier_count", -1)
    end
end

-- remove frontier alive
local ret = redis.call("DEL", frontier_alive_key)
return ret