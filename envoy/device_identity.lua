local nil_uuid = "00000000-0000-0000-0000-000000000000"

local function is_uuid(value)
  if value == nil or value == nil_uuid or value ~= string.lower(value) then
    return false
  end
  local a, b, c, d, e = string.match(value, "^(%x+)%-(%x+)%-(%x+)%-(%x+)%-(%x+)$")
  return a ~= nil and #a == 8 and #b == 4 and #c == 4 and #d == 4 and #e == 12
end

local function deny(request)
  request:respond({[":status"] = "403", ["content-type"] = "text/plain"}, "invalid device identity\n")
end

function envoy_on_request(request)
  local headers = request:headers()
  headers:remove("x-device-id")
  headers:remove("x-park-id")
  headers:remove("x-user-id")
  headers:remove("x-forwarded-client-cert")

  local tls = request:streamInfo():downstreamSslConnection()
  if tls == nil or not tls:peerCertificateValidated() then
    deny(request)
    return
  end

  local identities = tls:uriSanPeerCertificate()
  if #identities ~= 1 then
    deny(request)
    return
  end
  local park_id, device_id = string.match(identities[1], "^spiffe://sparkly/parks/([^/]+)/devices/([^/]+)$")
  if not is_uuid(park_id) or not is_uuid(device_id) then
    deny(request)
    return
  end

  headers:replace("x-device-id", device_id)
  headers:replace("x-park-id", park_id)
end
