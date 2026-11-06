-- 每个 Frontier 使用独立期限；Redis TIME 避免多个 Frontlas 的时钟偏差。
local clock = redis.call('TIME')
local now = tonumber(clock[1]) * 1000 + math.floor(tonumber(clock[2]) / 1000)
local expired = redis.call('ZRANGEBYSCORE', KEYS[2], '-inf', now)
for _, id in ipairs(expired) do
    redis.call('HDEL', KEYS[1], id)
    redis.call('ZREM', KEYS[2], id)
end
local op = ARGV[1]
local owner = ARGV[2]
local addr = ARGV[3]
local raw = redis.call('HGET', KEYS[1], owner)
local result = 1
if op == 'heartbeat' or op == 'offline' then
    if raw then
        local ok, current = pcall(cjson.decode, raw)
        -- 迁移旧版本的非 JSON Hash 值，仅完整心跳可转换旧值。
        local oldaddr = nil
        if ok and type(current) == 'table' then oldaddr = current.addr
        else oldaddr = string.match(raw, 'addr: (.-), update_time:') end
        if addr ~= '' and oldaddr ~= addr then return -1 end
    elseif op == 'heartbeat' and addr == '' then
        return -2
    end
end
if op == 'online' or op == 'heartbeat' then
    local data = ARGV[4]
    if addr == '' then
        local ok, current = pcall(cjson.decode, raw or '')
        if not ok or type(current) ~= 'table' then return -2 end
        data = raw
    end
    redis.call('HSET', KEYS[1], owner, data)
    redis.call('ZADD', KEYS[2], now + tonumber(ARGV[5]), owner)
    if op == 'online' and not raw then
        redis.call('HINCRBY', KEYS[4], 'service_count', 1)
    end
elseif op == 'offline' then
    local removed = redis.call('HDEL', KEYS[1], owner)
    redis.call('ZREM', KEYS[2], owner)
    if removed > 0 then
        local count = tonumber(redis.call('HGET', KEYS[4], 'service_count') or '0')
        if count > 0 then redis.call('HINCRBY', KEYS[4], 'service_count', -1) end
    end
end
local last = redis.call('ZREVRANGE', KEYS[2], 0, 0, 'WITHSCORES')
if #last == 0 then
    redis.call('DEL', KEYS[1], KEYS[2], KEYS[3])
else
    local ttl = tonumber(last[2]) - now
    redis.call('PEXPIRE', KEYS[1], ttl)
    redis.call('PEXPIRE', KEYS[2], ttl)
    redis.call('SET', KEYS[3], 1, 'PX', ttl)
end
if op == 'read' then
    local values = {}
    -- 旧格式中没有租约的映射不会被当作在线；完整心跳或 Online 可迁移。
    for _, id in ipairs(redis.call('ZRANGE', KEYS[2], 0, -1)) do
        local value = redis.call('HGET', KEYS[1], id)
        if value then table.insert(values, value) end
    end
    return values
end
return result
