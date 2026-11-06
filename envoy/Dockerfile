FROM envoyproxy/envoy:v1.40.0
COPY envoy.yaml device_identity.lua server-secret.yaml device-trust-secret.yaml /etc/envoy/
EXPOSE 10000 10001 10002
CMD ["envoy", "-c", "/etc/envoy/envoy.yaml", "--log-level", "info"]
