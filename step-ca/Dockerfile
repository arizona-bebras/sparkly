FROM smallstep/step-ca:0.30.2

USER root
# Only this export volume is shared with the backend, never /home/step.
RUN mkdir -p /run/device-pki /run/envoy-pki && chown step:step /run/device-pki /run/envoy-pki
COPY bootstrap.sh /usr/local/bin/sparkly-step-ca
COPY templates/device.tpl templates/envoy-server.tpl /opt/sparkly/
USER step
ENTRYPOINT ["/bin/bash", "/usr/local/bin/sparkly-step-ca"]
