local results = {}
for _, key in ipairs(KEYS) do
    local value = redis.call('HGETALL', key)
    results[#results+1] = value
end
-- see this: https://redis.io/docs/interact/programmability/eval-intro/
-- Lua's table arrays are returned as array
return results
