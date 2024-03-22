local edge_key = KEYS[1]
local edge_alive_key = KEYS[2]
local frontier_key_prefix = KEYS[3]

-- get edge and it's frontier_id
local edge = redis.call("GET", edge_key)
if edge then
    local value = cjson.decode(edge)
    local frontier_id = value['frontier_id']
    if frontier_id then
        -- decrement the edge_count in frontier
        local frontier_key = frontier_key_prefix .. tostring(frontier_id)
        redis.call("HINCRBY", frontier_key, "edge_count", -1)
    end
end

-- remove edge alive
local ret = redis.call("DEL", edge_alive_key)
return ret