-- 校验、补建与续期必须原子完成，不能覆盖已经迁移到其他连接的记录。
local raw = redis.call('GET', KEYS[1])
if raw then
    local current = cjson.decode(raw)
    if current.frontier_id ~= ARGV[1] then
        return -1
    end
    -- 兼容旧 Frontier：未携带地址时只允许续期已有记录。
    if ARGV[2] ~= '' and current.addr ~= ARGV[2] then
        return -1
    end
else
    if ARGV[2] == '' then
        return -2
    end
    redis.call('SET', KEYS[1], ARGV[3])
end
local ttl = tonumber(ARGV[4])
if ttl > 0 then
    redis.call('PEXPIRE', KEYS[1], ttl)
else
    redis.call('PERSIST', KEYS[1])
end
redis.call('SET', KEYS[2], 1, 'PX', ARGV[5])
return 1
