//! Filesystem credentials and rustls enrollment/client adapters for the shared
//! sparkly-core identity lifecycle. No recorder or global logging configuration.
use fs2::FileExt;
use rcgen::{CertificateParams, DistinguishedName, KeyPair};
use rustls::{
    RootCertStore,
    pki_types::{CertificateDer, PrivatePkcs8KeyDer, UnixTime},
    server::WebPkiClientVerifier,
    sign::CertifiedKey,
};
use serde::{Deserialize, Serialize};
use sparkly_core::identity::{CredentialStore, IdentityBackend};
use std::{
    fs::{self, File, OpenOptions},
    io::{self, Write},
    path::{Path, PathBuf},
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

/// Filesystem store holding an exclusive identity lock for its entire lifetime.
pub struct FileIdentityStore {
    directory: PathBuf,
    _lock: File,
}

impl FileIdentityStore {
    pub fn open(directory: impl AsRef<Path>) -> io::Result<Self> {
        let directory = directory.as_ref();
        let lock = lock_directory(directory)?;
        Ok(Self {
            directory: directory.to_owned(),
            _lock: lock,
        })
    }
}

impl CredentialStore for FileIdentityStore {
    type Error = io::Error;
    type Key = KeyPair;
    type Certificate = Vec<u8>;

    fn load_key(&mut self) -> io::Result<Option<KeyPair>> {
        match fs::read_to_string(self.directory.join(KEY_FILE)) {
            Ok(key) => KeyPair::from_pem(&key).map(Some).map_err(io::Error::other),
            Err(error) if error.kind() == io::ErrorKind::NotFound => Ok(None),
            Err(error) => Err(error),
        }
    }

    fn save_key(&mut self, key: &KeyPair) -> io::Result<()> {
        write_private_file(
            &self.directory.join(KEY_FILE),
            key.serialize_pem().as_bytes(),
        )
    }

    fn load_certificate(&mut self) -> io::Result<Option<Vec<u8>>> {
        match fs::read(self.directory.join(CERTIFICATE_FILE)) {
            Ok(certificate) => Ok(Some(certificate)),
            Err(error) if error.kind() == io::ErrorKind::NotFound => Ok(None),
            Err(error) => Err(error),
        }
    }

    fn save_certificate(&mut self, certificate: &Vec<u8>) -> io::Result<()> {
        write_certificate(&self.directory, certificate)
    }
}

/// PEM/PKCS#8 keys, signed CSRs, verified HTTPS enrollment and rustls mTLS clients.
/// Other platforms implement IdentityBackend using their native TLS and signer.
pub struct RustlsIdentity {
    endpoint: String,
    roots: Vec<Certificate<'static>>,
}

impl RustlsIdentity {
    pub fn new(endpoint: &str, ca_pem: &[u8]) -> io::Result<Self> {
        Ok(Self {
            endpoint: endpoint.to_owned(),
            roots: certificates(ca_pem)?,
        })
    }
}

impl IdentityBackend for RustlsIdentity {
    type Error = io::Error;
    type Key = KeyPair;
    type Certificate = Vec<u8>;
    type Csr = String;
    type Client = Agent;

    fn generate_key(&self) -> io::Result<KeyPair> {
        KeyPair::generate().map_err(io::Error::other)
    }

    fn create_csr(&self, key: &KeyPair) -> io::Result<String> {
        let mut params = CertificateParams::new(Vec::<String>::new()).map_err(io::Error::other)?;
        // The backend assigns the identity; its CSR contract forbids CN/SANs.
        params.distinguished_name = DistinguishedName::new();
        params
            .serialize_request(key)
            .map_err(io::Error::other)?
            .pem()
            .map_err(io::Error::other)
    }

    fn enroll(&self, csr: &String, token: &str) -> io::Result<Vec<u8>> {
        enroll(&self.endpoint, &self.roots, csr, token)
    }

    fn authenticated_client(&self, key: &KeyPair, certificate: &Vec<u8>) -> io::Result<Agent> {
        authenticated_agent(&self.roots, certificate, &key.serialize_pem())
    }
}

/// Convenience entry point: hold the filesystem lock through validation and
/// persistence, then return a client. Lifecycle decisions live in no_std code.
pub fn load_or_enroll(
    endpoint: &str,
    directory: &Path,
    ca_pem: &[u8],
    enrollment_token: &str,
) -> io::Result<Agent> {
    let backend = RustlsIdentity::new(endpoint, ca_pem)?;
    let mut store = FileIdentityStore::open(directory)?;
    let identity = sparkly_core::identity::load_or_enroll(&mut store, &backend, enrollment_token)
        .map_err(io::Error::other)?;
    Ok(identity.client)
}

fn enroll(
    endpoint: &str,
    roots: &[Certificate<'static>],
    csr_pem: &str,
    token: &str,
) -> io::Result<Vec<u8>> {
    require_https(endpoint)?;
    let mut response = agent(roots, None)
        .post(endpoint)
        .send_json(EnrollmentRequest {
            park_token: token,
            csr_pem,
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

#[cfg(test)]
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
    fn store_lock_is_held_for_its_lifetime_and_existing_keys_cannot_be_replaced() {
        let directory = private_directory();
        let mut store = FileIdentityStore::open(directory.path()).unwrap();
        assert!(FileIdentityStore::open(directory.path()).is_err());
        let key = KeyPair::generate().unwrap();
        store.save_key(&key).unwrap();
        let before = fs::read(directory.path().join(KEY_FILE)).unwrap();
        assert!(store.save_key(&KeyPair::generate().unwrap()).is_err());
        assert_eq!(before, fs::read(directory.path().join(KEY_FILE)).unwrap());
        drop(store);
        assert!(FileIdentityStore::open(directory.path()).is_ok());
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
