"""Render contracts for routing, certificate rotation, and the gateway trust boundary.

Run with Python 3 + PyYAML and Helm: python3 -m unittest discover -s charts/sparkly/tests -v
"""
import subprocess
import tempfile
import unittest
from pathlib import Path

import yaml

CHART = Path(__file__).resolve().parents[1]
ROOT = CHART.parents[1]


def render(*args):
    result = subprocess.run(
        ["helm", "template", "test", str(CHART), "--namespace", "sparkly", *args],
        text=True, capture_output=True, check=True,
    )
    return [doc for doc in yaml.safe_load_all(result.stdout) if doc]


def by_kind(docs, kind):
    return [doc for doc in docs if doc["kind"] == kind]


class ChartTests(unittest.TestCase):
    @classmethod
    def setUpClass(cls):
        cls.docs = render()

    def test_device_tls_passes_through_to_distinct_listeners(self):
        routes = by_kind(self.docs, "IngressRouteTCP")
        self.assertEqual(len(routes), 2)
        targets = {r["spec"]["routes"][0]["match"]:
                   r["spec"]["routes"][0]["services"][0]["port"] for r in routes}
        self.assertEqual(targets, {"HostSNI(`ingest.devices.sparkly.run`)": 10001,
                                   "HostSNI(`enroll.devices.sparkly.run`)": 10002})
        for r in routes:
            self.assertEqual(r["spec"]["tls"], {"passthrough": True})
        envoy = yaml.safe_load((ROOT / "envoy/envoy.yaml").read_text())
        listeners = envoy["static_resources"]["listeners"]
        ingest_tls = listeners[1]["filter_chains"][0]["transport_socket"]["typed_config"]
        enroll_tls = listeners[2]["filter_chains"][0]["transport_socket"]["typed_config"]
        self.assertTrue(ingest_tls["require_client_certificate"])
        self.assertNotIn("require_client_certificate", enroll_tls)
        self.assertIn("validation_context_sds_secret_config", ingest_tls["common_tls_context"])

    def test_certificates_cover_routes_and_use_step_issuer(self):
        certs = by_kind(self.docs, "Certificate")
        self.assertEqual(len(certs), 2)
        self.assertEqual({h for c in certs for h in c["spec"]["dnsNames"]},
                         {"sparkly.run", "ingest.devices.sparkly.run", "enroll.devices.sparkly.run"})
        for c in certs:
            self.assertEqual(c["spec"]["issuerRef"]["group"], "certmanager.step.sm")
            self.assertEqual(c["spec"]["privateKey"]["rotationPolicy"], "Always")
        ingress = by_kind(self.docs, "Ingress")[0]
        self.assertEqual(ingress["spec"]["tls"][0]["secretName"],
                         next(c["spec"]["secretName"] for c in certs if c["spec"]["dnsNames"] == ["sparkly.run"]))

    def test_rotation_mounts_and_provisioner_isolation(self):
        deployments = by_kind(self.docs, "Deployment")
        for d in deployments:
            pod = d["spec"]["template"]["spec"]
            mounts = pod["containers"][0]["volumeMounts"]
            self.assertFalse(any("subPath" in m for m in mounts))
            volumes = pod["volumes"]
            secrets = [src["secret"]["name"] for v in volumes for src in v.get("projected", {}).get("sources", [])]
            if d["metadata"]["name"].endswith("-envoy"):
                self.assertEqual(set(secrets), {"sparkly-devices-tls", "sparkly-device-trust"})
            elif d["metadata"]["name"].endswith("-backend"):
                self.assertEqual(set(secrets), {"sparkly-device-provisioner", "sparkly-device-trust"})
            else:
                self.assertEqual(secrets, [])

    def test_upstreams_only_accept_gateway(self):
        policies = by_kind(self.docs, "NetworkPolicy")
        self.assertEqual(len(policies), len(by_kind(self.docs, "Deployment")))
        for p in policies:
            component = p["spec"]["podSelector"]["matchLabels"]["app.kubernetes.io/component"]
            source = p["spec"]["ingress"][0]["from"][0]
            if component != "envoy":
                self.assertNotIn("namespaceSelector", source)
                self.assertEqual(source["podSelector"]["matchLabels"]["app.kubernetes.io/component"], "envoy")

    def test_uses_external_services_and_no_embedded_credentials(self):
        self.assertEqual({d["kind"] for d in self.docs},
                         {"Deployment", "Service", "Ingress", "IngressRouteTCP", "Certificate", "NetworkPolicy"})
        for d in by_kind(self.docs, "Deployment"):
            env = d["spec"]["template"]["spec"]["containers"][0]["env"]
            for item in env:
                if item["name"] in {"RABBITMQ_URL", "POSTGRES_PASSWORD", "CLICKHOUSE_PASSWORD", "JWKS_MASTER_KEY"}:
                    self.assertIn("secretKeyRef", item["valueFrom"])

    def test_service_names_match_image_config_independent_of_release_name(self):
        docs = render("--set", "fullnameOverride=custom", "--set", "routing.appHost=example.test",
                      "--set", "certificates.enabled=false", "--set", "networkPolicy.enabled=false",
                      "--set", "components.ml.enabled=true")
        self.assertFalse(by_kind(docs, "Certificate"))
        self.assertFalse(by_kind(docs, "NetworkPolicy"))
        services = {s["metadata"]["name"]: s for s in by_kind(docs, "Service")}
        self.assertEqual(set(services), {"envoy", "frontend", "backend", "analytics", "ml"})
        envoy = yaml.safe_load((ROOT / "envoy/envoy.yaml").read_text())
        for cluster in envoy["static_resources"]["clusters"]:
            endpoint = cluster["load_assignment"]["endpoints"][0]["lb_endpoints"][0]["endpoint"]
            address = endpoint["address"]["socket_address"]
            self.assertIn(address["address"], services)
            self.assertIn(address["port_value"], [p["port"] for p in services[address["address"]]["spec"]["ports"]])
        for service in services.values():
            self.assertTrue(any(
                service["spec"]["selector"] == d["spec"]["template"]["metadata"]["labels"]
                for d in by_kind(docs, "Deployment")))
        ingress_service = by_kind(docs, "Ingress")[0]["spec"]["rules"][0]["http"]["paths"][0]["backend"]["service"]
        self.assertEqual(ingress_service["name"], "envoy")
        for route in by_kind(docs, "IngressRouteTCP"):
            self.assertEqual(route["spec"]["routes"][0]["services"][0]["name"], "envoy")

    def test_packaged_release_selects_matching_images(self):
        with tempfile.TemporaryDirectory() as destination:
            subprocess.run(["helm", "package", str(CHART), "--version", "1.2.3",
                            "--app-version", "1.2.3", "--destination", destination],
                           check=True, capture_output=True, text=True)
            result = subprocess.run(["helm", "template", "release", f"{destination}/sparkly-1.2.3.tgz",
                                     "--set", "components.ml.enabled=true"],
                                    check=True, capture_output=True, text=True)
            docs = [d for d in yaml.safe_load_all(result.stdout) if d]
        for deployment in by_kind(docs, "Deployment"):
            component = deployment["metadata"]["labels"]["app.kubernetes.io/component"]
            self.assertEqual(deployment["spec"]["template"]["spec"]["containers"][0]["image"],
                             f"ghcr.io/arizona-bebras/sparkly/{component}:1.2.3")

    def test_registry_and_component_tag_overrides(self):
        docs = render("--set", "global.imageRegistry=ghcr.io/example/sparkly",
                      "--set-string", "components.backend.image.tag=hotfix")
        for deployment in by_kind(docs, "Deployment"):
            component = deployment["metadata"]["labels"]["app.kubernetes.io/component"]
            tag = "hotfix" if component == "backend" else "0.1.0"
            self.assertEqual(deployment["spec"]["template"]["spec"]["containers"][0]["image"],
                             f"ghcr.io/example/sparkly/{component}:{tag}")

    def test_optional_worker_has_firmware_and_broker(self):
        docs = render("--set", "components.symbolicator.enabled=true", "--set", "symbolicator.firmwareClaimName=firmware")
        worker = next(d for d in by_kind(docs, "Deployment") if d["metadata"]["name"].endswith("-symbolicator"))
        pod = worker["spec"]["template"]["spec"]
        self.assertEqual(pod["volumes"][-1]["persistentVolumeClaim"]["claimName"], "firmware")
        self.assertEqual(pod["containers"][0]["env"][0]["name"], "RABBITMQ_URL")
        self.assertEqual(pod["containers"][0]["args"], ["worker", "/app/fixtures/firmware.elf"])
        self.assertNotIn("readinessProbe", pod["containers"][0])

    def test_rejects_invalid_topology_and_values(self):
        for args in [("routing.enrollHost=sparkly.run",), ("components.envoy.enabled=false",),
                     ("components.symbolicator.enabled=true",), ("components.backend.replicas=0",),
                     ("analytics.rawPersistenceWorkers=0",)]:
            with self.subTest(args=args), self.assertRaises(subprocess.CalledProcessError):
                render("--set", *args)

    def test_gateway_uses_image_configuration(self):
        self.assertFalse(by_kind(self.docs, "ConfigMap"))
        gateway = next(d for d in by_kind(self.docs, "Deployment")
                       if d["metadata"]["name"].endswith("-envoy"))
        container = gateway["spec"]["template"]["spec"]["containers"][0]
        self.assertEqual(container["image"], "ghcr.io/arizona-bebras/sparkly/envoy:0.1.0")
        self.assertNotIn("command", container)
        self.assertNotIn("args", container)
        self.assertFalse(any(m["mountPath"] == "/etc/envoy" for m in container["volumeMounts"]))


if __name__ == "__main__":
    unittest.main()
