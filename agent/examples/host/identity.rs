use fs2::FileExt;
use rcgen::{CertificateParams, DistinguishedName, KeyPair};
use rustls::{
    RootCertStore,
    pki_types::{CertificateDer, PrivatePkcs8KeyDer, UnixTime},
    server::WebPkiClientVerifier,
    sign::CertifiedKey,
};
use serde::{Deserialize, Serialize};
use std::{
    fs::{self, File, OpenOptions},
    io::{self, Write},
    path::Path,
    sync::Arc,
    time::Duration,
};
use ureq::{
    Agent,
    tls::{Certificate, ClientCert, PemItem, PrivateKey, RootCerts, TlsConfig},
};

#[cfg(unix)]
use std::os::unix::fs::{DirBuilderExt, OpenOptionsExt, PermissionsExt};

const KEY_FILE: &str = "device.key";
const CERTIFICATE_FILE: &str = "device.crt";

#[derive(Serialize)]
struct EnrollmentRequest<'a> {
    park_token: &'a str,
    csr_pem: &'a str,
}

#[derive(Deserialize)]
struct EnrollmentResponse {
    certificate_chain_pem: String,
}

/// Load a device identity or enroll once. Neither the token nor the private key
/// is sent to the telemetry endpoint; TLS proves possession of the private key.
pub fn load_or_enroll(
    endpoint: &str,
    directory: &Path,
    ca_pem: &[u8],
    enrollment_token: &str,
) -> io::Result<Agent> {
    let roots = certificates(ca_pem).map_err(|_| {
        io::Error::other(
            "build the host example with SPARKLY_CA_CERT pointing to a valid PEM CA bundle",
        )
    })?;
    let _lock = lock_directory(directory)?;
    match fs::read(directory.join(CERTIFICATE_FILE)) {
        Ok(certificate) => {
            let key = fs::read_to_string(directory.join(KEY_FILE))?;
            return authenticated_agent(&roots, &certificate, &key);
        }
        Err(error) if error.kind() == io::ErrorKind::NotFound => {}
        Err(error) => return Err(error),
    }
    if enrollment_token.is_empty() {
        return Err(io::Error::other(
            "first enrollment requires SPARKLY_ENROLLMENT_TOKEN",
        ));
    }
    let key = load_or_generate_key(directory)?;
    let certificate = enroll(endpoint, &roots, &key, enrollment_token)?;
    let client = authenticated_agent(&roots, &certificate, &key.serialize_pem())?;
    write_certificate(directory, &certificate)?;
    println!(
        "enrolled device; credentials saved in {}",
        directory.display()
    );
    Ok(client)
}

fn enroll(
    endpoint: &str,
    roots: &[Certificate<'static>],
    key: &KeyPair,
    token: &str,
) -> io::Result<Vec<u8>> {
    let mut params = CertificateParams::new(Vec::<String>::new()).map_err(io::Error::other)?;
    // The backend assigns the identity; its CSR contract forbids CN/SANs.
    params.distinguished_name = DistinguishedName::new();
    let csr = params.serialize_request(key).map_err(io::Error::other)?;
    let csr_pem = csr.pem().map_err(io::Error::other)?;
    let mut response = agent(roots, None)
        .post(endpoint)
        .send_json(EnrollmentRequest {
            park_token: token,
            csr_pem: &csr_pem,
        })
        .map_err(io::Error::other)?;
    if response.status() != 201 {
        return Err(io::Error::other(format!(
            "enrollment returned {}; expected 201 Created",
            response.status()
        )));
    }
    let response: EnrollmentResponse = response
        .body_mut()
        .with_config()
        .limit(128 * 1024)
        .read_json()
        .map_err(io::Error::other)?;
    Ok(response.certificate_chain_pem.into_bytes())
}

pub fn require_https(endpoint: &str) -> io::Result<()> {
    let uri = endpoint
        .parse::<ureq::http::Uri>()
        .map_err(io::Error::other)?;
    if uri.scheme_str() != Some("https") || uri.authority().is_none() {
        return Err(io::Error::other(
            "enrollment and ingest URLs must use HTTPS",
        ));
    }
    Ok(())
}

fn agent(roots: &[Certificate<'static>], identity: Option<ClientCert>) -> Agent {
    Agent::config_builder()
        .https_only(true)
        .timeout_global(Some(Duration::from_secs(10)))
        .max_redirects(0)
        .tls_config(
            TlsConfig::builder()
                .root_certs(RootCerts::new_with_certs(roots))
                .client_cert(identity)
                .build(),
        )
        .build()
        .into()
}

fn certificates(pem: &[u8]) -> io::Result<Vec<Certificate<'static>>> {
    let certificates = ureq::tls::parse_pem(pem)
        .map(|item| match item.map_err(io::Error::other)? {
            PemItem::Certificate(certificate) => Ok(certificate),
            _ => Err(io::Error::other("expected a PEM certificate bundle")),
        })
        .collect::<io::Result<Vec<_>>>()?;
    if certificates.is_empty() {
        return Err(io::Error::other("empty PEM certificate bundle"));
    }
    Ok(certificates)
}

fn authenticated_agent(
    roots: &[Certificate<'static>],
    certificate: &[u8],
    key: &str,
) -> io::Result<Agent> {
    let chain = certificates(certificate)?;
    let der_chain = chain
        .iter()
        .map(|c| CertificateDer::from(c.der().to_vec()))
        .collect::<Vec<_>>();
    let mut trust = RootCertStore::empty();
    for root in roots {
        trust
            .add(CertificateDer::from(root.der().to_vec()))
            .map_err(io::Error::other)?;
    }
    let provider = Arc::new(rustls::crypto::ring::default_provider());
    let verifier = WebPkiClientVerifier::builder_with_provider(Arc::new(trust), provider.clone())
        .build()
        .map_err(io::Error::other)?;
    verifier
        .verify_client_cert(&der_chain[0], &der_chain[1..], UnixTime::now())
        .map_err(io::Error::other)?;
    let parsed_key = KeyPair::from_pem(key).map_err(io::Error::other)?;
    CertifiedKey::from_der(
        der_chain,
        PrivatePkcs8KeyDer::from(parsed_key.serialize_der()).into(),
        &provider,
    )
    .map_err(io::Error::other)?;
    let identity = ClientCert::new_with_certs(
        &chain,
        PrivateKey::from_pem(key.as_bytes()).map_err(io::Error::other)?,
    );
    Ok(agent(roots, Some(identity)))
}

fn lock_directory(directory: &Path) -> io::Result<File> {
    let mut builder = fs::DirBuilder::new();
    builder.recursive(true);
    #[cfg(unix)]
    builder.mode(0o700);
    builder.create(directory)?;
    #[cfg(unix)]
    if fs::metadata(directory)?.permissions().mode() & 0o077 != 0 {
        return Err(io::Error::other(
            "device credential directory must have permissions 0700",
        ));
    }
    let mut options = OpenOptions::new();
    options.read(true).write(true).create(true).truncate(false);
    #[cfg(unix)]
    options.mode(0o600);
    let lock = options.open(directory.join("identity.lock"))?;
    lock.try_lock_exclusive()
        .map_err(|_| io::Error::other("another process is using this device identity"))?;
    Ok(lock)
}

fn load_or_generate_key(directory: &Path) -> io::Result<KeyPair> {
    let path = directory.join(KEY_FILE);
    match fs::read_to_string(&path) {
        Ok(key) => KeyPair::from_pem(&key).map_err(io::Error::other),
        Err(error) if error.kind() == io::ErrorKind::NotFound => {
            let key = KeyPair::generate().map_err(io::Error::other)?;
            write_private_file(&path, key.serialize_pem().as_bytes())?;
            Ok(key)
        }
        Err(error) => Err(error),
    }
}

fn write_private_file(path: &Path, contents: &[u8]) -> io::Result<()> {
    let mut options = OpenOptions::new();
    options.write(true).create_new(true);
    #[cfg(unix)]
    options.mode(0o600);
    let mut file = options.open(path)?;
    file.write_all(contents)?;
    file.sync_all()
}

fn write_certificate(directory: &Path, certificate: &[u8]) -> io::Result<()> {
    let temporary = directory.join("device.crt.tmp");
    // A previous process may have stopped during the temporary write.
    match fs::remove_file(&temporary) {
        Ok(()) => {}
        Err(error) if error.kind() == io::ErrorKind::NotFound => {}
        Err(error) => return Err(error),
    }
    write_private_file(&temporary, certificate)?;
    fs::rename(temporary, directory.join(CERTIFICATE_FILE))
}

#[cfg(test)]
mod tests {
    use super::*;
    use rcgen::{BasicConstraints, ExtendedKeyUsagePurpose, IsCa, Issuer, KeyUsagePurpose};

    fn private_directory() -> tempfile::TempDir {
        let directory = tempfile::tempdir().unwrap();
        #[cfg(unix)]
        fs::set_permissions(directory.path(), fs::Permissions::from_mode(0o700)).unwrap();
        directory
    }

    fn authority() -> (rcgen::Certificate, Issuer<'static, KeyPair>) {
        let mut params = CertificateParams::default();
        params.is_ca = IsCa::Ca(BasicConstraints::Unconstrained);
        params.key_usages = vec![KeyUsagePurpose::KeyCertSign, KeyUsagePurpose::CrlSign];
        let key = KeyPair::generate().unwrap();
        let certificate = params.self_signed(&key).unwrap();
        (certificate, Issuer::new(params, key))
    }

    fn device_certificate(key: &KeyPair, issuer: &Issuer<'_, KeyPair>) -> rcgen::Certificate {
        let mut params = CertificateParams::default();
        params.extended_key_usages = vec![ExtendedKeyUsagePurpose::ClientAuth];
        params.signed_by(key, issuer).unwrap()
    }

    #[test]
    fn credentials_are_reused_without_an_enrollment_token_or_connection() {
        let directory = private_directory();
        let (root, issuer) = authority();
        let key = load_or_generate_key(directory.path()).unwrap();
        let certificate = device_certificate(&key, &issuer).pem();
        write_certificate(directory.path(), certificate.as_bytes()).unwrap();
        let before = fs::read(directory.path().join(KEY_FILE)).unwrap();
        for _ in 0..2 {
            load_or_enroll(
                "https://unused.invalid/devices/reg",
                directory.path(),
                root.pem().as_bytes(),
                "",
            )
            .unwrap();
        }
        assert_eq!(before, fs::read(directory.path().join(KEY_FILE)).unwrap());
        assert_eq!(
            certificate,
            fs::read_to_string(directory.path().join(CERTIFICATE_FILE)).unwrap()
        );
        #[cfg(unix)]
        {
            assert_eq!(
                fs::metadata(directory.path().join(KEY_FILE))
                    .unwrap()
                    .permissions()
                    .mode()
                    & 0o777,
                0o600
            );
            assert_eq!(
                fs::metadata(directory.path()).unwrap().permissions().mode() & 0o777,
                0o700
            );
        }
        // Corruption must fail instead of replacing the device identity.
        fs::write(directory.path().join(CERTIFICATE_FILE), "corrupt").unwrap();
        assert!(
            load_or_enroll(
                "https://unused.invalid",
                directory.path(),
                root.pem().as_bytes(),
                "token"
            )
            .is_err()
        );
        assert_eq!(before, fs::read(directory.path().join(KEY_FILE)).unwrap());
    }

    #[test]
    fn device_credentials_require_trust_validity_client_usage_and_matching_key() {
        let (root, issuer) = authority();
        let roots = certificates(root.pem().as_bytes()).unwrap();
        let key = KeyPair::generate().unwrap();
        let certificate = device_certificate(&key, &issuer);
        assert!(
            authenticated_agent(&roots, certificate.pem().as_bytes(), &key.serialize_pem()).is_ok()
        );
        let wrong_key = KeyPair::generate().unwrap();
        assert!(
            authenticated_agent(
                &roots,
                certificate.pem().as_bytes(),
                &wrong_key.serialize_pem()
            )
            .is_err()
        );
        let (foreign_root, _) = authority();
        let foreign_roots = certificates(foreign_root.pem().as_bytes()).unwrap();
        assert!(
            authenticated_agent(
                &foreign_roots,
                certificate.pem().as_bytes(),
                &key.serialize_pem()
            )
            .is_err()
        );
        let mut params = CertificateParams::default();
        params.extended_key_usages = vec![ExtendedKeyUsagePurpose::ServerAuth];
        let wrong_usage = params.signed_by(&key, &issuer).unwrap();
        assert!(
            authenticated_agent(&roots, wrong_usage.pem().as_bytes(), &key.serialize_pem())
                .is_err()
        );
        params.extended_key_usages = vec![ExtendedKeyUsagePurpose::ClientAuth];
        params.not_after = rcgen::date_time_ymd(2020, 1, 1);
        let expired = params.signed_by(&key, &issuer).unwrap();
        assert!(
            authenticated_agent(&roots, expired.pem().as_bytes(), &key.serialize_pem()).is_err()
        );
    }

    #[test]
    fn failed_enrollment_preserves_the_private_key_for_retry() {
        let directory = private_directory();
        let (root, _) = authority();
        // HTTPS-only policy rejects this before any network traffic is sent.
        let mut previous_key = None;
        for _ in 0..2 {
            assert!(
                load_or_enroll(
                    "http://localhost/devices/reg",
                    directory.path(),
                    root.pem().as_bytes(),
                    "token"
                )
                .is_err()
            );
            let current_key = fs::read(directory.path().join(KEY_FILE)).unwrap();
            if let Some(previous) = previous_key {
                assert_eq!(previous, current_key);
            }
            previous_key = Some(current_key);
        }
        let key = fs::read_to_string(directory.path().join(KEY_FILE)).unwrap();
        assert_eq!(
            key,
            load_or_generate_key(directory.path())
                .unwrap()
                .serialize_pem()
        );
        assert!(!directory.path().join(CERTIFICATE_FILE).exists());
        assert!(require_https("http://localhost/ingest").is_err());
        assert!(require_https("https://localhost:10001/ingest").is_ok());
    }
}
