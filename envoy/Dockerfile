FROM envoyproxy/envoy:v1.40.0
COPY envoy.yaml /etc/envoy/
EXPOSE 10000
CMD ["envoy", "-c", "/etc/envoy/envoy.yaml", "--log-level", "info"]
